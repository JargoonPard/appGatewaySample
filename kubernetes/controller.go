package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type loadBalancerController struct {
	client   *client.Client
	recorder record.EventRecorder

	ingressController *cache.Controller
	ingressStore      cache.Store
	ingressQueue      *taskQueue

	podInfo *podInfo

	stoplock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

var (
	keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// Frequency to poll on local stores to sync.
	storeSyncPollPeriod = 5 * time.Second
)

func newLoadBalancerController(kubeClient *client.Client, namespace string, resyncPeriod time.Duration) (*loadBalancerController, error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(kubeClient.Events(namespace))

	lbc := loadBalancerController{
		client: kubeClient,
		stopCh: make(chan struct{}),
		recorder: eventBroadcaster.NewRecorder(api.EventSource{
			Component: "azure-ingress-controller",
		}),
	}

	lbc.ingressQueue = NewTaskQueue(lbc.updateIngress)

	ingressEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIngress := obj.(*extensions.Ingress)
			if !isAzureIngress(addIngress) {
				glog.Infof("ignoring add for ingress %v based on annotation %v", addIngress.Name, ingressClassKey)
				return
			}
			lbc.recorder.Eventf(addIngress, api.EventTypeNormal, "CREATE", fmt.Sprintf("%s/%s", addIngress.Namespace, addIngress.Name))
			lbc.ingressQueue.enqueue(obj)
		},
	}

	lbc.ingressStore, lbc.ingressController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc:  ingressListFunc(lbc.client, namespace),
			WatchFunc: ingressWatchFunc(lbc.client, namespace),
		},
		&extensions.Ingress{}, resyncPeriod, ingressEventHandler)

	return &lbc, nil
}

func ingressListFunc(kubeClient *client.Client, namespace string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return kubeClient.Extensions().Ingress(namespace).List(opts)
	}
}

func ingressWatchFunc(kubeClient *client.Client, namespace string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return kubeClient.Extensions().Ingress(namespace).Watch(options)
	}
}

func (lbc *loadBalancerController) Stop() error {
	lbc.stoplock.Lock()
	defer lbc.stoplock.Unlock()

	if !lbc.shutdown {
		lbc.shutdown = true
		close(lbc.stopCh)

		glog.Infof("Shutting down controller queues")
		lbc.ingressQueue.shutdown()
	}

	return nil
}

func (lbc *loadBalancerController) updateIngress(key string) error {
	if !lbc.ingressController.HasSynced() {
		time.Sleep(storeSyncPollPeriod)
		return fmt.Errorf("deferring sync till endpoints controller has synced")
	}

	obj, ingressExists, err := lbc.ingressStore.GetByKey(key)
	if err != nil {
		return err
	}

	if !ingressExists {
		// TODO: what's the correct behavior here?
		return nil
	}

	ingress := obj.(*extensions.Ingress)
	glog.Infof("Ingress client retrieved %v", ingress.Name)

	return nil
}

func (lbc *loadBalancerController) Run() {
	glog.Infof("Starting Azure ingress controller")

	go lbc.ingressController.Run(lbc.stopCh)
	go lbc.ingressQueue.run(time.Second, lbc.stopCh)
	<-lbc.stopCh
	glog.Infof("Shutting down Azure ingress controller")
}

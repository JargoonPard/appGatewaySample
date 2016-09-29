package main

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
	ingressLister     StoreToIngressLister

	podInfo *podInfo

	syncQueue    *taskQueue
	ingressQueue *taskQueue

	stoplock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

var (
	keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

func newLoadBalancerController(kubeClient *client.Client, namespace string, resyncPeriod time.Duration) (*loadBalancerController, error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(kubeClient.Events(""))

	lbc := loadBalancerController{
		client: kubeClient,
		recorder: eventBroadcaster.NewRecorder(api.EventSource{
			Component: "azure-ingress-controller",
		}),
	}

	ingressEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIngress := obj.(*extensions.Ingress)
			if !isAzureIngress(addIngress) {
				glog.Infof("ignoring add for ingress %v based on annotation %v", addIngress.Name, ingressClassKey)
				return
			}
			lbc.recorder.Eventf(addIngress, api.EventTypeNormal, "CREATE", fmt.Sprintf("%s/%s", addIngress.Namespace, addIngress.Name))
			lbc.ingressQueue.enqueue(obj)
			lbc.syncQueue.enqueue(obj)
		},
	}

	lbc.ingressLister.Store, lbc.ingressController = cache.NewInformer(
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

		ingress := lbc.ingressLister.Store.List()
		glog.Infof("Removing IP address %v from ingress rules", lbc.podInfo.NodeIP)
		lbc.removeFromIngress(ingress)

		glog.Infof("Shutting down controller queues")
		lbc.syncQueue.shutdown()
		lbc.ingressQueue.shutdown()

		return nil
	}

	return fmt.Errorf("Shutdown already in progress")
}

func (lbc *loadBalancerController) removeFromIngress(ingress []interface{}) {

}

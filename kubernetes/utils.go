package main

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/util/workqueue"
)

type ingressAnnotations map[string]string

const (
	ingressClassKey   = "kubernetes.io/ingress.class"
	azureIngressClass = "azure"
)

func (ingress ingressAnnotations) ingressClass() string {
	val, ok := ingress[ingressClassKey]
	if !ok {
		return ""
	}
	return val
}

// enqueue enqueues ns/name of the given api object in the task queue.
func (t *taskQueue) enqueue(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		glog.Infof("could not get key for object %+v: %v", obj, err)
		return
	}
	t.queue.Add(key)
}

// shutdown shuts down the work queue and waits for the worker to ACK
func (t *taskQueue) shutdown() {
	t.queue.ShutDown()
	<-t.workerDone
}

// taskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type taskQueue struct {
	// queue is the work queue the worker polls
	queue workqueue.RateLimitingInterface
	// sync is called for each item in the queue
	sync func(string) error
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

func isAzureIngress(ingress *extensions.Ingress) bool {
	class := ingressAnnotations(ingress.ObjectMeta.Annotations).ingressClass()
	return class == "" || class == azureIngressClass
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(string) error) *taskQueue {
	return &taskQueue{
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

func (t *taskQueue) run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(t.worker, period, stopCh)
}

// worker processes work in the queue through sync.
func (t *taskQueue) worker() {
	for {
		key, quit := t.queue.Get()
		if quit {
			close(t.workerDone)
			return
		}
		glog.V(3).Infof("syncing %v", key)
		if err := t.sync(key.(string)); err != nil {
			glog.Warningf("requeuing %v, err %v", key, err)
			t.queue.AddRateLimited(key.(string))
		} else {
			t.queue.Forget(key)
		}

		t.queue.Done(key)
	}
}

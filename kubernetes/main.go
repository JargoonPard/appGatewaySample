package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm cloud resources this often.`)

	watchNamespace = flags.String("watch-namespace", api.NamespaceAll,
		`Namespace to watch for Ingress. Default is to watch all namespaces`)

	testNode = flags.Bool("TestNodes", false, "Indicate whether a test run of calling into the cluster to get a list of nodes should be run")

	tenantID       = flags.String("tenantID", "", "Azure tenantId")
	subscriptionID = flags.String("subscriptionID", "", "Azure subscription Id")
	clientID       = flags.String("clientID", "", "Azure client id")
	clientSecret   = flags.String("clientSecret", "", "Azure client secret key")
	region         = flags.String("region", "", "Azure region that hosts the Kubernetes cluster (e.g. westus, southcentralasia, etc.)")
	resourceGroup  = flags.String("resourceGroup", "", "Azure resource group that hosts the Kubernetes cluster")
)

// podInfo contains runtime information about the pod
type podInfo struct {
	PodName      string
	PodNamespace string
	NodeIP       string
}

func main() {
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)

	//work around to issue #17162
	//https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	clientConfig := kubectl_util.DefaultClientConfig(flags)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		glog.Fatalf("error getting the client configuration: %v", err)
	}

	config.Host = "http://localhost:8001"

	kubeclient, err := unversioned.New(config)
	if err != nil {
		glog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	if *testNode {
		nodes := kubeclient.Nodes()
		var opts api.ListOptions

		mynodelist, err := nodes.List(opts)
		if err != nil {
			glog.Fatalf("Failed to get the list of nodes: %v", err)
		}

		glog.Infof("Number of nodes is: %v", mynodelist.Items)
	}

	lbc, err := newLoadBalancerController(kubeclient, *watchNamespace, *resyncPeriod)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	go registerHTTPHandlers(lbc)
	go handleSigterm(lbc)

	lbc.Run()

	for {
		glog.Infof("Handled quit, awaiting pod deletion.")
		time.Sleep(30 * time.Second)
	}
}

func handleSigterm(lbc *loadBalancerController) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := lbc.Stop(); err != nil {
		glog.Infof("Error during shutdown: %v", err)
		exitCode = 1
	}

	glog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}

func registerHTTPHandlers(lbc *loadBalancerController) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		//TODO: add in determination of what defines healthy
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/delete-all-and-quit", func(w http.ResponseWriter, r *http.Request) {
		lbc.Stop()
	})

	//glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *healthzPort), nil))
}

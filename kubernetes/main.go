package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	testNode = flags.Bool("TestNodes", false, "Indicate whether a test run of calling into the cluster to get a list of nodes should be run")
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

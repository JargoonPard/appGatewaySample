package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)
	host  = flags.String("Host", "http://localhost:8001",
		`Service proxy host`)
)

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

	nodes := kubeclient.Nodes()
	var opts api.ListOptions

	mynodelist, err := nodes.List(opts)
	if err != nil {
		glog.Fatalf("Failed to get the list of nodes: %v", err)
	}

	glog.Infof("Number of nodes is: %v", mynodelist.Items)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/kubernetes/pkg/api"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/spf13/pflag"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm cloud resources this often.`)

	watchNamespace = flags.String("watch-namespace", api.NamespaceAll,
		`Namespace to watch for Ingress. Default is to watch all namespaces`)

	tenantID       = flags.String("tenantID", "", "Azure tenantId")
	subscriptionID = flags.String("subscriptionID", "", "Azure subscription Id")
	clientID       = flags.String("clientID", "", "Azure client id")
	clientSecret   = flags.String("clientSecret", "", "Azure client secret key")
	region         = flags.String("region", "", "Azure region that hosts the Kubernetes cluster (e.g. westus, southcentralasia, etc.)")
	resourceGroup  = flags.String("resourceGroup", "", "Azure resource group that hosts the Kubernetes cluster")
)

//this is main
func main() {
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)
	//work around to issue #17162
	//https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	fmt.Printf("TenantID: %s \nclientID: %s \nsecret: %s \nsubscription: %s \nregion: %s \nresourceGroup: %s \n",
		*tenantID, *clientID, *clientSecret, *subscriptionID, *region, *resourceGroup)

	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(*tenantID)
	if err != nil {
		return
	}

	servicePrincipalToken, err := azure.NewServicePrincipalToken(
		*oauthConfig,
		*clientID,
		*clientSecret,
		azure.PublicCloud.ServiceManagementEndpoint)

	if err != nil {
		fmt.Println("Kaboom!")
	} else {
		fmt.Printf("Got a service principal\n")
		fmt.Println(servicePrincipalToken)
	}

	gatewayClient := network.NewApplicationGatewaysClient(*subscriptionID)
	gatewayClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	gatewayClient.Authorizer = servicePrincipalToken

	gatewayList := getGatewayList(gatewayClient)

	for i, k := range *gatewayList {
		fmt.Printf("i is: %b\n", i)
		fmt.Printf("v is: %s\n", *k.Name)
	}

	createPublicIP(*subscriptionID, *resourceGroup, servicePrincipalToken)
}

//GatewayClient interface has been added to support unit testing
type GatewayClient interface {
	ListAll() (network.ApplicationGatewayListResult, error)
}

func getGatewayList(client GatewayClient) *[]network.ApplicationGateway {
	var value *[]network.ApplicationGateway
	gateways, err := client.ListAll()

	if err != nil {
		fmt.Printf("Error!!\n%s", err)
	} else {
		value = gateways.Value
	}

	return value
}

func createPublicIP(subscriptionID, resourceGroup string, servicePrincipalToken autorest.Authorizer) {
	ipClient := network.NewPublicIPAddressesClient(subscriptionID)
	ipClient.Authorizer = servicePrincipalToken

	name := "testPIPCreate"
	location := "westus2"
	props := network.PublicIPAddressPropertiesFormat{}
	params := network.PublicIPAddress{
		Name:       &name,
		Properties: &props,
		Location:   &location,
	}

	result, err := ipClient.CreateOrUpdate(resourceGroup, name, params, nil)

	if err != nil {
		fmt.Printf("[AZURE] Failed to create Public IP: %v", err)
		fmt.Printf("Result: %v\n", result)
	}

	var p []byte
	n, err2 := result.Response.Body.Read(p)

	fmt.Printf("length of response was %v\n", n)
	fmt.Printf("err2 is %v\n", err2)
	fmt.Printf("content of response was %v\n", p)
}

package main

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/go-autorest/autorest/azure"
)

//this is main
func main() {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	region := os.Getenv("AZURE_REGION")
	resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP")

	fmt.Printf("TenantID: %s \nclientID: %s \nsecret: %s \nsubscription: %s \nregion: %s \nresourceGroup: %s \n",
		tenantID, clientID, clientSecret, subscriptionID, region, resourceGroup)

	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(tenantID)
	if err != nil {
		return
	}

	servicePrincipalToken, err := azure.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		clientSecret,
		azure.PublicCloud.ServiceManagementEndpoint)

	if err != nil {
		fmt.Println("Kaboom!")
	} else {
		fmt.Printf("Got a service principal\n")
		fmt.Println(servicePrincipalToken)
	}

	gatewayClient := network.NewApplicationGatewaysClient(subscriptionID)
	gatewayClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	gatewayClient.Authorizer = servicePrincipalToken

	gatewayList := getGatewayList(gatewayClient)

	for i, k := range *gatewayList {
		fmt.Printf("i is: %b\n", i)
		fmt.Printf("v is: %s\n", *k.Name)
	}
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

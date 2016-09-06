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

	fmt.Printf("TenantID: %s \nclientID: %s \nsecret: %s \nsubscription: %s \n", tenantID, clientID, clientSecret, subscriptionID)

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

	result, err := gatewayClient.CheckDNSNameAvailability("westus", "demo")

	if err != nil {
		fmt.Printf("Erorr!!\n %s", err)
	} else {
		fmt.Println("Result:")
		yesno := result.Available
		fmt.Println(yesno)
		fmt.Println(*yesno)
	}
}

package azurecontroller

import (
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

//GatewayClient interface has been added to support unit testing
type GatewayClient interface {
	ListAll() (network.ApplicationGatewayListResult, error)
	Get(resourceGroupName string, applicationGatewayName string) (result network.ApplicationGateway, err error)
}

//AzureCredentialInfo holds credentials and security tokens for Azure
type AzureCredentialInfo struct {
	ResourceGroupName string
	Region            string
	SubscriptionID    string

	ServicePrincipalToken *azure.ServicePrincipalToken
}

//NewAzureGatewayClientController creates an object for interacting with Azure API
func NewAzureGatewayClientController(creds AzureCredentialInfo) *AzureGatewayClientController {
	return &AzureGatewayClientController{
		AzureCredentialInfo: creds,
	}
}

//AzureGatewayClientController handles api calls to Azure
type AzureGatewayClientController struct {
	AzureCredentialInfo
}

//SyncApplicationGateway synchronizes an ingress identifier with the matching Azure ApplicationGateway
func (controller *AzureGatewayClientController) SyncApplicationGateway(ingress *extensions.Ingress) {
	gatewayClient := network.NewApplicationGatewaysClient(controller.SubscriptionID)
	gatewayClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	gatewayClient.Authorizer = controller.ServicePrincipalToken

	gateway, err := gatewayClient.Get(controller.ResourceGroupName, ingress.Name)

	if err != nil {
		detailedError, ok := err.(autorest.DetailedError) //.Original.(*azure.RequestError).ServiceError
		if ok {
			requestError, ok := detailedError.Original.(*azure.RequestError)
			if ok {
				if requestError.ServiceError.Code == "ResourceNotFound" {
					glog.Infof("%v Attempting to create.", requestError.ServiceError.Message)
					//TODO: go create a new one
				}
			}
		} else {
			glog.Errorf("Failure retrieving the gateway %v in the resource group %v: %v", ingress.Name, controller.ResourceGroupName, err)
		}
	} else {
		//TODO: No errors therefore validate the configuration
		glog.Infof("Validating %v settings", gateway)
	}
}

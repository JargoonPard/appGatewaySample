package main

import (
	"testing/quick"

	"github.com/Azure/azure-sdk-for-go/arm/network"
)

type testGatewayClientPass struct{}
type testGatewayClientFail struct{}

func (t testGatewayClientFail) ListAll() (network.ApplicationGatewayListResult, error) {
	//var err quick.SetupError
	var err quick.SetupError = "error"
	var result network.ApplicationGatewayListResult

	return result, err
}

var _ GatewayClient = (*testGatewayClientFail)(nil)

func ExampleGatewayListGet() {
	client := testGatewayClientFail{}
	getGatewayList(client)
	//Output:
	//Error!!
	//error
}

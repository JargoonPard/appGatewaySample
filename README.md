# appGatewaySample

This is a simple sample that I am using as a learning exercise for manipulating Azure ApplicationGateways using Go and the Go SDK for Azure.

The goal is to use this as a stepping stone to adding a proper Azure Ingress Controller to Kubernetes

To run this sample you can either configure actual environment variables on your machine or if you are using vs code you can configure them in the launch.json file. Like so: 

 {
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "remotePath": "",
            "port": 2345,
            "host": "127.0.0.1",
            "program": "${workspaceRoot}",
            "env": {
                "AZURE_SUBSCRIPTION_ID": "1234-5678-9012"
            },
            "args": [],
            "showLog": true
        }
    ]
}

The required environment variables are:
    AZURE_TENANT_ID
    AZURE_CLIENT_ID
	AZURE_CLIENT_SECRET
	AZURE_SUBSCRIPTION_ID
    AZURE_REGION
    AZURE_RESOURCE_GROUP

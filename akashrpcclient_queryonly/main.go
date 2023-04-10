package main

import (
	"akashrpcclient/client"
	"context"
	"fmt"
	"log"

	types "github.com/akash-network/node/x/deployment/types/v1beta2"
)

func main() {
	ctx := context.Background()
	addressPrefix := "akash"

	// Create a Cosmos client instance
	client, err := client.New(ctx, client.WithAddressPrefix(addressPrefix))
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate a query client
	queryClient := types.NewQueryClient(client.Context())

	// // Query the blockchain using the client's `Deployments` method for a return of all deployments
	// queryResp, err := queryClient.Deployments(ctx, &types.QueryDeploymentsRequest{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Print response from querying all deployments on the blockchain
	// fmt.Print("\n\nAll Akash deployments on the blockchain:\n\n")
	// fmt.Println(queryResp)

	// Query the blockchain using the client's `Deployment` method for a return of a specific deployment
	deploymentid := types.DeploymentID{
		Owner: "akash1f53fp8kk470f7k26yr5gztd9npzpczqv4ufud7",
		DSeq:  10219997,
	}

	querydeploymentrequest := types.QueryDeploymentRequest{
		ID: deploymentid,
	}

	// Query the blockchain using the client's `Deployment` method
	queryResp, err := queryClient.Deployment(ctx, &querydeploymentrequest)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from querying a specific deployment on the blockchain
	fmt.Print("\n\nSpeciifc Akash deployment on the blockchain :\n\n")
	fmt.Println(queryResp)

}

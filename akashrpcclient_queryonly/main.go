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

	// fmt.Printf("%+v", client)
	// fmt.Println("RPC: ", client.RPC)

	// Instantiate a query client
	queryClient := types.NewQueryClient(client.Context())

	// // Query the blockchain using the client's `Deployments` method
	// queryResp, err := queryClient.Deployments(ctx, &types.QueryDeploymentsRequest{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// type DeploymentID struct {
	// 	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner" yaml:"owner"`
	// 	DSeq  uint64 `protobuf:"varint,2,opt,name=dseq,proto3" json:"dseq" yaml:"dseq"`
	// }

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

	// Print response from querying all the posts
	fmt.Print("\n\nAll posts:\n\n")
	fmt.Println(queryResp)

}

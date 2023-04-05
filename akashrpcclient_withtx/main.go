package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/akash-network/node/sdl"

	"github.com/akash-network/node/x/deployment/types/v1beta2"

	"akashrpcclient/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"akashrpcclient/client"
)

func main() {

	// Start of Msg Create logic
	// Logic derived from code initiated via: https://github.com/akash-network/node/blob/52d5ee5caa2c6e5a5e59893d903d22fe450d6045/x/deployment/client/cli/tx.go#L83

	sdlLocation := ("./testsdl/deploy.yml")
	accountPrefix := "akash"
	bech32Address := "akash1w3k6qpr4uz44py4z68chfrl7ltpxwtkngnc6xk"

	sdlManifest, err := sdl.ReadFile(sdlLocation)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("SDL Manifest: %+v\n", sdlManifest)

	groups, err := sdlManifest.DeploymentGroups()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println("Groups: ", groups)

	id := v1beta2.DeploymentID{}

	ownerBytes, err := sdk.GetFromBech32(bech32Address, accountPrefix)
	if err != nil {
		fmt.Println("Error from GetFromBech32: ", err)
	}

	accOwnerBytes := sdk.AccAddress(ownerBytes)

	id.Owner = utils.String(accOwnerBytes)

	fmt.Println("ID owner: ", id.Owner)

	id.DSeq, err = strconv.ParseUint(utils.BlockHeight(), 10, 64)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println("ID:  ", id)

	version, err := sdl.Version(sdlManifest)
	if err != nil {
		fmt.Println("Error from Version: ", err)
	}

	fmt.Println("Version: ", string(version))

	deposit := "5000000uakt"
	depositCoin, err := sdk.ParseCoinNormalized(deposit)
	if err != nil {
		fmt.Println("Error ParseCoinNormalized: ", err)
	}

	//depositorAcc := "akash1f53fp8kk470f7k26yr5gztd9npzpczqv4ufud7"

	msg := &v1beta2.MsgCreateDeployment{
		ID:      id,
		Version: version,
		Groups:  make([]v1beta2.GroupSpec, 0, len(groups)),
		Deposit: depositCoin,
		// Depositor: depositorAcc,
		Depositor: id.Owner,
	}

	for _, group := range groups {
		msg.Groups = append(msg.Groups, *group)
	}

	fmt.Println("Msg: ", msg)

	if err := msg.ValidateBasic(); err != nil {
		fmt.Println("Error from ValidateBasic: ", err)
	}

	fmt.Println("PAST ValidateBasic check")

	///////TX OPERATIONS/////////

	// Account `mywallet` was available in local OS keychain from test machine
	accountName := "chainzero"
	// accountAddress := "akash1w3k6qpr4uz44py4z68chfrl7ltpxwtkngnc6xk"

	ctx := context.Background()
	addressPrefix := "akash"

	// Create a Cosmos client instance
	client, err := client.New(ctx, client.WithAddressPrefix(addressPrefix))
	if err != nil {
		log.Fatal(err)
	}

	// Get account from the keyring
	account, err := client.Account(accountName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Account: ", account)

	addr, err := account.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("address: ", addr)

	// Broadcast a transaction from account with the message
	// to create a post store response in txResp
	txResp, err := client.BroadcastTx(ctx, account, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("MsgCreatePost:\n\n")
	fmt.Println(txResp)

	//////QUERY OPERATIONS//////////

	// // Instantiate a query client
	// queryClient := types.NewQueryClient(client.Context())

	// // Query the blockchain using the client's `Deployments` method
	// queryResp, err := queryClient.Deployments(ctx, &types.QueryDeploymentsRequest{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// type DeploymentID struct {
	// 	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner" yaml:"owner"`
	// 	DSeq  uint64 `protobuf:"varint,2,opt,name=dseq,proto3" json:"dseq" yaml:"dseq"`
	// }

	// 	deploymentid := types.DeploymentID{
	// 		Owner: "akash1f53fp8kk470f7k26yr5gztd9npzpczqv4ufud7",
	// 		DSeq:  10219997,
	// 	}

	// 	querydeploymentrequest := types.QueryDeploymentRequest{
	// 		ID: deploymentid,
	// 	}

	// 	// Query the blockchain using the client's `Deployment` method
	// 	queryResp, err := queryClient.Deployment(ctx, &querydeploymentrequest)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	// Print response from querying all the posts
	// 	fmt.Print("\n\nAll posts:\n\n")
	// 	fmt.Println(queryResp)

}

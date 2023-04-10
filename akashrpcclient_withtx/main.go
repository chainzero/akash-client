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

	// Replace the directory specified with the location of SDL associated with the current deployment
	sdlLocation := ("./testsdl/deploy.yml")
	accountPrefix := "akash"

	sdlManifest, err := sdl.ReadFile(sdlLocation)
	if err != nil {
		fmt.Println(err)
	}

	id := v1beta2.DeploymentID{}

	// Replace the Akash address below with the address that should be used for deployment creation
	bech32Address := "akash1w3k6qpr4uz44py4z68chfrl7ltpxwtkngnc6xk"

	ownerBytes, err := sdk.GetFromBech32(bech32Address, accountPrefix)
	if err != nil {
		fmt.Println("Error from GetFromBech32: ", err)
	}

	accOwnerBytes := sdk.AccAddress(ownerBytes)

	id.Owner = utils.String(accOwnerBytes)

	id.DSeq, err = strconv.ParseUint(utils.BlockHeight(), 10, 64)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	version, err := sdl.Version(sdlManifest)
	if err != nil {
		fmt.Println("Error from Version: ", err)
	}

	deposit := "5000000uakt"
	depositCoin, err := sdk.ParseCoinNormalized(deposit)
	if err != nil {
		fmt.Println("Error ParseCoinNormalized: ", err)
	}

	groups, err := sdlManifest.DeploymentGroups()
	if err != nil {
		fmt.Println("Error: ", err)
	}

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

	if err := msg.ValidateBasic(); err != nil {
		fmt.Println("Error from ValidateBasic: ", err)
	}

	///////TX OPERATIONS/////////

	// Account `chainzero` was available in local OS keychain from test machine
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

	// addr, err := account.Address(addressPrefix)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Broadcast a transaction from account with the message
	// to create a post store response in txResp
	txResp, err := client.BroadcastTx(ctx, account, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("Transaction broadcast result:\n\n")
	fmt.Println(txResp)

}

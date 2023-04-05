package client

import (
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

type rpcWrapper struct {
	rpcclient.Client
	nodeAddress string
}

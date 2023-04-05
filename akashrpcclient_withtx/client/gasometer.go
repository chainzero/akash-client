package client

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	gogogrpc "github.com/gogo/protobuf/grpc"
)

// gasometer implements the Gasometer interface.
type gasometer struct{}

func (gasometer) CalculateGas(clientCtx gogogrpc.ClientConn, txf tx.Factory, msgs ...sdktypes.Msg) (*txtypes.SimulateResponse, uint64, error) {
	fmt.Println("within CalculateGas in gasometer")
	resp, _, _ := tx.CalculateGas(clientCtx, txf, msgs...)
	fmt.Println("Response from CalculateGas: ", resp)
	return tx.CalculateGas(clientCtx, txf, msgs...)
}

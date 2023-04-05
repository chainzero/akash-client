package client

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"akashrpcclient/account"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	gogogrpc "github.com/gogo/protobuf/grpc"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	// GasAuto allows to calculate gas automatically when sending transaction.
	GasAuto = "auto"

	defaultNodeAddress   = "https://akash-rpc.polkachu.com:443"
	defaultGasAdjustment = 2.0
	defaultGasPrice      = "0.025uakt"
	// defaultGasLimit      = 300000

	defaultTXsPerPage = 30

	searchHeight = "tx.height"

	orderAsc = "asc"
)

type Gasometer interface {
	CalculateGas(clientCtx gogogrpc.ClientConn, txf tx.Factory, msgs ...sdktypes.Msg) (*txtypes.SimulateResponse, uint64, error)
}

type Signer interface {
	Sign(txf tx.Factory, name string, txBuilder client.TxBuilder, overwriteSig bool) error
}

// Option configures your client.
type Option func(*Client)

// New creates a new client with given options.
func New(ctx context.Context, options ...Option) (Client, error) {
	c := Client{
		nodeAddress:    defaultNodeAddress,
		keyringBackend: account.KeyringTest,
		addressPrefix:  "akash",
		out:            io.Discard,
		gas:            "auto",
	}

	var err error

	for _, apply := range options {
		apply(&c)
	}

	if c.RPC == nil {
		if c.RPC, err = rpchttp.New(c.nodeAddress, "/websocket"); err != nil {
			return Client{}, err
		}
	}
	// Wrap RPC client to have more contextualized errors
	c.RPC = rpcWrapper{
		Client:      c.RPC,
		nodeAddress: c.nodeAddress,
	}

	statusResp, err := c.RPC.Status(ctx)
	if err != nil {
		return Client{}, err
	}

	c.chainID = statusResp.NodeInfo.Network

	if c.homePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Client{}, err
		}
		c.homePath = filepath.Join(home, "."+c.chainID)
	}

	if c.keyringDir == "" {
		c.keyringDir = c.homePath
	}

	c.AccountRegistry, err = account.New(
		account.WithKeyringServiceName(c.keyringServiceName),
		account.WithKeyringBackend(c.keyringBackend),
		account.WithHome(c.keyringDir),
	)
	if err != nil {
		return Client{}, err
	}

	c.context = c.newContext()
	c.TxFactory = newFactory(c.context)

	if c.accountRetriever == nil {
		c.accountRetriever = authtypes.AccountRetriever{}
	}
	if c.bankQueryClient == nil {
		c.bankQueryClient = banktypes.NewQueryClient(c.context)
	}
	if c.gasometer == nil {
		c.gasometer = gasometer{}
	}
	if c.signer == nil {
		c.signer = signer{}
	}
	// set address prefix in SDK global config
	c.SetConfigAddressPrefix()

	return c, nil
}

type Client struct {
	// RPC is Tendermint RPC.
	RPC rpcclient.Client

	// TxFactory is a Cosmos SDK tx factory.
	TxFactory tx.Factory

	// context is a Cosmos SDK client context.
	context client.Context

	// AccountRegistry is the registry to access accounts.
	AccountRegistry account.Registry

	accountRetriever client.AccountRetriever
	bankQueryClient  banktypes.QueryClient
	gasometer        Gasometer
	signer           Signer

	addressPrefix string

	nodeAddress string
	out         io.Writer
	chainID     string

	homePath           string
	keyringServiceName string
	keyringBackend     account.KeyringBackend
	keyringDir         string

	gas           string
	gasPrices     string
	gasAdjustment float64
	fees          string
	generateOnly  bool
}

func WithAddressPrefix(prefix string) Option {
	return func(c *Client) {
		c.addressPrefix = prefix
	}
}

func (c Client) newContext() client.Context {
	var (
		amino             = codec.NewLegacyAmino()
		interfaceRegistry = codectypes.NewInterfaceRegistry()
		marshaler         = codec.NewProtoCodec(interfaceRegistry)
		txConfig          = authtx.NewTxConfig(marshaler, authtx.DefaultSignModes)
	)

	authtypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	sdktypes.RegisterInterfaces(interfaceRegistry)
	staking.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)

	return client.Context{}.
		WithChainID(c.chainID).
		WithInterfaceRegistry(interfaceRegistry).
		WithCodec(marshaler).
		WithTxConfig(txConfig).
		WithLegacyAmino(amino).
		WithInput(os.Stdin).
		WithOutput(c.out).
		WithAccountRetriever(c.accountRetriever).
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir(c.homePath).
		WithClient(c.RPC).
		WithSkipConfirmation(true).
		WithKeyring(c.AccountRegistry.Keyring).
		WithGenerateOnly(c.generateOnly)
}

func newFactory(clientCtx client.Context) tx.Factory {
	return tx.Factory{}.
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		// WithGas(defaultGasLimit).
		WithGasAdjustment(defaultGasAdjustment).
		WithGasPrices(defaultGasPrice).
		WithSignMode(signing.SignMode_SIGN_MODE_UNSPECIFIED).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithTxConfig(clientCtx.TxConfig)
}

// protects sdktypes.Config.
var mconf sync.Mutex

// SetConfigAddressPrefix sets the account prefix in the SDK global config.
func (c Client) SetConfigAddressPrefix() {
	mconf.Lock()
	defer mconf.Unlock()
	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount(c.addressPrefix, c.addressPrefix+"pub")
}

func (c Client) Context() client.Context {
	return c.context
}

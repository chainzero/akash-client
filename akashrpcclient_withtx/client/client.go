package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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
	"github.com/pkg/errors"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
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

	bankQueryClient banktypes.QueryClient
	gasometer       Gasometer
	signer          Signer

	addressPrefix string

	nodeAddress string
	out         io.Writer
	chainID     string

	useFaucet       bool
	faucetAddress   string
	faucetDenom     string
	faucetMinAmount uint64

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

// New creates a new client with given options.
func New(ctx context.Context, options ...Option) (Client, error) {
	c := Client{
		nodeAddress:    defaultNodeAddress,
		keyringBackend: account.KeyringOS,
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

	fmt.Println("account.WithKeyringBackend(c.keyringBackend): ", account.WithKeyringBackend(c.keyringBackend))
	fmt.Println("account.WithHome(c.keyringDir) ", account.WithHome(c.keyringDir))

	c.AccountRegistry, err = account.New(
		account.WithKeyringServiceName(c.keyringServiceName),
		account.WithKeyringBackend(c.keyringBackend),
		account.WithHome(c.keyringDir),
	)
	fmt.Println("Client after account new")
	fmt.Printf("%+v", c)
	fmt.Println("c.AccountRegistry after account new: ", c.AccountRegistry)

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

func (c Client) Account(nameOrAddress string) (account.Account, error) {
	defer c.lockBech32Prefix()()

	acc, err := c.AccountRegistry.GetByName(nameOrAddress)
	if err == nil {
		return acc, nil
	}
	return c.AccountRegistry.GetByAddress(nameOrAddress)
}

func (c Client) lockBech32Prefix() (unlockFn func()) {
	mconf.Lock()
	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount(c.addressPrefix, c.addressPrefix+"pub")
	return mconf.Unlock
}

func (c Client) BroadcastTx(ctx context.Context, account account.Account, msgs ...sdktypes.Msg) (Response, error) {
	txService, err := c.CreateTx(ctx, account, msgs...)
	if err != nil {
		return Response{}, err
	}

	return txService.Broadcast(ctx)
}

type Response struct {
	Codec codec.Codec

	// TxResponse is the underlying tx response.
	*sdktypes.TxResponse
}

func (c Client) CreateTx(goCtx context.Context, account account.Account, msgs ...sdktypes.Msg) (TxService, error) {
	fmt.Println("Within CreateTx")
	fmt.Println("Account within CreateTx: ", account)
	defer c.lockBech32Prefix()()

	if !c.generateOnly {
		addr, err := account.Address(c.addressPrefix)
		fmt.Println("addr within CreateTX: ", addr)
		if err != nil {
			return TxService{}, errors.WithStack(err)
		}
		if err := c.makeSureAccountHasTokens(goCtx, addr); err != nil {
			return TxService{}, err
		}
	}

	sdkaddr := account.Info.GetAddress()

	fmt.Println("within createtx account.Name: ", account.Name)
	fmt.Println("within createtx sdkaddr: ", sdkaddr)

	ctx := c.context.
		WithFromName(account.Name).
		WithFromAddress(sdkaddr)

	txf, err := c.prepareFactory(ctx)
	if err != nil {
		return TxService{}, err
	}

	fmt.Println("past c.prepareFactory(ctx)")
	fmt.Println("TXF after prepare factory: ", txf)

	var gas uint64
	if c.gas != "" && c.gas != GasAuto {
		fmt.Println("within c.gas conditional")
		gas, err = strconv.ParseUint(c.gas, 10, 64)
		if err != nil {
			fmt.Println("Error in ParseUint: ", err)
			return TxService{}, errors.WithStack(err)
		}
	} else {
		fmt.Println("within c.gas ELSE conditional")
		_, gas, err = c.gasometer.CalculateGas(ctx, txf, msgs...)
		if err != nil {
			fmt.Println("Error in CalculateGas: ", err)
			return TxService{}, errors.WithStack(err)
		}
		fmt.Println("Gas from gasometer: ", gas)
		// the simulated gas can vary from the actual gas needed for a real transaction
		// we add an amount to ensure sufficient gas is provided
		gas += 20000
	}

	fmt.Println("past gas")
	fmt.Println("Gas post gasometer: ", gas)
	txf = txf.WithGas(gas)
	//txf = txf.WithFees(c.fees)

	fmt.Println("TXF after prepare factory and before WithGasPrices: ", txf)

	if c.gasPrices != "" {
		txf = txf.WithGasPrices(c.gasPrices)
	}

	fmt.Println("TXF after prepare factory and before WithGasAdjustment: ", txf)

	if c.gasAdjustment != 0 && c.gasAdjustment != defaultGasAdjustment {
		txf = txf.WithGasAdjustment(c.gasAdjustment)
	}

	fmt.Println("TXF after prepare factory and before BuildUnsignedTx: ", txf)

	txUnsigned, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return TxService{}, errors.WithStack(err)
	}

	txUnsigned.SetFeeGranter(ctx.GetFeeGranterAddress())

	return TxService{
		client:        c,
		clientContext: ctx,
		txBuilder:     txUnsigned,
		txFactory:     txf,
	}, nil
}

// makeSureAccountHasTokens makes sure the address has a positive balance.
// It requests funds from the faucet if the address has an empty balance.
func (c *Client) makeSureAccountHasTokens(ctx context.Context, address string) error {
	fmt.Println("Within makeSureAccountHasTokens")
	if err := c.checkAccountBalance(ctx, address); err == nil {
		return nil
	} else {
		return err
	}

	// FundsEnsureDuration := time.Second * 40

	// // make sure funds are retrieved.
	// ctx, cancel := context.WithTimeout(ctx, FundsEnsureDuration)
	// defer cancel()

	// return backoff.Retry(func() error {
	// 	return c.checkAccountBalance(ctx, address)
	// }, backoff.WithContext(backoff.NewConstantBackOff(time.Second), ctx))
}

func (c *Client) checkAccountBalance(ctx context.Context, address string) error {
	fmt.Println("Within checkAccountBalance")
	fmt.Println("address: ", address)
	resp, err := c.bankQueryClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   "uakt",
	})
	if err != nil {
		return err
	}

	fmt.Println("Response from c.bankQueryClient.Balance: ", resp)

	if resp.Balance.Amount.Uint64() >= c.faucetMinAmount {
		return nil
	}

	return fmt.Errorf("account has not enough %q balance, min. required amount: %d", c.faucetDenom, c.faucetMinAmount)
}

// handleBroadcastResult handles the result of broadcast messages result and checks if an error occurred.
func handleBroadcastResult(resp *sdktypes.TxResponse, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return errors.New("make sure that your account has enough balance")
		}
		return err
	}

	if resp.Code > 0 {
		return errors.Errorf("error code: '%d' msg: '%s'", resp.Code, resp.RawLog)
	}
	return nil
}

func (c *Client) prepareFactory(clientCtx client.Context) (tx.Factory, error) {
	fmt.Println("Within prepareFactory")
	fmt.Println("Within prepareFactory - clientCtx: ", clientCtx)

	var (
		from = clientCtx.GetFromAddress()
		txf  = c.TxFactory
	)

	fmt.Println("from: ", from)

	if err := c.accountRetriever.EnsureExists(clientCtx, from); err != nil {
		fmt.Println("Error from c.accountRetriever.EnsureExist: ", err)
		return txf, errors.WithStack(err)
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := c.accountRetriever.GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, errors.WithStack(err)
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

// WaitForTx requests the tx from hash, if not found, waits for next block and
// tries again. Returns an error if ctx is canceled.
func (c Client) WaitForTx(ctx context.Context, hash string) (*ctypes.ResultTx, error) {
	bz, err := hex.DecodeString(hash)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to decode tx hash '%s'", hash)
	}
	for {
		resp, err := c.RPC.Tx(ctx, bz, false)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				// Tx not found, wait for next block and try again
				err := c.WaitForNextBlock(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "waiting for next block")
				}
				continue
			}
			return nil, errors.Wrapf(err, "fetching tx '%s'", hash)
		}
		// Tx found
		return resp, nil
	}
}

// WaitForNBlocks reads the current block height and then waits for another n
// blocks to be committed, or returns an error if ctx is canceled.
func (c Client) WaitForNBlocks(ctx context.Context, n int64) error {
	start, err := c.LatestBlockHeight(ctx)
	if err != nil {
		return err
	}
	return c.WaitForBlockHeight(ctx, start+n)
}

// WaitForNextBlock waits until next block is committed.
// It reads the current block height and then waits for another block to be
// committed, or returns an error if ctx is canceled.
func (c Client) WaitForNextBlock(ctx context.Context) error {
	return c.WaitForNBlocks(ctx, 1)
}

// WaitForBlockHeight waits until block height h is committed, or returns an
// error if ctx is canceled.
func (c Client) WaitForBlockHeight(ctx context.Context, h int64) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		latestHeight, err := c.LatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		if latestHeight >= h {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timeout exceeded waiting for block")
		case <-ticker.C:
		}
	}
}

func (c Client) LatestBlockHeight(ctx context.Context) (int64, error) {
	resp, err := c.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resp.SyncInfo.LatestBlockHeight, nil
}

func (c Client) Status(ctx context.Context) (*ctypes.ResultStatus, error) {
	return c.RPC.Status(ctx)
}

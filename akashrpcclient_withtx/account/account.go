package account

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	// "github.com/cosmos/cosmos-sdk/codec"
	dkeyring "github.com/99designs/keyring"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type Registry struct {
	homePath           string
	keyringServiceName string
	keyringBackend     KeyringBackend

	Keyring keyring.Keyring
}

const (
	// KeyringServiceName used for the name of keyring in OS backend.
	KeyringServiceName = "akash"

	// DefaultAccount is the name of the default account.
	DefaultAccount = "default"
)

// KeyringBackend is the backend for where keys are stored.
type KeyringBackend string

const (
	// KeyringTest is the test keyring backend. With this backend, your keys will be
	// stored under your app's data dir.
	KeyringTest KeyringBackend = "test"

	// KeyringOS is the OS keyring backend. With this backend, your keys will be
	// stored in your operating system's secured keyring.
	KeyringOS KeyringBackend = "os"

	// KeyringMemory is in memory keyring backend, your keys will be stored in application memory.
	KeyringMemory KeyringBackend = "memory"

	AccountPrefixCosmos = "akash"
)

// KeyringHome used to store account related data.
var KeyringHome = os.ExpandEnv("$HOME/.akash/accounts")

// New creates a new registry to manage accounts.
func New(options ...Option) (Registry, error) {
	r := Registry{
		keyringServiceName: sdktypes.KeyringServiceName(),
		keyringBackend:     KeyringOS,
		homePath:           KeyringHome,
	}

	for _, apply := range options {
		apply(&r)
	}

	var err error
	inBuf := bufio.NewReader(os.Stdin)
	interfaceRegistry := types.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	// cdc := codec.NewProtoCodec(interfaceRegistry)
	// r.Keyring, err = keyring.New(r.keyringServiceName, string(r.keyringBackend), r.homePath, inBuf, cdc)
	r.Keyring, err = keyring.New(r.keyringServiceName, string(r.keyringBackend), r.homePath, inBuf)

	fmt.Println("Wihtin account new - registry: ", r)

	if err != nil {
		return Registry{}, err
	}

	return r, nil
}

// Option configures your registry.
type Option func(*Registry)

func WithKeyringServiceName(name string) Option {
	return func(c *Registry) {
		c.keyringServiceName = name
	}
}

func WithKeyringBackend(backend KeyringBackend) Option {
	return func(c *Registry) {
		c.keyringBackend = backend
	}
}

func WithHome(path string) Option {
	return func(c *Registry) {
		c.homePath = path
	}
}

type Account struct {
	// Name of the account.
	Name string

	// // Record holds additional info about the account.
	// Record *keyring.Record

	// Record holds additional info about the account.
	Info keyring.Info
}

func (r Registry) GetByName(name string) (Account, error) {
	info, err := r.Keyring.Key(name)
	if errors.Is(err, dkeyring.ErrKeyNotFound) || errors.Is(err, sdkerrors.ErrKeyNotFound) {
		return Account{}, &AccountDoesNotExistError{name}
	}
	if err != nil {
		return Account{}, err
	}

	acc := Account{
		Name: name,
		Info: info,
	}

	return acc, nil
}

// GetByAddress returns an account by its address.
func (r Registry) GetByAddress(address string) (Account, error) {

	sdkAddr, err := sdktypes.AccAddressFromBech32(address)
	if err != nil {
		return Account{}, err
	}
	info, err := r.Keyring.KeyByAddress(sdkAddr)
	if errors.Is(err, dkeyring.ErrKeyNotFound) || errors.Is(err, sdkerrors.ErrKeyNotFound) {
		return Account{}, &AccountDoesNotExistError{address}
	}
	if err != nil {
		return Account{}, err
	}
	return Account{
		Name: address,
		Info: info,
	}, nil
}

type AccountDoesNotExistError struct {
	Name string
}

func (e *AccountDoesNotExistError) Error() string {
	return fmt.Sprintf("account %q does not exist", e.Name)
}

// Address returns the address of the account from given prefix.
func (a Account) Address(accPrefix string) (string, error) {
	if accPrefix == "" {
		accPrefix = AccountPrefixCosmos
	}

	pk := a.Info.GetPubKey()

	return toBech32(accPrefix, pk.Address())
}

func toBech32(prefix string, addr []byte) (string, error) {
	bech32Addr, err := bech32.ConvertAndEncode(prefix, addr)
	if err != nil {
		return "", err
	}
	return bech32Addr, nil
}

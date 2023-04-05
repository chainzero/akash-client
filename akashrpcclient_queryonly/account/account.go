package account

import (
	"bufio"
	"os"

	// "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

type Registry struct {
	homePath           string
	keyringServiceName string
	keyringBackend     KeyringBackend

	Keyring keyring.Keyring
}

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
)

// KeyringHome used to store account related data.
var KeyringHome = os.ExpandEnv("$HOME/.akash/accounts")

// New creates a new registry to manage accounts.
func New(options ...Option) (Registry, error) {
	r := Registry{
		keyringServiceName: sdktypes.KeyringServiceName(),
		keyringBackend:     KeyringTest,
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

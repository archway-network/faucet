package faucet

import (
	"log"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Option configures Faucet.
type Option func(*Faucet)

type Faucet struct {
	// HTTP Server configuration
	Port               int
	LogLevel           string
	MaxCoinsPerAccount map[string]sdkmath.Int
	MaxCoinsPerRequest map[string]sdkmath.Int

	// Faucet account configuration
	FaucetAccountMnemonic string
	faucetAccountAddress  string

	// Blockchain App configuration
	AppBinaryName string
	AppHome       string
	AppChainID    string
	AppNode       string // RPC Node to be used with the faucet.

	// Transfer tx configuration
	TxGasAdjustment string
	TxBroadcastMode string
	TxGasPrices     string
}

type Account struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic,omitempty"`
}

type TransferRequest struct {
	// AccountAddress to request coins for.
	AccountAddress string `json:"address"`

	// Coins that are requested.
	// Coins sdk.Coins `json:"coins"`
	Coins []string `json:"coins"`
}

type TransferResponse struct {
	Error string `json:"error,omitempty"`
}
type Event struct {
	Type       string
	Attributes []EventAttribute
	Time       time.Time
}

type EventAttribute struct {
	Key   string
	Value string
}

// WithPort
func WithPort(p int) Option {
	return func(f *Faucet) {
		f.Port = p
	}
}

// WithLogLevel sets the log level
// for the faucet server.
func WithLogLevel(l string) Option {
	return func(f *Faucet) {
		f.LogLevel = l
	}
}

// WithMaxCoinsPerAccount
func WithMaxCoinsPerAccount(c string) Option {
	coins, err := sdk.ParseCoinsNormalized(c)
	if err != nil {
		log.Fatal(err)
	}

	maxCoinsPerAccount := make(map[string]sdkmath.Int)
	for _, coin := range coins {
		maxCoinsPerAccount[coin.Denom] = coin.Amount
	}

	return func(f *Faucet) {
		f.MaxCoinsPerAccount = maxCoinsPerAccount
	}
}

// WithMaxCoinsPerRequest
func WithMaxCoinsPerRequest(c string) Option {
	coins, err := sdk.ParseCoinsNormalized(c)
	if err != nil {
		log.Fatal(err)
	}

	maxCoinsPerRequest := make(map[string]sdkmath.Int)
	for _, coin := range coins {
		maxCoinsPerRequest[coin.Denom] = coin.Amount
	}

	return func(f *Faucet) {
		f.MaxCoinsPerRequest = maxCoinsPerRequest
	}
}

// WithFaucetAccountMnemonic
func WithFaucetAccountMnemonic(m string) Option {
	return func(f *Faucet) {
		f.FaucetAccountMnemonic = m
	}
}

// WithAppBinaryName
func WithAppBinaryName(n string) Option {
	return func(f *Faucet) {
		f.AppBinaryName = n
	}
}

// WithAppHome
func WithAppHome(h string) Option {
	return func(f *Faucet) {
		f.AppHome = h
	}
}

// WithAppChainID
func WithAppChainID(c string) Option {
	return func(f *Faucet) {
		f.AppChainID = c
	}
}

// WithAppNode
func WithAppNode(n string) Option {
	return func(f *Faucet) {
		f.AppNode = n
	}
}

// WithTxGasAdjustment
func WithTxGasAdjustment(a string) Option {
	return func(f *Faucet) {
		f.TxGasAdjustment = a
	}
}

// WithTxBroadcastMode
func WithTxBroadcastMode(m string) Option {
	return func(f *Faucet) {
		f.TxBroadcastMode = m
	}
}

// WithTxGasPrices
func WithTxGasPrices(p string) Option {
	return func(f *Faucet) {
		f.TxGasPrices = p
	}
}

func New(port int, options ...Option) *Faucet {
	f := &Faucet{
		Port: port,
	}

	for _, opt := range options {
		opt(f)
	}
	return f
}

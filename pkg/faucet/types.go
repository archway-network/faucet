package faucet

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cmdrunner"
	"github.com/ignite/cli/ignite/pkg/cmdrunner/step"
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

	// White list file
	WhiteListFile      string
	WhiteListAddresses map[string]int

	// Command runner which faucet uses to execute blockchain app commands.
	runner *cmdrunner.Runner
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
	Coins sdk.Coins `json:"coins"`
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

// WithWhiteListFile
func WithWhiteListFile(w string) Option {
	return func(f *Faucet) {
		f.WhiteListFile = w
	}
}

func (f *Faucet) LoadWhiteList() error {
	log.Printf("Loading whitelist addresses from: %s", f.WhiteListFile)
	csvFile, err := os.Open(f.WhiteListFile)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	lines, err := reader.ReadAll()
	if err != nil {
		return err
	}

	f.WhiteListAddresses = make(map[string]int)
	for _, line := range lines {
		key := strings.Trim(line[0], " /n") // Value from the first column
		value, err := strconv.Atoi(line[1]) // Value from the second column
		if err != nil {
			return err
		}
		f.WhiteListAddresses[key] = value
	}

	return nil
}

func New(port int, options ...Option) (*Faucet, error) {
	f := &Faucet{
		Port:   port,
		runner: cmdrunner.New(),
	}

	for _, opt := range options {
		opt(f)
	}

	// If debug logs are enabled, command runner must be overridden
	// with a debug enabled command runner
	if f.LogLevel == "debug" {
		f.runner = cmdrunner.New(
			cmdrunner.EnableDebug(),
		)
	}

	// clean up the test keyring
	if err := f.ResetTestKeyring(filepath.Join(f.AppHome, "keyring-test")); err != nil {
		return nil, err
	}

	// init variables
	command := []string{}
	txStepOptions := []step.Option{}
	steps := []*step.Step{}
	cmdOutputBuffer := new(bytes.Buffer)

	// Add execution step to add faucet account
	input := &bytes.Buffer{}
	fmt.Fprintln(input, f.FaucetAccountMnemonic)

	command = []string{"keys", "add", "faucet-account", "--keyring-backend", "test", "--home",
		f.AppHome, "--recover", "--output", "json"}
	txStepOptions = []step.Option{
		step.Exec(f.AppBinaryName, command...),
		step.Stderr(os.Stderr),
		step.Stdout(io.MultiWriter(os.Stdout, cmdOutputBuffer)),
		step.Stdin(input),
	}
	steps = append(steps, step.New(txStepOptions...))
	err := f.runner.Run(context.Background(), steps...)

	if err != nil {
		return nil, err
	}

	data, err := JSONEnsuredBytes(cmdOutputBuffer.Bytes())
	if err != nil {
		return nil, err
	}
	// get and decodes all accounts of the chains
	var account Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}
	f.faucetAccountAddress = account.Address

	// load white list accounts
	f.LoadWhiteList()

	return f, nil
}

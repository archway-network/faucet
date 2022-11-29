// Package faucet is a faucet to request tokens for sdk accounts.
package faucet

import (
	"bytes"
	"context"
	"embed"
	_ "embed" // used for embedding openapi assets.
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ghodss/yaml"

	"github.com/ignite/cli/ignite/pkg/cmdrunner"
	"github.com/ignite/cli/ignite/pkg/cmdrunner/step"
)

const (
	DefaultFaucetAccountName = "faucet-account"
	msgEmptyKeyring          = "No records were found in keyring"
	fileNameOpenAPISpec      = "openapi/openapi.yml.tmpl"
)

var (
	// ErrAccountAlreadyExists returned when an already exists account attempted to be imported.
	ErrAccountAlreadyExists = errors.New("account already exists")

	// ErrAccountDoesNotExist returned when account does not exit.
	ErrAccountDoesNotExist = errors.New("account does not exit")

	//go:embed openapi/openapi.yml.tmpl
	bytesOpenAPISpec []byte

	tmplOpenAPISpec = template.Must(template.New(fileNameOpenAPISpec).Parse(string(bytesOpenAPISpec)))

	//go:embed openapi/index.tpl
	index embed.FS
)

func (f Faucet) CheckAccountAdded() error {
	command := []string{
		"keys", "list", "--keyring-backend", "test", "--output", "json",
	}

	cmdOutputBuffer := new(bytes.Buffer)
	keysListStepOptions := []step.Option{
		step.Exec(f.AppBinaryName, command...),
		step.Stderr(os.Stderr),
		step.Stdout(io.MultiWriter(os.Stdout, cmdOutputBuffer)),
	}

	step := step.New(keysListStepOptions...)
	err := cmdrunner.New().Run(context.Background(), step)
	if err != nil {
		return err
	}

	// Make sure that the command output is not the empty keyring message.
	// This need to be checked because when the keyring is empty the command
	// finishes with exit code 0 and a plain text message.
	// This behavior was added to Cosmos SDK v0.46.2. See the link
	// https://github.com/cosmos/cosmos-sdk/blob/d01aa5b4a8/client/keys/list.go#L37
	if strings.TrimSpace(cmdOutputBuffer.String()) == msgEmptyKeyring {
		return nil
	}

	data, err := JSONEnsuredBytes(cmdOutputBuffer.Bytes())
	if err != nil {
		return err
	}
	// get and decodes all accounts of the chains
	var accounts []Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return err
	}

	// search for the account name
	for _, account := range accounts {
		if account.Name == DefaultFaucetAccountName {
			return nil
		}
	}
	return ErrAccountDoesNotExist
}
func (f Faucet) ValidateRequest(req TransferRequest) error {
	if req.Coins == nil {
		return errors.New("no coins provided")
	}

	if req.AccountAddress == "" {
		return errors.New("no account address provided")
	}

	for _, coin := range req.Coins {
		if f.MaxCoinsPerRequest[coin.Denom].LT(coin.Amount) {
			return fmt.Errorf("coin amount %s is greater than max allowed per request of %s", coin.String(), f.MaxCoinsPerRequest[coin.Denom].String())
		}
	}

	// _, err := f.TotalTransferredAmount(req)
	// if err != nil {
	// 	return err
	// }
	return nil
	// if f.TotalMaxAmount < total {
	// 	return errors.New("total amount of coins requested is over the limit")
	// }

}

// TotalTransferredAmount returns the total transferred amount from faucet account to toAccountAddress.
func (f Faucet) TotalTransferredAmount(req TransferRequest) (coinsTransferred sdk.Coins, err error) {
	command := []string{"q", "txs", "--events",
		"message.sender=" + f.faucetAccountAddress + "&transfer.recipient=" + req.AccountAddress,
		"--page", "1", "--limit", "2", "--node", f.AppNode,
		"--output", "json", "--chain-id", f.AppChainID}

	cmdOutputBuffer := new(bytes.Buffer)

	cmdStdOut := io.MultiWriter(os.Stdout, cmdOutputBuffer)
	stepOptions := []step.Option{
		step.Exec(f.AppBinaryName, command...),
		step.Stderr(os.Stderr),
		step.Stdout(cmdStdOut),
	}

	step := step.New(stepOptions...)

	err = cmdrunner.New().Run(context.Background(), step)

	if err != nil {
		return nil, err
	}

	out := struct {
		Txs []struct {
			Logs []struct {
				Events []struct {
					Type  string `json:"type"`
					Attrs []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"attributes"`
				} `json:"events"`
			} `json:"logs"`
			TimeStamp string `json:"timestamp"`
		} `json:"txs"`
	}{}

	data, err := JSONEnsuredBytes(cmdOutputBuffer.Bytes())
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	var events []Event

	for _, tx := range out.Txs {
		for _, log := range tx.Logs {
			for _, e := range log.Events {
				var attrs []EventAttribute
				for _, attr := range e.Attrs {
					attrs = append(attrs, EventAttribute{
						Key:   attr.Key,
						Value: attr.Value,
					})
				}

				txTime, err := time.Parse(time.RFC3339, tx.TimeStamp)
				if err != nil {
					return nil, err
				}

				events = append(events, Event{
					Type:       e.Type,
					Attributes: attrs,
					Time:       txTime,
				})
			}
		}
	}

	coinsTransferred = sdk.Coins{}
	for _, event := range events {
		if event.Type == "transfer" {
			for _, attr := range event.Attributes {
				if attr.Key == "amount" {
					coins, err := sdk.ParseCoinsNormalized(attr.Value)
					if err != nil {
						return nil, err
					}

					coinsTransferred.Add(coins.Sort()...)

					// TODO: Enable refresh window
					// if amount > 0 && time.Since(event.Time) < f.limitRefreshWindow {
					// 	totalAmount += amount
					// }
				}
			}
		}
	}

	return coinsTransferred, nil
}

func (f Faucet) Transfer(req TransferRequest) error {
	// init variables
	command := []string{}
	txStepOptions := []step.Option{}
	steps := []*step.Step{}

	command = []string{
		"tx", "bank", "send", DefaultFaucetAccountName, req.AccountAddress, req.Coins.String(),
		"--node", f.AppNode, "--output", "json", "--chain-id", f.AppChainID, "--gas-adjustment",
		f.TxGasAdjustment, "--broadcast-mode", f.TxBroadcastMode, "--yes", "--gas-prices", f.TxGasPrices,
		"--log_level", f.LogLevel, "--keyring-backend", "test",
	}

	txStepOptions = []step.Option{
		step.Exec(f.AppBinaryName, command...),
		step.Stderr(os.Stderr),
		step.Stdout(os.Stdout),
	}

	steps = append(steps, step.New(txStepOptions...))
	err := cmdrunner.New().Run(context.Background(), steps...)

	if err != nil {
		return err
	}

	return nil
}

// JSONEnsuredBytes ensures that encoding format for returned bytes is always
// JSON even if the written data is originally encoded in YAML.
func JSONEnsuredBytes(bytes []byte) ([]byte, error) {

	var out interface{}

	if err := yaml.Unmarshal(bytes, &out); err == nil {
		return yaml.YAMLToJSON(bytes)
	}

	return bytes, nil
}

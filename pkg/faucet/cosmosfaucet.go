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
	"path/filepath"
	"time"

	sdkmath "cosmossdk.io/math"
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
	//go:embed openapi/openapi.yml.tmpl
	bytesOpenAPISpec []byte

	tmplOpenAPISpec = template.Must(template.New(fileNameOpenAPISpec).Parse(string(bytesOpenAPISpec)))

	//go:embed openapi/index.tpl
	index embed.FS
)

func (f Faucet) ValidateRequest(req TransferRequest) error {
	if req.Coins == nil {
		return errors.New("no coins provided")
	}

	if req.AccountAddress == "" {
		return errors.New("no account address provided")
	}

	for _, coin := range req.Coins {
		if _, ok := f.MaxCoinsPerAccount[coin.Denom]; ok {
			if coin.Amount.GT(f.MaxCoinsPerRequest[coin.Denom]) {
				return fmt.Errorf("%s is greater than max allowed per request of %s%s", coin.String(), f.MaxCoinsPerRequest[coin.Denom].String(), coin.Amount.String())
			}
		} else {
			return fmt.Errorf("Faucet not allowed to distribute %s", coin.Denom)
		}
	}

	TotalCoinsTransferred, err := f.TotalTransferredAmount(req)
	if err != nil {
		return err
	}

	for _, coin := range req.Coins {
		if _, ok := f.MaxCoinsPerAccount[coin.Denom]; ok {
			if _, ok := TotalCoinsTransferred[coin.Denom]; ok && TotalCoinsTransferred[coin.Denom].Add(coin.Amount).GT(f.MaxCoinsPerAccount[coin.Denom]) {
				return fmt.Errorf("quota exceeded. max %s%s allowed per account, %s%s already transferred", coin.Denom,
					f.MaxCoinsPerAccount[coin.Denom].String(), coin.Denom, TotalCoinsTransferred[coin.Denom].String())
			}
		} else {
			return fmt.Errorf("Faucet not allowed to distribute %s", coin.Denom)
		}
	}
	return nil
}

// TotalTransferredAmount returns the total transferred amount from faucet account to toAccountAddress.
func (f Faucet) TotalTransferredAmount(req TransferRequest) (coinsTransferred map[string]sdkmath.Int, err error) {
	//TODO: Add pagination support if there are more than 1 page of transactions
	command := []string{"q", "txs", "--events",
		"message.sender=" + f.faucetAccountAddress + "&transfer.recipient=" + req.AccountAddress,
		"--page", "1", "--limit", "50", "--node", f.AppNode,
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

	coinsTransferred = map[string]sdkmath.Int{}
	for _, event := range events {
		if event.Type == "transfer" {
			for _, attr := range event.Attributes {
				if attr.Key == "amount" {
					coins, err := sdk.ParseCoinsNormalized(attr.Value)
					if err != nil {
						return nil, err
					}

					for _, coin := range coins {
						if _, ok := coinsTransferred[coin.Denom]; !ok {
							coinsTransferred[coin.Denom] = coin.Amount
						} else {
							coinsTransferred[coin.Denom] = coinsTransferred[coin.Denom].Add(coin.Amount)
						}
					}
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

func (f *Faucet) ResetTestKeyring(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

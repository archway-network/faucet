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
	"net/http"
	"os"
	"strings"
	"time"

	// sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/ignite/cli/ignite/pkg/cmdrunner"
	"github.com/ignite/cli/ignite/pkg/cmdrunner/step"
	"github.com/ignite/cli/ignite/pkg/xhttp"
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

type Faucet struct {
	LogLevel            string
	Node                string
	Denoms              string
	TotalMaxAmount      string
	MaxAmountPerRequest string
	BinaryName          string
	Port                int
	AccountMnemonic     string
	AccountAddress      string
	Home                string
	GasAdjustment       string
	BroadcastMode       string
	GasPrices           string
	ChainID             string
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
	// default ones used when this one isn't provided.
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

func (f Faucet) CheckAccountAdded() error {
	command := []string{
		"keys", "list", "--keyring-backend", "test", "--output", "json",
	}

	cmdOutputBuffer := new(bytes.Buffer)
	keysListStepOptions := []step.Option{
		step.Exec(f.BinaryName, command...),
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
			f.AccountAddress = account.Address
			return nil
		}
	}
	return ErrAccountDoesNotExist
}

func (f Faucet) Transfer(req TransferRequest) error {
	// init variables
	command := []string{}
	txStepOptions := []step.Option{}
	steps := []*step.Step{}

	// Check if faucet account alredy exists
	err := f.CheckAccountAdded()
	if err == ErrAccountDoesNotExist {
		// Add execution step to add faucet account
		input := &bytes.Buffer{}
		fmt.Fprintln(input, f.AccountMnemonic)

		command = []string{"keys", "add", "faucet-account", "--keyring-backend", "test", "--recover"}
		txStepOptions = []step.Option{
			step.Exec(f.BinaryName, command...),
			step.Stderr(os.Stderr),
			step.Stdout(os.Stdout),
			step.Stdin(input),
		}
		steps = append(steps, step.New(txStepOptions...))
	} else if err != nil {
		return err
	}

	command = []string{
		"tx", "bank", "send", DefaultFaucetAccountName, req.AccountAddress, strings.Join(req.Coins, ","),
		"--node", f.Node, "--output", "json", "--chain-id", f.ChainID, "--gas-adjustment",
		f.GasAdjustment, "--broadcast-mode", f.BroadcastMode, "--yes", "--gas-prices", f.GasPrices,
		"--log_level", f.LogLevel, "--keyring-backend", "test",
	}

	txStepOptions = []step.Option{
		step.Exec(f.BinaryName, command...),
		step.Stderr(os.Stderr),
		step.Stdout(os.Stdout),
	}

	steps = append(steps, step.New(txStepOptions...))
	err = cmdrunner.New().Run(context.Background(), steps...)

	if err != nil {
		return err
	}

	return nil
}

// TotalTransferredAmount returns the total transferred amount from faucet account to toAccountAddress.
// func (f Faucet) TotalTransferredAmount(req TransferRequest) (totalAmount uint64, err error) {
// 	command := []string{"q", "txs", "--events",
// 		"message.sender=" + f.AccountAddress + "&transfer.recipient=" + req.AccountAddress,
// 		"--page", "1", "--limit", "2", "--node", f.Node,
// 		"--output", "json", "--chain-id", f.ChainID}

// 	//
// 	cmdOutputBuffer := new(bytes.Buffer)

// 	cmdStdOut := io.MultiWriter(os.Stdout, cmdOutputBuffer)
// 	stepOptions := []step.Option{
// 		step.Exec(f.BinaryName, command...),
// 		step.Stderr(os.Stderr),
// 		step.Stdout(cmdStdOut),
// 	}

// 	step := step.New(stepOptions...)

// 	err = cmdrunner.New().Run(context.Background(), step)

// 	if err != nil {
// 		fmt.Printf("Error: %v", err)
// 	}

// 	out := struct {
// 		Txs []struct {
// 			Logs []struct {
// 				Events []struct {
// 					Type  string `json:"type"`
// 					Attrs []struct {
// 						Key   string `json:"key"`
// 						Value string `json:"value"`
// 					} `json:"attributes"`
// 				} `json:"events"`
// 			} `json:"logs"`
// 			TimeStamp string `json:"timestamp"`
// 		} `json:"txs"`
// 	}{}

// 	data, err := JSONEnsuredBytes(cmdOutputBuffer.Bytes())
// 	if err != nil {
// 		return 0, err
// 	}
// 	if err := json.Unmarshal(data, &out); err != nil {
// 		return 0, err
// 	}

// 	if err := json.Unmarshal(data, &out); err != nil {
// 		return 0, err
// 	}

// 	var events []Event

// 	for _, tx := range out.Txs {
// 		for _, log := range tx.Logs {
// 			for _, e := range log.Events {
// 				var attrs []EventAttribute
// 				for _, attr := range e.Attrs {
// 					attrs = append(attrs, EventAttribute{
// 						Key:   attr.Key,
// 						Value: attr.Value,
// 					})
// 				}

// 				txTime, err := time.Parse(time.RFC3339, tx.TimeStamp)
// 				if err != nil {
// 					return 0, err
// 				}

// 				events = append(events, Event{
// 					Type:       e.Type,
// 					Attributes: attrs,
// 					Time:       txTime,
// 				})
// 			}
// 		}
// 	}

// 	for _, event := range events {
// 		if event.Type == "transfer" {
// 			for _, attr := range event.Attributes {
// 				if attr.Key == "amount" {
// 					coins, err := sdk.ParseCoinsNormalized(attr.Value)
// 					if err != nil {
// 						return 0, err
// 					}

// 					amount := coins.AmountOf("utitus").Uint64()
// 					totalAmount += amount

// 					// if amount > 0 && time.Since(event.Time) < f.limitRefreshWindow {
// 					// 	totalAmount += amount
// 					// }
// 				}
// 			}
// 		}
// 	}

// 	return totalAmount, nil
// }

// coinsFromRequest determines tokens to transfer from transfer request.
// func (f Faucet) coinsFromRequest(req TransferRequest) (sdk.Coins, error) {
// 	if len(req.Coins) == 0 {
// 		// TODO: return error message that no coins are provided.
// 		return nil, nil
// 	}

// 	var coins []sdk.Coin
// 	for _, c := range req.Coins {
// 		coin, err := sdk.ParseCoinNormalized(c)
// 		if err != nil {
// 			return nil, err
// 		}
// 		coins = append(coins, coin)
// 	}

// 	return coins, nil
// }

// JSONEnsuredBytes ensures that encoding format for returned bytes is always
// JSON even if the written data is originally encoded in YAML.
func JSONEnsuredBytes(bytes []byte) ([]byte, error) {

	var out interface{}

	if err := yaml.Unmarshal(bytes, &out); err == nil {
		return yaml.YAMLToJSON(bytes)
	}

	return bytes, nil
}

// ServeHTTP implements http.Handler to expose the functionality of Faucet.Transfer() via HTTP.
// request/response payloads are compatible with the previous implementation at allinbits/cosmos-faucet.
func (f Faucet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	router.
		Handle("/", cors.Default().Handler(http.HandlerFunc(f.faucetHandler))).
		Methods(http.MethodPost, http.MethodOptions)

	router.
		HandleFunc("/", OpenAPIHandler("Faucet", "openapi.yml")).
		Methods(http.MethodGet)

	router.
		HandleFunc("/openapi.yml", f.openAPISpecHandler).
		Methods(http.MethodGet)

	router.ServeHTTP(w, r)
}

func (f Faucet) faucetHandler(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest

	// decode request into req.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseError(w, http.StatusBadRequest, err)
		return
	}

	// try performing the transfer
	if err := f.Transfer(req); err != nil {
		if err == context.Canceled {
			return
		}
		responseError(w, http.StatusInternalServerError, err)
	} else {
		responseSuccess(w)
	}
}

func responseSuccess(w http.ResponseWriter) {
	xhttp.ResponseJSON(w, http.StatusOK, TransferResponse{})
}

func responseError(w http.ResponseWriter, code int, err error) {
	xhttp.ResponseJSON(w, code, TransferResponse{
		Error: err.Error(),
	})
}

func (f Faucet) openAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	tmplOpenAPISpec.Execute(w, f.ChainID)
}

// Handler returns an http handler that servers OpenAPI console for an OpenAPI spec at specURL.
func OpenAPIHandler(title, specURL string) http.HandlerFunc {
	t, _ := template.ParseFS(index, "openapi/index.tpl")

	return func(w http.ResponseWriter, req *http.Request) {
		t.Execute(w, struct {
			Title string
			URL   string
		}{
			title,
			specURL,
		})
	}
}

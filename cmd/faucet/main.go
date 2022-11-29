package main

import (
	"flag"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/archway-network/faucet/pkg/environ"
	"github.com/archway-network/faucet/pkg/faucet"
)

var (
	// HTTP Server configuration
	port                  int
	logLevel              string
	maxCoinsPerAccount    string
	maxCoinsPerRequest    string
	faucetAccountMnemonic string

	// Blockchain App configuration
	appBinaryName string
	appHome       string
	appChainID    string
	appNode       string // RPC Node to be used with the .

	// Transfer tx configuration
	txGasAdjustment string
	txBroadcastMode string
	txGasPrices     string
)

func main() {
	// HTTP Server configuration
	flag.IntVar(&port, "port",
		environ.EnvGetInt("PORT", 8000),
		"Port for the faucet server to listen on")
	flag.StringVar(&logLevel, "log-level",
		environ.EnvGetString("LOG_LEVEL", "info"),
		"log level (trace, debug, info, warn or error)")
	flag.StringVar(&maxCoinsPerAccount, "max-coins-per-account",
		environ.EnvGetString("MAX_COINS_PER_ACCOUNT", "10000000uarch"),
		"Comma seperated list of total amount of tokens allowed per address")
	flag.StringVar(&maxCoinsPerRequest, "max-coins-per-request",
		environ.EnvGetString("MAX_COINS_PER_REQUEST", "1000000uarch"),
		"Comma seperated list of total amount of tokens allowed per request")
	flag.StringVar(&faucetAccountMnemonic, "faucet-account-mnemonic",
		environ.EnvGetString("FAUCET_ACCOUNT_MNEMONIC", ""),
		"Mnemonic for the faucet account")

	// Blockchain App configuration
	flag.StringVar(&appBinaryName, "app-binary-name",
		environ.EnvGetString("APP_BINARY_NAME", "archwayd"),
		"Name of the chain binary to be used for the faucet")
	flag.StringVar(&appHome, "app-home",
		environ.EnvGetString("APP_HOME", ""),
		"Home directory for blockchain config")
	flag.StringVar(&appChainID, "app-chain-id",
		environ.EnvGetString("APP_CHAIN_ID", "archway-1"),
		"Chain ID for the transaction")
	flag.StringVar(&appNode, "app-node",
		environ.EnvGetString("APP_NODE", ""),
		"RPC address of the node to connect to")

	// Transfer tx configuration
	flag.StringVar(&txGasAdjustment, "tx-gas-adjustment",
		environ.EnvGetString("TX_GAS_ADJUSTMENT", "1.5"),
		"Gas adjustment for the transaction")
	flag.StringVar(&txBroadcastMode, "tx-broadcast-mode",
		environ.EnvGetString("TX_BROADCAST_MODE", "block"),
		"Broadcast mode for the transaction")
	flag.StringVar(&txGasPrices, "tx-gas-prices",
		environ.EnvGetString("TX_GAS_PRICES", "0.025uarch"),
		"Gas prices for the transaction")

	flag.Parse()

	// Create a new faucet.
	f, err := faucet.New(port,
		faucet.WithAppBinaryName(appBinaryName),
		faucet.WithAppHome(appHome),
		faucet.WithLogLevel(logLevel),
		faucet.WithMaxCoinsPerAccount(maxCoinsPerAccount),
		faucet.WithMaxCoinsPerRequest(maxCoinsPerRequest),
		faucet.WithFaucetAccountMnemonic(faucetAccountMnemonic),
		faucet.WithAppChainID(appChainID),
		faucet.WithAppNode(appNode),
		faucet.WithTxGasAdjustment(txGasAdjustment),
		faucet.WithTxBroadcastMode(txBroadcastMode),
		faucet.WithTxGasPrices(txGasPrices),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", f.ServeHTTP)
	log.Infof("listening on :%d", f.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", f.Port), nil))
}

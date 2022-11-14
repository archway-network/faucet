package main

import (
	"flag"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/archway-network/faucet/pkg/environ"
	"github.com/archway-network/faucet/pkg/faucet"
)

func main() {
	faucet := faucet.Faucet{}

	flag.StringVar(&faucet.LogLevel, "log-level",
		environ.EnvGetString("LOG_LEVEL", "info"),
		"log level (trace, debug, info, warn or error)")
	flag.StringVar(&faucet.Node, "node",
		environ.EnvGetString("NODE", ""),
		"RPC address of the node to connect to")
	flag.StringVar(&faucet.Denoms, "denom",
		environ.EnvGetString("DENOMs", "utitus"),
		"Comma seperated list of token denoms to be distributed by the fuacet")
	flag.StringVar(&faucet.TotalMaxAmount, "max-amount",
		environ.EnvGetString("TOTAL_MAX_AMOUNT", "10000000uarch"),
		"Total amount of tokens allowed per address")
	flag.StringVar(&faucet.MaxAmountPerRequest, "max-amount-per-request",
		environ.EnvGetString("MAX_AMOUNT_PER_REQUEST", "1000000uarch"),
		"Total amount of tokens allowed per request")
	flag.StringVar(&faucet.BinaryName, "binary-name",
		environ.EnvGetString("BINARY_NAME", "archwayd"),
		"Name of the chain binary to be used for the faucet")
	flag.IntVar(&faucet.Port, "port",
		environ.EnvGetInt("PORT", 8000),
		"Port on which the faucet server listens on")
	flag.StringVar(&faucet.Home, "home",
		environ.EnvGetString("HOME", ""),
		"Home directory for blockchain config")
	flag.StringVar(&faucet.GasAdjustment, "gas-adjustment",
		environ.EnvGetString("GAS_ADJUSTMENT", "1.5"),
		"Gas adjustment for the transaction")
	flag.StringVar(&faucet.BroadcastMode, "broadcast-mode",
		environ.EnvGetString("BROADCAST_MODE", "block"),
		"Broadcast mode for the transaction")
	flag.StringVar(&faucet.GasPrices, "gas-prices",
		environ.EnvGetString("GAS_PRICES", "0.025uarch"),
		"Gas prices for the transaction")
	flag.StringVar(&faucet.ChainID, "chain-id",
		environ.EnvGetString("CHAIN_ID", "archway-1"),
		"Chain ID for the transaction")
	flag.StringVar(&faucet.AccountMnemonic, "account-mnemonic",
		environ.EnvGetString("ACCOUNT_MNEMONIC", ""),
		"Mnemonic for the faucet account")

	flag.Parse()

	http.HandleFunc("/", faucet.ServeHTTP)
	log.Infof("listening on :%d", faucet.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", faucet.Port), nil))
}

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/ignite/cli/ignite/pkg/chaincmd"
	chaincmdrunner "github.com/ignite/cli/ignite/pkg/chaincmd/runner"
	"github.com/ignite/cli/ignite/pkg/cosmosver"
	log "github.com/sirupsen/logrus"

	"github.com/archway-network/faucet/pkg/faucet"
)

func main() {
	cfg := faucet.FaucetConfig{}

	flag.StringVar(&cfg.LogLevel, "log-level",
		faucet.EnvGetString("LOG_LEVEL", "info"),
		"log level (trace, debug, info, warn or error)")
	flag.StringVar(&cfg.Node, "node",
		faucet.EnvGetString("NODE", ""),
		"RPC address of the node to connect to")
	flag.StringVar(&cfg.Denom, "denom",
		faucet.EnvGetString("DENOM", "uarch"),
		"Denom of the token to be distributed by the fuacet")
	flag.StringVar(&cfg.TotalMaxAmount, "max-amount",
		faucet.EnvGetString("TOTAL_MAX_AMOUNT", "10000000uarch"),
		"Total amount of tokens allowed per address")
	flag.StringVar(&cfg.MaxAmountPerRequest, "max-amount-per-request",
		faucet.EnvGetString("MAX_AMOUNT_PER_REQUEST", "1000000uarch"),
		"Total amount of tokens allowed per request")
	flag.StringVar(&cfg.BinaryName, "binary-name",
		faucet.EnvGetString("BinaryName", "archwayd"),
		"Name of the chain binary to be used for the faucet")
	flag.IntVar(&cfg.Port, "port",
		faucet.EnvGetInt("PORT", 8000),
		"Port on which the faucet server listens on")
	flag.StringVar(&cfg.KeyringBackend, "keyring-backend",
		faucet.EnvGetString("KEYRING_BAKCEND", "test"),
		"Keyring backend to be used for the faucet account")
	flag.StringVar(&cfg.KeyringPassword, "keyring-password",
		faucet.EnvGetString("KEYRING_PASSWORD", ""),
		"Keyring password to be used for the faucet account")
	flag.StringVar(&cfg.Home, "home",
		faucet.EnvGetString("Home", ""),
		"Home directory for blockchain config")

	flag.Parse()

	configKeyringBackend, err := chaincmd.KeyringBackendFromString(cfg.KeyringBackend)
	if err != nil {
		log.Fatal(err)
	}

	ccoptions := []chaincmd.Option{
		chaincmd.WithKeyringPassword(cfg.KeyringPassword),
		chaincmd.WithKeyringBackend(configKeyringBackend),
		chaincmd.WithAutoChainIDDetection(),
		chaincmd.WithNodeAddress(cfg.Node),
		chaincmd.WithHome(cfg.Home),
		chaincmd.WithVersion(cosmosver.Latest),
	}

	cr, err := chaincmdrunner.New(context.Background(), chaincmd.New(cfg.BinaryName, ccoptions...))
	if err != nil {
		log.Fatal(err)
	}

	f := faucet.Faucet{
		runner: cr,
		config: cfg,
	}

	http.HandleFunc("/", faucet.ServeHTTP)
	log.Infof("listening on :%d", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil))
}

// Package faucet is a faucet to request tokens for sdk accounts.
package faucet

type FaucetConfig struct {
	LogLevel            string
	Node                string
	Denom               string
	TotalMaxAmount      string
	MaxAmountPerRequest string
	BinaryName          string
	Port                int
	AccountMnemonic     string
	KeyringBackend      string
	KeyringPassword     string
	Home                string
}

// Faucet represents a faucet.
type Faucet struct {
	// Faucet configuration.
	config FaucetConfig
}

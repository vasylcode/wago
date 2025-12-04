package wago

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "wago",
	Short: "Wago - A simple JSON-based wallet tracker",
	Long: `Wago is a simple CLI tool for tracking cryptocurrency wallets, 
their balances, and transactions across different blockchains.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Version = version.Version
}

func initConfig() {
	// Initialize configuration if needed
}

func er(msg interface{}) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", msg)
	os.Exit(1)
}

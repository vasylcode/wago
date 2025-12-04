package main

import (
	"fmt"
	"os"

	"github.com/vasylcode/wago/cmd/wago"
)

func main() {
	if err := wago.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PriceData represents the structure of prices.json
type PriceData map[string]float64

// GetCoinPrices reads prices from the manual prices.json file
func GetCoinPrices(coinSymbols []string) (map[string]float64, error) {
	// Get the data directory path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	pricesFile := filepath.Join(homeDir, ".wago", "prices.json")

	// Check if prices.json exists
	if _, err := os.Stat(pricesFile); os.IsNotExist(err) {
		// Create an empty prices.json file with some example entries
		examplePrices := PriceData{
			"btc":  45000.0,
			"eth":  3000.0,
			"sol":  200.0,
			"usdt": 1.0,
			"usdc": 1.0,
		}

		data, err := json.MarshalIndent(examplePrices, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal example prices: %w", err)
		}

		if err := os.WriteFile(pricesFile, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create prices.json: %w", err)
		}
	}

	// Read the prices.json file
	data, err := os.ReadFile(pricesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prices.json: %w", err)
	}

	// Parse the JSON
	var allPrices PriceData
	if err := json.Unmarshal(data, &allPrices); err != nil {
		return nil, fmt.Errorf("failed to parse prices.json: %w", err)
	}

	// Return only the requested prices (case-insensitive)
	result := make(map[string]float64)
	for _, symbol := range coinSymbols {
		lowerSymbol := strings.ToLower(symbol)
		if price, exists := allPrices[lowerSymbol]; exists {
			result[lowerSymbol] = price
		}
	}

	return result, nil
}

// FormatUSDValue formats a USD value for display
func FormatUSDValue(value float64) string {
	if value >= 1000000 {
		return fmt.Sprintf("$%.2fM", value/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("$%.2fK", value/1000)
	} else {
		return fmt.Sprintf("$%.2f", value)
	}
}

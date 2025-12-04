package util

import (
	"fmt"
	"strings"

	"github.com/vasylcode/wago/internal/storage"
)

// GetCoinPrices reads prices from storage config
func GetCoinPrices(coinSymbols []string) (map[string]float64, error) {
	s, err := storage.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load storage: %w", err)
	}

	allPrices := s.GetPrices()

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

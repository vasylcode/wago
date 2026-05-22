package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
	"github.com/vasylcode/wago/internal/version"
)

const coinGeckoSimplePriceURL = "https://api.coingecko.com/api/v3/simple/price"

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

// UpdateCoinPrices fetches current USD prices and stores them in wago.json.
func UpdateCoinPrices(s *storage.Storage, wallets []*model.Wallet) error {
	coins := make(map[string]bool)
	for coin := range s.GetPrices() {
		coin = strings.ToLower(strings.TrimSpace(coin))
		if coin != "" {
			coins[coin] = true
		}
	}
	for _, wallet := range wallets {
		for _, balance := range wallet.Balances {
			coin := strings.ToLower(strings.TrimSpace(balance.Coin))
			if coin != "" {
				coins[coin] = true
			}
		}
	}
	if len(coins) == 0 {
		return nil
	}

	symbols := make([]string, 0, len(coins))
	for coin := range coins {
		symbols = append(symbols, coin)
	}

	query := url.Values{}
	query.Set("symbols", strings.Join(symbols, ","))
	query.Set("vs_currencies", "usd")
	query.Set("precision", "full")

	client := &http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequest(http.MethodGet, coinGeckoSimplePriceURL+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "wago/"+strings.TrimPrefix(strings.TrimSpace(version.Version), "v"))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("coingecko request failed with HTTP %d", resp.StatusCode)
	}

	var payload map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}

	prices := make(map[string]float64)
	for coin := range coins {
		if coinData, ok := payload[coin]; ok {
			if price, ok := coinData["usd"]; ok {
				prices[coin] = price
			}
		}
	}
	if len(prices) == 0 {
		return nil
	}

	return s.SetPrices(prices)
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

package model

import (
	"time"
)

// Data represents the unified data structure stored in wago.json
type Data struct {
	Wallets      map[string]*Wallet   `json:"wallets"`
	Categories   map[string]*Category `json:"categories"`
	Contacts     map[string]*Contact  `json:"contacts"`
	Transactions map[string]*Tx       `json:"transactions"`
	Prices       map[string]float64   `json:"prices"`
}

// Wallet represents a crypto wallet
type Wallet struct {
	Name     string     `json:"name"`
	Address  string     `json:"address"`
	Category string     `json:"category,omitempty"`
	Chain    string     `json:"chain"`
	Type     string     `json:"type"`
	Note     string     `json:"note,omitempty"`
	Balances []*Balance `json:"balances,omitempty"`
}

// Balance represents a token balance in a wallet
type Balance struct {
	Coin   string  `json:"coin"`
	Amount float64 `json:"amount"`
}

// Category represents a wallet category with a color
type Category struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Contact represents a contact in the address book
type Contact struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Chain   string `json:"chain"`
	Note    string `json:"note,omitempty"`
}

// TxType represents the type of transaction
type TxType string

const (
	TxTypeDeposit  TxType = "deposit"
	TxTypeWithdraw TxType = "withdraw"
	TxTypeTransfer TxType = "transfer"
	TxTypeSwap     TxType = "swap"
)

// Tx represents a transaction
type Tx struct {
	ID          string    `json:"id"`
	Type        TxType    `json:"type"`
	FromWallet  string    `json:"from_wallet,omitempty"`
	ToWallet    string    `json:"to_wallet,omitempty"`
	FromAddress string    `json:"from_address,omitempty"`
	ToAddress   string    `json:"to_address,omitempty"`
	Coin        string    `json:"coin"`
	Amount      float64   `json:"amount"`
	Fee         float64   `json:"fee,omitempty"`
	FeeCoin     string    `json:"fee_coin,omitempty"`
	SwapWallet  string    `json:"swap_wallet,omitempty"`
	SellCoin    string    `json:"sell_coin,omitempty"`
	SellAmount  float64   `json:"sell_amount,omitempty"`
	BuyCoin     string    `json:"buy_coin,omitempty"`
	BuyAmount   float64   `json:"buy_amount,omitempty"`
	Date        time.Time `json:"date"`
	Note        string    `json:"note,omitempty"`
}

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vasylcode/wago/internal/model"
)

const dataFileName = "wago.json"

// Storage handles the persistence of data in a single JSON file
type Storage struct {
	dataDir  string
	dataFile string
	data     *model.Data
	txIndex  map[string]bool // Track tx IDs to prevent duplicates
}

// New creates a new Storage instance
func New() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".wago")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	s := &Storage{
		dataDir:  dataDir,
		dataFile: filepath.Join(dataDir, dataFileName),
		txIndex:  make(map[string]bool),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// load loads data from wago.json
func (s *Storage) load() error {
	// Initialize empty data structure with default prices
	s.data = &model.Data{
		Wallets:      make(map[string]*model.Wallet),
		Categories:   make(map[string]*model.Category),
		Contacts:     make(map[string]*model.Contact),
		Transactions: make(map[string]*model.Tx),
		Prices: map[string]float64{
			"usdc": 1.0,
			"usdt": 1.0,
		},
	}

	// Try to load existing wago.json
	if data, err := os.ReadFile(s.dataFile); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, s.data); err != nil {
			return fmt.Errorf("failed to parse data file: %w", err)
		}
	}

	// Ensure maps are initialized
	if s.data.Wallets == nil {
		s.data.Wallets = make(map[string]*model.Wallet)
	}
	if s.data.Categories == nil {
		s.data.Categories = make(map[string]*model.Category)
	}
	if s.data.Contacts == nil {
		s.data.Contacts = make(map[string]*model.Contact)
	}
	if s.data.Transactions == nil {
		s.data.Transactions = make(map[string]*model.Tx)
	}
	if s.data.Prices == nil {
		s.data.Prices = map[string]float64{"usdc": 1.0, "usdt": 1.0}
	}

	// Build transaction index for deduplication
	s.buildTxIndex()

	return nil
}

// buildTxIndex builds an index of all transaction IDs for deduplication
func (s *Storage) buildTxIndex() {
	s.txIndex = make(map[string]bool)
	for id := range s.data.Transactions {
		s.txIndex[id] = true
	}
}

// save writes all data to wago.json
func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(s.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	return nil
}

// GetPrices returns the price map
func (s *Storage) GetPrices() map[string]float64 {
	return s.data.Prices
}

// SetPrice sets a coin price
func (s *Storage) SetPrice(coin string, price float64) error {
	s.data.Prices[coin] = price
	return s.save()
}

// AddWallet adds a new wallet
func (s *Storage) AddWallet(wallet *model.Wallet) error {
	if _, exists := s.data.Wallets[wallet.Name]; exists {
		return fmt.Errorf("wallet with name '%s' already exists", wallet.Name)
	}

	s.data.Wallets[wallet.Name] = wallet
	return s.save()
}

// GetWallet gets a wallet by name
func (s *Storage) GetWallet(name string) (*model.Wallet, error) {
	wallet, exists := s.data.Wallets[name]
	if !exists {
		return nil, fmt.Errorf("wallet with name '%s' not found", name)
	}
	return wallet, nil
}

// UpdateWallet updates an existing wallet
func (s *Storage) UpdateWallet(name string, wallet *model.Wallet) error {
	if _, exists := s.data.Wallets[name]; !exists {
		return fmt.Errorf("wallet with name '%s' not found", name)
	}

	// If the name is changing, delete the old entry
	if name != wallet.Name {
		delete(s.data.Wallets, name)
	}

	s.data.Wallets[wallet.Name] = wallet
	return s.save()
}

// DeleteWallet deletes a wallet
func (s *Storage) DeleteWallet(name string) error {
	if _, exists := s.data.Wallets[name]; !exists {
		return fmt.Errorf("wallet with name '%s' not found", name)
	}

	delete(s.data.Wallets, name)
	return s.save()
}

// ListWallets returns all wallets
func (s *Storage) ListWallets() []*model.Wallet {
	wallets := make([]*model.Wallet, 0, len(s.data.Wallets))
	for _, wallet := range s.data.Wallets {
		wallets = append(wallets, wallet)
	}
	return wallets
}

// AddCategory adds a new category
func (s *Storage) AddCategory(category *model.Category) error {
	if _, exists := s.data.Categories[category.Name]; exists {
		return fmt.Errorf("category with name '%s' already exists", category.Name)
	}

	s.data.Categories[category.Name] = category
	return s.save()
}

// GetCategory gets a category by name
func (s *Storage) GetCategory(name string) (*model.Category, error) {
	category, exists := s.data.Categories[name]
	if !exists {
		return nil, fmt.Errorf("category with name '%s' not found", name)
	}
	return category, nil
}

// DeleteCategory deletes a category
func (s *Storage) DeleteCategory(name string) error {
	if _, exists := s.data.Categories[name]; !exists {
		return fmt.Errorf("category with name '%s' not found", name)
	}

	delete(s.data.Categories, name)

	// Update wallets that use this category
	for _, wallet := range s.data.Wallets {
		if wallet.Category == name {
			wallet.Category = ""
		}
	}

	return s.save()
}

// ListCategories returns all categories
func (s *Storage) ListCategories() []*model.Category {
	categories := make([]*model.Category, 0, len(s.data.Categories))
	for _, category := range s.data.Categories {
		categories = append(categories, category)
	}
	return categories
}

// AddContact adds a new contact
func (s *Storage) AddContact(contact *model.Contact) error {
	if _, exists := s.data.Contacts[contact.Name]; exists {
		return fmt.Errorf("contact with name '%s' already exists", contact.Name)
	}

	s.data.Contacts[contact.Name] = contact
	return s.save()
}

// GetContact gets a contact by name
func (s *Storage) GetContact(name string) (*model.Contact, error) {
	contact, exists := s.data.Contacts[name]
	if !exists {
		return nil, fmt.Errorf("contact with name '%s' not found", name)
	}
	return contact, nil
}

// DeleteContact deletes a contact
func (s *Storage) DeleteContact(name string) error {
	if _, exists := s.data.Contacts[name]; !exists {
		return fmt.Errorf("contact with name '%s' not found", name)
	}

	delete(s.data.Contacts, name)
	return s.save()
}

// ListContacts returns all contacts
func (s *Storage) ListContacts() []*model.Contact {
	contacts := make([]*model.Contact, 0, len(s.data.Contacts))
	for _, contact := range s.data.Contacts {
		contacts = append(contacts, contact)
	}
	return contacts
}

// AddTransaction adds a transaction and updates wallet balances
func (s *Storage) AddTransaction(tx *model.Tx) error {
	// Check for duplicate
	if tx.ID != "" && s.txIndex[tx.ID] {
		return fmt.Errorf("transaction with ID '%s' already exists", tx.ID)
	}

	// Validate and update balances based on tx type
	switch tx.Type {
	case model.TxTypeDeposit:
		wallet, err := s.GetWallet(tx.ToWallet)
		if err != nil {
			return err
		}
		s.updateBalance(wallet, tx.Coin, tx.Amount)

	case model.TxTypeWithdraw:
		wallet, err := s.GetWallet(tx.FromWallet)
		if err != nil {
			return err
		}
		s.updateBalance(wallet, tx.Coin, -tx.Amount)

	case model.TxTypeTransfer:
		var fromWallet, toWallet *model.Wallet
		var fromErr, toErr error

		if tx.FromWallet != "" {
			fromWallet, fromErr = s.GetWallet(tx.FromWallet)
		}
		if tx.ToWallet != "" {
			toWallet, toErr = s.GetWallet(tx.ToWallet)
		}

		if fromErr != nil && toErr != nil {
			return fmt.Errorf("both source and destination wallets are invalid")
		}

		if fromWallet != nil {
			s.updateBalance(fromWallet, tx.Coin, -tx.Amount)
		}
		if toWallet != nil {
			s.updateBalance(toWallet, tx.Coin, tx.Amount)
		}

	case model.TxTypeSwap:
		wallet, err := s.GetWallet(tx.SwapWallet)
		if err != nil {
			return err
		}
		s.updateBalance(wallet, tx.SellCoin, -tx.SellAmount)
		s.updateBalance(wallet, tx.BuyCoin, tx.BuyAmount)
	}

	// Store transaction in global map
	s.data.Transactions[tx.ID] = tx
	s.txIndex[tx.ID] = true

	return s.save()
}

// DeleteTransaction deletes a transaction and reverses balance changes
func (s *Storage) DeleteTransaction(txID string) error {
	tx, exists := s.data.Transactions[txID]
	if !exists {
		return fmt.Errorf("transaction with ID '%s' not found", txID)
	}

	// Reverse balance changes
	switch tx.Type {
	case model.TxTypeDeposit:
		if wallet, err := s.GetWallet(tx.ToWallet); err == nil {
			s.updateBalance(wallet, tx.Coin, -tx.Amount)
		}

	case model.TxTypeWithdraw:
		if wallet, err := s.GetWallet(tx.FromWallet); err == nil {
			s.updateBalance(wallet, tx.Coin, tx.Amount)
		}

	case model.TxTypeTransfer:
		if wallet, err := s.GetWallet(tx.FromWallet); err == nil {
			s.updateBalance(wallet, tx.Coin, tx.Amount)
		}
		if wallet, err := s.GetWallet(tx.ToWallet); err == nil {
			s.updateBalance(wallet, tx.Coin, -tx.Amount)
		}

	case model.TxTypeSwap:
		if wallet, err := s.GetWallet(tx.SwapWallet); err == nil {
			s.updateBalance(wallet, tx.SellCoin, tx.SellAmount)
			s.updateBalance(wallet, tx.BuyCoin, -tx.BuyAmount)
		}
	}

	// Remove from storage
	delete(s.data.Transactions, txID)
	delete(s.txIndex, txID)

	return s.save()
}

// GetTransaction returns a transaction by ID
func (s *Storage) GetTransaction(txID string) (*model.Tx, error) {
	tx, exists := s.data.Transactions[txID]
	if !exists {
		return nil, fmt.Errorf("transaction with ID '%s' not found", txID)
	}
	return tx, nil
}

// ListTransactions returns all transactions
func (s *Storage) ListTransactions() []*model.Tx {
	txs := make([]*model.Tx, 0, len(s.data.Transactions))
	for _, tx := range s.data.Transactions {
		txs = append(txs, tx)
	}
	return txs
}

// GetWalletTransactions returns transactions for a specific wallet
func (s *Storage) GetWalletTransactions(walletName string) []*model.Tx {
	var txs []*model.Tx
	for _, tx := range s.data.Transactions {
		if tx.FromWallet == walletName || tx.ToWallet == walletName || tx.SwapWallet == walletName {
			txs = append(txs, tx)
		}
	}
	return txs
}

// updateBalance updates a wallet's balance for a specific coin
func (s *Storage) updateBalance(wallet *model.Wallet, coin string, amount float64) {
	if wallet.Balances == nil {
		wallet.Balances = []*model.Balance{}
	}
	
	// Find existing balance for this coin
	for _, balance := range wallet.Balances {
		if balance.Coin == coin {
			balance.Amount += amount
			return
		}
	}
	
	// If no existing balance, create a new one
	wallet.Balances = append(wallet.Balances, &model.Balance{
		Coin:   coin,
		Amount: amount,
	})
}

// GenerateTxID generates a unique transaction ID
func (s *Storage) GenerateTxID() string {
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}

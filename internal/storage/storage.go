package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vasylcode/wago/internal/model"
)

// Storage handles the persistence of data
type Storage struct {
	dataDir     string
	walletsFile string
	categoriesFile string
	contactsFile   string
	wallets    map[string]*model.Wallet
	categories map[string]*model.Category
	contacts   map[string]*model.Contact
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
		dataDir:        dataDir,
		walletsFile:    filepath.Join(dataDir, "wallets.json"),
		categoriesFile: filepath.Join(dataDir, "categories.json"),
		contactsFile:   filepath.Join(dataDir, "contacts.json"),
		wallets:        make(map[string]*model.Wallet),
		categories:     make(map[string]*model.Category),
		contacts:       make(map[string]*model.Contact),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// load loads all data from disk
func (s *Storage) load() error {
	if err := s.loadWallets(); err != nil {
		return err
	}
	if err := s.loadCategories(); err != nil {
		return err
	}
	if err := s.loadContacts(); err != nil {
		return err
	}
	return nil
}

// loadWallets loads wallets from disk
func (s *Storage) loadWallets() error {
	if _, err := os.Stat(s.walletsFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.walletsFile)
	if err != nil {
		return fmt.Errorf("failed to read wallets file: %w", err)
	}

	var wallets map[string]*model.Wallet
	if err := json.Unmarshal(data, &wallets); err != nil {
		return fmt.Errorf("failed to unmarshal wallets: %w", err)
	}

	s.wallets = wallets
	return nil
}

// loadCategories loads categories from disk
func (s *Storage) loadCategories() error {
	if _, err := os.Stat(s.categoriesFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.categoriesFile)
	if err != nil {
		return fmt.Errorf("failed to read categories file: %w", err)
	}

	var categories map[string]*model.Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return fmt.Errorf("failed to unmarshal categories: %w", err)
	}

	s.categories = categories
	return nil
}

// loadContacts loads contacts from disk
func (s *Storage) loadContacts() error {
	if _, err := os.Stat(s.contactsFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.contactsFile)
	if err != nil {
		return fmt.Errorf("failed to read contacts file: %w", err)
	}

	var contacts map[string]*model.Contact
	if err := json.Unmarshal(data, &contacts); err != nil {
		return fmt.Errorf("failed to unmarshal contacts: %w", err)
	}

	s.contacts = contacts
	return nil
}

// saveWallets saves wallets to disk
func (s *Storage) saveWallets() error {
	data, err := json.MarshalIndent(s.wallets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallets: %w", err)
	}

	if err := os.WriteFile(s.walletsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write wallets file: %w", err)
	}

	return nil
}

// saveCategories saves categories to disk
func (s *Storage) saveCategories() error {
	data, err := json.MarshalIndent(s.categories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	if err := os.WriteFile(s.categoriesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write categories file: %w", err)
	}

	return nil
}

// saveContacts saves contacts to disk
func (s *Storage) saveContacts() error {
	data, err := json.MarshalIndent(s.contacts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contacts: %w", err)
	}

	if err := os.WriteFile(s.contactsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write contacts file: %w", err)
	}

	return nil
}

// AddWallet adds a new wallet
func (s *Storage) AddWallet(wallet *model.Wallet) error {
	if _, exists := s.wallets[wallet.Name]; exists {
		return fmt.Errorf("wallet with name '%s' already exists", wallet.Name)
	}

	s.wallets[wallet.Name] = wallet
	return s.saveWallets()
}

// GetWallet gets a wallet by name
func (s *Storage) GetWallet(name string) (*model.Wallet, error) {
	wallet, exists := s.wallets[name]
	if !exists {
		return nil, fmt.Errorf("wallet with name '%s' not found", name)
	}
	return wallet, nil
}

// UpdateWallet updates an existing wallet
func (s *Storage) UpdateWallet(name string, wallet *model.Wallet) error {
	if _, exists := s.wallets[name]; !exists {
		return fmt.Errorf("wallet with name '%s' not found", name)
	}

	// If the name is changing, delete the old entry
	if name != wallet.Name {
		delete(s.wallets, name)
	}

	s.wallets[wallet.Name] = wallet
	return s.saveWallets()
}

// DeleteWallet deletes a wallet
func (s *Storage) DeleteWallet(name string) error {
	if _, exists := s.wallets[name]; !exists {
		return fmt.Errorf("wallet with name '%s' not found", name)
	}

	delete(s.wallets, name)
	return s.saveWallets()
}

// ListWallets returns all wallets
func (s *Storage) ListWallets() []*model.Wallet {
	wallets := make([]*model.Wallet, 0, len(s.wallets))
	for _, wallet := range s.wallets {
		wallets = append(wallets, wallet)
	}
	return wallets
}

// AddCategory adds a new category
func (s *Storage) AddCategory(category *model.Category) error {
	if _, exists := s.categories[category.Name]; exists {
		return fmt.Errorf("category with name '%s' already exists", category.Name)
	}

	s.categories[category.Name] = category
	return s.saveCategories()
}

// GetCategory gets a category by name
func (s *Storage) GetCategory(name string) (*model.Category, error) {
	category, exists := s.categories[name]
	if !exists {
		return nil, fmt.Errorf("category with name '%s' not found", name)
	}
	return category, nil
}

// DeleteCategory deletes a category
func (s *Storage) DeleteCategory(name string) error {
	if _, exists := s.categories[name]; !exists {
		return fmt.Errorf("category with name '%s' not found", name)
	}

	delete(s.categories, name)

	// Update wallets that use this category
	for _, wallet := range s.wallets {
		if wallet.Category == name {
			wallet.Category = ""
		}
	}

	if err := s.saveWallets(); err != nil {
		return err
	}

	return s.saveCategories()
}

// ListCategories returns all categories
func (s *Storage) ListCategories() []*model.Category {
	categories := make([]*model.Category, 0, len(s.categories))
	for _, category := range s.categories {
		categories = append(categories, category)
	}
	return categories
}

// AddContact adds a new contact
func (s *Storage) AddContact(contact *model.Contact) error {
	if _, exists := s.contacts[contact.Name]; exists {
		return fmt.Errorf("contact with name '%s' already exists", contact.Name)
	}

	s.contacts[contact.Name] = contact
	return s.saveContacts()
}

// GetContact gets a contact by name
func (s *Storage) GetContact(name string) (*model.Contact, error) {
	contact, exists := s.contacts[name]
	if !exists {
		return nil, fmt.Errorf("contact with name '%s' not found", name)
	}
	return contact, nil
}

// DeleteContact deletes a contact
func (s *Storage) DeleteContact(name string) error {
	if _, exists := s.contacts[name]; !exists {
		return fmt.Errorf("contact with name '%s' not found", name)
	}

	delete(s.contacts, name)
	return s.saveContacts()
}

// ListContacts returns all contacts
func (s *Storage) ListContacts() []*model.Contact {
	contacts := make([]*model.Contact, 0, len(s.contacts))
	for _, contact := range s.contacts {
		contacts = append(contacts, contact)
	}
	return contacts
}

// AddTransaction adds a transaction to a wallet and updates balances
func (s *Storage) AddTransaction(tx *model.Tx) error {
	switch tx.Type {
	case model.TxTypeDeposit:
		// Handle deposit (add to wallet)
		toWallet, err := s.GetWallet(tx.ToWallet)
		if err != nil {
			return err
		}
		
		// Add transaction
		if toWallet.Txs == nil {
			toWallet.Txs = []*model.Tx{}
		}
		toWallet.Txs = append(toWallet.Txs, tx)
		
		// Update balance
		s.updateBalance(toWallet, tx.Coin, tx.Amount)
		
	case model.TxTypeWithdraw:
		// Handle withdraw (subtract from wallet)
		fromWallet, err := s.GetWallet(tx.FromWallet)
		if err != nil {
			return err
		}
		
		// Add transaction
		if fromWallet.Txs == nil {
			fromWallet.Txs = []*model.Tx{}
		}
		fromWallet.Txs = append(fromWallet.Txs, tx)
		
		// Update balance
		s.updateBalance(fromWallet, tx.Coin, -tx.Amount)
		
	case model.TxTypeTransfer:
		// Handle transfer (subtract from one wallet, add to another)
		// For transfers, at least one of FromWallet or ToWallet must be a valid wallet
		var fromWallet, toWallet *model.Wallet
		var fromErr, toErr error
		
		if tx.FromWallet != "" {
			fromWallet, fromErr = s.GetWallet(tx.FromWallet)
		}
		
		if tx.ToWallet != "" {
			toWallet, toErr = s.GetWallet(tx.ToWallet)
		}
		
		// Check if we have at least one valid wallet
		if fromErr != nil && toErr != nil {
			return fmt.Errorf("both source and destination wallets are invalid")
		}
		
		// Add transaction to wallets and update balances
		if fromWallet != nil {
			if fromWallet.Txs == nil {
				fromWallet.Txs = []*model.Tx{}
			}
			fromWallet.Txs = append(fromWallet.Txs, tx)
			s.updateBalance(fromWallet, tx.Coin, -tx.Amount)
		}
		
		if toWallet != nil {
			if toWallet.Txs == nil {
				toWallet.Txs = []*model.Tx{}
			}
			toWallet.Txs = append(toWallet.Txs, tx)
			s.updateBalance(toWallet, tx.Coin, tx.Amount)
		}
		
	case model.TxTypeSwap:
		// Handle swap transaction (sell one coin, buy another in same wallet)
		swapWallet, err := s.GetWallet(tx.SwapWallet)
		if err != nil {
			return err
		}
		
		// Add transaction to wallet
		if swapWallet.Txs == nil {
			swapWallet.Txs = []*model.Tx{}
		}
		swapWallet.Txs = append(swapWallet.Txs, tx)
		
		// Update balances: subtract sold coin, add bought coin
		s.updateBalance(swapWallet, tx.SellCoin, -tx.SellAmount)
		s.updateBalance(swapWallet, tx.BuyCoin, tx.BuyAmount)
	}
	
	return s.saveWallets()
}

// DeleteTransaction deletes a transaction and updates balances
func (s *Storage) DeleteTransaction(walletName, txID string) error {
	wallet, err := s.GetWallet(walletName)
	if err != nil {
		return err
	}
	
	var foundTx *model.Tx
	var foundIndex int
	
	for i, tx := range wallet.Txs {
		if tx.ID == txID {
			foundTx = tx
			foundIndex = i
			break
		}
	}
	
	if foundTx == nil {
		return fmt.Errorf("transaction with ID '%s' not found in wallet '%s'", txID, walletName)
	}
	
	// Remove transaction
	wallet.Txs = append(wallet.Txs[:foundIndex], wallet.Txs[foundIndex+1:]...)
	
	// Reverse the balance change
	switch foundTx.Type {
	case model.TxTypeDeposit:
		s.updateBalance(wallet, foundTx.Coin, -foundTx.Amount)
		
	case model.TxTypeWithdraw:
		s.updateBalance(wallet, foundTx.Coin, foundTx.Amount)
		
	case model.TxTypeTransfer:
		if walletName == foundTx.FromWallet {
			s.updateBalance(wallet, foundTx.Coin, foundTx.Amount)
			
			// Also update the other wallet
			otherWallet, err := s.GetWallet(foundTx.ToWallet)
			if err == nil {
				// Find and remove the transaction from the other wallet
				for i, tx := range otherWallet.Txs {
					if tx.ID == txID {
						otherWallet.Txs = append(otherWallet.Txs[:i], otherWallet.Txs[i+1:]...)
						s.updateBalance(otherWallet, foundTx.Coin, -foundTx.Amount)
						break
					}
				}
			}
		} else {
			s.updateBalance(wallet, foundTx.Coin, -foundTx.Amount)
			
			// Also update the other wallet
			otherWallet, err := s.GetWallet(foundTx.FromWallet)
			if err == nil {
				// Find and remove the transaction from the other wallet
				for i, tx := range otherWallet.Txs {
					if tx.ID == txID {
						otherWallet.Txs = append(otherWallet.Txs[:i], otherWallet.Txs[i+1:]...)
						s.updateBalance(otherWallet, foundTx.Coin, foundTx.Amount)
						break
					}
				}
			}
		}
		
	case model.TxTypeSwap:
		// Reverse the swap: add back sold coin, subtract bought coin
		s.updateBalance(wallet, foundTx.SellCoin, foundTx.SellAmount)
		s.updateBalance(wallet, foundTx.BuyCoin, -foundTx.BuyAmount)
	}
	
	return s.saveWallets()
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

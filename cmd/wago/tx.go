package wago

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
)

var (
	txFromWallet string
	txToWallet   string
	txCoin       string
	txAmount     float64
	txNote       string
	txFee        float64
	txFeeCoin    string
	txSwapWallet string
	txSellCoin   string
	txSellAmount float64
	txBuyCoin    string
	txBuyAmount  float64
)

func init() {
	// Transaction command
	txCmd := &cobra.Command{
		Use:     "tx",
		Short:   "Manage transactions",
		Long:    `Add and delete transactions.`,
		Run:     listTransactions,
	}

	// Add subcommand
	addTxCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new transaction",
		Long:  `Add a new transaction with the specified properties.`,
		Run:   addTransaction,
	}

	// Delete subcommand
	delTxCmd := &cobra.Command{
		Use:   "del [wallet] [txID]",
		Short: "Delete a transaction",
		Long:  `Delete a transaction from a wallet.`,
		Args:  cobra.ExactArgs(2),
		Run:   deleteTransaction,
	}

	// Add flags to add command
	addTxCmd.Flags().StringVarP(&txFromWallet, "from", "f", "", "Source wallet name (for withdraw or transfer)")
	addTxCmd.Flags().StringVarP(&txToWallet, "to", "t", "", "Destination wallet name (for deposit or transfer)")
	addTxCmd.Flags().StringVarP(&txSwapWallet, "swap", "s", "", "Wallet name for swap transaction")
	addTxCmd.Flags().StringVarP(&txCoin, "coin", "c", "", "Coin/token symbol")
	addTxCmd.Flags().Float64VarP(&txAmount, "amount", "a", 0, "Transaction amount")
	addTxCmd.Flags().StringVarP(&txNote, "note", "n", "", "Transaction note")
	addTxCmd.Flags().Float64VarP(&txFee, "fee", "F", 0, "Transaction fee amount")
	addTxCmd.Flags().StringVarP(&txFeeCoin, "fee-coin", "C", "", "Fee coin (defaults to transaction coin if not specified)")
	addTxCmd.Flags().StringVarP(&txSellCoin, "sell-coin", "S", "", "Coin to sell (swap transactions)")
	addTxCmd.Flags().Float64VarP(&txSellAmount, "sell-amount", "A", 0, "Amount to sell (swap transactions)")
	addTxCmd.Flags().StringVarP(&txBuyCoin, "buy-coin", "B", "", "Coin to buy (swap transactions)")
	addTxCmd.Flags().Float64VarP(&txBuyAmount, "buy-amount", "M", 0, "Amount to buy (swap transactions)")

	// Add subcommands to tx command
	txCmd.AddCommand(addTxCmd)
	txCmd.AddCommand(delTxCmd)

	// Add tx command to root command
	rootCmd.AddCommand(txCmd)
}

func addTransaction(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	// Validate transaction type based on provided flags
	if txFromWallet == "" && txToWallet == "" && txSwapWallet == "" {
		er("Either --from, --to, or --swap wallet must be specified")
		return
	}

	// Check for swap transaction
	if txSwapWallet != "" {
		// Validate swap-specific fields
		if txSellCoin == "" || txBuyCoin == "" {
			er("For swap transactions, both --sell-coin and --buy-coin must be specified")
			return
		}
		if txSellAmount <= 0 || txBuyAmount <= 0 {
			er("For swap transactions, both --sell-amount and --buy-amount must be greater than zero")
			return
		}
	} else {
		// Validate non-swap transactions
		if txCoin == "" {
			er("Coin must be specified with --coin flag")
			return
		}
		if txAmount <= 0 {
			er("Amount must be greater than zero")
			return
		}
	}

	// Determine transaction type
	var txType model.TxType
	var fromAddress, toAddress string

	if txSwapWallet != "" {
		// Handle swap transaction
		txType = model.TxTypeSwap
		
		// Verify wallet exists
		_, err := s.GetWallet(txSwapWallet)
		if err != nil {
			er(fmt.Sprintf("Swap wallet '%s' not found", txSwapWallet))
			return
		}
		
	} else if txFromWallet != "" && txToWallet != "" {
		// Transfer between wallets
		txType = model.TxTypeTransfer

		// Verify both wallets exist
		fromWallet, err := s.GetWallet(txFromWallet)
		if err != nil {
			// Check if it's a contact
			contact, err := s.GetContact(txFromWallet)
			if err != nil {
				er(fmt.Sprintf("Source wallet or contact '%s' not found", txFromWallet))
				return
			}
			fromAddress = contact.Address
			// Keep txFromWallet as the contact name
		} else {
			fromAddress = fromWallet.Address
		}

		toWallet, err := s.GetWallet(txToWallet)
		if err != nil {
			// Check if it's a contact
			contact, err := s.GetContact(txToWallet)
			if err != nil {
				er(fmt.Sprintf("Destination wallet or contact '%s' not found", txToWallet))
				return
			}
			toAddress = contact.Address
			// Keep txToWallet as the contact name
		} else {
			toAddress = toWallet.Address
		}

		// If both are contacts, that's invalid
		if fromWallet == nil && toWallet == nil {
			er("Cannot transfer between two contacts")
			return
		}

	} else if txFromWallet != "" {
		// Withdraw from wallet
		txType = model.TxTypeWithdraw

		// Verify wallet exists
		fromWallet, err := s.GetWallet(txFromWallet)
		if err != nil {
			er(fmt.Sprintf("Source wallet '%s' not found", txFromWallet))
			return
		}
		fromAddress = fromWallet.Address

		// If to address is a contact, use its address
		if txToWallet != "" {
			contact, err := s.GetContact(txToWallet)
			if err == nil {
				toAddress = contact.Address
				// Keep txToWallet as the contact name
			} else {
				toAddress = txToWallet // Assume it's an address
			}
		}

	} else if txToWallet != "" {
		// Deposit to wallet
		txType = model.TxTypeDeposit

		// Verify wallet exists
		toWallet, err := s.GetWallet(txToWallet)
		if err != nil {
			er(fmt.Sprintf("Destination wallet '%s' not found", txToWallet))
			return
		}
		toAddress = toWallet.Address

		// If from address is a contact, use its address
		if txFromWallet != "" {
			contact, err := s.GetContact(txFromWallet)
			if err == nil {
				fromAddress = contact.Address
				// Keep txFromWallet as the contact name
			} else {
				fromAddress = txFromWallet // Assume it's an address
			}
		}
	}

	// Determine fee coin (default to transaction coin if not specified)
	feeCoin := txFeeCoin
	if txFee > 0 && feeCoin == "" {
		if txCoin != "" {
			feeCoin = txCoin
		} else if txSellCoin != "" {
			feeCoin = txSellCoin
		}
	}

	// Create and add the transaction
	tx := &model.Tx{
		ID:          s.GenerateTxID(),
		Type:        txType,
		FromWallet:  txFromWallet,
		ToWallet:    txToWallet,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Coin:        txCoin,
		Amount:      txAmount,
		Fee:         txFee,
		FeeCoin:     feeCoin,
		SwapWallet:  txSwapWallet,
		SellCoin:    txSellCoin,
		SellAmount:  txSellAmount,
		BuyCoin:     txBuyCoin,
		BuyAmount:   txBuyAmount,
		Date:        time.Now(),
		Note:        txNote,
	}

	if err := s.AddTransaction(tx); err != nil {
		er(fmt.Sprintf("Failed to add transaction: %v", err))
		return
	}

	fmt.Printf("Transaction added successfully\n")
}

func deleteTransaction(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	walletName := args[0]
	txID := args[1]

	if err := s.DeleteTransaction(walletName, txID); err != nil {
		er(fmt.Sprintf("Failed to delete transaction: %v", err))
		return
	}

	fmt.Printf("Transaction deleted successfully\n")
}

func listTransactions(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	// Get all wallets to access their transactions
	wallets := s.ListWallets()
	if len(wallets) == 0 {
		fmt.Println("No wallets found")
		return
	}

	// Collect all transactions from all wallets, deduplicating by ID
	// (transfers are stored in both source and destination wallets)
	var allTxs []*model.Tx
	seenTxIDs := make(map[string]bool)
	walletMap := make(map[string]*model.Wallet)
	
	for _, wallet := range wallets {
		walletMap[wallet.Name] = wallet
		for _, tx := range wallet.Txs {
			if !seenTxIDs[tx.ID] {
				seenTxIDs[tx.ID] = true
				allTxs = append(allTxs, tx)
			}
		}
	}

	// Sort transactions by date (newest first)
	sort.Slice(allTxs, func(i, j int) bool {
		return allTxs[i].Date.After(allTxs[j].Date)
	})

	// Print transactions with enhanced formatting
	titleColor := color.New(color.Bold, color.Underline)
	titleColor.Println("Recent Transactions:")

	for _, tx := range allTxs {
		// Create colored elements for transaction
		txTypeColor := color.New(color.Bold)
		amountColor := color.New(color.FgGreen)
		amountPrefix := "+"
		
		// Set colors based on transaction type
		switch tx.Type {
		case model.TxTypeDeposit:
			txTypeColor = color.New(color.FgGreen, color.Bold)
		case model.TxTypeWithdraw:
			txTypeColor = color.New(color.FgRed, color.Bold)
			amountColor = color.New(color.FgRed)
			amountPrefix = "-"
		case model.TxTypeTransfer:
			txTypeColor = color.New(color.FgYellow, color.Bold)
			// For transfers in the global list, we don't use +/- prefixes
			amountPrefix = ""
		case model.TxTypeSwap:
			txTypeColor = color.New(color.FgMagenta, color.Bold)
			// For swaps, we'll show a special format
			amountPrefix = ""
		}
		
		// Format transaction details
		txType := string(tx.Type)
		coloredType := txTypeColor.Sprint(strings.ToUpper(txType))
		
		// Format amount and details based on transaction type
		var coloredAmount, coloredCoin, details string
		
		switch tx.Type {
		case model.TxTypeSwap:
			// Special formatting for swap transactions
			sellColor := color.New(color.FgRed)
			buyColor := color.New(color.FgGreen)
			coloredAmount = fmt.Sprintf("%s %s → %s %s",
				sellColor.Sprintf("-%.2f", tx.SellAmount),
				color.New(color.Bold).Sprint(tx.SellCoin),
				buyColor.Sprintf("+%.2f", tx.BuyAmount),
				color.New(color.Bold).Sprint(tx.BuyCoin))
			coloredCoin = ""
			details = fmt.Sprintf("in %s", tx.SwapWallet)
		default:
			// Standard formatting for other transaction types
			coloredAmount = amountColor.Sprintf("%s%.2f", amountPrefix, tx.Amount)
			coloredCoin = color.New(color.Bold).Sprint(tx.Coin)
			
			switch tx.Type {
			case model.TxTypeDeposit:
				details = fmt.Sprintf("to %s", tx.ToWallet)
			case model.TxTypeWithdraw:
				details = fmt.Sprintf("from %s", tx.FromWallet)
			case model.TxTypeTransfer:
				details = fmt.Sprintf("from %s to %s", tx.FromWallet, tx.ToWallet)
			}
		}
		
		// Format date in local time with color
		localTime := tx.Date.Local()
		dateStr := color.New(color.FgHiBlack).Sprintf("[%s]", localTime.Format("2006-01-02 15:04"))
		
		// Format fee if present
		feeStr := ""
		if tx.Fee > 0 {
			feeCoin := tx.FeeCoin
			if feeCoin == "" {
				feeCoin = tx.Coin
			}
			feeStr = color.New(color.FgHiBlack).Sprintf(" [fee: %.2f %s]", tx.Fee, feeCoin)
		}

		// Format note with color if present
		noteStr := ""
		if tx.Note != "" {
			noteStr = color.New(color.FgYellow).Sprintf(" \"%s\"", tx.Note)
		}
		
		// Print the transaction with all the colored elements
		if tx.Type == model.TxTypeSwap {
			// For swap transactions, coloredCoin is empty so we skip it
			fmt.Printf("  %s %s %s %s%s%s\n", 
				coloredType,
				coloredAmount, 
				details,
				dateStr,
				feeStr,
				noteStr)
		} else {
			// For other transactions, include the coin
			fmt.Printf("  %s %s %s %s %s%s%s\n", 
				coloredType,
				coloredAmount, 
				coloredCoin, 
				details,
				dateStr,
				feeStr,
				noteStr)
		}
	}
}

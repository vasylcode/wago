package wago

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
	"github.com/vasylcode/wago/internal/util"
)

var (
	walletAddress  string
	walletCategory string
	walletChain    string
	walletType     string
	walletNote     string
	showBalances   bool
	showTxs        bool
)

func init() {
	// Wallet command
	walletCmd := &cobra.Command{
		Use:     "wallet",
		Aliases: []string{"w"},
		Short:   "Manage wallets",
		Long:    `Add, delete, update, and list wallets.`,
		Run:     listWallets,
	}

	// Add subcommand
	addWalletCmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new wallet",
		Long:  `Add a new wallet with the specified name and properties.`,
		Args:  cobra.ExactArgs(1),
		Run:   addWallet,
	}

	// Delete subcommand
	delWalletCmd := &cobra.Command{
		Use:   "del [name]",
		Short: "Delete a wallet",
		Long:  `Delete a wallet and all its transactions and balances.`,
		Args:  cobra.ExactArgs(1),
		Run:   deleteWallet,
	}

	// Update subcommand
	updWalletCmd := &cobra.Command{
		Use:   "upd [name]",
		Short: "Update a wallet",
		Long:  `Update a wallet's properties.`,
		Args:  cobra.ExactArgs(1),
		Run:   updateWallet,
	}

	// Add flags to add command
	addWalletCmd.Flags().StringVarP(&walletAddress, "address", "a", "", "Wallet address")
	addWalletCmd.Flags().StringVarP(&walletCategory, "category", "c", "", "Wallet category (optional)")
	addWalletCmd.Flags().StringVarP(&walletChain, "chain", "n", "", "Blockchain")
	addWalletCmd.Flags().StringVarP(&walletType, "type", "t", "", "Wallet type")
	addWalletCmd.Flags().StringVarP(&walletNote, "note", "", "", "Note to describe wallet")

	addWalletCmd.MarkFlagRequired("address")
	addWalletCmd.MarkFlagRequired("chain")
	addWalletCmd.MarkFlagRequired("type")

	// Add flags to update command
	updWalletCmd.Flags().StringVarP(&walletAddress, "address", "a", "", "Wallet address")
	updWalletCmd.Flags().StringVarP(&walletCategory, "category", "c", "", "Wallet category (optional)")
	updWalletCmd.Flags().StringVarP(&walletChain, "chain", "n", "", "Blockchain")
	updWalletCmd.Flags().StringVarP(&walletType, "type", "t", "", "Wallet type")
	updWalletCmd.Flags().StringVarP(&walletNote, "note", "", "", "Note to describe wallet")

	// Add flags to wallet list command
	walletCmd.Flags().BoolVarP(&showBalances, "balances", "b", false, "Show wallet balances")
	walletCmd.Flags().BoolVarP(&showTxs, "txs", "t", false, "Show wallet transactions")

	// Add subcommands to wallet command
	walletCmd.AddCommand(addWalletCmd)
	walletCmd.AddCommand(delWalletCmd)
	walletCmd.AddCommand(updWalletCmd)

	// Add wallet command to root command
	rootCmd.AddCommand(walletCmd)
}

func addWallet(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	
	// Check if category exists, if not, create it
	if walletCategory != "" {
		_, err := s.GetCategory(walletCategory)
		if err != nil {
			// Category doesn't exist, create it with a random color
			category := &model.Category{
				Name:  walletCategory,
				Color: generateRandomColor(),
			}
			if err := s.AddCategory(category); err != nil {
				er(fmt.Sprintf("Failed to create category: %v", err))
				return
			}
			fmt.Printf("Category '%s' created automatically with color %s\n", walletCategory, category.Color)
		}
	}
	
	wallet := &model.Wallet{
		Name:     name,
		Address:  walletAddress,
		Category: walletCategory,
		Chain:    walletChain,
		Type:     walletType,
		Note:     walletNote,
	}

	if err := s.AddWallet(wallet); err != nil {
		er(fmt.Sprintf("Failed to add wallet: %v", err))
		return
	}

	fmt.Printf("Wallet '%s' added successfully\n", name)
}

func deleteWallet(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	if err := s.DeleteWallet(name); err != nil {
		er(fmt.Sprintf("Failed to delete wallet: %v", err))
		return
	}

	fmt.Printf("Wallet '%s' deleted successfully\n", name)
}

func updateWallet(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	wallet, err := s.GetWallet(name)
	if err != nil {
		er(fmt.Sprintf("Failed to get wallet: %v", err))
		return
	}

	// Update only the fields that were provided
	if cmd.Flags().Changed("address") {
		wallet.Address = walletAddress
	}
	if cmd.Flags().Changed("category") {
		wallet.Category = walletCategory
	}
	if cmd.Flags().Changed("chain") {
		wallet.Chain = walletChain
	}
	if cmd.Flags().Changed("type") {
		wallet.Type = walletType
	}
	if cmd.Flags().Changed("note") {
		wallet.Note = walletNote
	}

	if err := s.UpdateWallet(name, wallet); err != nil {
		er(fmt.Sprintf("Failed to update wallet: %v", err))
		return
	}

	fmt.Printf("Wallet '%s' updated successfully\n", name)
}

func listWallets(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	// If a specific wallet name is provided, show that wallet
	if len(args) == 1 {
		showWallet(s, args[0])
		return
	}

	// Otherwise, list all wallets
	wallets := s.ListWallets()
	if len(wallets) == 0 {
		fmt.Println("No wallets found")
		return
	}

	// Get categories for coloring
	categories := s.ListCategories()
	categoryColors := make(map[string]*color.Color)
	for _, cat := range categories {
		// Convert color name to terminal color
		colorName := cat.Color
		if colorName == "" {
			colorName = "white" // Default to white
		}
		
		// Create a color based on the terminal color name
		categoryColors[cat.Name] = util.GetTerminalColor(colorName, color.FgHiWhite)
	}

	// Print wallets
	for _, wallet := range wallets {
		var txs []*model.Tx
		if showTxs {
			txs = s.GetWalletTransactions(wallet.Name)
		}
		printWallet(wallet, categoryColors, showBalances, showTxs, txs)
	}
}

func showWallet(s *storage.Storage, name string) {
	wallet, err := s.GetWallet(name)
	if err != nil {
		er(fmt.Sprintf("Failed to get wallet: %v", err))
		return
	}

	// Get categories for coloring
	categories := s.ListCategories()
	categoryColors := make(map[string]*color.Color)
	for _, cat := range categories {
		// Convert color name to terminal color
		colorName := cat.Color
		if colorName == "" {
			colorName = "white" // Default to white
		}
		
		// Create a color based on the terminal color name
		categoryColors[cat.Name] = util.GetTerminalColor(colorName, color.FgHiWhite)
	}

	// Always show balances and transactions for a specific wallet
	txs := s.GetWalletTransactions(wallet.Name)
	printWallet(wallet, categoryColors, true, true, txs)
}

func printWallet(wallet *model.Wallet, categoryColors map[string]*color.Color, showBalances, showTxs bool, txs []*model.Tx) {
	// Format the wallet information
	catPrefix := ""
	if wallet.Category != "" {
		catPrefix = fmt.Sprintf("[%s] ", wallet.Category)
		
		// Apply color if available
		if col, ok := categoryColors[wallet.Category]; ok {
			catPrefix = col.Sprint(catPrefix)
		}
	}
	
	// Create colored elements
	boldName := color.New(color.Bold).Sprint(wallet.Name)
	grayAddress := color.New(color.FgHiBlack).Sprintf("(%s)", wallet.Address)
	blueChainType := color.New(color.FgBlue).Sprintf("%s-%s", wallet.Chain, wallet.Type)
	
	noteStr := ""
	if wallet.Note != "" {
		noteStr = color.New(color.FgYellow).Sprintf(" (%s)", wallet.Note)
	}
	
	fmt.Printf("%s%s %s %s%s\n", 
		catPrefix, 
		boldName, 
		grayAddress, 
		blueChainType,
		noteStr)
	
	// Show balances if requested
	if showBalances && len(wallet.Balances) > 0 {
		fmt.Println("  Balances:")
		
		// Get coin symbols for price fetching
		coins := make([]string, 0, len(wallet.Balances))
		for _, balance := range wallet.Balances {
			coins = append(coins, balance.Coin)
		}
		
		// Fetch USD prices from manual prices.json
		prices, err := util.GetCoinPrices(coins)
		
		for _, balance := range wallet.Balances {
			// Round to 2 decimal places for display
			displayAmount := fmt.Sprintf("%.2f", balance.Amount)
			
			// Color based on amount (green for positive, red for negative)
			amountColor := color.New(color.FgGreen)
			if balance.Amount < 0 {
				amountColor = color.New(color.FgRed)
			}
			
			coloredAmount := amountColor.Sprint(displayAmount)
			coinName := color.New(color.Bold).Sprint(balance.Coin)
			
			// Add USD value if available
			usdStr := ""
			if err == nil {
				if price, exists := prices[strings.ToLower(balance.Coin)]; exists {
					usdValue := balance.Amount * price
					usdColor := color.New(color.FgHiBlack)
					if balance.Amount < 0 {
						usdColor = color.New(color.FgRed)
					}
					usdStr = usdColor.Sprintf(" (%s)", util.FormatUSDValue(usdValue))
				}
			}
			
			fmt.Printf("    %s: %s%s\n", coinName, coloredAmount, usdStr)
		}
	}
	
	// Show transactions if requested
	if showTxs && len(txs) > 0 {
		fmt.Println("  Transactions:")
		for _, tx := range txs {
			txType := string(tx.Type)
			details := ""
			
			switch tx.Type {
			case model.TxTypeDeposit:
				details = fmt.Sprintf("from %s", tx.FromAddress)
			case model.TxTypeWithdraw:
				details = fmt.Sprintf("to %s", tx.ToAddress)
			case model.TxTypeTransfer:
				details = fmt.Sprintf("from %s to %s", tx.FromWallet, tx.ToWallet)
			}
			
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
				// For transfers, color depends on whether this wallet is sender or receiver
				if tx.FromWallet == wallet.Name {
					amountColor = color.New(color.FgRed)
					amountPrefix = "-"
				} else {
					amountColor = color.New(color.FgGreen)
				}
			}
			
			// Format amount with prefix and color, rounded to 2 decimals
			coloredAmount := amountColor.Sprintf("%s%.2f", amountPrefix, tx.Amount)
			coloredType := txTypeColor.Sprint(strings.ToUpper(txType))
			coloredCoin := color.New(color.Bold).Sprint(tx.Coin)
			
			// Format details with colors
			coloredDetails := color.New(color.FgHiBlack).Sprint(details)
			
			// Format date in local time
			localTime := tx.Date.Local()
			dateStr := color.New(color.FgHiBlack).Sprintf("[%s]", localTime.Format("2006-01-02 15:04"))
			
			noteStr := ""
			if tx.Note != "" {
				noteStr = color.New(color.FgYellow).Sprintf(" (%s)", tx.Note)
			}
			
			fmt.Printf("    %s: %s %s %s %s%s\n", 
				coloredType, 
				coloredAmount, 
				coloredCoin, 
				coloredDetails,
				dateStr,
				noteStr)
		}
	}
}

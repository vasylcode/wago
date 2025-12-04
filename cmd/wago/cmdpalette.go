package wago

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
)

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success  bool
	Message  string
	IsHelp   bool   // Show as popup
	HelpText string // Multi-line help content
	Quit     bool   // Signal to quit app
}

// CommandPalette handles command parsing and execution
type CommandPalette struct {
	storage *storage.Storage
	history []string
	histIdx int
}

// NewCommandPalette creates a new command palette
func NewCommandPalette(s *storage.Storage) *CommandPalette {
	return &CommandPalette{
		storage: s,
		history: []string{},
		histIdx: -1,
	}
}

// Execute parses and executes a command string
func (cp *CommandPalette) Execute(input string) CommandResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return CommandResult{Success: false, Message: ""}
	}

	// Add to history
	cp.history = append(cp.history, input)
	cp.histIdx = len(cp.history)

	// Parse command
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return CommandResult{Success: false, Message: "Empty command"}
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "q", "quit", "exit":
		return CommandResult{Quit: true}
	case "add", "a":
		return cp.cmdAdd(args)
	case "del", "d", "delete", "rm":
		return cp.cmdDelete(args)
	case "deposit", "dep":
		return cp.cmdDeposit(args)
	case "withdraw", "wd":
		return cp.cmdWithdraw(args)
	case "transfer", "tf":
		return cp.cmdTransfer(args)
	case "swap", "sw":
		return cp.cmdSwap(args)
	case "balance", "bal", "b":
		return cp.cmdBalance(args)
	case "price", "p":
		return cp.cmdPrice(args)
	case "help", "h", "?":
		return cp.cmdHelp()
	default:
		return CommandResult{Success: false, Message: fmt.Sprintf("Unknown command: %s (:help for commands)", cmd)}
	}
}

// GetHistory returns previous command (for up arrow)
func (cp *CommandPalette) GetHistory(direction int) string {
	if len(cp.history) == 0 {
		return ""
	}
	cp.histIdx += direction
	if cp.histIdx < 0 {
		cp.histIdx = 0
	}
	if cp.histIdx >= len(cp.history) {
		cp.histIdx = len(cp.history)
		return ""
	}
	return cp.history[cp.histIdx]
}

// --- Command implementations ---

func (cp *CommandPalette) cmdAdd(args []string) CommandResult {
	if len(args) < 1 {
		return CommandResult{Success: false, Message: "Usage: add wallet|category|contact ..."}
	}

	sub := strings.ToLower(args[0])
	subArgs := args[1:]

	switch sub {
	case "wallet", "w":
		// add wallet <name> <address> <chain-type> [category] [note]
		// chain-type format: solana-hot, eth-cold, btc-exchange
		if len(subArgs) < 3 {
			return CommandResult{Success: false, Message: "Usage: add wallet NAME ADDR CHAIN-TYPE (CAT) (NOTE)"}
		}
		
		// Parse chain-type (e.g., "solana-hot" -> chain="solana", type="hot")
		chainType := subArgs[2]
		chain := chainType
		walletType := "hot" // default
		if idx := strings.LastIndex(chainType, "-"); idx > 0 {
			chain = chainType[:idx]
			walletType = chainType[idx+1:]
		}
		
		wallet := &model.Wallet{
			Name:    subArgs[0],
			Address: subArgs[1],
			Chain:   chain,
			Type:    walletType,
		}
		if len(subArgs) > 3 {
			wallet.Category = subArgs[3]
		}
		if len(subArgs) > 4 {
			wallet.Note = strings.Join(subArgs[4:], " ")
		}
		if err := cp.storage.AddWallet(wallet); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Added wallet: %s (%s-%s)", wallet.Name, chain, walletType)}

	case "category", "cat", "c":
		// add category <name> [color]
		if len(subArgs) < 1 {
			return CommandResult{Success: false, Message: "Usage: add category NAME (COLOR)"}
		}
		
		// Random color if not specified
		colors := []string{"red", "green", "blue", "yellow", "magenta", "cyan", "orange", "pink", "purple"}
		randomColor := colors[rand.Intn(len(colors))]
		
		cat := &model.Category{
			Name:  subArgs[0],
			Color: randomColor,
		}
		if len(subArgs) > 1 {
			cat.Color = subArgs[1]
		}
		if err := cp.storage.AddCategory(cat); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Added category: %s (%s)", cat.Name, cat.Color)}

	case "contact", "con":
		// add contact <name> <address> [chain] [note]
		if len(subArgs) < 2 {
			return CommandResult{Success: false, Message: "Usage: add contact NAME ADDR (CHAIN) (NOTE)"}
		}
		contact := &model.Contact{
			Name:    subArgs[0],
			Address: subArgs[1],
		}
		if len(subArgs) > 2 {
			contact.Chain = subArgs[2]
		}
		if len(subArgs) > 3 {
			contact.Note = strings.Join(subArgs[3:], " ")
		}
		if err := cp.storage.AddContact(contact); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Added contact: %s", contact.Name)}

	default:
		return CommandResult{Success: false, Message: fmt.Sprintf("Unknown type: %s (use wallet, category, or contact)", sub)}
	}
}

func (cp *CommandPalette) cmdDelete(args []string) CommandResult {
	if len(args) < 2 {
		return CommandResult{Success: false, Message: "Usage: del wallet|category|contact|tx NAME|ID"}
	}

	sub := strings.ToLower(args[0])
	name := args[1]

	switch sub {
	case "wallet", "w":
		if err := cp.storage.DeleteWallet(name); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Deleted wallet: %s", name)}

	case "category", "cat", "c":
		if err := cp.storage.DeleteCategory(name); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Deleted category: %s", name)}

	case "contact", "con":
		if err := cp.storage.DeleteContact(name); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Deleted contact: %s", name)}

	case "tx", "transaction":
		if err := cp.storage.DeleteTransaction(name); err != nil {
			return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
		}
		return CommandResult{Success: true, Message: fmt.Sprintf("Deleted transaction: %s", name)}

	default:
		return CommandResult{Success: false, Message: fmt.Sprintf("Unknown type: %s", sub)}
	}
}

func (cp *CommandPalette) cmdDeposit(args []string) CommandResult {
	// deposit <wallet> <amount> <coin> [note]
	if len(args) < 3 {
		return CommandResult{Success: false, Message: "Usage: deposit WALLET AMOUNT COIN (NOTE)"}
	}

	wallet := args[0]
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid amount: %s", args[1])}
	}
	coin := strings.ToUpper(args[2])

	tx := &model.Tx{
		ID:       cp.storage.GenerateTxID(),
		Type:     model.TxTypeDeposit,
		ToWallet: wallet,
		Coin:     coin,
		Amount:   amount,
		Date:     time.Now(),
	}
	if len(args) > 3 {
		tx.Note = strings.Join(args[3:], " ")
	}

	if err := cp.storage.AddTransaction(tx); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Deposited %.2f %s to %s", amount, coin, wallet)}
}

func (cp *CommandPalette) cmdWithdraw(args []string) CommandResult {
	// withdraw <wallet> <amount> <coin> [note]
	if len(args) < 3 {
		return CommandResult{Success: false, Message: "Usage: withdraw WALLET AMOUNT COIN (NOTE)"}
	}

	wallet := args[0]
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid amount: %s", args[1])}
	}
	coin := strings.ToUpper(args[2])

	tx := &model.Tx{
		ID:         cp.storage.GenerateTxID(),
		Type:       model.TxTypeWithdraw,
		FromWallet: wallet,
		Coin:       coin,
		Amount:     amount,
		Date:       time.Now(),
	}
	if len(args) > 3 {
		tx.Note = strings.Join(args[3:], " ")
	}

	if err := cp.storage.AddTransaction(tx); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Withdrew %.2f %s from %s", amount, coin, wallet)}
}

func (cp *CommandPalette) cmdTransfer(args []string) CommandResult {
	// transfer <from> <to> <amount> <coin> [note]
	if len(args) < 4 {
		return CommandResult{Success: false, Message: "Usage: transfer FROM TO AMOUNT COIN (NOTE)"}
	}

	from := args[0]
	to := args[1]
	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid amount: %s", args[2])}
	}
	coin := strings.ToUpper(args[3])

	tx := &model.Tx{
		ID:         cp.storage.GenerateTxID(),
		Type:       model.TxTypeTransfer,
		FromWallet: from,
		ToWallet:   to,
		Coin:       coin,
		Amount:     amount,
		Date:       time.Now(),
	}
	if len(args) > 4 {
		tx.Note = strings.Join(args[4:], " ")
	}

	if err := cp.storage.AddTransaction(tx); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Transferred %.2f %s: %s → %s", amount, coin, from, to)}
}

func (cp *CommandPalette) cmdSwap(args []string) CommandResult {
	// swap <wallet> <sell_amount> <sell_coin> <buy_amount> <buy_coin> [note]
	if len(args) < 5 {
		return CommandResult{Success: false, Message: "Usage: swap WALLET SELL_AMT SELL_COIN BUY_AMT BUY_COIN (NOTE)"}
	}

	wallet := args[0]
	sellAmount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid sell amount: %s", args[1])}
	}
	sellCoin := strings.ToUpper(args[2])
	buyAmount, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid buy amount: %s", args[3])}
	}
	buyCoin := strings.ToUpper(args[4])

	tx := &model.Tx{
		ID:         cp.storage.GenerateTxID(),
		Type:       model.TxTypeSwap,
		SwapWallet: wallet,
		SellCoin:   sellCoin,
		SellAmount: sellAmount,
		BuyCoin:    buyCoin,
		BuyAmount:  buyAmount,
		Date:       time.Now(),
	}
	if len(args) > 5 {
		tx.Note = strings.Join(args[5:], " ")
	}

	if err := cp.storage.AddTransaction(tx); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Swapped %.2f %s → %.2f %s in %s", sellAmount, sellCoin, buyAmount, buyCoin, wallet)}
}

func (cp *CommandPalette) cmdBalance(args []string) CommandResult {
	// balance <wallet> <amount> <coin>
	if len(args) < 3 {
		return CommandResult{Success: false, Message: "Usage: balance WALLET AMOUNT COIN"}
	}

	walletName := args[0]
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid amount: %s", args[1])}
	}
	coin := strings.ToUpper(args[2])

	wallet, err := cp.storage.GetWallet(walletName)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}

	// Find or create balance
	found := false
	for _, bal := range wallet.Balances {
		if strings.EqualFold(bal.Coin, coin) {
			bal.Amount = amount
			found = true
			break
		}
	}
	if !found {
		wallet.Balances = append(wallet.Balances, &model.Balance{Coin: coin, Amount: amount})
	}

	if err := cp.storage.UpdateWallet(walletName, wallet); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Set %s balance: %.2f %s", walletName, amount, coin)}
}

func (cp *CommandPalette) cmdPrice(args []string) CommandResult {
	// price <coin> <usd_price>
	if len(args) < 2 {
		return CommandResult{Success: false, Message: "Usage: price COIN USD_PRICE"}
	}

	coin := strings.ToLower(args[0])
	price, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Invalid price: %s", args[1])}
	}

	if err := cp.storage.SetPrice(coin, price); err != nil {
		return CommandResult{Success: false, Message: fmt.Sprintf("Error: %v", err)}
	}
	return CommandResult{Success: true, Message: fmt.Sprintf("Set %s price: $%.2f", strings.ToUpper(coin), price)}
}

func (cp *CommandPalette) cmdHelp() CommandResult {
	help := `[yellow]Commands:[white]

[green]add wallet[white] NAME ADDR CHAIN-TYPE (CAT) (NOTE)

[green]add category[white] NAME (COLOR)
[green]add contact[white] NAME ADDR (CHAIN) (NOTE)

[green]del[white] wallet|category|contact|tx NAME|ID

[green]deposit[white] WALLET AMOUNT COIN (NOTE)
[green]withdraw[white] WALLET AMOUNT COIN (NOTE)
[green]transfer[white] FROM TO AMOUNT COIN (NOTE)
[green]swap[white] WALLET SELL_AMT SELL_COIN BUY_AMT BUY_COIN

[green]balance[white] WALLET AMOUNT COIN
[green]price[white] COIN USD_PRICE

[green]q[white] quit

[yellow]Shortcuts:[white] a=add d=del dep=deposit wd=withdraw
          tf=transfer sw=swap b=balance p=price`
	return CommandResult{Success: true, IsHelp: true, HelpText: help}
}

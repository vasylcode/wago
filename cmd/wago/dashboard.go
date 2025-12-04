package wago

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
	"github.com/vasylcode/wago/internal/util"
)

func init() {
	// Dashboard command
	dashboardCmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d"},
		Short:   "Display wallet statistics dashboard",
		Long:    `Display a dashboard with wallet statistics, balances by coin, category distribution, and other metrics.`,
		Run:     showDashboard,
	}

	rootCmd.AddCommand(dashboardCmd)
}

// ViewMode represents the current dashboard view
type ViewMode int

const (
	ViewMain ViewMode = iota
	ViewStats
)

// StatsState holds the state for the stats view
type StatsState struct {
	Months       []string // sorted month keys (newest first)
	CurrentMonth int      // index into Months
}

// MainDashboardState holds the state for the main dashboard
type MainDashboardState struct {
	SelectedWallet int
}

func showDashboard(cmd *cobra.Command, args []string) {
	// Create a new tview application
	app := tview.NewApplication()
	
	// Current view mode
	currentView := ViewMain
	
	// Stats state
	statsState := &StatsState{}
	
	// Main dashboard state
	mainState := &MainDashboardState{SelectedWallet: 0}

	// buildMainDashboard creates the main dashboard UI
	buildMainDashboard := func(s *storage.Storage, wallets []*model.Wallet, categories []*model.Category) *tview.Flex {
		// Create a flex layout for the main container
		flex := tview.NewFlex().SetDirection(tview.FlexRow)

		// Add a header
		header := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[::b][#00FFFF]Wallet Aggregator[white] [#666666]│[white] [#FF6600]Balances[white]")
		header.SetBorder(true)
		flex.AddItem(header, 3, 0, false)

		// Sort wallets by name
		sort.Slice(wallets, func(i, j int) bool {
			return wallets[i].Name < wallets[j].Name
		})

		// Ensure selected wallet is in bounds
		if mainState.SelectedWallet >= len(wallets) {
			mainState.SelectedWallet = 0
		}

		// Get selected wallet
		var selectedWallet *model.Wallet
		if len(wallets) > 0 {
			selectedWallet = wallets[mainState.SelectedWallet]
		}

		// TOP SECTION (80% height): Wallets | Balances | Transactions
		topSection := tview.NewFlex().SetDirection(tview.FlexColumn)

		// Wallets panel (30% width)
		walletsView := createWalletsPanel(wallets, categories, mainState.SelectedWallet)
		topSection.AddItem(walletsView, 0, 30, false)

		// Balances panel (20% width)
		balancesView := createWalletBalancesPanel(selectedWallet)
		topSection.AddItem(balancesView, 0, 20, false)

		// Transactions panel (50% width)
		var walletTxs []*model.Tx
		if selectedWallet != nil {
			walletTxs = s.GetWalletTransactions(selectedWallet.Name)
		}
		txsView := createWalletTransactionsPanel(walletTxs)
		topSection.AddItem(txsView, 0, 50, false)

		// BOTTOM SECTION (20% height): Total Balance | Category Balance | Category Distribution
		bottomSection := tview.NewFlex().SetDirection(tview.FlexColumn)

		// Total Balance by Coin (larger)
		totalBalanceView := createTotalBalanceView(wallets)
		bottomSection.AddItem(totalBalanceView, 0, 2, false)

		// Balance by Category (smaller, middle)
		categoryBalanceView := createCategoryBalanceView(wallets, categories)
		bottomSection.AddItem(categoryBalanceView, 0, 1, false)

		// Category Distribution (larger)
		categoryChartView := createCategoryChartView(wallets, categories)
		bottomSection.AddItem(categoryChartView, 0, 2, false)

		// Add sections to main flex
		flex.AddItem(topSection, 0, 4, false)    // 80%
		flex.AddItem(bottomSection, 0, 1, false) // 20%

		// Add a footer with instructions
		footer := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[::b][#AAAAAA]Press [#FFFFFF]:[#AAAAAA] commands | [#FFFFFF]↑↓[#AAAAAA] select wallet | [#FFFFFF]Enter[#AAAAAA] copy addr | [#FFFFFF]s[#AAAAAA] stats | [#FFFFFF]r[#AAAAAA] reload")
		footer.SetBorder(false)
		flex.AddItem(footer, 1, 0, false)

		return flex
	}

	// buildStatsDashboard creates the stats dashboard UI with month tabs
	buildStatsDashboard := func(s *storage.Storage, wallets []*model.Wallet, categories []*model.Category) *tview.Flex {
		// Collect all transactions
		allTxs := collectAllTransactions(s)
		
		// Group transactions by month
		txsByMonth := groupTransactionsByMonth(allTxs)
		
		// Get sorted month keys (newest first)
		if len(statsState.Months) == 0 || statsState.CurrentMonth >= len(txsByMonth) {
			statsState.Months = getSortedMonthKeys(txsByMonth)
			statsState.CurrentMonth = 0
		}

		// Create a flex layout for the main container
		flex := tview.NewFlex().SetDirection(tview.FlexRow)

		// Add a header with month navigation
		currentMonthDisplay := "No transactions"
		if len(statsState.Months) > 0 && statsState.CurrentMonth < len(statsState.Months) {
			monthKey := statsState.Months[statsState.CurrentMonth]
			currentMonthDisplay = formatMonthKey(monthKey)
		}
		
		header := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText(fmt.Sprintf("[::b][#00FFFF]Wallet Aggregator[white] [#666666]│[white] [#FF6600]Flow & Transactions[white]\n[#666666]◀[white] [::b]%s[:-] [#666666]▶[white]", currentMonthDisplay))
		header.SetBorder(true)
		flex.AddItem(header, 4, 0, false)

		// Main content area - horizontal split (flow left, transactions right)
		contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

		// Flow canvas (left side) - horizontally scrollable
		var flowCanvas *tview.TextView
		var monthTxs []*model.Tx
		if len(statsState.Months) > 0 && statsState.CurrentMonth < len(statsState.Months) {
			monthKey := statsState.Months[statsState.CurrentMonth]
			monthTxs = txsByMonth[monthKey]
			flowCanvas = createFlowCanvas(monthTxs, wallets)
		} else {
			flowCanvas = tview.NewTextView().
				SetDynamicColors(true).
				SetText("[#AAAAAA]No transactions found[white]")
			flowCanvas.SetBorder(true).SetTitle(" Flow ")
		}
		contentFlex.AddItem(flowCanvas, 0, 2, false)  // ~65% of width

		// Transactions panel (right side, filtered by current month)
		transactionsView := createTransactionsView(monthTxs)
		contentFlex.AddItem(transactionsView, 0, 1, false)  // ~35% of width

		// Add the content flex to the main flex
		flex.AddItem(contentFlex, 0, 1, true)

		// Add a footer with instructions
		footer := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[::b][#AAAAAA]Press [#FFFFFF]:[#AAAAAA] commands | [#FFFFFF]←/→[#AAAAAA] month | [#FFFFFF]s[#AAAAAA] balances | [#FFFFFF]r[#AAAAAA] reload")
		footer.SetBorder(false)
		flex.AddItem(footer, 1, 0, false)

		return flex
	}

	// Initialize storage once
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	// buildDashboard creates the appropriate dashboard based on current view
	buildDashboard := func() *tview.Flex {
		// Reload storage data
		s, _ = storage.New()

		// Get all wallets
		wallets := s.ListWallets()
		if len(wallets) == 0 {
			// Return a simple message view with help
			flex := tview.NewFlex().SetDirection(tview.FlexRow)
			msg := tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetDynamicColors(true).
				SetText("[yellow]No wallets found.[white]\n\nPress [yellow]:[white] and type:\n[green]add wallet <name> <address> <chain>[white]")
			flex.AddItem(msg, 0, 1, false)
			return flex
		}

		// Get all categories
		categories := s.ListCategories()

		switch currentView {
		case ViewStats:
			return buildStatsDashboard(s, wallets, categories)
		default:
			return buildMainDashboard(s, wallets, categories)
		}
	}

	// Command palette
	cmdPalette := NewCommandPalette(s)
	
	// Command input field
	cmdInput := tview.NewInputField().
		SetLabel(": ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorYellow)
	
	// Status message
	statusMsg := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	statusMsg.SetBackgroundColor(tcell.ColorBlack)
	
	// Command mode flag
	cmdMode := false
	
	// Build the full UI with command palette
	buildFullUI := func() *tview.Flex {
		dashboard := buildDashboard()
		if dashboard == nil {
			return nil
		}
		
		// Bottom bar with status and input
		bottomBar := tview.NewFlex().SetDirection(tview.FlexColumn)
		if cmdMode {
			bottomBar.AddItem(cmdInput, 0, 1, true)
		} else {
			bottomBar.AddItem(statusMsg, 0, 1, false)
		}
		
		// Main layout
		main := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(dashboard, 0, 1, !cmdMode).
			AddItem(bottomBar, 1, 0, cmdMode)
		
		return main
	}
	
	// Status message timeout
	var statusTimeout *time.Timer
	
	// Update status message
	setStatus := func(msg string, isError bool) {
		if statusTimeout != nil {
			statusTimeout.Stop()
		}
		if isError {
			statusMsg.SetText("[red]" + msg + "[white]")
		} else if msg != "" {
			statusMsg.SetText("[green]" + msg + "[white]")
			// Auto-clear success messages after 3 seconds
			statusTimeout = time.AfterFunc(3*time.Second, func() {
				app.QueueUpdateDraw(func() {
					statusMsg.SetText("")
				})
			})
		} else {
			statusMsg.SetText("") // Empty by default
		}
	}
	setStatus("", false)
	
	// Show help popup
	showHelp := func(helpText string) {
		modal := tview.NewModal().
			SetText(helpText).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(buildFullUI(), true)
			})
		modal.SetBackgroundColor(tcell.ColorBlack)
		
		// Use a frame for better styling
		frame := tview.NewFrame(modal).
			SetBorders(1, 1, 1, 1, 1, 1)
		frame.SetBackgroundColor(tcell.ColorBlack)
		
		app.SetRoot(frame, true)
	}
	
	// Handle command input
	cmdInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			input := cmdInput.GetText()
			cmdInput.SetText("")
			cmdMode = false
			
			if input != "" {
				result := cmdPalette.Execute(input)
				
				// Handle quit
				if result.Quit {
					app.Stop()
					return
				}
				
				// Handle help popup
				if result.IsHelp {
					showHelp(result.HelpText)
					return
				}
				
				setStatus(result.Message, !result.Success)
				
				// Reload dashboard
				statsState.Months = nil
				app.SetRoot(buildFullUI(), true)
			} else {
				setStatus("", false)
				app.SetRoot(buildFullUI(), true)
			}
		} else if key == tcell.KeyEscape {
			cmdInput.SetText("")
			cmdMode = false
			setStatus("", false)
			app.SetRoot(buildFullUI(), true)
		}
	})
	
	// Build initial UI
	flex := buildFullUI()
	if flex == nil {
		return
	}

	// Set up keyboard shortcuts
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// In command mode, let input field handle everything
		if cmdMode {
			if event.Key() == tcell.KeyUp {
				cmdInput.SetText(cmdPalette.GetHistory(-1))
				return nil
			}
			if event.Key() == tcell.KeyDown {
				cmdInput.SetText(cmdPalette.GetHistory(1))
				return nil
			}
			return event
		}
		
		// Normal mode shortcuts - only : to enter command mode
		if event.Rune() == ':' {
			cmdMode = true
			setStatus("", false)
			app.SetRoot(buildFullUI(), true)
			app.SetFocus(cmdInput)
			return nil
		}
		if event.Rune() == 'r' {
			statsState.Months = nil
			setStatus("Reloaded", false)
			app.SetRoot(buildFullUI(), true)
			return nil
		}
		if event.Rune() == 's' {
			if currentView == ViewMain {
				currentView = ViewStats
			} else {
				currentView = ViewMain
			}
			app.SetRoot(buildFullUI(), true)
			return nil
		}
		// Arrow keys for navigation
		if currentView == ViewStats && len(statsState.Months) > 0 {
			// Stats view: left/right for month navigation
			if event.Key() == tcell.KeyLeft {
				if statsState.CurrentMonth > 0 {
					statsState.CurrentMonth--
					app.SetRoot(buildFullUI(), true)
				}
				return nil
			}
			if event.Key() == tcell.KeyRight {
				if statsState.CurrentMonth < len(statsState.Months)-1 {
					statsState.CurrentMonth++
					app.SetRoot(buildFullUI(), true)
				}
				return nil
			}
		}
		
		// Main view: up/down for wallet selection
		if currentView == ViewMain {
			walletCount := len(s.ListWallets())
			if event.Key() == tcell.KeyUp {
				if mainState.SelectedWallet > 0 {
					mainState.SelectedWallet--
					app.SetRoot(buildFullUI(), true)
				}
				return nil
			}
			if event.Key() == tcell.KeyDown {
				if mainState.SelectedWallet < walletCount-1 {
					mainState.SelectedWallet++
					app.SetRoot(buildFullUI(), true)
				}
				return nil
			}
			// Enter to copy wallet address
			if event.Key() == tcell.KeyEnter {
				wallets := s.ListWallets()
				sort.Slice(wallets, func(i, j int) bool {
					return wallets[i].Name < wallets[j].Name
				})
				if mainState.SelectedWallet < len(wallets) {
					addr := wallets[mainState.SelectedWallet].Address
					// Use pbcopy on macOS
					copyCmd := exec.Command("pbcopy")
					copyCmd.Stdin = strings.NewReader(addr)
					if err := copyCmd.Run(); err == nil {
						setStatus("Copied: "+addr, false)
						app.SetRoot(buildFullUI(), true)
					}
				}
				return nil
			}
		}
		return event
	})

	// Set the root and run the application
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		er(fmt.Sprintf("Failed to run dashboard: %v", err))
	}
}

// groupTransactionsByMonth groups transactions by year-month
func groupTransactionsByMonth(txs []*model.Tx) map[string][]*model.Tx {
	result := make(map[string][]*model.Tx)
	for _, tx := range txs {
		key := fmt.Sprintf("%d-%02d", tx.Date.Year(), tx.Date.Month())
		result[key] = append(result[key], tx)
	}
	return result
}

// getSortedMonthKeys returns month keys sorted newest first
func getSortedMonthKeys(txsByMonth map[string][]*model.Tx) []string {
	keys := make([]string, 0, len(txsByMonth))
	for k := range txsByMonth {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys
}

// formatMonthKey formats "2025-04" to "April 2025"
func formatMonthKey(key string) string {
	t, err := time.Parse("2006-01", key)
	if err != nil {
		return key
	}
	return t.Format("January 2006")
}

// FlowNode represents a node in the flow diagram
type FlowNode struct {
	Name      string
	Address   string
	IsWallet  bool
	IsContact bool
}

// FlowEdge represents an edge (transaction) between nodes
type FlowEdge struct {
	From      string
	To        string
	Coin      string
	Amount    float64
	Count     int
	Dates     []time.Time
	TxType    model.TxType
	// For swaps
	SellCoin   string
	SellAmount float64
	BuyCoin    string
	BuyAmount  float64
}

// createFlowCanvas creates the flow visualization for a month's transactions
func createFlowCanvas(txs []*model.Tx, wallets []*model.Wallet) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(false) // Allow horizontal scrolling

	view.SetBorder(true).SetTitle(" Monthly Flow ")

	if len(txs) == 0 {
		view.SetText("[#AAAAAA]No transactions this month[white]")
		return view
	}

	// Build wallet lookup
	walletNames := make(map[string]bool)
	walletAddresses := make(map[string]string) // name -> address
	for _, w := range wallets {
		walletNames[w.Name] = true
		walletAddresses[w.Name] = w.Address
	}

	// Aggregate edges by from->to->coin
	edgeKey := func(from, to, coin string, txType model.TxType) string {
		return fmt.Sprintf("%s|%s|%s|%s", from, to, coin, txType)
	}
	
	edges := make(map[string]*FlowEdge)
	swaps := []*FlowEdge{} // Keep swaps separate

	for _, tx := range txs {
		switch tx.Type {
		case model.TxTypeDeposit:
			from := "External"
			to := tx.ToWallet
			key := edgeKey(from, to, tx.Coin, tx.Type)
			if e, exists := edges[key]; exists {
				e.Amount += tx.Amount
				e.Count++
				e.Dates = append(e.Dates, tx.Date)
			} else {
				edges[key] = &FlowEdge{
					From:   from,
					To:     to,
					Coin:   tx.Coin,
					Amount: tx.Amount,
					Count:  1,
					Dates:  []time.Time{tx.Date},
					TxType: tx.Type,
				}
			}

		case model.TxTypeWithdraw:
			from := tx.FromWallet
			to := "External"
			key := edgeKey(from, to, tx.Coin, tx.Type)
			if e, exists := edges[key]; exists {
				e.Amount += tx.Amount
				e.Count++
				e.Dates = append(e.Dates, tx.Date)
			} else {
				edges[key] = &FlowEdge{
					From:   from,
					To:     to,
					Coin:   tx.Coin,
					Amount: tx.Amount,
					Count:  1,
					Dates:  []time.Time{tx.Date},
					TxType: tx.Type,
				}
			}

		case model.TxTypeTransfer:
			from := tx.FromWallet
			to := tx.ToWallet
			// If to is empty but ToAddress exists, it's to a contact
			if to == "" && tx.ToAddress != "" {
				to = tx.ToAddress
				if len(to) > 10 {
					to = to[:6] + "..." + to[len(to)-4:]
				}
			}
			if from == "" && tx.FromAddress != "" {
				from = tx.FromAddress
				if len(from) > 10 {
					from = from[:6] + "..." + from[len(from)-4:]
				}
			}
			key := edgeKey(from, to, tx.Coin, tx.Type)
			if e, exists := edges[key]; exists {
				e.Amount += tx.Amount
				e.Count++
				e.Dates = append(e.Dates, tx.Date)
			} else {
				edges[key] = &FlowEdge{
					From:   from,
					To:     to,
					Coin:   tx.Coin,
					Amount: tx.Amount,
					Count:  1,
					Dates:  []time.Time{tx.Date},
					TxType: tx.Type,
				}
			}

		case model.TxTypeSwap:
			swaps = append(swaps, &FlowEdge{
				From:       tx.SwapWallet,
				SellCoin:   tx.SellCoin,
				SellAmount: tx.SellAmount,
				BuyCoin:    tx.BuyCoin,
				BuyAmount:  tx.BuyAmount,
				Count:      1,
				Dates:      []time.Time{tx.Date},
				TxType:     tx.Type,
			})
		}
	}

	// Build the flow visualization
	var content strings.Builder

	// Helper to format address snippet
	addrSnippet := func(name string) string {
		if addr, exists := walletAddresses[name]; exists && len(addr) > 8 {
			return fmt.Sprintf("[#666666](%s...%s)[white]", addr[:4], addr[len(addr)-4:])
		}
		return ""
	}

	// Helper to format dates
	formatDates := func(dates []time.Time) string {
		if len(dates) == 1 {
			return dates[0].Format("Jan 02")
		}
		// Sort dates
		sort.Slice(dates, func(i, j int) bool {
			return dates[i].Before(dates[j])
		})
		return fmt.Sprintf("%s - %s", dates[0].Format("Jan 02"), dates[len(dates)-1].Format("Jan 02"))
	}

	// Collect all edges and sort by date
	allEdges := make([]*FlowEdge, 0, len(edges))
	for _, e := range edges {
		allEdges = append(allEdges, e)
	}
	sort.Slice(allEdges, func(i, j int) bool {
		if len(allEdges[i].Dates) == 0 || len(allEdges[j].Dates) == 0 {
			return false
		}
		return allEdges[i].Dates[0].Before(allEdges[j].Dates[0])
	})

	// Helper to render a node header (wallet/external)
	renderNodeHeader := func(name string) string {
		if name == "External" {
			return "[#888888]External[white]"
		}
		addr := addrSnippet(name)
		if walletNames[name] {
			return fmt.Sprintf("[#00FFFF]%s[white] %s", name, addr)
		}
		// Contact or external address
		return fmt.Sprintf("[#FF6600]%s[white]", name)
	}

	// Helper to render a target node (for arrow destination)
	renderTargetNode := func(name string) string {
		if name == "External" {
			return "[#888888]External[white]"
		}
		if walletNames[name] {
			return fmt.Sprintf("[#00FFFF]%s[white]", name)
		}
		// Contact or external address
		return fmt.Sprintf("[#FF6600]%s[white]", name)
	}

	// Group edges by source wallet
	edgesBySource := make(map[string][]*FlowEdge)
	sourceOrder := []string{} // Track order of sources
	seenSources := make(map[string]bool)
	
	for _, edge := range allEdges {
		if !seenSources[edge.From] {
			seenSources[edge.From] = true
			sourceOrder = append(sourceOrder, edge.From)
		}
		edgesBySource[edge.From] = append(edgesBySource[edge.From], edge)
	}

	// Calculate max widths for alignment
	maxAmountLen := 0
	maxCoinLen := 0
	maxCountLen := 0
	maxTargetLen := 0
	maxDateLen := 0

	// Helper to get plain target name (without color codes)
	plainTargetName := func(name string) string {
		if name == "External" {
			return "External"
		}
		return name
	}

	for _, edge := range allEdges {
		amountStr := fmt.Sprintf("%.2f", edge.Amount)
		if len(amountStr) > maxAmountLen {
			maxAmountLen = len(amountStr)
		}
		if len(edge.Coin) > maxCoinLen {
			maxCoinLen = len(edge.Coin)
		}
		countStr := ""
		if edge.Count > 1 {
			countStr = fmt.Sprintf("(×%d)", edge.Count)
		}
		if len(countStr) > maxCountLen {
			maxCountLen = len(countStr)
		}
		targetName := plainTargetName(edge.To)
		if len(targetName) > maxTargetLen {
			maxTargetLen = len(targetName)
		}
		dateStr := formatDates(edge.Dates)
		if len(dateStr) > maxDateLen {
			maxDateLen = len(dateStr)
		}
	}

	// Ensure minimum widths
	if maxCountLen == 0 {
		maxCountLen = 0 // No padding needed if no counts
	}

	content.WriteString("\n")

	// Render each source wallet with its outgoing edges
	for _, source := range sourceOrder {
		edges := edgesBySource[source]
		if len(edges) == 0 {
			continue
		}

		// Render source node header
		content.WriteString(renderNodeHeader(source) + "\n")

		// Render each outgoing edge
		for i, edge := range edges {
			// Arrow color based on type
			var arrowColor string
			switch edge.TxType {
			case model.TxTypeDeposit:
				arrowColor = "#00FF00"
			case model.TxTypeWithdraw:
				arrowColor = "#FF5555"
			case model.TxTypeTransfer:
				arrowColor = "#FFFF00"
			}

			// Tree branch character
			branch := "├──"
			if i == len(edges)-1 {
				branch = "└──"
			}

			// Format each part with padding
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, edge.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, edge.Coin)
			
			countStr := ""
			if edge.Count > 1 {
				countStr = fmt.Sprintf("(×%d)", edge.Count)
			}
			countPadded := fmt.Sprintf("%-*s", maxCountLen, countStr)
			
			targetName := plainTargetName(edge.To)
			target := renderTargetNode(edge.To)
			// Add padding after colored target name
			if len(targetName) < maxTargetLen {
				target = target + strings.Repeat(" ", maxTargetLen-len(targetName))
			}
			
			dateLabel := formatDates(edge.Dates)

			// Build the line: branch + amount + coin + count + arrow + target + date
			// Arrow length adjusts based on count field usage
			arrow := "──>"
			
			content.WriteString(fmt.Sprintf("    %s [%s]%s %s %s %s[white] %s   [#666666]%s[white]\n",
				branch, arrowColor, amountStr, coinStr, countPadded, arrow, target, dateLabel))
		}
		content.WriteString("\n")
	}

	// Render swaps at the end with totals
	if len(swaps) > 0 {
		content.WriteString("[::b]Swaps:[:-]\n")
		
		// Group swaps by wallet+sellCoin+buyCoin for totals
		type swapKey struct {
			wallet   string
			sellCoin string
			buyCoin  string
		}
		swapGroups := make(map[swapKey][]*FlowEdge)
		
		for _, swap := range swaps {
			key := swapKey{swap.From, swap.SellCoin, swap.BuyCoin}
			swapGroups[key] = append(swapGroups[key], swap)
		}

		for key, group := range swapGroups {
			walletDisplay := fmt.Sprintf("[#00FFFF]%s[white]", key.wallet)
			walletAddr := addrSnippet(key.wallet)
			
			// Calculate padding for total line to align with amounts
			// The prefix is: "  " + wallet + " " + addr + "  "
			walletPrefix := key.wallet
			addrPrefix := ""
			if addr, exists := walletAddresses[key.wallet]; exists && len(addr) > 8 {
				addrPrefix = fmt.Sprintf("(%s...%s)", addr[:4], addr[len(addr)-4:])
			}
			prefixLen := 2 + len(walletPrefix) + 1 + len(addrPrefix) + 2
			
			// Render individual swaps
			for _, swap := range group {
				dateStr := fmt.Sprintf("[#666666]%s[white]", formatDates(swap.Dates))
				content.WriteString(fmt.Sprintf("  %s %s  [#FF00FF]%.2f %s  ⇄  %.2f %s[white]  %s\n",
					walletDisplay, walletAddr,
					swap.SellAmount, swap.SellCoin,
					swap.BuyAmount, swap.BuyCoin,
					dateStr))
			}
			
			// Show total if 2+ swaps in this group
			if len(group) >= 2 {
				var totalSell, totalBuy float64
				for _, swap := range group {
					totalSell += swap.SellAmount
					totalBuy += swap.BuyAmount
				}
				// Pad to align with the amounts column
				padding := strings.Repeat(" ", prefixLen)
				content.WriteString(fmt.Sprintf("%s[#FF00FF][::b]Σ %.2f %s  ⇄  %.2f %s[:-][white]\n",
					padding, totalSell, key.sellCoin, totalBuy, key.buyCoin))
			}
		}
	}

	view.SetText(content.String())
	return view
}

// collectAllTransactions gathers all transactions from storage
func collectAllTransactions(s *storage.Storage) []*model.Tx {
	allTxs := s.ListTransactions()

	// Sort by date (newest first)
	sort.Slice(allTxs, func(i, j int) bool {
		return allTxs[i].Date.After(allTxs[j].Date)
	})

	return allTxs
}

// createAnnualSummaryView creates a view showing annual statistics
// NOTE: Commented out for now, will be redone later
/*
func createAnnualSummaryView(txs []*model.Tx) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Annual Summary ")

	if len(txs) == 0 {
		view.SetText("[#AAAAAA]No transactions found[white]")
		return view
	}

	// Get prices
	coins := make(map[string]bool)
	for _, tx := range txs {
		if tx.Coin != "" {
			coins[tx.Coin] = true
		}
	}
	coinList := make([]string, 0, len(coins))
	for c := range coins {
		coinList = append(coinList, c)
	}
	prices, _ := util.GetCoinPrices(coinList)

	// Calculate annual stats
	type AnnualStats struct {
		Inflow   float64
		Outflow  float64
		TxCount  int
		Deposits int
		Withdraws int
		Transfers int
		Swaps    int
	}
	annualStats := make(map[int]*AnnualStats)

	for _, tx := range txs {
		year := tx.Date.Year()
		if _, exists := annualStats[year]; !exists {
			annualStats[year] = &AnnualStats{}
		}
		stats := annualStats[year]
		stats.TxCount++

		var usdValue float64
		if tx.Coin != "" {
			if price, exists := prices[strings.ToLower(tx.Coin)]; exists {
				usdValue = tx.Amount * price
			} else {
				usdValue = tx.Amount
			}
		}

		switch tx.Type {
		case model.TxTypeDeposit:
			stats.Deposits++
			stats.Inflow += usdValue
		case model.TxTypeWithdraw:
			stats.Withdraws++
			stats.Outflow += usdValue
		case model.TxTypeTransfer:
			stats.Transfers++
		case model.TxTypeSwap:
			stats.Swaps++
		}
	}

	// Sort years
	years := make([]int, 0, len(annualStats))
	for y := range annualStats {
		years = append(years, y)
	}
	sort.Ints(years)

	// Build content
	var content strings.Builder

	for _, year := range years {
		stats := annualStats[year]
		netFlow := stats.Inflow - stats.Outflow
		netColor := "#00FF00"
		netSign := "+"
		if netFlow < 0 {
			netColor = "#FF5555"
			netSign = ""
		}

		content.WriteString(fmt.Sprintf("[::b][#FFFF00]%d[white][:-]\n", year))
		content.WriteString(fmt.Sprintf("  [#00FF00]▲ Inflow:[white]  %s\n", util.FormatUSDValue(stats.Inflow)))
		content.WriteString(fmt.Sprintf("  [#FF5555]▼ Outflow:[white] %s\n", util.FormatUSDValue(stats.Outflow)))
		content.WriteString(fmt.Sprintf("  [%s]◆ Net:[white]     %s%s\n", netColor, netSign, util.FormatUSDValue(netFlow)))
		content.WriteString(fmt.Sprintf("  [#AAAAAA]Txs: %d (D:%d W:%d T:%d S:%d)[white]\n\n",
			stats.TxCount, stats.Deposits, stats.Withdraws, stats.Transfers, stats.Swaps))
	}

	view.SetText(content.String())
	return view
}
*/

// createTransactionsView creates a view showing transactions for the current month
func createTransactionsView(txs []*model.Tx) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Transactions ")

	if len(txs) == 0 {
		view.SetText("[#AAAAAA]No transactions this month[white]")
		return view
	}

	// Sort by date (newest first)
	sortedTxs := make([]*model.Tx, len(txs))
	copy(sortedTxs, txs)
	sort.Slice(sortedTxs, func(i, j int) bool {
		return sortedTxs[i].Date.After(sortedTxs[j].Date)
	})

	// Calculate max widths for alignment
	maxAmountLen := 0
	maxCoinLen := 0
	maxFromLen := 0
	maxToLen := 0
	maxSellAmountLen := 0
	maxSellCoinLen := 0
	maxBuyAmountLen := 0
	maxBuyCoinLen := 0
	maxSwapWalletLen := 0

	for _, tx := range sortedTxs {
		switch tx.Type {
		case model.TxTypeDeposit, model.TxTypeWithdraw:
			amountStr := fmt.Sprintf("%.2f", tx.Amount)
			if len(amountStr) > maxAmountLen {
				maxAmountLen = len(amountStr)
			}
			if len(tx.Coin) > maxCoinLen {
				maxCoinLen = len(tx.Coin)
			}
			if len(tx.ToWallet) > maxToLen {
				maxToLen = len(tx.ToWallet)
			}
			if len(tx.FromWallet) > maxFromLen {
				maxFromLen = len(tx.FromWallet)
			}
		case model.TxTypeTransfer:
			amountStr := fmt.Sprintf("%.2f", tx.Amount)
			if len(amountStr) > maxAmountLen {
				maxAmountLen = len(amountStr)
			}
			if len(tx.Coin) > maxCoinLen {
				maxCoinLen = len(tx.Coin)
			}
			if len(tx.FromWallet) > maxFromLen {
				maxFromLen = len(tx.FromWallet)
			}
			toWallet := tx.ToWallet
			if toWallet == "" && tx.ToAddress != "" {
				toWallet = tx.ToAddress
				if len(toWallet) > 10 {
					toWallet = toWallet[:6] + "..." + toWallet[len(toWallet)-4:]
				}
			}
			if len(toWallet) > maxToLen {
				maxToLen = len(toWallet)
			}
		case model.TxTypeSwap:
			if len(tx.SwapWallet) > maxSwapWalletLen {
				maxSwapWalletLen = len(tx.SwapWallet)
			}
			sellAmountStr := fmt.Sprintf("%.2f", tx.SellAmount)
			if len(sellAmountStr) > maxSellAmountLen {
				maxSellAmountLen = len(sellAmountStr)
			}
			if len(tx.SellCoin) > maxSellCoinLen {
				maxSellCoinLen = len(tx.SellCoin)
			}
			buyAmountStr := fmt.Sprintf("%.2f", tx.BuyAmount)
			if len(buyAmountStr) > maxBuyAmountLen {
				maxBuyAmountLen = len(buyAmountStr)
			}
			if len(tx.BuyCoin) > maxBuyCoinLen {
				maxBuyCoinLen = len(tx.BuyCoin)
			}
		}
	}

	var content strings.Builder

	for _, tx := range sortedTxs {
		// Format date
		dateStr := tx.Date.Format("Jan 02")

		// Format based on type with alignment
		var typeIcon, typeColor, details string
		switch tx.Type {
		case model.TxTypeDeposit:
			typeIcon = "▼"
			typeColor = "#00FF00"
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			toStr := fmt.Sprintf("%-*s", maxToLen, tx.ToWallet)
			details = fmt.Sprintf("%s %s  →  %s", amountStr, coinStr, toStr)
		case model.TxTypeWithdraw:
			typeIcon = "▲"
			typeColor = "#FF5555"
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			fromStr := fmt.Sprintf("%-*s", maxFromLen, tx.FromWallet)
			details = fmt.Sprintf("%s %s  ←  %s", amountStr, coinStr, fromStr)
		case model.TxTypeTransfer:
			typeIcon = "↔"
			typeColor = "#FFFF00"
			toWallet := tx.ToWallet
			if toWallet == "" && tx.ToAddress != "" {
				toWallet = tx.ToAddress
				if len(toWallet) > 10 {
					toWallet = toWallet[:6] + "..." + toWallet[len(toWallet)-4:]
				}
			}
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			fromStr := fmt.Sprintf("%-*s", maxFromLen, tx.FromWallet)
			toStr := fmt.Sprintf("%-*s", maxToLen, toWallet)
			details = fmt.Sprintf("%s %s  %s  →  %s", amountStr, coinStr, fromStr, toStr)
		case model.TxTypeSwap:
			typeIcon = "⇄"
			typeColor = "#FF00FF"
			walletStr := fmt.Sprintf("%-*s", maxSwapWalletLen, tx.SwapWallet)
			sellAmountStr := fmt.Sprintf("%*.2f", maxSellAmountLen, tx.SellAmount)
			sellCoinStr := fmt.Sprintf("%-*s", maxSellCoinLen, tx.SellCoin)
			buyAmountStr := fmt.Sprintf("%*.2f", maxBuyAmountLen, tx.BuyAmount)
			buyCoinStr := fmt.Sprintf("%-*s", maxBuyCoinLen, tx.BuyCoin)
			details = fmt.Sprintf("%s  %s %s  →  %s %s", walletStr, sellAmountStr, sellCoinStr, buyAmountStr, buyCoinStr)
		}

		line := fmt.Sprintf("[#666666]%s[white] [%s]%s[white] %s", dateStr, typeColor, typeIcon, details)
		
		// Add note if present
		if tx.Note != "" {
			line += fmt.Sprintf("  [#666666]// %s[white]", tx.Note)
		}
		
		content.WriteString(line + "\n")
	}

	view.SetText(content.String())
	return view
}

// createWalletsPanel creates the wallets list panel with selection highlighting
func createWalletsPanel(wallets []*model.Wallet, categories []*model.Category, selectedIdx int) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Wallets ")

	// Create a map of category name to color
	categoryColors := make(map[string]string)
	for _, cat := range categories {
		colorName := cat.Color
		if colorName == "" {
			colorName = "white"
		}
		tviewColor := terminalColorToTviewColor(colorName)
		categoryColors[cat.Name] = tviewColor
	}

	var content strings.Builder
	for i, wallet := range wallets {
		// Highlight selected wallet
		prefix := "  "
		if i == selectedIdx {
			prefix = "[#FF6600]▶[white] "
		}

		// Get category color
		catColor := categoryColors[wallet.Category]
		if catColor == "" {
			catColor = "#FFFFFF"
		}

		// Format: prefix name [cat] chain-type address (all one line)
		content.WriteString(prefix)
		if i == selectedIdx {
			content.WriteString(fmt.Sprintf("[::b][#FFFFFF]%s[:-]", wallet.Name))
		} else {
			content.WriteString(fmt.Sprintf("[#AAAAAA]%s[white]", wallet.Name))
		}
		
		if wallet.Category != "" {
			content.WriteString(fmt.Sprintf(" [%s]■[white]", catColor))
		}
		
		content.WriteString(fmt.Sprintf(" [#666666]%s-%s[white]", wallet.Chain, wallet.Type))
		
		// Show address on same line (truncated)
		addr := wallet.Address
		if len(addr) > 16 {
			addr = addr[:8] + "..." + addr[len(addr)-6:]
		}
		content.WriteString(fmt.Sprintf(" [#888888]%s[white]\n", addr))
		
		// Show note on second line only for selected wallet
		if i == selectedIdx && wallet.Note != "" {
			content.WriteString(fmt.Sprintf("   [#666666]%s[white]\n", wallet.Note))
		}
	}

	view.SetText(content.String())
	return view
}

// createWalletBalancesPanel creates the balances panel for selected wallet
func createWalletBalancesPanel(wallet *model.Wallet) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Balances ")

	if wallet == nil {
		view.SetText("[#AAAAAA]No wallet selected[white]")
		return view
	}

	if len(wallet.Balances) == 0 {
		view.SetText("[#AAAAAA]No balances[white]")
		return view
	}

	// Get coins for price fetching
	coins := make([]string, 0, len(wallet.Balances))
	for _, bal := range wallet.Balances {
		if bal.Amount > 0 {
			coins = append(coins, bal.Coin)
		}
	}
	sort.Strings(coins)

	prices, _ := util.GetCoinPrices(coins)

	// Same format as Total Balance by Coin: COIN: amount (usd)
	var content strings.Builder
	for _, bal := range wallet.Balances {
		if bal.Amount == 0 {
			continue
		}
		if prices != nil {
			if price, exists := prices[strings.ToLower(bal.Coin)]; exists {
				usdValue := bal.Amount * price
				content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white] [#AAAAAA](%s)[white]\n",
					bal.Coin, bal.Amount, util.FormatUSDValue(usdValue)))
			} else {
				content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white]\n", bal.Coin, bal.Amount))
			}
		} else {
			content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white]\n", bal.Coin, bal.Amount))
		}
	}

	view.SetText(content.String())
	return view
}

// createWalletTransactionsPanel creates the transactions panel for selected wallet
// Uses exact same format as createTransactionsView in Stats
func createWalletTransactionsPanel(txs []*model.Tx) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Transactions ")

	if len(txs) == 0 {
		view.SetText("[#AAAAAA]No transactions[white]")
		return view
	}

	// Sort by date (newest first)
	sortedTxs := make([]*model.Tx, len(txs))
	copy(sortedTxs, txs)
	sort.Slice(sortedTxs, func(i, j int) bool {
		return sortedTxs[i].Date.After(sortedTxs[j].Date)
	})

	// Calculate max widths for alignment (same as createTransactionsView)
	maxAmountLen := 0
	maxCoinLen := 0
	maxFromLen := 0
	maxToLen := 0
	maxSellAmountLen := 0
	maxSellCoinLen := 0
	maxBuyAmountLen := 0
	maxBuyCoinLen := 0
	maxSwapWalletLen := 0

	for _, tx := range sortedTxs {
		switch tx.Type {
		case model.TxTypeDeposit, model.TxTypeWithdraw:
			amountStr := fmt.Sprintf("%.2f", tx.Amount)
			if len(amountStr) > maxAmountLen {
				maxAmountLen = len(amountStr)
			}
			if len(tx.Coin) > maxCoinLen {
				maxCoinLen = len(tx.Coin)
			}
			if len(tx.ToWallet) > maxToLen {
				maxToLen = len(tx.ToWallet)
			}
			if len(tx.FromWallet) > maxFromLen {
				maxFromLen = len(tx.FromWallet)
			}
		case model.TxTypeTransfer:
			amountStr := fmt.Sprintf("%.2f", tx.Amount)
			if len(amountStr) > maxAmountLen {
				maxAmountLen = len(amountStr)
			}
			if len(tx.Coin) > maxCoinLen {
				maxCoinLen = len(tx.Coin)
			}
			if len(tx.FromWallet) > maxFromLen {
				maxFromLen = len(tx.FromWallet)
			}
			toWallet := tx.ToWallet
			if toWallet == "" && tx.ToAddress != "" {
				toWallet = tx.ToAddress
				if len(toWallet) > 10 {
					toWallet = toWallet[:6] + "..." + toWallet[len(toWallet)-4:]
				}
			}
			if len(toWallet) > maxToLen {
				maxToLen = len(toWallet)
			}
		case model.TxTypeSwap:
			if len(tx.SwapWallet) > maxSwapWalletLen {
				maxSwapWalletLen = len(tx.SwapWallet)
			}
			sellAmountStr := fmt.Sprintf("%.2f", tx.SellAmount)
			if len(sellAmountStr) > maxSellAmountLen {
				maxSellAmountLen = len(sellAmountStr)
			}
			if len(tx.SellCoin) > maxSellCoinLen {
				maxSellCoinLen = len(tx.SellCoin)
			}
			buyAmountStr := fmt.Sprintf("%.2f", tx.BuyAmount)
			if len(buyAmountStr) > maxBuyAmountLen {
				maxBuyAmountLen = len(buyAmountStr)
			}
			if len(tx.BuyCoin) > maxBuyCoinLen {
				maxBuyCoinLen = len(tx.BuyCoin)
			}
		}
	}

	var content strings.Builder

	for _, tx := range sortedTxs {
		dateStr := tx.Date.Format("Jan 02")

		var typeIcon, typeColor, details string
		switch tx.Type {
		case model.TxTypeDeposit:
			typeIcon = "▼"
			typeColor = "#00FF00"
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			toStr := fmt.Sprintf("%-*s", maxToLen, tx.ToWallet)
			details = fmt.Sprintf("%s %s  →  %s", amountStr, coinStr, toStr)
		case model.TxTypeWithdraw:
			typeIcon = "▲"
			typeColor = "#FF5555"
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			fromStr := fmt.Sprintf("%-*s", maxFromLen, tx.FromWallet)
			details = fmt.Sprintf("%s %s  ←  %s", amountStr, coinStr, fromStr)
		case model.TxTypeTransfer:
			typeIcon = "↔"
			typeColor = "#FFFF00"
			toWallet := tx.ToWallet
			if toWallet == "" && tx.ToAddress != "" {
				toWallet = tx.ToAddress
				if len(toWallet) > 10 {
					toWallet = toWallet[:6] + "..." + toWallet[len(toWallet)-4:]
				}
			}
			amountStr := fmt.Sprintf("%*.2f", maxAmountLen, tx.Amount)
			coinStr := fmt.Sprintf("%-*s", maxCoinLen, tx.Coin)
			fromStr := fmt.Sprintf("%-*s", maxFromLen, tx.FromWallet)
			toStr := fmt.Sprintf("%-*s", maxToLen, toWallet)
			details = fmt.Sprintf("%s %s  %s  →  %s", amountStr, coinStr, fromStr, toStr)
		case model.TxTypeSwap:
			typeIcon = "⇄"
			typeColor = "#FF00FF"
			walletStr := fmt.Sprintf("%-*s", maxSwapWalletLen, tx.SwapWallet)
			sellAmountStr := fmt.Sprintf("%*.2f", maxSellAmountLen, tx.SellAmount)
			sellCoinStr := fmt.Sprintf("%-*s", maxSellCoinLen, tx.SellCoin)
			buyAmountStr := fmt.Sprintf("%*.2f", maxBuyAmountLen, tx.BuyAmount)
			buyCoinStr := fmt.Sprintf("%-*s", maxBuyCoinLen, tx.BuyCoin)
			details = fmt.Sprintf("%s  %s %s  →  %s %s", walletStr, sellAmountStr, sellCoinStr, buyAmountStr, buyCoinStr)
		}

		line := fmt.Sprintf("[#666666]%s[white] [%s]%s[white] %s", dateStr, typeColor, typeIcon, details)

		if tx.Note != "" {
			line += fmt.Sprintf("  [#666666]// %s[white]", tx.Note)
		}

		content.WriteString(line + "\n")
	}

	view.SetText(content.String())
	return view
}

// createTotalBalanceView creates a view showing total balance by coin
func createTotalBalanceView(wallets []*model.Wallet) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Total Balance by Coin ")

	// Calculate total balance by coin
	balanceByCoin := make(map[string]float64)
	for _, wallet := range wallets {
		for _, balance := range wallet.Balances {
			balanceByCoin[balance.Coin] += balance.Amount
		}
	}

	// Get all coin symbols for price fetching (skip zero balances)
	coins := make([]string, 0, len(balanceByCoin))
	for coin, amount := range balanceByCoin {
		if amount > 0 {
			coins = append(coins, coin)
		}
	}
	sort.Strings(coins)

	// Fetch USD prices from manual prices.json
	prices, err := util.GetCoinPrices(coins)
	if err != nil {
		// If price fetching fails, show without USD values
		var content strings.Builder
		for _, coin := range coins {
			balance := balanceByCoin[coin]
			content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white]\n", coin, balance))
		}
		view.SetText(content.String())
		return view
	}

	// Calculate total net worth and format display
	var content strings.Builder
	totalNetWorth := 0.0
	liquidNetWorth := 0.0
	nonLiquidNetWorth := 0.0

	// Define stablecoins (liquid assets)
	stablecoins := map[string]bool{
		"usdt": true,
		"usdc": true,
		"dai":  true,
		"busd": true,
		"tusd": true,
		"frax": true,
		"lusd": true,
		"susd": true,
	}

	for _, coin := range coins {
		balance := balanceByCoin[coin]
		if price, exists := prices[strings.ToLower(coin)]; exists {
			usdValue := balance * price
			totalNetWorth += usdValue
			
			// Categorize as liquid or non-liquid
			if stablecoins[strings.ToLower(coin)] {
				liquidNetWorth += usdValue
			} else {
				nonLiquidNetWorth += usdValue
			}
			
			content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white] [#AAAAAA](%s)[white]\n", 
				coin, balance, util.FormatUSDValue(usdValue)))
		} else {
			content.WriteString(fmt.Sprintf("[::b]%s:[:-]  [#00FF00]%.2f[white]\n", coin, balance))
		}
	}

	// Add net worth breakdown at the bottom
	if totalNetWorth > 0 {
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("[::b][#FF6600]Non-Stables: %s[white]\n", util.FormatUSDValue(nonLiquidNetWorth)))
		content.WriteString(fmt.Sprintf("[::b][#00FF00]Stables: %s[white]\n", util.FormatUSDValue(liquidNetWorth)))
		content.WriteString(fmt.Sprintf("[::b][#FFFF00]Total: %s[white]", util.FormatUSDValue(totalNetWorth)))
	}

	view.SetText(content.String())
	return view
}

// createWalletListView creates a view showing all wallets and their balances
func createWalletListView(wallets []*model.Wallet, categories []*model.Category) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Wallet Balances ")

	// Get all coins for price fetching
	allCoins := make(map[string]bool)
	for _, wallet := range wallets {
		for _, balance := range wallet.Balances {
			allCoins[balance.Coin] = true
		}
	}
	coins := make([]string, 0, len(allCoins))
	for coin := range allCoins {
		coins = append(coins, coin)
	}

	// Fetch USD prices from manual prices.json
	prices, err := util.GetCoinPrices(coins)

	// Create a map of category name to color
	categoryColors := make(map[string]string)
	for _, cat := range categories {
		colorName := cat.Color
		if colorName == "" {
			colorName = "white"
		}
		// Convert terminal color to tview color
		tviewColor := terminalColorToTviewColor(colorName)
		categoryColors[cat.Name] = tviewColor
	}

	// Sort wallets by name
	sort.Slice(wallets, func(i, j int) bool {
		return wallets[i].Name < wallets[j].Name
	})

	// Format and display the wallets
	var content strings.Builder
	for _, wallet := range wallets {
		// Get category color
		catColor := categoryColors[wallet.Category]
		if catColor == "" {
			catColor = "#FFFFFF"
		}

		// Add wallet name and address
		content.WriteString(fmt.Sprintf("[::b]%s[:-]", wallet.Name))
		if wallet.Category != "" {
			content.WriteString(fmt.Sprintf(" [%s]■[white]", catColor))
		}
		content.WriteString(fmt.Sprintf(" [#888888](%s)[white]\n", wallet.Address))

		// Get coins from balances
		coinMap := make(map[string]float64)
		for _, balance := range wallet.Balances {
			coinMap[balance.Coin] = balance.Amount
		}

		// Sort coins
		coins := make([]string, 0, len(coinMap))
		for coin := range coinMap {
			coins = append(coins, coin)
		}
		sort.Strings(coins)

		// Add balances for each coin (skip zero balances)
		for _, coin := range coins {
			balance := coinMap[coin]
			if balance == 0 {
				continue
			}
			// Add USD value if available
			if err == nil {
				if price, exists := prices[strings.ToLower(coin)]; exists {
					usdValue := balance * price
					content.WriteString(fmt.Sprintf("  %s: [#00FF00]%.2f[white] [#AAAAAA](%s)[white]\n", 
						coin, balance, util.FormatUSDValue(usdValue)))
				} else {
					content.WriteString(fmt.Sprintf("  %s: [#00FF00]%.2f[white]\n", coin, balance))
				}
			} else {
				content.WriteString(fmt.Sprintf("  %s: [#00FF00]%.2f[white]\n", coin, balance))
			}
		}
		content.WriteString("\n")
	}

	view.SetText(content.String())
	return view
}

// createCategoryBalanceView creates a view showing balances by category
func createCategoryBalanceView(wallets []*model.Wallet, categories []*model.Category) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Balance by Category ")

	// Create a map of category name to color
	categoryColors := make(map[string]string)
	for _, cat := range categories {
		colorName := cat.Color
		if colorName == "" {
			colorName = "white"
		}
		// Convert terminal color to tview color
		tviewColor := terminalColorToTviewColor(colorName)
		categoryColors[cat.Name] = tviewColor
	}

	// Calculate balance by category
	balanceByCategory := make(map[string]map[string]float64)
	for _, wallet := range wallets {
		category := wallet.Category
		if category == "" {
			category = "Uncategorized"
		}

		if _, ok := balanceByCategory[category]; !ok {
			balanceByCategory[category] = make(map[string]float64)
		}

		for _, balance := range wallet.Balances {
			balanceByCategory[category][balance.Coin] += balance.Amount
		}
	}

	// Sort categories by name
	categoryNames := make([]string, 0, len(balanceByCategory))
	for catName := range balanceByCategory {
		categoryNames = append(categoryNames, catName)
	}
	sort.Strings(categoryNames)

	// Format and display the balances by category
	var content strings.Builder
	for _, catName := range categoryNames {
		// Skip if no balances
		if len(balanceByCategory[catName]) == 0 {
			continue
		}

		// Get color for category
		catColor := categoryColors[catName]
		if catColor == "" {
			catColor = "#FFFFFF"
		}

		// Add category name
		content.WriteString(fmt.Sprintf("[%s]■[white] [::b]%s[:-]\n", catColor, catName))

		// Sort coins for this category
		coins := make([]string, 0, len(balanceByCategory[catName]))
		for coin := range balanceByCategory[catName] {
			coins = append(coins, coin)
		}
		sort.Strings(coins)

		// Add balances for each coin in this category (skip zero balances)
		for _, coin := range coins {
			balance := balanceByCategory[catName][coin]
			if balance == 0 {
				continue
			}
			content.WriteString(fmt.Sprintf("  %s: [#00FF00]%.2f[white]\n", coin, balance))
		}
		content.WriteString("\n")
	}

	view.SetText(content.String())
	return view
}

// createCategoryChartView creates a view showing a chart of category distribution
func createCategoryChartView(wallets []*model.Wallet, categories []*model.Category) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	view.SetBorder(true).SetTitle(" Category Distribution by Coin ")

	// Create a map of category name to color
	categoryColors := make(map[string]string)
	for _, cat := range categories {
		colorName := cat.Color
		if colorName == "" {
			colorName = "white"
		}
		// Convert terminal color to tview color
		tviewColor := terminalColorToTviewColor(colorName)
		categoryColors[cat.Name] = tviewColor
	}
	categoryColors["Uncategorized"] = "#FFFFFF"

	// Calculate balances by category and coin
	balanceByCategoryAndCoin := make(map[string]map[string]float64)
	allCoins := make(map[string]bool)
	
	for _, wallet := range wallets {
		category := wallet.Category
		if category == "" {
			category = "Uncategorized"
		}

		if _, ok := balanceByCategoryAndCoin[category]; !ok {
			balanceByCategoryAndCoin[category] = make(map[string]float64)
		}

		for _, balance := range wallet.Balances {
			balanceByCategoryAndCoin[category][balance.Coin] += balance.Amount
			allCoins[balance.Coin] = true
		}
	}

	// Skip if no categories with balances
	if len(balanceByCategoryAndCoin) == 0 {
		view.SetText("No category data available")
		return view
	}

	// Get all coins
	coinsList := make([]string, 0, len(allCoins))
	for coin := range allCoins {
		coinsList = append(coinsList, coin)
	}
	sort.Strings(coinsList)

	// Format and display the chart for each coin
	var content strings.Builder
	maxBarLength := 30

	for _, coin := range coinsList {
		// Calculate total balance for this coin
		totalCoinBalance := 0.0
		for _, balances := range balanceByCategoryAndCoin {
			if amount, ok := balances[coin]; ok {
				totalCoinBalance += amount
			}
		}
		
		if totalCoinBalance == 0 {
			continue
		}

		// Add coin header
		content.WriteString(fmt.Sprintf("\n[::b]%s[:-]\n", coin))

		// Sort categories by balance for this coin (descending)
		type categoryStat struct {
			name    string
			balance float64
		}

		stats := make([]categoryStat, 0)
		for cat, balances := range balanceByCategoryAndCoin {
			if amount, ok := balances[coin]; ok && amount > 0 {
				stats = append(stats, categoryStat{cat, amount})
			}
		}

		sort.Slice(stats, func(i, j int) bool {
			return stats[i].balance > stats[j].balance
		})

		// Find the maximum balance for scaling
		maxBalance := 0.0
		if len(stats) > 0 {
			maxBalance = stats[0].balance
		}

		// Display bars for each category with this coin
		for _, stat := range stats {
			// Get color for category
			catColor := categoryColors[stat.name]
			if catColor == "" {
				catColor = "#FFFFFF"
			}

			// Calculate bar length
			barLength := int((stat.balance / maxBalance) * float64(maxBarLength))
			if barLength < 1 {
				barLength = 1
			}

			// Calculate percentage
			percentage := (stat.balance / totalCoinBalance) * 100

			// Create the bar
			bar := strings.Repeat("█", barLength)

			// Add category name, bar, balance, and percentage
			content.WriteString(fmt.Sprintf(" [%s]■[white] [::b]%s[:-] [%s]%s[white] [#00FF00]%.2f[white] ([#FFFF00]%.1f%%[white])\n",
				catColor,
				stat.name,
				catColor,
				bar,
				stat.balance,
				percentage,
			))
		}
	}

	view.SetText(content.String())
	return view
}

// terminalColorToTviewColor converts terminal color names to tview color codes
func terminalColorToTviewColor(colorName string) string {
	colorMap := map[string]string{
		"black":         "#000000",
		"red":           "#FF0000",
		"green":         "#00FF00",
		"yellow":        "#FFFF00",
		"blue":          "#0000FF",
		"magenta":       "#FF00FF",
		"cyan":          "#00FFFF",
		"white":         "#FFFFFF",
		"brightred":     "#FF5555",
		"brightgreen":   "#55FF55",
		"brightyellow":  "#FFFF55",
		"brightblue":    "#5555FF",
		"brightmagenta": "#FF55FF",
		"brightcyan":    "#55FFFF",
		"brightwhite":   "#FFFFFF",
	}

	if color, ok := colorMap[colorName]; ok {
		return color
	}
	return "#FFFFFF" // Default to white
}

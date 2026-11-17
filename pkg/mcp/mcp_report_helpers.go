package mcp

import (
	"math/big"
	"sort"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Textual group by values of transaction totals
const (
	mcpTotalsGroupByAccount         = "account"
	mcpTotalsGroupByCategory        = "category"
	mcpTotalsGroupByPrimaryCategory = "primary_category"
	mcpTotalsGroupByTag             = "tag"
	mcpTotalsGroupByCurrency        = "currency"
)

// mcpTotalsNoTagsGroupName is the group name of transactions without tags when grouping by tag
const mcpTotalsNoTagsGroupName = "(no tags)"

// MCPTransactionTotalsItem represents the activity totals of a group in one currency
type MCPTransactionTotalsItem struct {
	Id          string `json:"id,omitempty" jsonschema_description:"ID of the group (account, category or tag ID, empty for currency groups and for transactions without tags)"`
	Name        string `json:"name,omitempty" jsonschema_description:"Name of the group"`
	Currency    string `json:"currency" jsonschema_description:"Currency code of the amounts (amounts are in the currency of the account of each transaction)"`
	Income      string `json:"income" jsonschema_description:"Total income amount"`
	Expense     string `json:"expense" jsonschema_description:"Total expense amount"`
	TransferIn  string `json:"transfer_in" jsonschema_description:"Total amount transferred in"`
	TransferOut string `json:"transfer_out" jsonschema_description:"Total amount transferred out"`
	Net         string `json:"net" jsonschema_description:"Net amount (income - expense + transfer_in - transfer_out)"`
}

// mcpTotalsAccumulator accumulates the activity totals of a group in one currency
type mcpTotalsAccumulator struct {
	id          string
	name        string
	currency    string
	income      *big.Int
	expense     *big.Int
	transferIn  *big.Int
	transferOut *big.Int
}

// mcpTransactionTotalsAggregator aggregates the total amounts returned by the transaction statistics service into groups
type mcpTransactionTotalsAggregator struct {
	items map[string]*mcpTotalsAccumulator
}

// newMCPTransactionTotalsAggregator returns a new transaction totals aggregator
func newMCPTransactionTotalsAggregator() *mcpTransactionTotalsAggregator {
	return &mcpTransactionTotalsAggregator{
		items: make(map[string]*mcpTotalsAccumulator),
	}
}

// Add adds the amount of the specified transaction type to the group in the currency
func (a *mcpTransactionTotalsAggregator) Add(groupId string, groupName string, currency string, transactionType models.TransactionDbType, amount *big.Int) {
	if amount == nil {
		return
	}

	key := groupId + "\x00" + groupName + "\x00" + currency
	item, exists := a.items[key]

	if !exists {
		item = &mcpTotalsAccumulator{
			id:          groupId,
			name:        groupName,
			currency:    currency,
			income:      big.NewInt(0),
			expense:     big.NewInt(0),
			transferIn:  big.NewInt(0),
			transferOut: big.NewInt(0),
		}

		a.items[key] = item
	}

	switch transactionType {
	case models.TRANSACTION_DB_TYPE_INCOME:
		item.income.Add(item.income, amount)
	case models.TRANSACTION_DB_TYPE_EXPENSE:
		item.expense.Add(item.expense, amount)
	case models.TRANSACTION_DB_TYPE_TRANSFER_IN:
		item.transferIn.Add(item.transferIn, amount)
	case models.TRANSACTION_DB_TYPE_TRANSFER_OUT:
		item.transferOut.Add(item.transferOut, amount)
	}
}

// Items returns the aggregated groups ordered by name, id and currency
func (a *mcpTransactionTotalsAggregator) Items() []*MCPTransactionTotalsItem {
	accumulators := make([]*mcpTotalsAccumulator, 0, len(a.items))

	for _, item := range a.items {
		accumulators = append(accumulators, item)
	}

	sort.Slice(accumulators, func(i, j int) bool {
		if accumulators[i].name != accumulators[j].name {
			return accumulators[i].name < accumulators[j].name
		}

		if accumulators[i].id != accumulators[j].id {
			return accumulators[i].id < accumulators[j].id
		}

		return accumulators[i].currency < accumulators[j].currency
	})

	items := make([]*MCPTransactionTotalsItem, 0, len(accumulators))

	for i := 0; i < len(accumulators); i++ {
		accumulator := accumulators[i]
		net := new(big.Int).Set(accumulator.income)
		net.Sub(net, accumulator.expense)
		net.Add(net, accumulator.transferIn)
		net.Sub(net, accumulator.transferOut)

		items = append(items, &MCPTransactionTotalsItem{
			Id:          accumulator.id,
			Name:        accumulator.name,
			Currency:    accumulator.currency,
			Income:      utils.FormatBigIntAmount(accumulator.income),
			Expense:     utils.FormatBigIntAmount(accumulator.expense),
			TransferIn:  utils.FormatBigIntAmount(accumulator.transferIn),
			TransferOut: utils.FormatBigIntAmount(accumulator.transferOut),
			Net:         utils.FormatBigIntAmount(net),
		})
	}

	return items
}

// mcpTotalsGetGroup returns the group id and name of the total amount row according to the group by value
func mcpTotalsGetGroup(groupBy string, row *models.TransactionTotalAmount, accounts map[int64]*models.Account, categories map[int64]*models.TransactionCategory) (string, string) {
	switch groupBy {
	case mcpTotalsGroupByAccount:
		name := ""

		if account, exists := accounts[row.AccountId]; exists && account != nil {
			name = account.Name
		}

		return utils.Int64ToString(row.AccountId), name
	case mcpTotalsGroupByCategory:
		name := ""

		if category, exists := categories[row.CategoryId]; exists && category != nil {
			name = category.Name
		}

		return utils.Int64ToString(row.CategoryId), name
	case mcpTotalsGroupByPrimaryCategory:
		categoryId := row.CategoryId

		if category, exists := categories[row.CategoryId]; exists && category != nil && category.ParentCategoryId > models.LevelOneTransactionCategoryParentId {
			categoryId = category.ParentCategoryId
		}

		name := ""

		if category, exists := categories[categoryId]; exists && category != nil {
			name = category.Name
		}

		return utils.Int64ToString(categoryId), name
	case mcpTotalsGroupByCurrency:
		return "", mcpTotalsGetCurrency(row, accounts)
	default:
		return "", ""
	}
}

// mcpTotalsGetCurrency returns the currency of the amount of the total amount row (the currency of the account of the row)
func mcpTotalsGetCurrency(row *models.TransactionTotalAmount, accounts map[int64]*models.Account) string {
	if account, exists := accounts[row.AccountId]; exists && account != nil {
		return account.Currency
	}

	return ""
}

// mcpTotalsRowMatches returns whether the total amount row matches the account filter and the category filter (secondary or primary category ids)
func mcpTotalsRowMatches(row *models.TransactionTotalAmount, accountFilter map[int64]bool, categoryFilter map[int64]bool, categories map[int64]*models.TransactionCategory) bool {
	if len(accountFilter) > 0 && !accountFilter[row.AccountId] {
		return false
	}

	if len(categoryFilter) > 0 && !categoryFilter[row.CategoryId] {
		category, exists := categories[row.CategoryId]

		if !exists || category == nil || !categoryFilter[category.ParentCategoryId] {
			return false
		}
	}

	return true
}

// mcpReportFormatBalances returns the displayed balance, the signed accounting balance and the outstanding balance (only for liability accounts) of the signed balance
func mcpReportFormatBalances(category models.AccountCategory, signedBalance *big.Int) (displayBalance string, signed string, outstandingBalance string) {
	if signedBalance == nil {
		signedBalance = big.NewInt(0)
	}

	signed = utils.FormatBigIntAmount(signedBalance)

	if category.IsLiability() {
		outstandingBalance = utils.FormatBigIntAmount(new(big.Int).Neg(signedBalance))
		displayBalance = outstandingBalance
	} else {
		displayBalance = signed
	}

	return displayBalance, signed, outstandingBalance
}

// mcpReportGetLatestClosingBalances returns the closing balance of every account on its latest day in the daily balances returned by the transaction service
func mcpReportGetLatestClosingBalances(dailyBalances map[int32][]*models.TransactionWithAccountBalance) map[int64]*big.Int {
	days := make([]int32, 0, len(dailyBalances))

	for day := range dailyBalances {
		days = append(days, day)
	}

	sort.Slice(days, func(i, j int) bool {
		return days[i] < days[j]
	})

	balances := make(map[int64]*big.Int)

	for _, day := range days {
		for _, accountBalance := range dailyBalances[day] {
			if accountBalance == nil || accountBalance.Transaction == nil || accountBalance.AccountClosingBalance == nil {
				continue
			}

			balances[accountBalance.AccountId] = new(big.Int).Set(accountBalance.AccountClosingBalance)
		}
	}

	return balances
}

// mcpReportParseTimeRange parses the optional start and end time, and returns the unix times (0 if not set) and the timezone used to determine the date range
func mcpReportParseTimeRange(startTimeText string, endTimeText string) (int64, int64, *time.Location, error) {
	startTime, err := parseMCPTime("start_time", startTimeText, false)

	if err != nil {
		return 0, 0, nil, err
	}

	endTime, err := parseMCPTime("end_time", endTimeText, false)

	if err != nil {
		return 0, 0, nil, err
	}

	location := time.UTC
	startUnixTime := int64(0)
	endUnixTime := int64(0)

	if endTime != nil {
		endUnixTime = endTime.Unix()
		location = endTime.Location()
	}

	if startTime != nil {
		startUnixTime = startTime.Unix()
		location = startTime.Location()
	}

	if startTime != nil && endTime != nil && startUnixTime > endUnixTime {
		return 0, 0, nil, NewMCPInvalidFieldError("end_time", endTimeText, "\"end_time\" must not be earlier than \"start_time\"")
	}

	return startUnixTime, endUnixTime, location, nil
}

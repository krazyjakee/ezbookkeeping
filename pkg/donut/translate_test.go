package donut

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testToday = time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)

const testPrompt = `## Role
...
### Expense categories:
Groceries
Fuel
Restaurants

### Income categories:
Salary

### Transfer categories:

### Account names:
Current Account
Amex

### Tags:
business

### Current time:
2026-11-18 11:00:00
`

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestParseAmount(t *testing.T) {
	cases := []struct{ text, amount, currency string }{
		{"£20.00", "20.00", "GBP"},
		{"€1.234,56", "1234.56", "EUR"},
		{"1,234.5", "1234.50", ""},
		{"Rp 28,000", "28000.00", "IDR"},
		{"60.000", "60000.00", ""},
		{"6.65", "6.65", ""},
		{"USD 99", "99.00", "USD"},
		{"RM 12.50", "12.50", "MYR"},
		{"", "", ""},
		{"n/a", "", ""},
		{"0.00", "", ""},
	}

	for _, c := range cases {
		amount, currency := ParseAmount(c.text)
		assert.Equal(t, c.amount, amount, c.text)
		assert.Equal(t, c.currency, currency, c.text)
	}
}

func TestDetectCurrency(t *testing.T) {
	assert.Equal(t, "GBP", DetectCurrency("Total £20"))
	assert.Equal(t, "", DetectCurrency("FARM 12.00"))
	assert.Equal(t, "MYR", DetectCurrency("RM12.00"))
	assert.Equal(t, "EUR", DetectCurrency("Total EUR 5"))
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		text     string
		order    DateOrder
		expected time.Time
	}{
		{"01/10/2026", DateOrderDMY, date(2026, 10, 1)},
		{"10/01/2026", DateOrderMDY, date(2026, 10, 1)},
		{"22-08-2026", DateOrderDMY, date(2026, 8, 22)},
		{"2026-09-15 08:31", DateOrderDMY, date(2026, 9, 15)},
		{"3rd Nov 2026", DateOrderDMY, date(2026, 11, 3)},
		{"November 3, 2026", DateOrderDMY, date(2026, 11, 3)},
		{"09/13/2026", DateOrderDMY, date(2026, 9, 13)},
		{"03/11/2016", DateOrderDMY, date(2026, 11, 3)},
		{"15/09/2016", DateOrderDMY, date(2026, 9, 15)},
		{"15/09/2025", DateOrderDMY, date(2025, 9, 15)},
	}

	for _, c := range cases {
		actual, ok := ParseDate(c.text, c.order, testToday)
		assert.True(t, ok, c.text)
		assert.Equal(t, c.expected, actual, c.text)
	}

	for _, text := range []string{"no date here", "31/02/2026", "15/09/02", "01/01/2027"} {
		_, ok := ParseDate(text, DateOrderDMY, testToday)
		assert.False(t, ok, text)
	}
}

func TestParseTimeOfDay(t *testing.T) {
	assert.Equal(t, &TimeOfDay{12, 4, 0}, ParseTimeOfDay("12:04"))
	assert.Equal(t, &TimeOfDay{17, 2, 11}, ParseTimeOfDay("17:02:11"))
	assert.Equal(t, &TimeOfDay{19, 45, 0}, ParseTimeOfDay("7.45pm"))
	assert.Nil(t, ParseTimeOfDay("25:00"))
	assert.Nil(t, ParseTimeOfDay("no time"))
}

func TestTokensToJSON(t *testing.T) {
	assert.Equal(t, map[string]any{"total": "2.65", "date": "01/10/2016", "company": "CORNER SHOP LTD"},
		TokensToJSON("<s><s_total> 2.65</s_total><s_date> 01/10/2016</s_date><s_company> CORNER SHOP LTD</s_company>"))

	assert.Equal(t, map[string]any{
		"store_name": "TESCOEXPRESS",
		"total":      "",
		"line_items": []any{
			map[string]any{"item_name": "Croissant", "item_value": "0.95"},
			map[string]any{"item_name": "Juice", "item_value": "1.85"},
		},
	}, TokensToJSON("<s_store_name> TESCOEXPRESS</s_store_name><s_total></s_total><s_line_items><s_item_name> Croissant</s_item_name><s_item_value> 0.95</s_item_value><sep/><s_item_name> Juice</s_item_name><s_item_value> 1.85</s_item_value></s_line_items>"))

	assert.Equal(t, map[string]any{"menu": map[string]any{"nm": "TICKET", "price": "60,000"}},
		TokensToJSON("<s_menu><s_nm> TICKET</s_nm><s_price> 60,000</s_price></s_menu>"))

	assert.Equal(t, map[string]any{"total": "5"}, TokensToJSON("<s_unclosed> x <s_total>5</s_total>"))
	assert.Equal(t, map[string]any{"text_sequence": "garbage"}, TokensToJSON("garbage"))
}

func TestFromDonut(t *testing.T) {
	receipt := FromDonut(map[string]any{
		"store_name": "CORNERSHOPLTD", "date": "01/10/2026", "time": "12:04", "subtotal": "£20.00", "total": "£20.00",
		"line_items": []any{map[string]any{"item_name": "Semi-skinmedmilk2L", "item_value": "£1.40"}},
	}, DateOrderDMY, testToday)
	assert.Equal(t, "20.00", receipt.Amount)
	assert.Equal(t, "GBP", receipt.Currency)
	assert.Equal(t, date(2026, 10, 1), receipt.Date)
	assert.Equal(t, &TimeOfDay{12, 4, 0}, receipt.Time)
	assert.Equal(t, "CORNERSHOPLTD", receipt.Supplier)
	assert.Equal(t, []string{"Semi-skinmedmilk2L"}, receipt.Items)

	sroie := FromDonut(map[string]any{"total": "40.92", "date": "22-08-2026", "company": "Table 6 COVERS 2 DATE: 22-08-2026"}, DateOrderDMY, testToday)
	assert.Equal(t, "40.92", sroie.Amount)
	assert.Equal(t, date(2026, 8, 22), sroie.Date)
	assert.Equal(t, "", sroie.Supplier)

	cord := FromDonut(map[string]any{
		"menu":  []any{map[string]any{"nm": "COCONUT JELLY", "price": "4,000"}},
		"total": map[string]any{"total_price": "28,000", "cashprice": "100,000"},
	}, DateOrderDMY, testToday)
	assert.Equal(t, "28000.00", cord.Amount)
	assert.Equal(t, []string{"COCONUT JELLY"}, cord.Items)
}

func TestMerge(t *testing.T) {
	receipts := &Recognized{Amount: "2.83", Currency: "GBP", Date: date(2026, 9, 15), Time: &TimeOfDay{8, 31, 0}, Supplier: "TESCOEXPRESS"}
	sroie := &Recognized{Amount: "6.65", Supplier: "TESCO EXPRESS OXFORD ROAD"}
	merged := Merge([]*Recognized{receipts, sroie}, FieldSources{Amount: []int{1}, Supplier: []int{0}, DateTime: []int{0}})

	assert.Equal(t, "6.65", merged.Amount)
	assert.Equal(t, "TESCO EXPRESS", merged.Supplier)
	assert.Equal(t, date(2026, 9, 15), merged.Date)
	assert.Equal(t, &TimeOfDay{8, 31, 0}, merged.Time)
	assert.Equal(t, "GBP", merged.Currency)
}

func TestMerge_AgreedAmountWins(t *testing.T) {
	merged := Merge([]*Recognized{{Amount: "1.40"}, {Amount: "20.00"}, {Amount: "20.00"}}, FieldSources{Amount: []int{0}})
	assert.Equal(t, "20.00", merged.Amount)
}

func TestMerge_SplitsCamelCase(t *testing.T) {
	assert.Equal(t, "The Red Lion", Merge([]*Recognized{{Supplier: "TheRedLion"}}, FieldSources{}).Supplier)
	assert.Equal(t, "CORNERSHOPLTD", Merge([]*Recognized{{Supplier: "CORNERSHOPLTD"}}, FieldSources{}).Supplier)
	assert.Equal(t, "", Merge([]*Recognized{nil, {}}, FieldSources{}).Supplier)
}

func TestFromText_Invoice(t *testing.T) {
	text := "ACME Supplies Ltd\nINVOICE\nInvoice No: INV-2026-0042\nDate: 3rd November 2026\nWidgets\t2\t£50.00\nSubtotal £100.00\nVAT £20.00\nTotal due £120.00"
	recognized := FromText(text, DateOrderDMY, testToday)

	assert.Equal(t, "120.00", recognized.Amount)
	assert.Equal(t, "GBP", recognized.Currency)
	assert.Equal(t, date(2026, 11, 3), recognized.Date)
	assert.Equal(t, "INV-2026-0042", recognized.InvoiceNumber)
	assert.Equal(t, "ACME Supplies Ltd", recognized.Supplier)
}

func TestFromText_PrefersLastTotal(t *testing.T) {
	assert.Equal(t, "10.80", FromText("Shop\nTotal items 3\nSubtotal 9.00\nTOTAL £10.80\nVAT 1.80", DateOrderDMY, testToday).Amount)
}

func TestFromText_SingleRowCSV(t *testing.T) {
	recognized := FromText("date,description,amount\n2026-10-01,Corner Shop,20.00", DateOrderDMY, testToday)
	assert.Equal(t, "20.00", recognized.Amount)
	assert.Equal(t, date(2026, 10, 1), recognized.Date)
	assert.Equal(t, "Corner Shop", recognized.Supplier)
}

func TestParsePromptOptionsAndChooseCategory(t *testing.T) {
	options := ParsePromptOptions(testPrompt)

	assert.Equal(t, []string{"Groceries", "Fuel", "Restaurants"}, options.ExpenseCategories)
	assert.Equal(t, []string{"Salary"}, options.IncomeCategories)
	assert.Equal(t, []string{"Current Account", "Amex"}, options.Accounts)
	assert.Equal(t, []string{"business"}, options.Tags)
	assert.Equal(t, time.Date(2026, 11, 18, 11, 0, 0, 0, time.UTC), options.Now)

	assert.Equal(t, "Fuel", ChooseCategory(&Recognized{Supplier: "SHELL WATFORD"}, options.ExpenseCategories))
	assert.Equal(t, "Groceries", ChooseCategory(&Recognized{Supplier: "TESCO EXPRESS"}, options.ExpenseCategories))
	assert.Equal(t, "", ChooseCategory(&Recognized{Supplier: "The Red Lion", Items: []string{"Fish & chips"}}, options.ExpenseCategories))
	assert.Equal(t, "Restaurants", ChooseCategory(&Recognized{Supplier: "Red Lion Pub"}, options.ExpenseCategories))
	assert.Equal(t, "Groceries", ChooseCategory(&Recognized{Supplier: "CORNER SHOP LTD", Items: []string{"Semi-skinmedmilk2L", "Wholemealbread"}}, options.ExpenseCategories))
}

func TestToRecognitionResult(t *testing.T) {
	recognized := &Recognized{Amount: "20.00", Currency: "GBP", Date: date(2026, 10, 1), Time: &TimeOfDay{12, 4, 0}, Supplier: "Tesco Express", ReceiptNumber: "004512"}
	options := ParsePromptOptions(testPrompt)

	assert.Equal(t, map[string]any{
		"type": "expense", "time": "2026-10-01 12:04:00", "amount": "20.00", "currency": "GBP", "supplier": "Tesco Express",
		"description": "Tesco Express", "receipt_number": "004512", "category": "Groceries", "account": "Amex",
	}, ToRecognitionResult(recognized, options, "Amex"))

	assert.NotContains(t, ToRecognitionResult(recognized, options, ""), "account")
	assert.Equal(t, "2026-10-01 12:00:00", ToRecognitionResult(&Recognized{Amount: "1.00", Date: date(2026, 10, 1)}, nil, "")["time"])
	assert.Empty(t, ToRecognitionResult(&Recognized{}, options, ""))
}

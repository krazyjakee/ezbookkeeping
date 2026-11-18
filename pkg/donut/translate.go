package donut

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Recognized contains the fields recognized from a document
type Recognized struct {
	Amount        string
	Currency      string
	Date          time.Time // zero if not recognized
	Time          *TimeOfDay
	Supplier      string
	ReceiptNumber string
	InvoiceNumber string
	Items         []string
}

// TimeOfDay is a time of day
type TimeOfDay struct {
	Hour, Minute, Second int
}

// PromptOptions contains the options listed in the ezBookkeeping recognition system prompt
type PromptOptions struct {
	ExpenseCategories []string
	IncomeCategories  []string
	Accounts          []string
	Tags              []string
	Now               time.Time // zero if not found
}

// DateOrder is the order of day and month in ambiguous numeric dates
type DateOrder string

// Supported date orders
const (
	DateOrderDMY DateOrder = "DMY"
	DateOrderMDY DateOrder = "MDY"
)

type currencySymbol struct {
	symbol string
	code   string
}

var currencySymbols = []currencySymbol{
	{"US$", "USD"}, {"A$", "AUD"}, {"C$", "CAD"}, {"NZ$", "NZD"}, {"HK$", "HKD"}, {"S$", "SGD"}, {"R$", "BRL"},
	{"£", "GBP"}, {"€", "EUR"}, {"¥", "JPY"}, {"₹", "INR"}, {"₩", "KRW"}, {"₽", "RUB"}, {"₺", "TRY"}, {"₫", "VND"},
	{"₱", "PHP"}, {"฿", "THB"}, {"RM", "MYR"}, {"Rp", "IDR"}, {"$", "USD"},
}

var currencyCodes = map[string]bool{
	"USD": true, "AUD": true, "CAD": true, "NZD": true, "HKD": true, "SGD": true, "BRL": true, "GBP": true, "EUR": true, "JPY": true,
	"INR": true, "KRW": true, "RUB": true, "TRY": true, "VND": true, "PHP": true, "THB": true, "MYR": true, "IDR": true, "CHF": true,
	"SEK": true, "NOK": true, "DKK": true, "PLN": true, "CZK": true, "HUF": true, "ZAR": true, "AED": true, "CNY": true,
}

var months = map[string]int{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6,
	"jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11,
	"november": 11, "dec": 12, "december": 12,
}

type categoryHint struct {
	merchantWords []string
	categoryWords []string
}

// categoryHints map merchant / item keywords to words which are looked for in the user's category names (first match wins)
var categoryHints = []categoryHint{
	{[]string{"unleaded", "diesel", "petrol", "fuel", "pump", "shell", "esso", "texaco", "bp ", "jet ", "gulf"}, []string{"fuel", "petrol", "gas", "car", "transport"}},
	{[]string{"tesco", "sainsbury", "asda", "aldi", "lidl", "morrisons", "waitrose", "co-op", "coop", "iceland", "ocado", "supermarket", "grocer", "spar", "m&s food",
		"milk", "bread", "eggs", "banana", "cheese", "cheddar", "butter", "yogurt", "yoghurt", "vegetable", "fruit", "cereal"}, []string{"grocer", "supermarket", "food"}},
	{[]string{"restaurant", "covers", "table ", "service charge", "pizza", "kitchen", "grill", "bistro", "brasserie", "pub", "inn", "arms", "tavern", "bar "}, []string{"dining", "restaurant", "eating out", "meal", "food"}},
	{[]string{"cafe", "café", "coffee", "costa", "starbucks", "pret", "greggs", "bakery"}, []string{"coffee", "cafe", "snack", "dining", "food"}},
	{[]string{"pharmacy", "chemist", "boots", "superdrug"}, []string{"pharmacy", "health", "medical"}},
	{[]string{"train", "rail", "tfl", "bus ", "taxi", "uber", "parking", "toll"}, []string{"transport", "travel", "parking", "taxi"}},
	{[]string{"hotel", "airbnb", "booking.com", "lodge"}, []string{"hotel", "accommodation", "travel"}},
}

var (
	amountPattern         = regexp.MustCompile(`-?\d{1,3}(?:[ ,.\x{00a0}]\d{3})*(?:[.,]\d{1,2})?|-?\d+(?:[.,]\d{1,2})?`)
	upperCodePattern      = regexp.MustCompile(`\b[A-Z]{3}\b`)
	isoDatePattern        = regexp.MustCompile(`(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})`)
	numericDatePattern    = regexp.MustCompile(`(\d{1,2})[-/. ](\d{1,2})[-/. ](\d{2,4})`)
	dayMonthNamePattern   = regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?[ -]*([a-z]{3,9})[ ,-]*(\d{2,4})`)
	monthNameDayPattern   = regexp.MustCompile(`([a-z]{3,9})[ -]*(\d{1,2})(?:st|nd|rd|th)?[ ,-]+(\d{2,4})`)
	timePattern           = regexp.MustCompile(`\b(\d{1,2})[:.](\d{2})(?::(\d{2}))?\s*([ap]\.?m\.?)?`)
	nameNoisePattern      = regexp.MustCompile(`(?i)\b(date|table|covers|tel|total|receipt|invoice)\b\s*[:#]?`)
	textTotalPattern      = regexp.MustCompile(`(?i)\b(grand\s+total|total\s+due|amount\s+due|balance\s+due|total\s+to\s+pay|amount\s+payable|total)\b[^\n\d£$€¥₹-]*(-?[£$€¥₹]?\s?\d[\d,.]*)`)
	textInvoicePattern    = regexp.MustCompile(`(?i)\binvoice[ \t]*(?:no\.?|number|#)?[ \t]*[:#]?[ \t]*([A-Z0-9/\-]*\d[A-Z0-9/\-]*)`)
	textReceiptPattern    = regexp.MustCompile(`(?i)\b(?:receipt|order|trans(?:action)?)[ \t]*(?:no\.?|number|#|id)?[ \t]*[:#]?[ \t]*([A-Z0-9/\-]*\d[A-Z0-9/\-]*)`)
	textTimeLinePattern   = regexp.MustCompile(`(?i)\btime\b`)
	textDateRemovePattern = regexp.MustCompile(`\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}`)
	supplierSkipPattern   = regexp.MustCompile(`(?i)(invoice|receipt|statement|date,|description,)`)
	whitespacePattern     = regexp.MustCompile(`\s+`)
	columnAmountPattern   = regexp.MustCompile(`amount|total|debit|paid|value`)
	columnNamePattern     = regexp.MustCompile(`description|payee|merchant|name|supplier|narrative`)
)

// ParseAmount returns the first amount in the text as a plain decimal string with 2 decimals, and the currency code next to it (if any)
func ParseAmount(text string) (string, string) {
	text = strings.TrimSpace(text)

	if text == "" {
		return "", ""
	}

	currency := DetectCurrency(text)
	match := amountPattern.FindString(strings.ReplaceAll(text, " ", " "))

	if match == "" {
		return "", currency
	}

	number := strings.ReplaceAll(match, " ", "")
	negative := strings.HasPrefix(number, "-")
	number = strings.TrimLeft(number, "-")
	decimalIndex := max(strings.LastIndex(number, "."), strings.LastIndex(number, ","))
	integerPart, fractionPart := number, ""

	if decimalIndex >= 0 && (len(number)-decimalIndex-1 == 1 || len(number)-decimalIndex-1 == 2) {
		integerPart, fractionPart = number[:decimalIndex], number[decimalIndex+1:]
	}

	integerPart = strings.NewReplacer(".", "", ",", "").Replace(integerPart)

	if integerPart == "" {
		integerPart = "0"
	}

	integerValue, err := strconv.ParseInt(integerPart, 10, 64)

	if err != nil {
		return "", currency
	}

	cents := int64(0)

	if fractionPart != "" {
		cents, _ = strconv.ParseInt((fractionPart + "0")[:2], 10, 64)
	}

	if integerValue == 0 && cents == 0 || integerValue > math.MaxInt64/100-1 {
		return "", currency
	}

	result := fmt.Sprintf("%d.%02d", integerValue, cents)

	if negative {
		result = "-" + result
	}

	return result, currency
}

// DetectCurrency returns the ISO 4217 code of the first currency symbol or code in the text
func DetectCurrency(text string) string {
	for _, symbol := range currencySymbols {
		if isLetters(symbol.symbol) {
			index := strings.Index(text, symbol.symbol)

			for index >= 0 {
				before := index == 0 || !unicode.IsLetter(lastRune(text[:index]))
				rest := strings.TrimPrefix(text[index+len(symbol.symbol):], " ")

				if before && rest != "" && rest[0] >= '0' && rest[0] <= '9' {
					return symbol.code
				}

				next := strings.Index(text[index+1:], symbol.symbol)

				if next < 0 {
					break
				}

				index += 1 + next
			}
		} else if strings.Contains(text, symbol.symbol) {
			return symbol.code
		}
	}

	for _, code := range upperCodePattern.FindAllString(text, -1) {
		if currencyCodes[code] {
			return code
		}
	}

	return ""
}

func isLetters(text string) bool {
	for _, r := range text {
		if !unicode.IsLetter(r) {
			return false
		}
	}

	return true
}

func lastRune(text string) rune {
	runes := []rune(text)

	if len(runes) == 0 {
		return 0
	}

	return runes[len(runes)-1]
}

// ParseDate parses a date in common receipt formats, ambiguous numeric dates use the date order.
// Years which are implausibly old (Donut models often read 2026 as 2016) are corrected by whole decades when that makes the date recent
func ParseDate(text string, order DateOrder, today time.Time) (time.Time, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	type candidate struct{ year, month, day int }
	candidates := make([]candidate, 0)

	for _, match := range isoDatePattern.FindAllStringSubmatch(text, -1) {
		candidates = append(candidates, candidate{atoi(match[1]), atoi(match[2]), atoi(match[3])})
	}

	if len(candidates) == 0 {
		for _, match := range numericDatePattern.FindAllStringSubmatch(text, -1) {
			a, b, year := atoi(match[1]), atoi(match[2]), atoi(match[3])

			if len(match[3]) == 2 {
				year += 2000
			}

			var day, month int

			switch {
			case order == DateOrderMDY && a <= 12:
				month, day = a, b
			case a > 12 && b <= 12:
				day, month = a, b
			case b > 12 && a <= 12:
				month, day = a, b
			default:
				day, month = a, b
			}

			candidates = append(candidates, candidate{year, month, day})
		}
	}

	if len(candidates) == 0 {
		for _, match := range dayMonthNamePattern.FindAllStringSubmatch(text, -1) {
			if month, exists := months[match[2]]; exists {
				candidates = append(candidates, candidate{twoDigitYear(match[3]), month, atoi(match[1])})
			}
		}

		for _, match := range monthNameDayPattern.FindAllStringSubmatch(text, -1) {
			if month, exists := months[match[1]]; exists {
				candidates = append(candidates, candidate{twoDigitYear(match[3]), month, atoi(match[2])})
			}
		}
	}

	for _, c := range candidates {
		if c.month < 1 || c.month > 12 || c.day < 1 || c.day > 31 {
			continue
		}

		date := time.Date(c.year, time.Month(c.month), c.day, 0, 0, 0, 0, time.UTC)

		if date.Day() != c.day {
			continue
		}

		return correctYear(date, today)
	}

	return time.Time{}, false
}

func twoDigitYear(text string) int {
	year := atoi(text)

	if len(text) == 2 {
		year += 2000
	}

	return year
}

func atoi(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}

// correctYear moves a date which is more than 2 years old forward by whole decades if that makes it recent,
// and rejects dates which are still implausible (older than 10 years or more than a week in the future)
func correctYear(date time.Time, today time.Time) (time.Time, bool) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	earliestRecent := today.AddDate(0, 0, -730)
	latest := today.AddDate(0, 0, 7)

	if date.After(latest) {
		return time.Time{}, false
	}

	if date.Before(earliestRecent) {
		for decades := 1; decades <= 2; decades++ {
			shifted := time.Date(date.Year()+10*decades, date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

			if shifted.Day() == date.Day() && !shifted.Before(earliestRecent) && !shifted.After(latest) {
				return shifted, true
			}
		}
	}

	if date.Before(today.AddDate(0, 0, -3650)) {
		return time.Time{}, false
	}

	return date, true
}

// ParseTimeOfDay parses a time of day such as 12:04, 17:02:11 or 7.45pm
func ParseTimeOfDay(text string) *TimeOfDay {
	match := timePattern.FindStringSubmatch(strings.ToLower(text))

	if match == nil {
		return nil
	}

	hour, minute, second := atoi(match[1]), atoi(match[2]), atoi(match[3])

	if strings.HasPrefix(match[4], "p") && hour < 12 {
		hour += 12
	} else if strings.HasPrefix(match[4], "a") && hour == 12 {
		hour = 0
	}

	if hour > 23 || minute > 59 || second > 59 {
		return nil
	}

	return &TimeOfDay{hour, minute, second}
}

// cleanName returns a merchant name, or empty if the text does not look like one
func cleanName(value string) string {
	name := strings.Trim(whitespacePattern.ReplaceAllString(value, " "), " -:*,.")
	digits := 0

	for _, r := range name {
		if unicode.IsDigit(r) {
			digits++
		}
	}

	if name == "" || len([]rune(name)) > 60 || float64(digits) > float64(len([]rune(name)))/3 || nameNoisePattern.MatchString(name) {
		return ""
	}

	return name
}

// flatten returns the leaf values of a Donut parse by key over all nesting levels
func flatten(parse any) map[string][]string {
	result := make(map[string][]string)
	var walk func(node any)

	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			keys := make([]string, 0, len(value))

			for key := range value {
				keys = append(keys, key)
			}

			sort.Strings(keys)

			for _, key := range keys {
				switch child := value[key].(type) {
				case string:
					if child != "" {
						result[key] = append(result[key], child)
					}
				case []any:
					allText := true

					for _, item := range child {
						if text, ok := item.(string); ok {
							if text != "" {
								result[key] = append(result[key], text)
							}
						} else {
							allText = false
						}
					}

					if !allText {
						walk(child)
					}
				default:
					walk(child)
				}
			}
		case []any:
			for _, item := range value {
				walk(item)
			}
		}
	}

	walk(parse)
	return result
}

var (
	totalKeys    = []string{"total", "total_price", "total_gross_worth", "subtotal", "subtotal_price"}
	supplierKeys = []string{"company", "store_name", "seller"}
	dateKeys     = []string{"date", "invoice_date"}
)

// FromDonut converts one Donut parse (SROIE, CORD, invoice / receipt schemas) into recognized fields
func FromDonut(parse map[string]any, order DateOrder, today time.Time) *Recognized {
	flat := flatten(parse)
	recognized := &Recognized{}

	for _, key := range totalKeys {
		for _, value := range flat[key] {
			if amount, currency := ParseAmount(value); amount != "" {
				recognized.Amount, recognized.Currency = amount, currency
				break
			}
		}

		if recognized.Amount != "" {
			break
		}
	}

	if recognized.Currency == "" {
		keys := make([]string, 0, len(flat))

		for key := range flat {
			keys = append(keys, key)
		}

		sort.Strings(keys)

	currencyLoop:
		for _, key := range keys {
			for _, value := range flat[key] {
				if currency := DetectCurrency(value); currency != "" {
					recognized.Currency = currency
					break currencyLoop
				}
			}
		}
	}

dateLoop:
	for _, key := range dateKeys {
		for _, value := range flat[key] {
			if date, ok := ParseDate(value, order, today); ok {
				recognized.Date = date
				recognized.Time = ParseTimeOfDay(value)
				break dateLoop
			}
		}
	}

	for _, value := range flat["time"] {
		if parsed := ParseTimeOfDay(value); parsed != nil {
			recognized.Time = parsed
			break
		}
	}

supplierLoop:
	for _, key := range supplierKeys {
		for _, value := range flat[key] {
			if name := cleanName(value); name != "" {
				recognized.Supplier = name
				break supplierLoop
			}
		}
	}

	if values := flat["invoice_no"]; len(values) > 0 {
		recognized.InvoiceNumber = strings.TrimSpace(values[0])
	}

	for _, key := range []string{"item_desc", "item_name", "nm"} {
		for _, value := range flat[key] {
			if value = strings.TrimSpace(value); value != "" {
				recognized.Items = append(recognized.Items, value)
			}
		}
	}

	return recognized
}

// FieldSources lists which recognitions (by index) are preferred for each field when merging, the remaining ones follow in their original order
type FieldSources struct {
	Amount   []int
	DateTime []int
	Supplier []int
}

func orderedSources(preferred []int, count int) []int {
	result := make([]int, 0, count)
	seen := make(map[int]bool)

	for _, index := range preferred {
		if index >= 0 && index < count && !seen[index] {
			result = append(result, index)
			seen[index] = true
		}
	}

	for index := 0; index < count; index++ {
		if !seen[index] {
			result = append(result, index)
		}
	}

	return result
}

// Merge merges the fields recognized by several models / extractors. An amount which two of them agree on wins,
// otherwise each field is taken from the first source (in the preferred order) which recognized it. A merchant name without
// spaces ("CORNERSHOPLTD") takes the spacing of another source's reading of the same name, and CamelCase names are split
func Merge(parses []*Recognized, sources FieldSources) *Recognized {
	merged := &Recognized{}
	count := len(parses)
	amounts := make(map[string]int)

	for _, p := range parses {
		if p != nil && p.Amount != "" {
			amounts[p.Amount]++
		}
	}

	for _, index := range orderedSources(sources.Amount, count) {
		if p := parses[index]; p != nil && p.Amount != "" && amounts[p.Amount] > 1 {
			merged.Amount = p.Amount
			break
		}
	}

	if merged.Amount == "" {
		for _, index := range orderedSources(sources.Amount, count) {
			if p := parses[index]; p != nil && p.Amount != "" {
				merged.Amount = p.Amount
				break
			}
		}
	}

	for _, index := range orderedSources(sources.DateTime, count) {
		if p := parses[index]; p != nil {
			if merged.Date.IsZero() && !p.Date.IsZero() {
				merged.Date = p.Date

				if merged.Time == nil {
					merged.Time = p.Time
				}
			}
		}
	}

	for _, index := range orderedSources(sources.DateTime, count) {
		if p := parses[index]; p != nil && merged.Time == nil && p.Time != nil {
			merged.Time = p.Time
		}
	}

	suppliers := make([]string, 0)

	for _, index := range orderedSources(sources.Supplier, count) {
		if p := parses[index]; p != nil && p.Supplier != "" {
			suppliers = append(suppliers, p.Supplier)
		}
	}

	for _, p := range parses {
		if p == nil {
			continue
		}

		if merged.Currency == "" {
			merged.Currency = p.Currency
		}

		if merged.ReceiptNumber == "" {
			merged.ReceiptNumber = p.ReceiptNumber
		}

		if merged.InvoiceNumber == "" {
			merged.InvoiceNumber = p.InvoiceNumber
		}

		if len(merged.Items) == 0 {
			merged.Items = p.Items
		}
	}

	if len(suppliers) > 0 {
		merged.Supplier = suppliers[0]

		if !strings.Contains(merged.Supplier, " ") {
			for _, other := range suppliers[1:] {
				if strings.Contains(other, " ") {
					if respaced := respace(merged.Supplier, other); respaced != "" {
						merged.Supplier = respaced
						break
					}
				}
			}
		}

		if !strings.Contains(merged.Supplier, " ") {
			merged.Supplier = splitCamelCase(merged.Supplier)
		}
	}

	return merged
}

func squash(text string) string {
	var builder strings.Builder

	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// respace returns the prefix of spaced which has the same letters as name ("CORNERSHOPLTD" and "CORNER SHOP LTD, 12 HIGH ST" -> "CORNER SHOP LTD")
func respace(name string, spaced string) string {
	target := squash(name)

	if target == "" || !strings.HasPrefix(squash(spaced), target) {
		return ""
	}

	count := 0

	for index, r := range spaced {
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			count++
		}

		if count == len(target) {
			return spaced[:index+len(string(r))]
		}
	}

	return ""
}

// splitCamelCase returns "The Red Lion" for "TheRedLion"
func splitCamelCase(name string) string {
	runes := []rune(name)
	var builder strings.Builder

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			builder.WriteRune(' ')
		}

		builder.WriteRune(r)
	}

	return builder.String()
}

type textTotal struct {
	rank  int
	index int
	value string
}

// FromText extracts the fields from document text (office documents, CSV, text layers of PDF documents) with regular expressions
func FromText(text string, order DateOrder, today time.Time) *Recognized {
	recognized := &Recognized{}
	lines := make([]string, 0)

	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}

	totals := make([]textTotal, 0)

	for index, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "sub") {
			continue
		}

		for _, match := range textTotalPattern.FindAllStringSubmatch(line, -1) {
			rank := 1

			if strings.EqualFold(match[1], "total") {
				rank = 0
			}

			totals = append(totals, textTotal{rank, index, match[2]})
		}
	}

	// prefer the specific keywords (amount due, grand total...), then the last occurrence (totals are usually at the bottom)
	sort.SliceStable(totals, func(i, j int) bool {
		if totals[i].rank != totals[j].rank {
			return totals[i].rank > totals[j].rank
		}

		return totals[i].index > totals[j].index
	})

	for _, total := range totals {
		if amount, currency := ParseAmount(total.value); amount != "" {
			recognized.Amount, recognized.Currency = amount, currency
			break
		}
	}

	if recognized.Amount == "" {
		readSingleRowTable(lines, recognized, order, today)
	}

	if recognized.Currency == "" {
		recognized.Currency = DetectCurrency(text)
	}

	for _, line := range lines {
		if date, ok := ParseDate(line, order, today); ok {
			recognized.Date = date
			recognized.Time = ParseTimeOfDay(textDateRemovePattern.ReplaceAllString(line, ""))
			break
		}
	}

	if !recognized.Date.IsZero() && recognized.Time == nil {
		for _, line := range lines {
			if textTimeLinePattern.MatchString(line) {
				if parsed := ParseTimeOfDay(line); parsed != nil {
					recognized.Time = parsed
					break
				}
			}
		}
	}

	if match := textInvoicePattern.FindStringSubmatch(text); match != nil {
		recognized.InvoiceNumber = match[1]
	}

	if match := textReceiptPattern.FindStringSubmatch(text); match != nil {
		recognized.ReceiptNumber = match[1]
	}

	if recognized.Supplier == "" {
		for i := 0; i < len(lines) && i < 5; i++ {
			candidate := cleanName(strings.Split(lines[i], ",")[0])

			if candidate != "" && len([]rune(candidate)) >= 3 && !supplierSkipPattern.MatchString(candidate) {
				recognized.Supplier = candidate
				break
			}
		}
	}

	return recognized
}

// readSingleRowTable reads amount, date and description from a table with a header row and exactly one data row (e.g. a CSV export of one payment)
func readSingleRowTable(lines []string, recognized *Recognized, order DateOrder, today time.Time) {
	if len(lines) != 2 {
		return
	}

	delimiter := ","

	for _, candidate := range []string{";", "\t", "|"} {
		if strings.Count(lines[0], candidate) > strings.Count(lines[0], delimiter) {
			delimiter = candidate
		}
	}

	header := strings.Split(lines[0], delimiter)
	row := strings.Split(lines[1], delimiter)

	if len(header) < 2 || len(header) != len(row) {
		return
	}

	for index, name := range header {
		name = strings.ToLower(strings.Trim(strings.TrimSpace(name), `"`))
		cell := strings.Trim(strings.TrimSpace(row[index]), `"`)

		switch {
		case recognized.Amount == "" && columnAmountPattern.MatchString(name):
			var currency string
			recognized.Amount, currency = ParseAmount(cell)

			if recognized.Currency == "" {
				recognized.Currency = currency
			}
		case recognized.Date.IsZero() && strings.Contains(name, "date"):
			if date, ok := ParseDate(cell, order, today); ok {
				recognized.Date = date
			}
		case recognized.Supplier == "" && columnNamePattern.MatchString(name):
			recognized.Supplier = cleanName(cell)
		}
	}
}

// ParsePromptOptions reads the option lists from the ezBookkeeping recognition system prompt (sections such as "### Expense categories:")
func ParsePromptOptions(systemPrompt string) *PromptOptions {
	options := &PromptOptions{}
	var current *[]string
	readNow := false

	for _, raw := range strings.Split(systemPrompt, "\n") {
		line := strings.TrimSpace(raw)

		if strings.HasPrefix(line, "#") {
			title := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(strings.TrimLeft(line, "#")), ":"))
			current = nil
			readNow = false

			switch title {
			case "expense categories":
				current = &options.ExpenseCategories
			case "income categories":
				current = &options.IncomeCategories
			case "account names":
				current = &options.Accounts
			case "tags":
				current = &options.Tags
			case "current time":
				readNow = true
			}

			continue
		}

		if line == "" {
			continue
		}

		if readNow {
			if now, err := time.Parse("2006-01-02 15:04:05", line); err == nil {
				options.Now = now
			}

			readNow = false
			continue
		}

		if current != nil {
			*current = append(*current, line)
		}
	}

	return options
}

// ChooseCategory chooses an expense category from the user's categories by merchant / item keywords
func ChooseCategory(recognized *Recognized, categories []string) string {
	if len(categories) == 0 {
		return ""
	}

	haystack := " " + strings.ToLower(strings.Join(append([]string{recognized.Supplier}, recognized.Items...), " ")) + " "

	for _, hint := range categoryHints {
		matched := false

		for _, word := range hint.merchantWords {
			if strings.Contains(haystack, word) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		for _, categoryWord := range hint.categoryWords {
			for _, category := range categories {
				if strings.Contains(strings.ToLower(category), categoryWord) {
					return category
				}
			}
		}
	}

	return ""
}

// ToRecognitionResult returns the transaction in the json schema of the ezBookkeeping recognition prompts (empty if nothing useful was found)
func ToRecognitionResult(recognized *Recognized, options *PromptOptions, defaultAccount string) map[string]any {
	result := map[string]any{}

	if recognized.Amount == "" && recognized.Supplier == "" {
		return result
	}

	result["type"] = "expense"

	if !recognized.Date.IsZero() {
		timeOfDay := TimeOfDay{Hour: 12}

		if recognized.Time != nil {
			timeOfDay = *recognized.Time
		}

		result["time"] = fmt.Sprintf("%s %02d:%02d:%02d", recognized.Date.Format("2006-01-02"), timeOfDay.Hour, timeOfDay.Minute, timeOfDay.Second)
	}

	if recognized.Amount != "" {
		result["amount"] = strings.TrimPrefix(recognized.Amount, "-")
	}

	if recognized.Currency != "" {
		result["currency"] = recognized.Currency
	}

	if recognized.Supplier != "" {
		result["supplier"] = recognized.Supplier
		result["description"] = recognized.Supplier
	}

	if recognized.ReceiptNumber != "" {
		result["receipt_number"] = recognized.ReceiptNumber
	}

	if recognized.InvoiceNumber != "" {
		result["invoice_number"] = recognized.InvoiceNumber
	}

	if options != nil {
		if category := ChooseCategory(recognized, options.ExpenseCategories); category != "" {
			result["category"] = category
		}

		if defaultAccount != "" {
			for _, account := range options.Accounts {
				if account == defaultAccount {
					result["account"] = defaultAccount
					break
				}
			}
		} else if len(options.Accounts) == 1 {
			result["account"] = options.Accounts[0]
		}
	}

	return result
}

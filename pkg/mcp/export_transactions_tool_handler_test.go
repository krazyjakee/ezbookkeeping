package mcp

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPExportBuildCSV_HeaderAndEscaping(t *testing.T) {
	details := []*MCPTransactionDetail{
		{
			Id:               "100",
			Type:             transactionTypeExpense,
			Time:             "2024-01-02T03:04:05+01:00",
			UtcOffsetMinutes: 60,
			Amount:           "120.00",
			Currency:         "GBP",
			AccountId:        "1",
			AccountName:      "Director's Loan, GBP",
			CategoryId:       "11",
			CategoryName:     "Software \"SaaS\"",
			Tags: []*MCPTransactionDetailTag{
				{Id: "7", Name: "Director"},
				{Id: "8", Name: "Business"},
			},
			Comment: "line 1\nline 2",
			Attachments: []*MCPTransactionDetailAttachment{
				{Id: "9", SHA256: "abc", OriginalFileName: "receipt.pdf"},
			},
			References: []*MCPTransactionReference{
				{Type: "supplier", Value: "ACME Ltd"},
				{Type: "invoice_number", Value: "INV-001"},
			},
		},
	}

	csvText, err := mcpExportBuildCSV(details)
	assert.Nil(t, err)

	records, err := csv.NewReader(strings.NewReader(csvText)).ReadAll()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(records))
	assert.Equal(t, mcpExportCSVColumns, records[0])
	assert.Equal(t, len(mcpExportCSVColumns), len(records[1]))

	record := make(map[string]string)

	for i, column := range mcpExportCSVColumns {
		record[column] = records[1][i]
	}

	assert.Equal(t, "100", record["transaction_id"])
	assert.Equal(t, "60", record["utc_offset_minutes"])
	assert.Equal(t, "Director's Loan, GBP", record["account_name"])
	assert.Equal(t, "Software \"SaaS\"", record["category_name"])
	assert.Equal(t, "line 1\nline 2", record["comment"])
	assert.Equal(t, "7; 8", record["tag_ids"])
	assert.Equal(t, "Director; Business", record["tags"])
	assert.Equal(t, "9", record["attachment_ids"])
	assert.Equal(t, "abc", record["attachment_sha256"])
	assert.Equal(t, "receipt.pdf", record["attachment_file_names"])
	assert.Equal(t, "supplier=ACME Ltd; invoice_number=INV-001", record["references"])

	assert.True(t, strings.Contains(csvText, "\"Director's Loan, GBP\""))
	assert.True(t, strings.Contains(csvText, "\"Software \"\"SaaS\"\"\""))
	assert.True(t, strings.Contains(csvText, "\"line 1\nline 2\""))
}

func TestMCPExportBuildCSV_FormulaInjection(t *testing.T) {
	details := []*MCPTransactionDetail{
		{
			Id:          "100",
			Type:        transactionTypeModifyBalance,
			Amount:      "-12.00",
			AccountName: "=HYPERLINK(\"http://example.com\")",
			Comment:     "@SUM(A1)",
			References: []*MCPTransactionReference{
				{Type: "supplier", Value: "+cmd"},
			},
		},
	}

	csvText, err := mcpExportBuildCSV(details)
	assert.Nil(t, err)

	records, err := csv.NewReader(strings.NewReader(csvText)).ReadAll()
	assert.Nil(t, err)

	record := make(map[string]string)

	for i, column := range mcpExportCSVColumns {
		record[column] = records[1][i]
	}

	assert.Equal(t, "-12.00", record["amount"])
	assert.Equal(t, "'=HYPERLINK(\"http://example.com\")", record["account_name"])
	assert.Equal(t, "'@SUM(A1)", record["comment"])
	assert.Equal(t, "supplier=+cmd", record["references"])
}

func TestMCPExportSanitizeCSVTextCell(t *testing.T) {
	assert.Equal(t, "", mcpExportSanitizeCSVTextCell(""))
	assert.Equal(t, "Normal", mcpExportSanitizeCSVTextCell("Normal"))
	assert.Equal(t, "'=1+1", mcpExportSanitizeCSVTextCell("=1+1"))
	assert.Equal(t, "'+1", mcpExportSanitizeCSVTextCell("+1"))
	assert.Equal(t, "'-1", mcpExportSanitizeCSVTextCell("-1"))
	assert.Equal(t, "'@x", mcpExportSanitizeCSVTextCell("@x"))
	assert.Equal(t, "'\tx", mcpExportSanitizeCSVTextCell("\tx"))
}

func TestMCPExportBuildCSV_Empty(t *testing.T) {
	csvText, err := mcpExportBuildCSV(nil)
	assert.Nil(t, err)
	assert.Equal(t, strings.Join(mcpExportCSVColumns, ",")+"\n", csvText)
}

func TestMCPTxnParseQueryTransactionType(t *testing.T) {
	transactionType, err := mcpTxnParseQueryTransactionType("type", "")
	assert.Nil(t, err)
	assert.Equal(t, 0, int(transactionType))

	_, err = mcpTxnParseQueryTransactionType("type", "unknown")
	assert.Equal(t, "type", mcpTestGetToolError(t, err).Field)
}

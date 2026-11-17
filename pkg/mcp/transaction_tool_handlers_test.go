package mcp

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func mcpTxnTestGetToolError(t *testing.T, err error) *MCPToolError {
	t.Helper()
	var toolErr *MCPToolError

	if !errors.As(err, &toolErr) {
		t.Fatalf("expected MCPToolError, but got %v", err)
	}

	return toolErr
}

func mcpTxnTestNewAddRequest() *MCPAddTransactionRequest {
	return &MCPAddTransactionRequest{
		Type:                  transactionTypeExpense,
		Time:                  "2024-01-02T03:04:05+01:00",
		AccountName:           "Cash",
		SecondaryCategoryName: "Food",
		Amount:                "12.34",
		DryRun:                true,
	}
}

func TestMCPTxnAddTransactionValidateRequest_CommentLength(t *testing.T) {
	request := mcpTxnTestNewAddRequest()
	request.Comment = strings.Repeat("中", mcpTransactionCommentMaxLength)
	assert.Nil(t, MCPAddTransactionToolHandler.validateRequest(request))

	request.Comment = strings.Repeat("a", mcpTransactionCommentMaxLength+1)
	toolErr := mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request))
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "comment", toolErr.Field)
	assert.Equal(t, int64(mcpTransactionCommentMaxLength), *toolErr.Limit)
	assert.Equal(t, int64(mcpTransactionCommentMaxLength+1), *toolErr.Actual)
}

func TestMCPTxnAddTransactionValidateRequest_CommentMaxLengthMatchesModel(t *testing.T) {
	assert.Equal(t, getModelBindingMaxValue(reflect.TypeOf(models.TransactionCreateRequest{}), "Comment", -1), mcpTransactionCommentMaxLength)
}

func TestMCPTxnAddTransactionValidateRequest_InvalidFields(t *testing.T) {
	request := mcpTxnTestNewAddRequest()
	request.Type = "unknown"
	assert.Equal(t, "type", mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Field)

	request = mcpTxnTestNewAddRequest()
	request.Type = transactionTypeTransfer
	assert.Equal(t, MCPToolErrorCodeRequired, mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Code)

	request.DestinationAccountName = "Bank"
	toolErr := mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request))
	assert.Equal(t, "destination_amount", toolErr.Field)

	request = mcpTxnTestNewAddRequest()
	request.DestinationAmount = "1"
	assert.Equal(t, "destination_amount", mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Field)

	request = mcpTxnTestNewAddRequest()
	request.Time = "2024-01-02 03:04:05"
	assert.Equal(t, "time", mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Field)

	request = mcpTxnTestNewAddRequest()
	request.Amount = "12.345"
	assert.Equal(t, "amount", mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Field)

	request = mcpTxnTestNewAddRequest()
	request.GeoLocation = &MCPTransactionDetailGeoLocation{Latitude: 91, Longitude: 0}
	assert.Equal(t, "geo_location.latitude", mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request)).Field)

	request = mcpTxnTestNewAddRequest()
	request.Tags = make([]string, models.MaximumTagsCountOfTransaction+1)
	toolErr = mcpTxnTestGetToolError(t, MCPAddTransactionToolHandler.validateRequest(request))
	assert.Equal(t, MCPToolErrorCodeTooMany, toolErr.Code)
	assert.Equal(t, int64(models.MaximumTagsCountOfTransaction), *toolErr.Limit)
}

func TestMCPTxnResolveTagIds(t *testing.T) {
	allTags := []*models.TransactionTag{
		{TagId: 1, Name: "Business"},
		{TagId: 2, Name: "Business"},
		{TagId: 3, Name: "Receipt"},
		{TagId: 4, Name: "Archived", Hidden: true},
	}

	tagIds, err := mcpTxnResolveTagIds(allTags, "tag_ids", nil, "tags", []string{"Receipt", "receipt "})
	assert.Nil(t, err)
	assert.Equal(t, []int64{3}, tagIds)

	tagIds, err = mcpTxnResolveTagIds(allTags, "tag_ids", []string{"4", "3"}, "tags", []string{"Receipt"})
	assert.Nil(t, err)
	assert.Equal(t, []int64{4, 3}, tagIds)

	_, err = mcpTxnResolveTagIds(allTags, "tag_ids", nil, "tags", []string{"Business"})
	toolErr := mcpTxnTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, "tags", toolErr.Field)
	assert.Equal(t, 2, len(toolErr.Candidates))
	assert.Equal(t, "1", toolErr.Candidates[0].Id)
	assert.Equal(t, "2", toolErr.Candidates[1].Id)

	_, err = mcpTxnResolveTagIds(allTags, "tag_ids", nil, "tags", []string{"Unknown"})
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveTagIds(allTags, "tag_ids", nil, "tags", []string{"Archived"})
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveTagIds(allTags, "tag_ids", []string{"99"}, "tags", nil)
	toolErr = mcpTxnTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotFound, toolErr.Code)
	assert.Equal(t, "tag_ids", toolErr.Field)

	_, err = mcpTxnResolveTagIds(allTags, "tag_ids", []string{"abc"}, "tags", nil)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTxnTestGetToolError(t, err).Code)
}

func TestMCPTxnResolveTagIds_TooMany(t *testing.T) {
	allTags := make([]*models.TransactionTag, 0)
	tagIds := make([]string, 0)

	for i := 1; i <= models.MaximumTagsCountOfTransaction+1; i++ {
		allTags = append(allTags, &models.TransactionTag{TagId: int64(i), Name: strconv.Itoa(i)})
		tagIds = append(tagIds, strconv.Itoa(i))
	}

	_, err := mcpTxnResolveTagIds(allTags, "tag_ids", tagIds, "tags", nil)
	assert.Equal(t, MCPToolErrorCodeTooMany, mcpTxnTestGetToolError(t, err).Code)
}

func TestMCPTxnResolveAccount(t *testing.T) {
	allAccounts := []*models.Account{
		{AccountId: 1, Name: "Cash", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP"},
		{AccountId: 2, Name: "Bank", Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS},
		{AccountId: 3, Name: "Current", ParentAccountId: 2, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP"},
		{AccountId: 4, Name: "Current", ParentAccountId: 2, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "USD"},
		{AccountId: 5, Name: "Old", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Hidden: true},
	}

	account, err := mcpTxnResolveAccount(allAccounts, "account_id", "", "account_name", "Cash")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), account.AccountId)

	account, err = mcpTxnResolveAccount(allAccounts, "account_id", "4", "account_name", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(4), account.AccountId)

	_, err = mcpTxnResolveAccount(allAccounts, "account_id", "", "account_name", "Current")
	toolErr := mcpTxnTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))

	_, err = mcpTxnResolveAccount(allAccounts, "account_id", "2", "account_name", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveAccount(allAccounts, "account_id", "", "account_name", "Bank")
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveAccount(allAccounts, "account_id", "", "account_name", "Old")
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveAccount(allAccounts, "account_id", "", "account_name", "")
	assert.Equal(t, MCPToolErrorCodeRequired, mcpTxnTestGetToolError(t, err).Code)
}

func TestMCPTxnResolveCategory(t *testing.T) {
	allCategories := []*models.TransactionCategory{
		{CategoryId: 1, Name: "Business", Type: models.CATEGORY_TYPE_EXPENSE},
		{CategoryId: 2, Name: "Software", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1},
		{CategoryId: 3, Name: "Personal", Type: models.CATEGORY_TYPE_EXPENSE},
		{CategoryId: 4, Name: "Software", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 3},
		{CategoryId: 5, Name: "Salary", Type: models.CATEGORY_TYPE_INCOME},
		{CategoryId: 6, Name: "Salary", Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 5},
	}

	category, err := mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_INCOME, "category_id", "", "category_name", "Salary")
	assert.Nil(t, err)
	assert.Equal(t, int64(6), category.CategoryId)

	_, err = mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_EXPENSE, "category_id", "", "category_name", "Software")
	toolErr := mcpTxnTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))
	assert.Contains(t, toolErr.Candidates[0].Description, "Business")
	assert.Contains(t, toolErr.Candidates[1].Description, "Personal")

	category, err = mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_EXPENSE, "category_id", "4", "category_name", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(4), category.CategoryId)

	_, err = mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_EXPENSE, "category_id", "1", "category_name", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_EXPENSE, "category_id", "6", "category_name", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnResolveCategory(allCategories, models.CATEGORY_TYPE_TRANSFER, "category_id", "", "category_name", "Salary")
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTxnTestGetToolError(t, err).Code)
}

func TestMCPTxnParseAmountFilter(t *testing.T) {
	testCases := map[string]string{
		"":            "",
		"gt:12.34":    "gt:1234",
		"lt:5":        "lt:500",
		"eq:-1.50":    "eq:-150",
		"ne:0":        "ne:0",
		"bt:10:20.50": "bt:1000:2050",
		"nb:1:1":      "nb:100:100",
	}

	for value, expected := range testCases {
		actual, err := mcpTxnParseAmountFilter("amount_filter", value)
		assert.Nil(t, err, value)
		assert.Equal(t, expected, actual, value)
	}

	for _, value := range []string{"gt", "gt:abc", "gt:1:2", "bt:1", "bt:20:10", "xx:1", "gt:1.234"} {
		_, err := mcpTxnParseAmountFilter("amount_filter", value)
		toolErr := mcpTxnTestGetToolError(t, err)
		assert.Equal(t, "amount_filter", toolErr.Field, value)
	}
}

func TestMCPTxnGetTagFilters(t *testing.T) {
	allTags := []*models.TransactionTag{
		{TagId: 1, Name: "A"},
		{TagId: 2, Name: "B"},
	}

	filters, err := mcpTxnGetTagFilters(allTags, nil, nil, "", false)
	assert.Nil(t, err)
	assert.Nil(t, filters)

	filters, err = mcpTxnGetTagFilters(allTags, []string{"1"}, []string{"B", "A"}, "has_all", false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(filters))
	assert.Equal(t, []int64{1, 2}, filters[0].TagIds)
	assert.Equal(t, models.TRANSACTION_TAG_FILTER_HAS_ALL, filters[0].Type)

	_, err = mcpTxnGetTagFilters(allTags, []string{"1"}, nil, "invalid", false)
	assert.Equal(t, "tag_filter_mode", mcpTxnTestGetToolError(t, err).Field)

	_, err = mcpTxnGetTagFilters(allTags, []string{"1"}, nil, "", true)
	assert.Equal(t, "no_tags", mcpTxnTestGetToolError(t, err).Field)
}

func TestMCPTxnBuildDetail_Transfer(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:        100,
		Type:                 models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		CategoryId:           11,
		TransactionTime:      utils.GetMinTransactionTimeFromUnixTime(1704164645),
		TimezoneUtcOffset:    60,
		AccountId:            1,
		Amount:               12000,
		RelatedAccountId:     2,
		RelatedAccountAmount: 15000,
		Comment:              "Reimbursement",
		GeoLatitude:          51.5,
		GeoLongitude:         -0.12,
	}

	data := &mcpTxnDetailData{
		accounts: map[int64]*models.Account{
			1: {AccountId: 1, Name: "Bank", Currency: "GBP"},
			2: {AccountId: 2, Name: "Director Loan", Currency: "USD"},
		},
		categories: map[int64]*models.TransactionCategory{
			10: {CategoryId: 10, Name: "Business"},
			11: {CategoryId: 11, Name: "Reimbursement", ParentCategoryId: 10},
		},
		tags: map[int64]*models.TransactionTag{
			7: {TagId: 7, Name: "Director"},
		},
		tagIds: map[int64][]int64{
			100: {7},
		},
		attachments: map[int64][]*models.TransactionPictureInfo{
			100: {{PictureId: 9, TransactionId: 100, PictureExtension: "jpg"}},
		},
	}

	user := &models.User{TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL}
	detail := mcpTxnBuildDetail(transaction, data, user, true)

	assert.Equal(t, "100", detail.Id)
	assert.Equal(t, transactionTypeTransfer, detail.Type)
	assert.Equal(t, "2024-01-02T04:04:05+01:00", detail.Time)
	assert.Equal(t, int16(60), detail.UtcOffsetMinutes)
	assert.Equal(t, "120.00", detail.Amount)
	assert.Equal(t, "GBP", detail.Currency)
	assert.Equal(t, "1", detail.AccountId)
	assert.Equal(t, "Bank", detail.AccountName)
	assert.Equal(t, "150.00", detail.DestinationAmount)
	assert.Equal(t, "USD", detail.DestinationCurrency)
	assert.Equal(t, "2", detail.DestinationAccountId)
	assert.Equal(t, "Director Loan", detail.DestinationAccountName)
	assert.Equal(t, "11", detail.CategoryId)
	assert.Equal(t, "Reimbursement", detail.CategoryName)
	assert.Equal(t, "10", detail.PrimaryCategoryId)
	assert.Equal(t, "Business", detail.PrimaryCategoryName)
	assert.Equal(t, 1, len(detail.Tags))
	assert.Equal(t, "7", detail.Tags[0].Id)
	assert.Equal(t, "Director", detail.Tags[0].Name)
	assert.Equal(t, 51.5, detail.GeoLocation.Latitude)
	assert.Equal(t, -0.12, detail.GeoLocation.Longitude)
	assert.Equal(t, 1, detail.AttachmentCount)
	assert.Equal(t, 1, len(detail.Attachments))
	assert.Equal(t, "9", detail.Attachments[0].Id)
	assert.Equal(t, "jpg", detail.Attachments[0].Extension)
	assert.True(t, *detail.Editable)

	data.accounts[2].Hidden = true
	detail = mcpTxnBuildDetail(transaction, data, user, false)
	assert.False(t, *detail.Editable)
	assert.Nil(t, detail.Attachments)
	assert.Equal(t, 1, detail.AttachmentCount)

	detail = mcpTxnBuildDetail(transaction, data, nil, false)
	assert.Nil(t, detail.Editable)
}

func TestMCPTxnBuildDetail_BalanceModificationWithoutRelatedData(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:   1,
		Type:            models.TRANSACTION_DB_TYPE_MODIFY_BALANCE,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(1704164645),
		AccountId:       1,
		Amount:          100,
	}

	detail := mcpTxnBuildDetail(transaction, nil, nil, true)
	assert.Equal(t, transactionTypeModifyBalance, detail.Type)
	assert.Equal(t, "", detail.CategoryId)
	assert.Equal(t, 0, len(detail.Tags))
	assert.Nil(t, detail.GeoLocation)
	assert.Equal(t, 0, detail.AttachmentCount)
}

func TestMCPTxnValidateAccounts(t *testing.T) {
	gbp1 := &models.Account{AccountId: 1, Name: "Bank", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT}
	gbp2 := &models.Account{AccountId: 2, Name: "Loan", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT}
	usd := &models.Account{AccountId: 3, Name: "USD", Currency: "USD", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT}
	hidden := &models.Account{AccountId: 4, Name: "Hidden", Currency: "GBP", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Hidden: true}

	transfer := &models.Transaction{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, AccountId: 1, Amount: 100, RelatedAccountId: 2, RelatedAccountAmount: 100}
	assert.Nil(t, mcpTxnValidateAccounts(transfer, gbp1, gbp2))

	transfer.RelatedAccountAmount = 200
	assert.Equal(t, "destination_amount", mcpTxnTestGetToolError(t, mcpTxnValidateAccounts(transfer, gbp1, gbp2)).Field)
	assert.Nil(t, mcpTxnValidateAccounts(transfer, gbp1, usd))
	assert.Equal(t, "destination_account_id", mcpTxnTestGetToolError(t, mcpTxnValidateAccounts(transfer, gbp1, gbp1)).Field)
	assert.Equal(t, MCPToolErrorCodeNotAllowed, mcpTxnTestGetToolError(t, mcpTxnValidateAccounts(transfer, gbp1, hidden)).Code)

	expense := &models.Transaction{Type: models.TRANSACTION_DB_TYPE_EXPENSE, AccountId: 4, Amount: 100}
	assert.Equal(t, MCPToolErrorCodeNotAllowed, mcpTxnTestGetToolError(t, mcpTxnValidateAccounts(expense, hidden, nil)).Code)
}

func TestMCPTxnGetUpdatedTransactionType(t *testing.T) {
	newType, err := mcpTxnGetUpdatedTransactionType(models.TRANSACTION_DB_TYPE_EXPENSE, transactionTypeTransfer)
	assert.Nil(t, err)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, newType)

	_, err = mcpTxnGetUpdatedTransactionType(models.TRANSACTION_DB_TYPE_EXPENSE, transactionTypeModifyBalance)
	assert.Equal(t, MCPToolErrorCodeNotSupported, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnGetUpdatedTransactionType(models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, transactionTypeIncome)
	assert.Equal(t, MCPToolErrorCodeNotSupported, mcpTxnTestGetToolError(t, err).Code)

	_, err = mcpTxnGetUpdatedTransactionType(models.TRANSACTION_DB_TYPE_EXPENSE, "invalid")
	assert.Equal(t, "type", mcpTxnTestGetToolError(t, err).Field)
}

func TestMCPTxnBuildUpdatedTransaction(t *testing.T) {
	gbp1 := &models.Account{AccountId: 1, Currency: "GBP"}
	gbp2 := &models.Account{AccountId: 2, Currency: "GBP"}
	oldTransaction := &models.Transaction{
		TransactionId:        1,
		Uid:                  1,
		Type:                 models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		CategoryId:           3,
		TransactionTime:      utils.GetMinTransactionTimeFromUnixTime(1704164645) + 5,
		TimezoneUtcOffset:    60,
		AccountId:            1,
		Amount:               100,
		RelatedAccountId:     2,
		RelatedAccountAmount: 100,
		Comment:              "old",
		GeoLatitude:          1,
		GeoLongitude:         2,
	}

	amount := "2.50"
	comment := "new"
	request := &MCPUpdateTransactionRequest{
		Amount:           &amount,
		Comment:          &comment,
		ClearGeoLocation: true,
	}

	newTransaction, err := mcpTxnBuildUpdatedTransaction(oldTransaction, request, oldTransaction.Type, 3, gbp1, gbp2)
	assert.Nil(t, err)
	assert.Equal(t, int64(250), newTransaction.Amount)
	assert.Equal(t, int64(250), newTransaction.RelatedAccountAmount)
	assert.Equal(t, oldTransaction.TransactionTime, newTransaction.TransactionTime)
	assert.Equal(t, "new", newTransaction.Comment)
	assert.Equal(t, float64(0), newTransaction.GeoLatitude)
	assert.Equal(t, []string{"amount", "destination_amount", "comment", "geo_location"}, mcpTxnGetChangedFields(oldTransaction, newTransaction, []int64{1, 2}, []int64{2, 1}))

	newTime := "2024-01-02T04:04:05+02:00"
	request = &MCPUpdateTransactionRequest{
		Time: &newTime,
	}

	newTransaction, err = mcpTxnBuildUpdatedTransaction(oldTransaction, request, oldTransaction.Type, 3, gbp1, gbp2)
	assert.Nil(t, err)
	assert.Equal(t, int16(120), newTransaction.TimezoneUtcOffset)
	assert.Equal(t, []string{"time", "utc_offset_minutes"}, mcpTxnGetChangedFields(oldTransaction, newTransaction, nil, nil))

	newTransaction, err = mcpTxnBuildUpdatedTransaction(oldTransaction, &MCPUpdateTransactionRequest{}, oldTransaction.Type, 3, gbp1, gbp2)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(mcpTxnGetChangedFields(oldTransaction, newTransaction, []int64{1}, []int64{1})))
	assert.Equal(t, []string{"tags"}, mcpTxnGetChangedFields(oldTransaction, newTransaction, []int64{1}, []int64{}))

	expense := &models.Transaction{TransactionId: 2, Type: models.TRANSACTION_DB_TYPE_EXPENSE, AccountId: 1, Amount: 100, CategoryId: 3}
	usd := &models.Account{AccountId: 3, Currency: "USD"}
	_, err = mcpTxnBuildUpdatedTransaction(expense, &MCPUpdateTransactionRequest{}, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, 4, gbp1, usd)
	assert.Equal(t, "destination_amount", mcpTxnTestGetToolError(t, err).Field)

	newTransaction, err = mcpTxnBuildUpdatedTransaction(expense, &MCPUpdateTransactionRequest{}, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, 4, gbp1, gbp2)
	assert.Nil(t, err)
	assert.Equal(t, int64(100), newTransaction.RelatedAccountAmount)
}

func TestMCPTxnValidateUpdateRequest(t *testing.T) {
	comment := strings.Repeat("a", mcpTransactionCommentMaxLength+1)
	toolErr := mcpTxnTestGetToolError(t, mcpTxnValidateUpdateRequest(&MCPUpdateTransactionRequest{Comment: &comment, DryRun: true}))
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "comment", toolErr.Field)

	assert.Equal(t, "clear_tags", mcpTxnTestGetToolError(t, mcpTxnValidateUpdateRequest(&MCPUpdateTransactionRequest{ClearTags: true, TagNames: []string{"a"}})).Field)
	assert.Equal(t, "clear_geo_location", mcpTxnTestGetToolError(t, mcpTxnValidateUpdateRequest(&MCPUpdateTransactionRequest{ClearGeoLocation: true, GeoLocation: &MCPTransactionDetailGeoLocation{}})).Field)
	assert.Nil(t, mcpTxnValidateUpdateRequest(&MCPUpdateTransactionRequest{}))
}

func TestMCPTxnInt64SetEquals(t *testing.T) {
	assert.True(t, mcpTxnInt64SetEquals(nil, []int64{}))
	assert.True(t, mcpTxnInt64SetEquals([]int64{1, 2}, []int64{2, 1}))
	assert.False(t, mcpTxnInt64SetEquals([]int64{1, 2}, []int64{1}))
	assert.False(t, mcpTxnInt64SetEquals([]int64{1, 2}, []int64{1, 3}))
}

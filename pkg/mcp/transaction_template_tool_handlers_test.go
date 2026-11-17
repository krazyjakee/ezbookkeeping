package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func transactionTemplateTestUnixTime(value string) int64 {
	t, err := time.Parse(time.RFC3339, value)

	if err != nil {
		panic(err)
	}

	return t.Unix()
}

func transactionTemplateTestInt64Pointer(value int64) *int64 {
	return &value
}

func transactionTemplateTestScheduledTemplate(frequencyType models.TransactionScheduleFrequencyType, frequency string, location *time.Location, timezoneName string, startTime *int64, endTime *int64) *models.TransactionTemplate {
	createdUnixTime := transactionTemplateTestUnixTime("2024-01-01T00:00:00Z")

	return &models.TransactionTemplate{
		TemplateId:                 1,
		TemplateType:               models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE,
		Name:                       "test",
		Type:                       models.TRANSACTION_TYPE_EXPENSE,
		ScheduledFrequencyType:     frequencyType,
		ScheduledFrequency:         frequency,
		ScheduledStartTime:         startTime,
		ScheduledEndTime:           endTime,
		ScheduledAt:                utils.GetScheduledTransactionScheduledAt(createdUnixTime, location),
		ScheduledTimezoneName:      timezoneName,
		ScheduledTimezoneUtcOffset: utils.GetTimezoneOffsetMinutes(createdUnixTime, location),
		CreatedUnixTime:            createdUnixTime,
	}
}

func transactionTemplateTestAssertToolError(t *testing.T, err error, expectedCode string, expectedField string) *MCPToolError {
	t.Helper()

	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, expectedCode, toolErr.Code)

	if expectedField != "" {
		assert.Equal(t, expectedField, toolErr.Field)
	}

	return toolErr
}

func transactionTemplateTestReferenceData() *mcpTransactionTemplateReferenceData {
	accounts := []*models.Account{
		{AccountId: 1, Name: "Bank", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP"},
		{AccountId: 2, Name: "Cash", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP"},
		{AccountId: 3, Name: "Cash", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "USD"},
		{AccountId: 4, Name: "Old", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP", Hidden: true},
		{AccountId: 5, Name: "Parent", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS, Currency: "GBP"},
		{AccountId: 6, Name: "Director Loan", Category: models.ACCOUNT_CATEGORY_DEBT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP"},
	}

	categories := []*models.TransactionCategory{
		{CategoryId: 10, Name: "Business", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		{CategoryId: 11, Name: "Software", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 10},
		{CategoryId: 12, Name: "Salary", Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 13},
		{CategoryId: 13, Name: "Income", Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		{CategoryId: 14, Name: "Archived", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 10, Hidden: true},
		{CategoryId: 15, Name: "Transfer", Type: models.CATEGORY_TYPE_TRANSFER, ParentCategoryId: 16},
		{CategoryId: 16, Name: "Transfers", Type: models.CATEGORY_TYPE_TRANSFER, ParentCategoryId: models.LevelOneTransactionCategoryParentId},
	}

	tags := []*models.TransactionTag{
		{TagId: 20, Name: "a"},
		{TagId: 21, Name: "b"},
		{TagId: 22, Name: "hidden", Hidden: true},
	}

	return mcpTransactionTemplateNewReferenceData(accounts, categories, tags)
}

func transactionTemplateTestBuild(input *mcpTransactionTemplateInput, templateType models.TransactionTemplateType) (*models.TransactionTemplate, error) {
	now := transactionTemplateTestUnixTime("2024-01-10T12:00:00Z")
	return mcpTransactionTemplateBuildModel(input, templateType, transactionTemplateTestReferenceData(), 1, now, now)
}

func TestTransactionTemplateNormalizeFrequency(t *testing.T) {
	value, err := mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "3, 1,1")
	assert.Nil(t, err)
	assert.Equal(t, "1,3", value)

	value, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "15,-1")
	assert.Nil(t, err)
	assert.Equal(t, "-1,15", value)

	value, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "1225,0229")
	assert.Nil(t, err)
	assert.Equal(t, "229,1225", value)

	value, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "3")
	assert.Nil(t, err)
	assert.Equal(t, "3", value)

	value, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "")
	assert.Nil(t, err)
	assert.Equal(t, "0", value)

	value, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED, "1,2")
	assert.Nil(t, err)
	assert.Equal(t, "", value)
}

func TestTransactionTemplateNormalizeFrequency_InvalidValues(t *testing.T) {
	_, err := mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "7")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeRequired, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "a")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "32")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "0")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "1301")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "0230")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "0")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")

	_, err = mcpTransactionTemplateNormalizeFrequency(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "2,3")
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "frequency")
}

func TestTransactionTemplateGetOccurrences_Daily(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", nil, nil)
	occurrences, completed := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00Z"), 400, 3)

	assert.False(t, completed)
	assert.Equal(t, 3, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-11T00:00:00Z"), occurrences[0].Unix())
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-12T00:00:00Z"), occurrences[1].Unix())
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-13T00:00:00Z"), occurrences[2].Unix())
}

func TestTransactionTemplateGetOccurrences_Weekly(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "1", time.UTC, "UTC", nil, nil)
	occurrences, _ := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00Z"), 400, 2)

	assert.Equal(t, 2, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-15T00:00:00Z"), occurrences[0].Unix())
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-22T00:00:00Z"), occurrences[1].Unix())
}

func TestTransactionTemplateGetOccurrences_MonthlyLastDayAndMissingDay(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-1", time.UTC, "UTC", nil, nil)
	occurrences, _ := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-02-01T12:00:00Z"), 400, 1)

	assert.Equal(t, 1, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-02-29T00:00:00Z"), occurrences[0].Unix())

	template = transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "31", time.UTC, "UTC", nil, nil)
	occurrences, _ = mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-02-01T12:00:00Z"), 400, 1)

	assert.Equal(t, 1, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-03-31T00:00:00Z"), occurrences[0].Unix())
}

func TestTransactionTemplateGetOccurrences_YearlyBeyondWindow(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "229", time.UTC, "UTC", nil, nil)
	occurrences, completed := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2025-01-01T00:00:00Z"), 400, 1)

	assert.False(t, completed)
	assert.Equal(t, 0, len(occurrences))

	occurrences, _ = mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2025-01-01T00:00:00Z"), 1500, 1)
	assert.Equal(t, 1, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2028-02-29T00:00:00Z"), occurrences[0].Unix())
}

func TestTransactionTemplateGetOccurrences_EveryNDays(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "3", time.UTC, "UTC", transactionTemplateTestInt64Pointer(transactionTemplateTestUnixTime("2024-01-01T00:00:00Z")), nil)
	occurrences, _ := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-02T12:00:00Z"), 400, 2)

	assert.Equal(t, 2, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-04T00:00:00Z"), occurrences[0].Unix())
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-07T00:00:00Z"), occurrences[1].Unix())
}

func TestTransactionTemplateGetOccurrences_StartAndEndDate(t *testing.T) {
	startTime := transactionTemplateTestInt64Pointer(transactionTemplateTestUnixTime("2024-03-01T00:00:00Z"))
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", startTime, nil)
	occurrences, _ := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00Z"), 400, 1)

	assert.Equal(t, 1, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-03-01T00:00:00Z"), occurrences[0].Unix())

	endTime := transactionTemplateTestInt64Pointer(transactionTemplateTestUnixTime("2024-01-10T23:59:59Z"))
	template = transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", nil, endTime)
	occurrences, completed := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-05T12:00:00Z"), 3700, 1000)

	assert.True(t, completed)
	assert.Equal(t, 5, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-06T00:00:00Z"), occurrences[0].Unix())
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-10T00:00:00Z"), occurrences[4].Unix())
}

func TestTransactionTemplateGetOccurrences_FixedOffsetTimezone(t *testing.T) {
	location := time.FixedZone("Test Timezone", 8*60*60)
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", location, "", nil, nil)
	assert.Equal(t, int16(960), template.ScheduledAt)

	occurrences, _ := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00+08:00"), 400, 1)

	assert.Equal(t, 1, len(occurrences))
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-11T00:00:00+08:00"), occurrences[0].Unix())
}

func TestTransactionTemplateGetOccurrences_DisabledOrInvalid(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED, "", time.UTC, "UTC", nil, nil)
	occurrences, completed := mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00Z"), 400, 1)
	assert.True(t, completed)
	assert.Equal(t, 0, len(occurrences))

	template = transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "3", time.UTC, "UTC", nil, nil)
	occurrences, _ = mcpTransactionTemplateGetOccurrences(template, transactionTemplateTestUnixTime("2024-01-10T12:00:00Z"), 400, 1)
	assert.Equal(t, 0, len(occurrences))
}

func TestTransactionTemplateBuildScheduleInfo(t *testing.T) {
	now := transactionTemplateTestUnixTime("2024-01-05T12:00:00Z")

	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED, "", time.UTC, "UTC", nil, nil)
	info := mcpTransactionTemplateBuildScheduleInfo(template, now)
	assert.Equal(t, mcpTransactionTemplateStatusDisabled, info.Status)
	assert.Equal(t, "", info.NextOccurrence)
	assert.Equal(t, mcpTransactionTemplateFrequencyDisabled, info.FrequencyType)

	endTime := transactionTemplateTestInt64Pointer(transactionTemplateTestUnixTime("2024-01-10T23:59:59Z"))
	template = transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", nil, endTime)
	info = mcpTransactionTemplateBuildScheduleInfo(template, now)
	assert.Equal(t, mcpTransactionTemplateStatusActive, info.Status)
	assert.True(t, strings.HasPrefix(info.NextOccurrence, "2024-01-06T00:00:00"))
	assert.NotNil(t, info.RemainingOccurrences)
	assert.Equal(t, 5, *info.RemainingOccurrences)
	assert.Equal(t, "2024-01-10", info.EndDate)

	info = mcpTransactionTemplateBuildScheduleInfo(template, transactionTemplateTestUnixTime("2024-02-01T00:00:00Z"))
	assert.Equal(t, mcpTransactionTemplateStatusEnded, info.Status)
	assert.Equal(t, 0, *info.RemainingOccurrences)

	startTime := transactionTemplateTestInt64Pointer(transactionTemplateTestUnixTime("2024-03-01T00:00:00Z"))
	template = transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", startTime, nil)
	info = mcpTransactionTemplateBuildScheduleInfo(template, now)
	assert.Equal(t, mcpTransactionTemplateStatusNotStarted, info.Status)
	assert.True(t, strings.HasPrefix(info.NextOccurrence, "2024-03-01T00:00:00"))
	assert.Nil(t, info.RemainingOccurrences)
	assert.NotEqual(t, "", info.Note)
	assert.Equal(t, "2024-03-01", info.StartDate)
}

func TestTransactionTemplateBuildModel_Valid(t *testing.T) {
	template, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:         "Software",
		Type:         transactionTypeExpense,
		CategoryName: "Software",
		AccountName:  "Bank",
		Amount:       "12.34",
		TagIds:       []string{"21"},
		TagNames:     []string{"a", "b"},
		Comment:      "monthly subscription",
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	assert.Nil(t, err)
	assert.Equal(t, int64(11), template.CategoryId)
	assert.Equal(t, int64(1), template.AccountId)
	assert.Equal(t, int64(1234), template.Amount)
	assert.Equal(t, "21,20", template.TagIds)
	assert.Equal(t, int64(0), template.RelatedAccountId)
	assert.Equal(t, models.TRANSACTION_TYPE_EXPENSE, template.Type)
	assert.Equal(t, models.TRANSACTION_TEMPLATE_TYPE_NORMAL, template.TemplateType)
}

func TestTransactionTemplateBuildModel_TransferDefaultDestinationAmount(t *testing.T) {
	template, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:                 "Repay director",
		Type:                 transactionTypeTransfer,
		CategoryName:         "Transfer",
		AccountId:            "1",
		DestinationAccountId: "6",
		Amount:               "120",
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	assert.Nil(t, err)
	assert.Equal(t, int64(6), template.RelatedAccountId)
	assert.Equal(t, int64(12000), template.RelatedAccountAmount)

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:         "Repay director",
		Type:         transactionTypeTransfer,
		CategoryName: "Transfer",
		AccountId:    "1",
		Amount:       "120",
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeRequired, "")
}

func TestTransactionTemplateBuildModel_FieldLengths(t *testing.T) {
	_, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:         strings.Repeat("a", mcpTemplateNameMaxLength+1),
		Type:         transactionTypeExpense,
		CategoryName: "Software",
		AccountName:  "Bank",
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	toolErr := transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeTooLong, "name")
	assert.Equal(t, int64(mcpTemplateNameMaxLength), *toolErr.Limit)

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:         "test",
		Type:         transactionTypeExpense,
		CategoryName: "Software",
		AccountName:  "Bank",
		Comment:      strings.Repeat("中", mcpTemplateCommentMaxLength+1),
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	toolErr = transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeTooLong, "comment")
	assert.Equal(t, int64(mcpTemplateCommentMaxLength), *toolErr.Limit)
	assert.Equal(t, int64(mcpTemplateCommentMaxLength+1), *toolErr.Actual)

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{
		Name:         "test",
		Type:         transactionTypeExpense,
		CategoryName: "Software",
		AccountName:  "Bank",
		Comment:      strings.Repeat("中", mcpTemplateCommentMaxLength),
	}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)

	assert.Nil(t, err)
}

func TestTransactionTemplateBuildModel_InvalidReferences(t *testing.T) {
	_, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountName: "Cash"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	toolErr := transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeAmbiguousName, "account_name")
	assert.Equal(t, 2, len(toolErr.Candidates))

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "4"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotAllowed, "account_id")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountName: "Old"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotFound, "account_name")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "5"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotAllowed, "account_id")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryId: "14", AccountId: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotAllowed, "category_id")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryId: "10", AccountId: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotAllowed, "category_id")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryId: "12", AccountId: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "category_id")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Salary", AccountId: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotFound, "category_name")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", TagIds: []string{"22"}}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotAllowed, "tag_ids")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", TagNames: []string{"missing"}}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotFound, "tag_names")

	tooManyTags := make([]string, mcpTransactionTemplateMaximumTagsCount+1)

	for i := 0; i < len(tooManyTags); i++ {
		tooManyTags[i] = "a"
	}

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", TagNames: tooManyTags}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeTooMany, "")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: "balance_modification", CategoryName: "Software", AccountId: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "type")
}

func TestTransactionTemplateBuildModel_Schedule(t *testing.T) {
	_, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyWeekly, Frequency: "1"}, models.TRANSACTION_TEMPLATE_TYPE_NORMAL)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeNotSupported, "")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyWeekly, Frequency: "1"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeRequired, "timezone")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", Frequency: "1", Timezone: "UTC"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeRequired, "frequency_type")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyWeekly, Frequency: "1", Timezone: "Invalid/Timezone"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "timezone")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyEveryNDays, Frequency: "3", Timezone: "UTC"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeRequired, "start_date")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyDaily, Timezone: "UTC", StartDate: "2024-02-01", EndDate: "2024-01-01"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "end_date")

	_, err = transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyDaily, Timezone: "UTC", StartDate: "2024/02/01"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	transactionTemplateTestAssertToolError(t, err, MCPToolErrorCodeInvalidValue, "start_date")

	template, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", FrequencyType: mcpTransactionTemplateFrequencyWeekly, Frequency: "5,1", Timezone: "UTC", StartDate: "2024-01-01", EndDate: "2024-12-31"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	assert.Nil(t, err)
	assert.Equal(t, models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, template.ScheduledFrequencyType)
	assert.Equal(t, "1,5", template.ScheduledFrequency)
	assert.Equal(t, int16(0), template.ScheduledAt)
	assert.Equal(t, int16(0), template.ScheduledTimezoneUtcOffset)
	assert.Equal(t, transactionTemplateTestUnixTime("2024-01-01T00:00:00Z"), *template.ScheduledStartTime)
	assert.Equal(t, transactionTemplateTestUnixTime("2024-12-31T23:59:59Z"), *template.ScheduledEndTime)
}

func TestTransactionTemplateIsSame(t *testing.T) {
	template := transactionTemplateTestScheduledTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", time.UTC, "UTC", transactionTemplateTestInt64Pointer(100), nil)
	newTemplate := *template
	newTemplate.ScheduledStartTime = transactionTemplateTestInt64Pointer(100)
	assert.True(t, mcpTransactionTemplateIsSame(template, &newTemplate))

	newTemplate.ScheduledEndTime = transactionTemplateTestInt64Pointer(200)
	assert.False(t, mcpTransactionTemplateIsSame(template, &newTemplate))

	newTemplate = *template
	newTemplate.Comment = "changed"
	assert.False(t, mcpTransactionTemplateIsSame(template, &newTemplate))
}

func TestTransactionTemplateBuildInputFromTemplate_RoundTrip(t *testing.T) {
	original, err := transactionTemplateTestBuild(&mcpTransactionTemplateInput{Name: "t", Type: transactionTypeExpense, CategoryName: "Software", AccountId: "1", Amount: "9.99", TagNames: []string{"a"}, FrequencyType: mcpTransactionTemplateFrequencyMonthly, Frequency: "-1", Timezone: "UTC", StartDate: "2024-01-01"}, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)
	assert.Nil(t, err)

	original.CreatedUnixTime = transactionTemplateTestUnixTime("2024-01-10T12:00:00Z")
	input := mcpTransactionTemplateBuildInputFromTemplate(original)
	rebuilt, err := transactionTemplateTestBuild(input, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE)

	assert.Nil(t, err)
	assert.True(t, mcpTransactionTemplateIsSame(original, rebuilt))
}

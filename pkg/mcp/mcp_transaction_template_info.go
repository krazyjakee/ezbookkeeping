package mcp

import (
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Textual transaction template types used in MCP tools
const (
	mcpTransactionTemplateTypeNormal    = "normal"
	mcpTransactionTemplateTypeScheduled = "scheduled"
)

// Textual scheduled transaction frequency types used in MCP tools
const (
	mcpTransactionTemplateFrequencyDisabled   = "disabled"
	mcpTransactionTemplateFrequencyDaily      = "daily"
	mcpTransactionTemplateFrequencyWeekly     = "weekly"
	mcpTransactionTemplateFrequencyMonthly    = "monthly"
	mcpTransactionTemplateFrequencyYearly     = "yearly"
	mcpTransactionTemplateFrequencyEveryNDays = "every_n_days"
)

// Scheduled transaction template statuses used in MCP tools
const (
	mcpTransactionTemplateStatusDisabled   = "disabled"
	mcpTransactionTemplateStatusNotStarted = "not_started"
	mcpTransactionTemplateStatusActive     = "active"
	mcpTransactionTemplateStatusEnded      = "ended"
)

// Limits of scheduled transaction occurrence computation
const (
	mcpTransactionTemplateNextOccurrenceMaxDays        = 400
	mcpTransactionTemplateRemainingOccurrencesMaxDays  = 3700
	mcpTransactionTemplateRemainingOccurrencesMaxCount = 1000
)

// mcpTransactionTemplateMaximumTagsCount is the maximum tags count of a transaction template (same as the HTTP API)
const mcpTransactionTemplateMaximumTagsCount = models.MaximumTagsCountOfTransaction

var mcpTransactionTemplateTimezoneMaxLength = getModelBindingMaxValue(reflect.TypeOf(models.TransactionTemplateCreateRequest{}), "ScheduledTimezoneName", 100)

var mcpTransactionTemplateFrequencyTypeNameMap = map[models.TransactionScheduleFrequencyType]string{
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED:     mcpTransactionTemplateFrequencyDisabled,
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY:        mcpTransactionTemplateFrequencyDaily,
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY:       mcpTransactionTemplateFrequencyWeekly,
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY:      mcpTransactionTemplateFrequencyMonthly,
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY:       mcpTransactionTemplateFrequencyYearly,
	models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS: mcpTransactionTemplateFrequencyEveryNDays,
}

// MCPTransactionTemplateInfo represents the transaction template information returned by MCP tools
type MCPTransactionTemplateInfo struct {
	Id                     string                              `json:"id,omitempty" jsonschema_description:"Stable transaction template ID (empty for dry run results)"`
	TemplateType           string                              `json:"template_type" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type (normal or scheduled)"`
	Name                   string                              `json:"name" jsonschema_description:"Template name"`
	Type                   string                              `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction type"`
	CategoryId             string                              `json:"category_id" jsonschema_description:"Secondary category ID"`
	CategoryName           string                              `json:"category_name,omitempty" jsonschema_description:"Secondary category name"`
	AccountId              string                              `json:"account_id" jsonschema_description:"Source account ID"`
	AccountName            string                              `json:"account_name,omitempty" jsonschema_description:"Source account name"`
	AccountCurrency        string                              `json:"account_currency,omitempty" jsonschema_description:"Currency of the source account"`
	DestinationAccountId   string                              `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID (only for transfer)"`
	DestinationAccountName string                              `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name (only for transfer)"`
	Amount                 string                              `json:"amount" jsonschema_description:"Amount (e.g. 12.34)"`
	DestinationAmount      string                              `json:"destination_amount,omitempty" jsonschema_description:"Destination amount (only for transfer)"`
	HideAmount             bool                                `json:"hide_amount" jsonschema_description:"Whether the amount is hidden in the web interface"`
	TagIds                 []string                            `json:"tag_ids" jsonschema_description:"Tag IDs"`
	TagNames               []string                            `json:"tag_names,omitempty" jsonschema_description:"Tag names"`
	Comment                string                              `json:"comment" jsonschema_description:"Transaction description"`
	Hidden                 bool                                `json:"hidden" jsonschema_description:"Whether the template is hidden (hiding does not stop scheduled transactions)"`
	DisplayOrder           int32                               `json:"display_order" jsonschema_description:"Display order"`
	Schedule               *MCPTransactionTemplateScheduleInfo `json:"schedule,omitempty" jsonschema_description:"Schedule information (only for scheduled templates)"`
}

// MCPTransactionTemplateScheduleInfo represents the schedule information of a scheduled transaction template
type MCPTransactionTemplateScheduleInfo struct {
	FrequencyType        string `json:"frequency_type" jsonschema:"enum=disabled,enum=daily,enum=weekly,enum=monthly,enum=yearly,enum=every_n_days" jsonschema_description:"Frequency type, disabled means the scheduled transaction will not be created"`
	Frequency            string `json:"frequency" jsonschema_description:"Comma-separated frequency values (weekly: weekday 0-6 with 0 as Sunday; monthly: day of month 1-31, negative value counts from the end of month, e.g. -1 is the last day; yearly: MMDD, e.g. 0131; every_n_days: N; daily: 0)"`
	StartDate            string `json:"start_date,omitempty" jsonschema_description:"Start date (YYYY-MM-DD) in the template timezone"`
	EndDate              string `json:"end_date,omitempty" jsonschema_description:"End date (YYYY-MM-DD, inclusive) in the template timezone"`
	Timezone             string `json:"timezone" jsonschema_description:"IANA timezone name of the schedule"`
	Status               string `json:"status" jsonschema:"enum=disabled,enum=not_started,enum=active,enum=ended" jsonschema_description:"Schedule status"`
	NextOccurrence       string `json:"next_occurrence,omitempty" jsonschema_description:"Transaction time of the next scheduled transaction in RFC 3339 format (computed with the server scheduling rules, omitted if there is none within the look-ahead window)"`
	RemainingOccurrences *int   `json:"remaining_occurrences,omitempty" jsonschema_description:"Number of remaining scheduled transactions until the end date (only when the end date is set and the count is computable)"`
	Note                 string `json:"note,omitempty" jsonschema_description:"Explanation when next occurrence or remaining occurrences are omitted"`
}

// mcpTransactionTemplateReferenceData contains the accounts, categories and tags of the user for resolving and validating template references
type mcpTransactionTemplateReferenceData struct {
	accounts    []*models.Account
	accountMap  map[int64]*models.Account
	categories  []*models.TransactionCategory
	categoryMap map[int64]*models.TransactionCategory
	tags        []*models.TransactionTag
	tagMap      map[int64]*models.TransactionTag
}

// mcpTransactionTemplateInput represents the textual fields of a transaction template to be created or updated
type mcpTransactionTemplateInput struct {
	Name                   string
	Type                   string
	CategoryId             string
	CategoryName           string
	AccountId              string
	AccountName            string
	DestinationAccountId   string
	DestinationAccountName string
	Amount                 string
	DestinationAmount      string
	HideAmount             bool
	TagIds                 []string
	TagNames               []string
	Comment                string
	FrequencyType          string
	Frequency              string
	StartDate              string
	EndDate                string
	Timezone               string
}

func mcpTransactionTemplateLoadReferenceData(c *core.WebContext, uid int64, services MCPAvailableServices) (*mcpTransactionTemplateReferenceData, error) {
	accounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_template_info.mcpTransactionTemplateLoadReferenceData] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	categories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_template_info.mcpTransactionTemplateLoadReferenceData] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	tags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_template_info.mcpTransactionTemplateLoadReferenceData] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	return mcpTransactionTemplateNewReferenceData(accounts, categories, tags), nil
}

func mcpTransactionTemplateNewReferenceData(accounts []*models.Account, categories []*models.TransactionCategory, tags []*models.TransactionTag) *mcpTransactionTemplateReferenceData {
	data := &mcpTransactionTemplateReferenceData{
		accounts:    accounts,
		accountMap:  make(map[int64]*models.Account, len(accounts)),
		categories:  categories,
		categoryMap: make(map[int64]*models.TransactionCategory, len(categories)),
		tags:        tags,
		tagMap:      make(map[int64]*models.TransactionTag, len(tags)),
	}

	for i := 0; i < len(accounts); i++ {
		data.accountMap[accounts[i].AccountId] = accounts[i]
	}

	for i := 0; i < len(categories); i++ {
		data.categoryMap[categories[i].CategoryId] = categories[i]
	}

	for i := 0; i < len(tags); i++ {
		data.tagMap[tags[i].TagId] = tags[i]
	}

	return data
}

func (d *mcpTransactionTemplateReferenceData) resolveAccount(idField string, id string, nameField string, name string) (*models.Account, error) {
	resolver := mcpAccountNewResolver()

	var account *models.Account
	var err error

	if strings.TrimSpace(id) != "" {
		account, err = resolver.Resolve(d.accounts, idField, id, nameField, "")
	} else {
		usableAccounts := make([]*models.Account, 0, len(d.accounts))

		for i := 0; i < len(d.accounts); i++ {
			if !d.accounts[i].Hidden && d.accounts[i].Type == models.ACCOUNT_TYPE_SINGLE_ACCOUNT {
				usableAccounts = append(usableAccounts, d.accounts[i])
			}
		}

		account, err = resolver.Resolve(usableAccounts, idField, "", nameField, name)
	}

	if err != nil {
		return nil, err
	}

	if account.Hidden {
		return nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, idField, errs.ErrCannotUseHiddenAccount.Message)
	}

	if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, idField, errs.ErrCannotAddTransactionToParentAccount.Message+", please use one of its sub-accounts")
	}

	return account, nil
}

func (d *mcpTransactionTemplateReferenceData) resolveCategory(id string, name string, categoryType models.TransactionCategoryType) (*models.TransactionCategory, error) {
	resolver := mcpTransactionCategoryNewNamedResolver("category", d.categoryMap)

	var category *models.TransactionCategory
	var err error

	if strings.TrimSpace(id) != "" {
		category, err = resolver.Resolve(d.categories, "category_id", id, "category_name", "")
	} else {
		usableCategories := make([]*models.TransactionCategory, 0, len(d.categories))

		for i := 0; i < len(d.categories); i++ {
			if !d.categories[i].Hidden && d.categories[i].ParentCategoryId != models.LevelOneTransactionCategoryParentId && d.categories[i].Type == categoryType {
				usableCategories = append(usableCategories, d.categories[i])
			}
		}

		category, err = resolver.Resolve(usableCategories, "category_id", "", "category_name", name)
	}

	if err != nil {
		return nil, err
	}

	if category.Hidden {
		return nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "category_id", errs.ErrCannotUseHiddenTransactionCategory.Message)
	}

	if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
		return nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "category_id", errs.ErrCannotUsePrimaryCategoryForTransaction.Message+", please use one of its sub-categories")
	}

	if category.Type != categoryType {
		return nil, NewMCPInvalidFieldError("category_id", utils.Int64ToString(category.CategoryId), "category \""+category.Name+"\" is a "+getMCPCategoryTypeName(category.Type)+" category, but the transaction type requires a "+getMCPCategoryTypeName(categoryType)+" category")
	}

	return category, nil
}

// resolveTagIds resolves the tag ids by ids or names, the caller (mcpTransactionTemplateBuildModel) must check the maximum tags count first
func (d *mcpTransactionTemplateReferenceData) resolveTagIds(tagIds []string, tagNames []string) ([]int64, error) {
	resolver := newMCPTransactionTagResolver(nil)

	visibleTags := make([]*models.TransactionTag, 0, len(d.tags))

	for i := 0; i < len(d.tags); i++ {
		if !d.tags[i].Hidden {
			visibleTags = append(visibleTags, d.tags[i])
		}
	}

	result := make([]int64, 0, len(tagIds)+len(tagNames))
	resultSet := make(map[int64]bool)

	appendTag := func(tag *models.TransactionTag, field string) error {
		if tag.Hidden {
			return NewMCPToolError(MCPToolErrorCodeNotAllowed, field, errs.ErrCannotUseHiddenTransactionTag.Message+" (\""+tag.Name+"\")")
		}

		if !resultSet[tag.TagId] {
			resultSet[tag.TagId] = true
			result = append(result, tag.TagId)
		}

		return nil
	}

	for i := 0; i < len(tagIds); i++ {
		tag, err := resolver.Resolve(d.tags, "tag_ids", tagIds[i], "tag_names", "")

		if err != nil {
			return nil, err
		}

		if err := appendTag(tag, "tag_ids"); err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(tagNames); i++ {
		tag, err := resolver.Resolve(visibleTags, "tag_ids", "", "tag_names", tagNames[i])

		if err != nil {
			return nil, err
		}

		if err := appendTag(tag, "tag_names"); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// mcpTransactionTemplateParseTemplateType returns the template type by the textual template type (default is normal)
func mcpTransactionTemplateParseTemplateType(field string, value string) (models.TransactionTemplateType, error) {
	switch value {
	case "", mcpTransactionTemplateTypeNormal:
		return models.TRANSACTION_TEMPLATE_TYPE_NORMAL, nil
	case mcpTransactionTemplateTypeScheduled:
		return models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE, nil
	default:
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of normal, scheduled")
	}
}

// mcpTransactionTemplateGetTemplateTypeName returns the textual template type
func mcpTransactionTemplateGetTemplateTypeName(templateType models.TransactionTemplateType) string {
	if templateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		return mcpTransactionTemplateTypeScheduled
	}

	return mcpTransactionTemplateTypeNormal
}

// mcpTransactionTemplateParseTransactionType returns the transaction type by the textual transaction type
func mcpTransactionTemplateParseTransactionType(field string, value string) (models.TransactionType, error) {
	switch value {
	case transactionTypeIncome:
		return models.TRANSACTION_TYPE_INCOME, nil
	case transactionTypeExpense:
		return models.TRANSACTION_TYPE_EXPENSE, nil
	case transactionTypeTransfer:
		return models.TRANSACTION_TYPE_TRANSFER, nil
	case "":
		return 0, NewMCPRequiredFieldError(field)
	default:
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of income, expense, transfer")
	}
}

// mcpTransactionTemplateGetTransactionTypeName returns the textual transaction type of the template
func mcpTransactionTemplateGetTransactionTypeName(transactionType models.TransactionType) string {
	switch transactionType {
	case models.TRANSACTION_TYPE_INCOME:
		return transactionTypeIncome
	case models.TRANSACTION_TYPE_EXPENSE:
		return transactionTypeExpense
	case models.TRANSACTION_TYPE_TRANSFER:
		return transactionTypeTransfer
	default:
		return ""
	}
}

// mcpTransactionTemplateParseFrequencyType returns the frequency type by the textual frequency type
func mcpTransactionTemplateParseFrequencyType(field string, value string) (models.TransactionScheduleFrequencyType, error) {
	if value == "" {
		return 0, NewMCPRequiredFieldError(field)
	}

	for frequencyType, name := range mcpTransactionTemplateFrequencyTypeNameMap {
		if name == value {
			return frequencyType, nil
		}
	}

	return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of disabled, daily, weekly, monthly, yearly, every_n_days")
}

// mcpTransactionTemplateNormalizeFrequency validates the textual frequency values of the frequency type and returns the sorted and deduplicated frequency values
func mcpTransactionTemplateNormalizeFrequency(frequencyType models.TransactionScheduleFrequencyType, frequency string) (string, error) {
	const field = "frequency"

	if frequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED {
		return "", nil
	}

	if frequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY {
		return "0", nil
	}

	if strings.TrimSpace(frequency) == "" {
		return "", NewMCPRequiredFieldError(field)
	}

	items := strings.Split(frequency, ",")
	values := make([]int, 0, len(items))
	valueSet := make(map[int]bool)

	for i := 0; i < len(items); i++ {
		item := strings.TrimSpace(items[i])
		value, err := utils.StringToInt(item)

		if err != nil {
			return "", NewMCPInvalidFieldError(field, frequency, "\""+field+"\" must be comma-separated integers")
		}

		valid := false

		switch frequencyType {
		case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY:
			valid = value >= 0 && value <= 6
		case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY:
			valid = (value >= 1 && value <= 31) || (value >= -31 && value <= -1)
		case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY:
			month := value / 100
			day := value % 100
			valid = month >= 1 && month <= 12 && day >= 1 && day <= utils.GetMaxDayOfMonth(2000, time.Month(month))
		case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS:
			valid = value > 0 && len(items) == 1
		}

		if !valid {
			switch frequencyType {
			case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY:
				return "", NewMCPInvalidFieldError(field, frequency, "weekly \""+field+"\" values must be weekdays from 0 (Sunday) to 6 (Saturday)")
			case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY:
				return "", NewMCPInvalidFieldError(field, frequency, "monthly \""+field+"\" values must be days of month from 1 to 31, or from -1 (last day of month) to -31")
			case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY:
				return "", NewMCPInvalidFieldError(field, frequency, "yearly \""+field+"\" values must be dates in MMDD format (e.g. 0131 or 1225)")
			default:
				return "", NewMCPInvalidFieldError(field, frequency, "every_n_days \""+field+"\" must be one positive integer N")
			}
		}

		if !valueSet[value] {
			valueSet[value] = true
			values = append(values, value)
		}
	}

	sort.Ints(values)

	textualValues := make([]string, len(values))

	for i := 0; i < len(values); i++ {
		textualValues[i] = utils.IntToString(values[i])
	}

	return strings.Join(textualValues, ","), nil
}

// mcpTransactionTemplateBuildModel validates the input with the same rules as the HTTP API and the service layer, and returns the template model
// (without template id, display order and hidden status). The validation is the same for dry run and actual write.
func mcpTransactionTemplateBuildModel(input *mcpTransactionTemplateInput, templateType models.TransactionTemplateType, refs *mcpTransactionTemplateReferenceData, uid int64, createdUnixTime int64, nowUnixTime int64) (*models.TransactionTemplate, error) {
	if err := checkMCPStringLength("name", input.Name, true, mcpTemplateNameMaxLength); err != nil {
		return nil, err
	}

	if err := checkMCPStringLength("comment", input.Comment, false, mcpTemplateCommentMaxLength); err != nil {
		return nil, err
	}

	transactionType, err := mcpTransactionTemplateParseTransactionType("type", input.Type)

	if err != nil {
		return nil, err
	}

	if len(input.TagIds)+len(input.TagNames) > mcpTransactionTemplateMaximumTagsCount {
		return nil, NewMCPTooManyItemsError("tag_ids/tag_names", mcpTransactionTemplateMaximumTagsCount, len(input.TagIds)+len(input.TagNames))
	}

	sourceAccount, err := refs.resolveAccount("account_id", input.AccountId, "account_name", input.AccountName)

	if err != nil {
		return nil, err
	}

	var destinationAccount *models.Account

	if transactionType == models.TRANSACTION_TYPE_TRANSFER {
		if strings.TrimSpace(input.DestinationAccountId) == "" && input.DestinationAccountName == "" {
			return nil, NewMCPRequiredFieldError("destination_account_id or destination_account_name")
		}

		destinationAccount, err = refs.resolveAccount("destination_account_id", input.DestinationAccountId, "destination_account_name", input.DestinationAccountName)

		if err != nil {
			return nil, err
		}
	}

	category, err := refs.resolveCategory(input.CategoryId, input.CategoryName, getMCPCategoryTypeByTransactionType(mcpTransactionTemplateGetTransactionTypeName(transactionType)))

	if err != nil {
		return nil, err
	}

	amount, err := parseMCPAmount("amount", input.Amount, false)

	if err != nil {
		return nil, err
	}

	destinationAmount := int64(0)

	if transactionType == models.TRANSACTION_TYPE_TRANSFER {
		if strings.TrimSpace(input.DestinationAmount) == "" {
			destinationAmount = amount
		} else {
			destinationAmount, err = parseMCPAmount("destination_amount", input.DestinationAmount, false)

			if err != nil {
				return nil, err
			}
		}
	}

	tagIds, err := refs.resolveTagIds(input.TagIds, input.TagNames)

	if err != nil {
		return nil, err
	}

	textualTagIds := make([]string, len(tagIds))

	for i := 0; i < len(tagIds); i++ {
		textualTagIds[i] = utils.Int64ToString(tagIds[i])
	}

	template := &models.TransactionTemplate{
		Uid:          uid,
		TemplateType: templateType,
		Name:         input.Name,
		Type:         transactionType,
		CategoryId:   category.CategoryId,
		AccountId:    sourceAccount.AccountId,
		TagIds:       strings.Join(textualTagIds, ","),
		Amount:       amount,
		HideAmount:   input.HideAmount,
		Comment:      input.Comment,
	}

	if destinationAccount != nil {
		template.RelatedAccountId = destinationAccount.AccountId
		template.RelatedAccountAmount = destinationAmount
	}

	if templateType != models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		if input.FrequencyType != "" || input.Frequency != "" || input.StartDate != "" || input.EndDate != "" || input.Timezone != "" {
			return nil, NewMCPNotSupportedError("frequency_type", "schedule fields (frequency_type, frequency, start_date, end_date, timezone) are only supported for scheduled templates")
		}

		return template, nil
	}

	frequencyType, err := mcpTransactionTemplateParseFrequencyType("frequency_type", input.FrequencyType)

	if err != nil {
		return nil, err
	}

	frequency, err := mcpTransactionTemplateNormalizeFrequency(frequencyType, input.Frequency)

	if err != nil {
		return nil, err
	}

	if err := checkMCPStringLength("timezone", input.Timezone, true, mcpTransactionTemplateTimezoneMaxLength); err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(input.Timezone)

	if err != nil || len(input.Timezone) < 3 {
		return nil, NewMCPInvalidFieldError("timezone", input.Timezone, "\"timezone\" must be a valid IANA timezone name (e.g. Europe/London)")
	}

	template.ScheduledFrequencyType = frequencyType
	template.ScheduledFrequency = frequency
	template.ScheduledTimezoneName = input.Timezone
	template.ScheduledTimezoneUtcOffset = utils.GetTimezoneOffsetMinutes(nowUnixTime, location)
	template.ScheduledAt = utils.GetScheduledTransactionScheduledAt(createdUnixTime, location)

	if strings.TrimSpace(input.StartDate) != "" {
		startTime, err := utils.ParseFromLongDateFirstTimeInTimezone(strings.TrimSpace(input.StartDate), location)

		if err != nil {
			return nil, NewMCPInvalidFieldError("start_date", input.StartDate, "\"start_date\" must be a valid date in YYYY-MM-DD format")
		}

		startUnixTime := startTime.Unix()
		template.ScheduledStartTime = &startUnixTime
	}

	if strings.TrimSpace(input.EndDate) != "" {
		endTime, err := utils.ParseFromLongDateLastTimeInTimezone(strings.TrimSpace(input.EndDate), location)

		if err != nil {
			return nil, NewMCPInvalidFieldError("end_date", input.EndDate, "\"end_date\" must be a valid date in YYYY-MM-DD format")
		}

		endUnixTime := endTime.Unix()
		template.ScheduledEndTime = &endUnixTime
	}

	if frequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS && template.ScheduledStartTime == nil {
		return nil, NewMCPToolError(MCPToolErrorCodeRequired, "start_date", errs.ErrScheduledTransactionStartDateRequired.Message+" for every_n_days frequency")
	}

	if template.ScheduledStartTime != nil && template.ScheduledEndTime != nil && *template.ScheduledStartTime > *template.ScheduledEndTime {
		return nil, NewMCPInvalidFieldError("end_date", input.EndDate, "\"end_date\" must not be earlier than \"start_date\"")
	}

	return template, nil
}

// mcpTransactionTemplateIsScheduleValid returns whether the schedule of the template can create transactions
func mcpTransactionTemplateIsScheduleValid(template *models.TransactionTemplate) bool {
	if template.TemplateType != models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE || template.ScheduledFrequency == "" {
		return false
	}

	switch template.ScheduledFrequencyType {
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY,
		models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY,
		models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY,
		models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY:
		return true
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS:
		return template.ScheduledStartTime != nil
	default:
		return false
	}
}

// mcpTransactionTemplateGetOccurrences returns the transaction times of the scheduled transactions which would be created
// by the server-side scheduler (see TransactionService.CreateScheduledTransactions) at or after the specified unix time.
// It scans at most maxDays days and returns at most maxCount occurrences. The second return value indicates whether the
// scan reached the end time of the schedule, i.e. the returned occurrences are all remaining occurrences.
func mcpTransactionTemplateGetOccurrences(template *models.TransactionTemplate, fromUnixTime int64, maxDays int, maxCount int) ([]time.Time, bool) {
	occurrences := make([]time.Time, 0)

	if !mcpTransactionTemplateIsScheduleValid(template) {
		return occurrences, true
	}

	frequencyValues, err := utils.StringArrayToInt64Array(strings.Split(template.ScheduledFrequency, ","))

	if err != nil || len(frequencyValues) < 1 {
		return occurrences, true
	}

	if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS && (len(frequencyValues) != 1 || frequencyValues[0] <= 0) {
		return occurrences, true
	}

	location := services.GetScheduledTransactionTemplateLocation(template)
	fromTimeInUTC := time.Unix(fromUnixTime, 0).UTC()
	firstDayUnixTimeInUTC := time.Date(fromTimeInUTC.Year(), fromTimeInUTC.Month(), fromTimeInUTC.Day(), 0, 0, 0, 0, time.UTC).Unix()

	for i := 0; i < maxDays; i++ {
		todayFirstUnixTimeInUTC := firstDayUnixTimeInUTC + int64(i)*24*60*60
		runUnixTime := todayFirstUnixTimeInUTC + int64(template.ScheduledAt)*60

		if runUnixTime < fromUnixTime {
			continue
		}

		if template.ScheduledEndTime != nil && *template.ScheduledEndTime < runUnixTime {
			return occurrences, true
		}

		if template.ScheduledStartTime != nil && *template.ScheduledStartTime > runUnixTime {
			continue
		}

		transactionTime := utils.GetScheduledTransactionFinalTime(todayFirstUnixTimeInUTC, template.ScheduledAt, template.CreatedUnixTime, location)

		if transactionTime == nil {
			continue
		}

		if !services.IsScheduledTransactionTemplateOccurrence(template, frequencyValues, *transactionTime, location) {
			continue
		}

		occurrences = append(occurrences, *transactionTime)

		if maxCount > 0 && len(occurrences) >= maxCount {
			return occurrences, false
		}
	}

	return occurrences, false
}

// mcpTransactionTemplateBuildScheduleInfo returns the schedule information of the scheduled transaction template at the specified time
func mcpTransactionTemplateBuildScheduleInfo(template *models.TransactionTemplate, nowUnixTime int64) *MCPTransactionTemplateScheduleInfo {
	location := services.GetScheduledTransactionTemplateLocation(template)
	scheduleInfo := &MCPTransactionTemplateScheduleInfo{
		FrequencyType: mcpTransactionTemplateFrequencyTypeNameMap[template.ScheduledFrequencyType],
		Frequency:     template.ScheduledFrequency,
		Timezone:      template.ScheduledTimezoneName,
	}

	if template.ScheduledStartTime != nil {
		scheduleInfo.StartDate = utils.FormatUnixTimeToLongDate(*template.ScheduledStartTime, location)
	}

	if template.ScheduledEndTime != nil {
		scheduleInfo.EndDate = utils.FormatUnixTimeToLongDate(*template.ScheduledEndTime, location)
	}

	if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED {
		scheduleInfo.Status = mcpTransactionTemplateStatusDisabled
		scheduleInfo.Note = "the schedule is disabled, no transaction will be created"
		return scheduleInfo
	}

	if !mcpTransactionTemplateIsScheduleValid(template) {
		scheduleInfo.Status = mcpTransactionTemplateStatusDisabled
		scheduleInfo.Note = "the schedule frequency is invalid, no transaction will be created"
		return scheduleInfo
	}

	if template.ScheduledEndTime != nil && *template.ScheduledEndTime < nowUnixTime {
		scheduleInfo.Status = mcpTransactionTemplateStatusEnded
		zero := 0
		scheduleInfo.RemainingOccurrences = &zero
		return scheduleInfo
	}

	if template.ScheduledStartTime != nil && *template.ScheduledStartTime > nowUnixTime {
		scheduleInfo.Status = mcpTransactionTemplateStatusNotStarted
	} else {
		scheduleInfo.Status = mcpTransactionTemplateStatusActive
	}

	nextOccurrences, scanCompleted := mcpTransactionTemplateGetOccurrences(template, nowUnixTime, mcpTransactionTemplateNextOccurrenceMaxDays, 1)

	if len(nextOccurrences) > 0 {
		scheduleInfo.NextOccurrence = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(nextOccurrences[0].Unix(), location)
	} else if scanCompleted {
		scheduleInfo.Status = mcpTransactionTemplateStatusEnded
		zero := 0
		scheduleInfo.RemainingOccurrences = &zero
		scheduleInfo.Note = "no more transaction will be created before the end date"
		return scheduleInfo
	} else {
		scheduleInfo.Note = "no transaction will be created in the next " + utils.IntToString(mcpTransactionTemplateNextOccurrenceMaxDays) + " days"
	}

	if template.ScheduledEndTime == nil {
		if scheduleInfo.Note == "" {
			scheduleInfo.Note = "remaining occurrences are unlimited because no end date is set"
		}

		return scheduleInfo
	}

	allOccurrences, scanCompleted := mcpTransactionTemplateGetOccurrences(template, nowUnixTime, mcpTransactionTemplateRemainingOccurrencesMaxDays, mcpTransactionTemplateRemainingOccurrencesMaxCount)

	if scanCompleted {
		remainingOccurrences := len(allOccurrences)
		scheduleInfo.RemainingOccurrences = &remainingOccurrences
	} else if scheduleInfo.Note == "" {
		scheduleInfo.Note = "remaining occurrences are not computed because there are more than " + utils.IntToString(mcpTransactionTemplateRemainingOccurrencesMaxCount) + " occurrences or the end date is too far away"
	}

	return scheduleInfo
}

// mcpTransactionTemplateBuildInfo returns the MCP view-object of the transaction template
func mcpTransactionTemplateBuildInfo(template *models.TransactionTemplate, refs *mcpTransactionTemplateReferenceData, nowUnixTime int64) *MCPTransactionTemplateInfo {
	info := &MCPTransactionTemplateInfo{
		TemplateType: mcpTransactionTemplateGetTemplateTypeName(template.TemplateType),
		Name:         template.Name,
		Type:         mcpTransactionTemplateGetTransactionTypeName(template.Type),
		CategoryId:   utils.Int64ToString(template.CategoryId),
		AccountId:    utils.Int64ToString(template.AccountId),
		Amount:       utils.FormatAmount(template.Amount),
		HideAmount:   template.HideAmount,
		TagIds:       make([]string, 0),
		Comment:      template.Comment,
		Hidden:       template.Hidden,
		DisplayOrder: template.DisplayOrder,
	}

	if template.TemplateId > 0 {
		info.Id = utils.Int64ToString(template.TemplateId)
	}

	if refs != nil {
		if category, exists := refs.categoryMap[template.CategoryId]; exists {
			info.CategoryName = category.Name
		}

		if account, exists := refs.accountMap[template.AccountId]; exists {
			info.AccountName = account.Name
			info.AccountCurrency = account.Currency
		}
	}

	if template.Type == models.TRANSACTION_TYPE_TRANSFER {
		info.DestinationAccountId = utils.Int64ToString(template.RelatedAccountId)
		info.DestinationAmount = utils.FormatAmount(template.RelatedAccountAmount)

		if refs != nil {
			if account, exists := refs.accountMap[template.RelatedAccountId]; exists {
				info.DestinationAccountName = account.Name
			}
		}
	}

	tagIds := template.GetTagIds()

	for i := 0; i < len(tagIds); i++ {
		info.TagIds = append(info.TagIds, utils.Int64ToString(tagIds[i]))

		if refs != nil {
			if tag, exists := refs.tagMap[tagIds[i]]; exists {
				info.TagNames = append(info.TagNames, tag.Name)
			}
		}
	}

	if template.TemplateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		info.Schedule = mcpTransactionTemplateBuildScheduleInfo(template, nowUnixTime)
	}

	return info
}

// mcpTransactionTemplateBuildInputFromTemplate returns the textual input of the existed template, which is used as the base of updating
func mcpTransactionTemplateBuildInputFromTemplate(template *models.TransactionTemplate) *mcpTransactionTemplateInput {
	input := &mcpTransactionTemplateInput{
		Name:       template.Name,
		Type:       mcpTransactionTemplateGetTransactionTypeName(template.Type),
		CategoryId: utils.Int64ToString(template.CategoryId),
		AccountId:  utils.Int64ToString(template.AccountId),
		Amount:     utils.FormatAmount(template.Amount),
		HideAmount: template.HideAmount,
		TagIds:     make([]string, 0),
		Comment:    template.Comment,
	}

	if template.Type == models.TRANSACTION_TYPE_TRANSFER && template.RelatedAccountId > 0 {
		input.DestinationAccountId = utils.Int64ToString(template.RelatedAccountId)
		input.DestinationAmount = utils.FormatAmount(template.RelatedAccountAmount)
	}

	tagIds := template.GetTagIds()

	for i := 0; i < len(tagIds); i++ {
		input.TagIds = append(input.TagIds, utils.Int64ToString(tagIds[i]))
	}

	if template.TemplateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		location := services.GetScheduledTransactionTemplateLocation(template)
		input.FrequencyType = mcpTransactionTemplateFrequencyTypeNameMap[template.ScheduledFrequencyType]
		input.Frequency = template.ScheduledFrequency
		input.Timezone = template.ScheduledTimezoneName

		if template.ScheduledStartTime != nil {
			input.StartDate = utils.FormatUnixTimeToLongDate(*template.ScheduledStartTime, location)
		}

		if template.ScheduledEndTime != nil {
			input.EndDate = utils.FormatUnixTimeToLongDate(*template.ScheduledEndTime, location)
		}
	}

	return input
}

// mcpTransactionTemplateFindTemplate finds the template by the textual id or by the name in the templates of the given template types
func mcpTransactionTemplateFindTemplate(c *core.WebContext, uid int64, services MCPAvailableServices, templateId string, templateName string, templateTypes []models.TransactionTemplateType) (*models.TransactionTemplate, error) {
	if strings.TrimSpace(templateId) != "" {
		id, err := parseMCPRequiredId("template_id", templateId)

		if err != nil {
			return nil, err
		}

		template, err := services.GetTransactionTemplateService().GetTemplateByTemplateId(c, uid, id)

		if err != nil {
			if err == errs.ErrTransactionTemplateNotFound {
				return nil, NewMCPNotFoundError("template_id", "transaction template", templateId)
			}

			log.Errorf(c, "[mcp_transaction_template_info.mcpTransactionTemplateFindTemplate] failed to get template \"id:%d\" for user \"uid:%d\", because %s", id, uid, err.Error())
			return nil, err
		}

		if templateName != "" && template.Name != templateName {
			return nil, NewMCPInvalidFieldError("template_name", templateName, "\"template_name\" does not match the name of transaction template \""+templateId+"\" (\""+template.Name+"\")")
		}

		return template, nil
	}

	if templateName == "" {
		return nil, NewMCPRequiredFieldError("template_id or template_name")
	}

	allTemplates := make([]*models.TransactionTemplate, 0)

	for i := 0; i < len(templateTypes); i++ {
		templates, err := services.GetTransactionTemplateService().GetAllTemplatesByUid(c, uid, templateTypes[i])

		if err != nil {
			log.Errorf(c, "[mcp_transaction_template_info.mcpTransactionTemplateFindTemplate] failed to get templates for user \"uid:%d\", because %s", uid, err.Error())
			return nil, err
		}

		allTemplates = append(allTemplates, templates...)
	}

	resolver := mcpTransactionTemplateNewResolver()
	return resolver.Resolve(allTemplates, "template_id", "", "template_name", templateName)
}

// mcpTransactionTemplateNewResolver returns the entity resolver of transaction templates
func mcpTransactionTemplateNewResolver() *mcpEntityResolver[*models.TransactionTemplate] {
	return newMCPEntityResolver[*models.TransactionTemplate]("transaction template",
		func(template *models.TransactionTemplate) int64 { return template.TemplateId },
		func(template *models.TransactionTemplate) string { return template.Name },
		func(template *models.TransactionTemplate) string {
			description := mcpTransactionTemplateGetTemplateTypeName(template.TemplateType) + ", " + mcpTransactionTemplateGetTransactionTypeName(template.Type)

			if template.Hidden {
				description = description + ", hidden"
			}

			return description
		})
}

// mcpTransactionTemplateIsSame returns whether the two templates have the same editable fields
func mcpTransactionTemplateIsSame(template *models.TransactionTemplate, newTemplate *models.TransactionTemplate) bool {
	if newTemplate.Name != template.Name ||
		newTemplate.Type != template.Type ||
		newTemplate.CategoryId != template.CategoryId ||
		newTemplate.AccountId != template.AccountId ||
		newTemplate.TagIds != template.TagIds ||
		newTemplate.Amount != template.Amount ||
		newTemplate.RelatedAccountId != template.RelatedAccountId ||
		newTemplate.RelatedAccountAmount != template.RelatedAccountAmount ||
		newTemplate.HideAmount != template.HideAmount ||
		newTemplate.Comment != template.Comment {
		return false
	}

	if template.TemplateType != models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		return true
	}

	return newTemplate.ScheduledFrequencyType == template.ScheduledFrequencyType &&
		newTemplate.ScheduledFrequency == template.ScheduledFrequency &&
		mcpTransactionTemplateIsSameOptionalTime(newTemplate.ScheduledStartTime, template.ScheduledStartTime) &&
		mcpTransactionTemplateIsSameOptionalTime(newTemplate.ScheduledEndTime, template.ScheduledEndTime) &&
		newTemplate.ScheduledAt == template.ScheduledAt &&
		newTemplate.ScheduledTimezoneName == template.ScheduledTimezoneName &&
		newTemplate.ScheduledTimezoneUtcOffset == template.ScheduledTimezoneUtcOffset
}

func mcpTransactionTemplateIsSameOptionalTime(a *int64, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

package mcp

import (
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPTransactionDetailTag represents a tag of the transaction
type MCPTransactionDetailTag struct {
	Id   string `json:"id" jsonschema_description:"Tag ID"`
	Name string `json:"name" jsonschema_description:"Tag name"`
}

// MCPTransactionDetailGeoLocation represents the geographic location of the transaction
type MCPTransactionDetailGeoLocation struct {
	Latitude  float64 `json:"latitude" jsonschema_description:"Latitude (-90 to 90)"`
	Longitude float64 `json:"longitude" jsonschema_description:"Longitude (-180 to 180)"`
}

// MCPTransactionDetailAttachment represents an attachment (picture or document) of the transaction
type MCPTransactionDetailAttachment struct {
	Id               string `json:"id" jsonschema_description:"Attachment ID"`
	Extension        string `json:"extension" jsonschema_description:"File extension of the attachment"`
	ContentType      string `json:"content_type,omitempty" jsonschema_description:"Content type of the attachment"`
	OriginalFileName string `json:"original_file_name,omitempty" jsonschema_description:"Original file name recorded at upload (empty for attachments uploaded before original file names were recorded)"`
	SizeBytes        int64  `json:"size_bytes,omitempty" jsonschema_description:"Size in bytes recorded at upload (empty for older attachments)"`
	SHA256           string `json:"sha256,omitempty" jsonschema_description:"Hex encoded SHA-256 checksum recorded at upload (empty for older attachments, use get_transaction_attachment to compute it from the stored file)"`
}

// MCPTransactionDetail represents the full information of a transaction
type MCPTransactionDetail struct {
	Id                     string                            `json:"id" jsonschema_description:"Transaction ID"`
	Type                   string                            `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer,enum=balance_modification" jsonschema_description:"Transaction type (income, expense, transfer, balance_modification)"`
	Time                   string                            `json:"time" jsonschema_description:"Transaction time in RFC 3339 format in the transaction timezone"`
	UtcOffsetMinutes       int16                             `json:"utc_offset_minutes" jsonschema_description:"UTC offset of the transaction timezone in minutes"`
	Amount                 string                            `json:"amount" jsonschema_description:"Amount of the transaction in the source account currency (e.g. 12.34)"`
	Currency               string                            `json:"currency,omitempty" jsonschema_description:"Currency code of the source account"`
	AccountId              string                            `json:"account_id" jsonschema_description:"Source account ID"`
	AccountName            string                            `json:"account_name,omitempty" jsonschema_description:"Source account name"`
	DestinationAmount      string                            `json:"destination_amount,omitempty" jsonschema_description:"Destination amount (only for transfer transactions)"`
	DestinationCurrency    string                            `json:"destination_currency,omitempty" jsonschema_description:"Currency code of the destination account (only for transfer transactions)"`
	DestinationAccountId   string                            `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID (only for transfer transactions)"`
	DestinationAccountName string                            `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name (only for transfer transactions)"`
	CategoryId             string                            `json:"category_id,omitempty" jsonschema_description:"Secondary category ID (empty for balance modification transactions)"`
	CategoryName           string                            `json:"category_name,omitempty" jsonschema_description:"Secondary category name"`
	PrimaryCategoryId      string                            `json:"primary_category_id,omitempty" jsonschema_description:"Primary (parent) category ID"`
	PrimaryCategoryName    string                            `json:"primary_category_name,omitempty" jsonschema_description:"Primary (parent) category name"`
	Tags                   []*MCPTransactionDetailTag        `json:"tags" jsonschema_description:"Tags of the transaction"`
	Comment                string                            `json:"comment" jsonschema_description:"Transaction description"`
	GeoLocation            *MCPTransactionDetailGeoLocation  `json:"geo_location,omitempty" jsonschema_description:"Geographic location of the transaction (omitted if not set)"`
	HideAmount             bool                              `json:"hide_amount,omitzero" jsonschema_description:"Whether the amount is hidden in the web interface"`
	ScheduledCreated       bool                              `json:"scheduled_created,omitzero" jsonschema_description:"Whether the transaction was created by a scheduled transaction"`
	AttachmentCount        int                               `json:"attachment_count" jsonschema_description:"Count of attachments of the transaction"`
	Attachments            []*MCPTransactionDetailAttachment `json:"attachments,omitempty" jsonschema_description:"Attachments of the transaction"`
	References             []*MCPTransactionReference        `json:"references,omitempty" jsonschema_description:"Structured evidence / source references of the transaction (supplier, invoice / receipt number, external source ID, source document and row)"`
	CreatedTime            string                            `json:"created_time,omitempty" jsonschema_description:"Creation time in RFC 3339 format (UTC)"`
	UpdatedTime            string                            `json:"updated_time,omitempty" jsonschema_description:"Last update time in RFC 3339 format (UTC)"`
	Editable               *bool                             `json:"editable,omitempty" jsonschema_description:"Whether the current user can still edit or delete this transaction according to the transaction edit scope setting"`
}

// mcpTxnDetailData contains all related data used to render transaction details
type mcpTxnDetailData struct {
	accounts    map[int64]*models.Account
	categories  map[int64]*models.TransactionCategory
	tags        map[int64]*models.TransactionTag
	tagIds      map[int64][]int64
	attachments map[int64][]*models.TransactionPictureInfo
	references  map[int64][]*models.TransactionReference
}

// mcpTxnLoadDetailData loads all related data of the given transactions which are used to render transaction details
func mcpTxnLoadDetailData(c *core.WebContext, uid int64, transactions []*models.Transaction, currentConfig *settings.Config, services MCPAvailableServices) (*mcpTxnDetailData, error) {
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_detail.mcpTxnLoadDetailData] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_detail.mcpTxnLoadDetailData] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	allTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_detail.mcpTxnLoadDetailData] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	data := &mcpTxnDetailData{
		accounts:    services.GetAccountService().GetAccountMapByList(allAccounts),
		categories:  services.GetTransactionCategoryService().GetCategoryMapByList(allCategories),
		tags:        services.GetTransactionTagService().GetTagMapByList(allTags),
		tagIds:      make(map[int64][]int64),
		attachments: make(map[int64][]*models.TransactionPictureInfo),
	}

	if len(transactions) < 1 {
		return data, nil
	}

	transactionIds := make([]int64, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transactionIds = append(transactionIds, transactions[i].TransactionId)
	}

	tagIds, err := services.GetTransactionTagService().GetAllTagIdsOfTransactions(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_detail.mcpTxnLoadDetailData] failed to get tag ids of transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	if tagIds != nil {
		data.tagIds = tagIds
	}

	if currentConfig != nil && currentConfig.EnableTransactionPictures {
		attachments, err := services.GetTransactionPictureService().GetPictureInfosByTransactionIds(c, uid, transactionIds)

		if err != nil {
			log.Errorf(c, "[mcp_transaction_detail.mcpTxnLoadDetailData] failed to get attachments of transactions for user \"uid:%d\", because %s", uid, err.Error())
			return nil, err
		}

		if attachments != nil {
			data.attachments = attachments
		}
	}

	data.references = mcpReferenceLoad(c, uid, transactionIds, services)

	return data, nil
}

// mcpTxnBuildDetail returns the transaction detail according to the transaction model and related data
func mcpTxnBuildDetail(transaction *models.Transaction, data *mcpTxnDetailData, user *models.User, withAttachments bool) *MCPTransactionDetail {
	detail := &MCPTransactionDetail{
		Id:               utils.Int64ToString(transaction.TransactionId),
		Type:             getMCPTransactionTypeName(transaction.Type),
		Time:             formatMCPTransactionTime(transaction.TransactionTime, transaction.TimezoneUtcOffset),
		UtcOffsetMinutes: transaction.TimezoneUtcOffset,
		Amount:           utils.FormatAmount(transaction.Amount),
		AccountId:        utils.Int64ToString(transaction.AccountId),
		Tags:             make([]*MCPTransactionDetailTag, 0),
		Comment:          transaction.Comment,
		HideAmount:       transaction.HideAmount,
		ScheduledCreated: transaction.ScheduledCreated,
	}

	if transaction.CreatedUnixTime > 0 {
		detail.CreatedTime = formatMCPUnixTime(transaction.CreatedUnixTime)
	}

	if transaction.UpdatedUnixTime > 0 {
		detail.UpdatedTime = formatMCPUnixTime(transaction.UpdatedUnixTime)
	}

	var account *models.Account
	var relatedAccount *models.Account

	if data != nil && data.accounts != nil {
		account = data.accounts[transaction.AccountId]
	}

	if account != nil {
		detail.AccountName = account.Name
		detail.Currency = account.Currency
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		detail.DestinationAmount = utils.FormatAmount(transaction.RelatedAccountAmount)
		detail.DestinationAccountId = utils.Int64ToString(transaction.RelatedAccountId)

		if data != nil && data.accounts != nil {
			relatedAccount = data.accounts[transaction.RelatedAccountId]
		}

		if relatedAccount != nil {
			detail.DestinationAccountName = relatedAccount.Name
			detail.DestinationCurrency = relatedAccount.Currency
		}
	}

	if transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && transaction.CategoryId > 0 {
		detail.CategoryId = utils.Int64ToString(transaction.CategoryId)

		if data != nil && data.categories != nil {
			if category, exists := data.categories[transaction.CategoryId]; exists && category != nil {
				detail.CategoryName = category.Name

				if category.ParentCategoryId > models.LevelOneTransactionCategoryParentId {
					detail.PrimaryCategoryId = utils.Int64ToString(category.ParentCategoryId)

					if parentCategory, exists := data.categories[category.ParentCategoryId]; exists && parentCategory != nil {
						detail.PrimaryCategoryName = parentCategory.Name
					}
				}
			}
		}
	}

	if data != nil && data.tagIds != nil {
		for _, tagId := range data.tagIds[transaction.TransactionId] {
			tag := &MCPTransactionDetailTag{
				Id: utils.Int64ToString(tagId),
			}

			if data.tags != nil {
				if tagModel, exists := data.tags[tagId]; exists && tagModel != nil {
					tag.Name = tagModel.Name
				}
			}

			detail.Tags = append(detail.Tags, tag)
		}
	}

	if transaction.GeoLongitude != 0 || transaction.GeoLatitude != 0 {
		detail.GeoLocation = &MCPTransactionDetailGeoLocation{
			Latitude:  transaction.GeoLatitude,
			Longitude: transaction.GeoLongitude,
		}
	}

	if data != nil && data.attachments != nil {
		attachments := data.attachments[transaction.TransactionId]
		detail.AttachmentCount = len(attachments)

		if withAttachments && len(attachments) > 0 {
			detail.Attachments = make([]*MCPTransactionDetailAttachment, 0, len(attachments))

			for i := 0; i < len(attachments); i++ {
				detail.Attachments = append(detail.Attachments, &MCPTransactionDetailAttachment{
					Id:               utils.Int64ToString(attachments[i].PictureId),
					Extension:        attachments[i].PictureExtension,
					ContentType:      utils.GetTransactionAttachmentContentType(attachments[i].PictureExtension),
					OriginalFileName: attachments[i].OriginalFileName,
					SizeBytes:        attachments[i].FileSize,
					SHA256:           attachments[i].FileSha256,
				})
			}
		}
	}

	if data != nil && data.references != nil {
		detail.References = mcpReferenceBuildInfos(data.references[transaction.TransactionId])
	}

	if user != nil {
		transactionTimezone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
		editable := transaction.IsEditable(user, transactionTimezone, account, relatedAccount)
		detail.Editable = &editable
	}

	return detail
}

// mcpTxnResolveAccount returns the account which is specified by id (preferred) or name.
// Accounts are matched by name only among visible accounts which can have transactions (parent accounts with sub-accounts are excluded)
func mcpTxnResolveAccount(allAccounts []*models.Account, idField string, id string, nameField string, name string) (*models.Account, error) {
	resolver := mcpAccountNewResolver()

	if strings.TrimSpace(id) != "" {
		account, err := resolver.Resolve(allAccounts, idField, id, nameField, "")

		if err != nil {
			return nil, err
		}

		if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			return nil, NewMCPInvalidFieldError(idField, id, "account \""+account.Name+"\" has sub-accounts, please specify one of its sub-accounts instead")
		}

		return account, nil
	}

	candidates := make([]*models.Account, 0, len(allAccounts))

	for i := 0; i < len(allAccounts); i++ {
		account := allAccounts[i]

		if account.Hidden || account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		candidates = append(candidates, account)
	}

	return resolver.Resolve(candidates, idField, "", nameField, name)
}

// mcpTxnResolveCategory returns the secondary category of the given category type which is specified by id (preferred) or name.
// Categories are matched by name only among visible secondary categories of the given category type
func mcpTxnResolveCategory(allCategories []*models.TransactionCategory, categoryType models.TransactionCategoryType, idField string, id string, nameField string, name string) (*models.TransactionCategory, error) {
	categoryMap := make(map[int64]*models.TransactionCategory, len(allCategories))

	for i := 0; i < len(allCategories); i++ {
		categoryMap[allCategories[i].CategoryId] = allCategories[i]
	}

	resolver := mcpTransactionCategoryNewNamedResolver("category", categoryMap)

	if strings.TrimSpace(id) != "" {
		category, err := resolver.Resolve(allCategories, idField, id, nameField, "")

		if err != nil {
			return nil, err
		}

		if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			return nil, NewMCPInvalidFieldError(idField, id, "category \""+category.Name+"\" is a primary category, please specify a secondary category instead")
		}

		if category.Type != categoryType {
			return nil, NewMCPInvalidFieldError(idField, id, "category \""+category.Name+"\" is a "+getMCPCategoryTypeName(category.Type)+" category, but a "+getMCPCategoryTypeName(categoryType)+" category is required")
		}

		return category, nil
	}

	candidates := make([]*models.TransactionCategory, 0, len(allCategories))

	for i := 0; i < len(allCategories); i++ {
		category := allCategories[i]

		if category.Hidden || category.ParentCategoryId == models.LevelOneTransactionCategoryParentId || category.Type != categoryType {
			continue
		}

		candidates = append(candidates, category)
	}

	return resolver.Resolve(candidates, idField, "", nameField, name)
}

// mcpTxnResolveTagIds returns the unique tag ids which are specified by ids or names.
// Tag ids must exist, tag names are matched only among visible tags, unknown or ambiguous tag names are returned as errors
func mcpTxnResolveTagIds(allTags []*models.TransactionTag, idField string, ids []string, nameField string, names []string) ([]int64, error) {
	resolver := newMCPTransactionTagResolver(nil)

	visibleTags := make([]*models.TransactionTag, 0, len(allTags))

	for i := 0; i < len(allTags); i++ {
		if !allTags[i].Hidden {
			visibleTags = append(visibleTags, allTags[i])
		}
	}

	tagIds := make([]int64, 0, len(ids)+len(names))

	for i := 0; i < len(ids); i++ {
		tag, err := resolver.Resolve(allTags, idField, ids[i], nameField, "")

		if err != nil {
			return nil, err
		}

		tagIds = append(tagIds, tag.TagId)
	}

	for i := 0; i < len(names); i++ {
		if strings.TrimSpace(names[i]) == "" {
			continue
		}

		tag, err := resolver.Resolve(visibleTags, idField, "", nameField, names[i])

		if err != nil {
			return nil, err
		}

		tagIds = append(tagIds, tag.TagId)
	}

	tagIds = utils.ToUniqueInt64Slice(tagIds)

	if len(tagIds) > models.MaximumTagsCountOfTransaction {
		return nil, NewMCPTooManyItemsError(nameField, models.MaximumTagsCountOfTransaction, len(tagIds))
	}

	return tagIds, nil
}

// mcpTxnParseGeoLocation checks the geographic location and returns the latitude and longitude
func mcpTxnParseGeoLocation(field string, geoLocation *MCPTransactionDetailGeoLocation) (float64, float64, error) {
	if geoLocation == nil {
		return 0, 0, nil
	}

	if geoLocation.Latitude < -90 || geoLocation.Latitude > 90 {
		return 0, 0, NewMCPToolError(MCPToolErrorCodeOutOfRange, field+".latitude", "\""+field+".latitude\" must be between -90 and 90")
	}

	if geoLocation.Longitude < -180 || geoLocation.Longitude > 180 {
		return 0, 0, NewMCPToolError(MCPToolErrorCodeOutOfRange, field+".longitude", "\""+field+".longitude\" must be between -180 and 180")
	}

	return geoLocation.Latitude, geoLocation.Longitude, nil
}

// mcpTxnParseAmountFilter converts the amount filter with decimal amounts (e.g. "gt:12.34", "bt:10:20") into the amount filter syntax used by the transaction service (amounts in cents)
func mcpTxnParseAmountFilter(field string, value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", nil
	}

	invalidErr := NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of gt:<amount>, lt:<amount>, eq:<amount>, ne:<amount>, bt:<min>:<max> (between) or nb:<min>:<max> (not between), e.g. gt:12.34")
	items := strings.Split(value, ":")

	if len(items) < 2 {
		return "", invalidErr
	}

	operator := items[0]

	if operator == "gt" || operator == "lt" || operator == "eq" || operator == "ne" {
		if len(items) != 2 {
			return "", invalidErr
		}

		amount, err := utils.ParseAmount(strings.TrimSpace(items[1]))

		if err != nil {
			return "", invalidErr
		}

		return operator + ":" + utils.Int64ToString(amount), nil
	} else if operator == "bt" || operator == "nb" {
		if len(items) != 3 {
			return "", invalidErr
		}

		amount1, err := utils.ParseAmount(strings.TrimSpace(items[1]))

		if err != nil {
			return "", invalidErr
		}

		amount2, err := utils.ParseAmount(strings.TrimSpace(items[2]))

		if err != nil || amount2 < amount1 {
			return "", invalidErr
		}

		return operator + ":" + utils.Int64ToString(amount1) + ":" + utils.Int64ToString(amount2), nil
	}

	return "", invalidErr
}

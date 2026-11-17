package mcp

import (
	"reflect"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPUpdateTransactionRequest represents all parameters of the update transaction request, only the specified fields are changed
type MCPUpdateTransactionRequest struct {
	TransactionId          string                           `json:"transaction_id" jsonschema_description:"ID of the transaction to update (for transfers, use the ID returned by query_transactions or get_transaction)"`
	Type                   *string                          `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"New transaction type (optional, income, expense or transfer; balance modification transactions cannot change type; a new category is required when the type changes)"`
	Time                   *string                          `json:"time,omitempty" jsonschema:"format=date-time" jsonschema_description:"New transaction time in RFC 3339 format (optional), the UTC offset of the time is saved as the transaction timezone"`
	Amount                 *string                          `json:"amount,omitempty" jsonschema_description:"New amount (optional, e.g. 12.34)"`
	AccountId              *string                          `json:"account_id,omitempty" jsonschema_description:"New source account ID (optional, takes precedence over account_name)"`
	AccountName            *string                          `json:"account_name,omitempty" jsonschema_description:"New source account name (optional)"`
	DestinationAccountId   *string                          `json:"destination_account_id,omitempty" jsonschema_description:"New destination account ID for transfer transactions (optional, takes precedence over destination_account_name)"`
	DestinationAccountName *string                          `json:"destination_account_name,omitempty" jsonschema_description:"New destination account name for transfer transactions (optional)"`
	DestinationAmount      *string                          `json:"destination_amount,omitempty" jsonschema_description:"New destination amount for transfer transactions (optional, defaults to amount when both accounts use the same currency)"`
	CategoryId             *string                          `json:"category_id,omitempty" jsonschema_description:"New secondary category ID (optional, takes precedence over category_name)"`
	CategoryName           *string                          `json:"category_name,omitempty" jsonschema_description:"New secondary category name (optional)"`
	TagIds                 []string                         `json:"tag_ids,omitempty" jsonschema_description:"New tag IDs (optional, when tag_ids or tags is set, the tags of the transaction are replaced with the union of both lists)"`
	TagNames               []string                         `json:"tags,omitempty" jsonschema_description:"New tag names (optional, when tag_ids or tags is set, the tags of the transaction are replaced with the union of both lists)"`
	ClearTags              bool                             `json:"clear_tags,omitempty" jsonschema_description:"If true, remove all tags from the transaction (optional, cannot be used with tag_ids or tags)"`
	Comment                *string                          `json:"comment,omitempty" jsonschema_description:"New transaction description (optional, set to empty string to clear, maximum length is reported by get_server_capabilities)"`
	GeoLocation            *MCPTransactionDetailGeoLocation `json:"geo_location,omitempty" jsonschema_description:"New geographic location (optional)"`
	ClearGeoLocation       bool                             `json:"clear_geo_location,omitempty" jsonschema_description:"If true, remove the geographic location from the transaction (optional, cannot be used with geo_location)"`
	AddAttachmentIds       []string                         `json:"add_attachment_ids,omitempty" jsonschema_description:"IDs of uploaded but unattached attachments (returned by upload_transaction_attachment) to attach to the transaction (optional, existing attachments are kept; use detach_transaction_attachment or delete_transaction_attachment to remove attachments)"`
	DryRun                 bool                             `json:"dry_run,omitempty" jsonschema_description:"If true, the transaction will not be saved, only validated with the same rules as a real write (optional)"`
}

// MCPTransactionDetailAccountBalance represents the balance of an account affected by a transaction change
type MCPTransactionDetailAccountBalance struct {
	AccountId     string `json:"account_id" jsonschema_description:"Account ID"`
	AccountName   string `json:"account_name" jsonschema_description:"Account name"`
	Currency      string `json:"currency" jsonschema_description:"Currency code of the account"`
	Balance       string `json:"balance" jsonschema_description:"Displayed balance (outstanding balance for liability accounts, positive means money owed)"`
	SignedBalance string `json:"signed_balance" jsonschema_description:"Signed accounting balance (negative means money owed for liability accounts)"`
}

// MCPUpdateTransactionResponse represents the response structure for update transaction
type MCPUpdateTransactionResponse struct {
	Success         bool                                  `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun          bool                                  `json:"dry_run,omitzero" jsonschema_description:"Indicates whether this operation is a dry run (transaction not saved actually)"`
	Committed       bool                                  `json:"committed" jsonschema_description:"Indicates whether the change has been saved to the database"`
	Status          string                                `json:"status" jsonschema:"enum=updated,enum=would_be_updated,enum=unchanged" jsonschema_description:"Result status (updated, would_be_updated for dry run, unchanged if nothing would change)"`
	ChangedFields   []string                              `json:"changed_fields" jsonschema_description:"Fields which are (or would be) changed"`
	Transaction     *MCPTransactionDetail                 `json:"transaction,omitempty" jsonschema_description:"The persisted transaction after update (when committed or unchanged) or the validated transaction which would be saved (when dry run)"`
	AccountBalances []*MCPTransactionDetailAccountBalance `json:"account_balances,omitempty" jsonschema_description:"Balances of the affected accounts after the update (only when committed)"`
}

type mcpUpdateTransactionToolHandler struct{}

var MCPUpdateTransactionToolHandler = &mcpUpdateTransactionToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateTransactionToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_TRANSACTION
}

// Name returns the name of the MCP tool
func (h *mcpUpdateTransactionToolHandler) Name() string {
	return "update_transaction"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateTransactionToolHandler) Description() string {
	return "Update an existing transaction in ezBookkeeping. Only the specified fields are changed (amount, accounts, category, time and timezone, description, tags, location). The user's transaction edit scope is applied to both the original and the new transaction time. Uploaded attachments can be added via add_attachment_ids, existing attachments are never removed by this tool. Use dry_run to validate without saving."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateTransactionToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateTransactionToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateTransactionToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateTransactionToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateTransactionToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateTransactionToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUpdateTransactionRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	transactionId, err := parseMCPRequiredId("transaction_id", request.TransactionId)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpTxnValidateUpdateRequest(request); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, transactionId)

	if err != nil {
		log.Warnf(c, "[update_transaction_tool_handler.Handle] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())

		if err == errs.ErrTransactionNotFound {
			return nil, nil, NewMCPNotFoundError("transaction_id", "transaction", request.TransactionId)
		}

		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "transaction_id", "transaction \""+request.TransactionId+"\" is the incoming side of a transfer, please update the transfer by its ID \""+utils.Int64ToString(transaction.RelatedId)+"\" instead")
	}

	newTransactionType := transaction.Type

	if request.Type != nil {
		newTransactionType, err = mcpTxnGetUpdatedTransactionType(transaction.Type, *request.Type)

		if err != nil {
			return nil, nil, err
		}
	}

	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Warnf(c, "[update_transaction_tool_handler.Handle] get account error, because %s", err.Error())
		return nil, nil, err
	}

	accountMap := services.GetAccountService().GetAccountMapByList(allAccounts)
	sourceAccount := accountMap[transaction.AccountId]

	if request.AccountId != nil || request.AccountName != nil {
		sourceAccount, err = mcpTxnResolveAccount(allAccounts, "account_id", mcpTxnStringValue(request.AccountId), "account_name", mcpTxnStringValue(request.AccountName))

		if err != nil {
			return nil, nil, err
		}
	}

	if sourceAccount == nil {
		log.Warnf(c, "[update_transaction_tool_handler.Handle] account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transactionId, uid)
		return nil, nil, errs.ErrSourceAccountNotFound
	}

	var destinationAccount *models.Account

	if newTransactionType == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if request.DestinationAccountId != nil || request.DestinationAccountName != nil {
			destinationAccount, err = mcpTxnResolveAccount(allAccounts, "destination_account_id", mcpTxnStringValue(request.DestinationAccountId), "destination_account_name", mcpTxnStringValue(request.DestinationAccountName))

			if err != nil {
				return nil, nil, err
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			destinationAccount = accountMap[transaction.RelatedAccountId]

			if destinationAccount == nil {
				log.Warnf(c, "[update_transaction_tool_handler.Handle] related account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transactionId, uid)
				return nil, nil, errs.ErrDestinationAccountNotFound
			}
		} else {
			return nil, nil, NewMCPRequiredFieldError("destination_account_id or destination_account_name")
		}
	} else if request.DestinationAccountId != nil || request.DestinationAccountName != nil || request.DestinationAmount != nil {
		return nil, nil, NewMCPInvalidFieldError("destination_account_id", "", "destination account and destination amount can only be set for transfer transactions")
	}

	categoryId := int64(0)

	if newTransactionType == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		if request.CategoryId != nil || request.CategoryName != nil {
			return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "category_id", "balance modification transactions cannot have a category")
		}
	} else if request.CategoryId != nil || request.CategoryName != nil {
		allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

		if err != nil {
			log.Warnf(c, "[update_transaction_tool_handler.Handle] get transaction category error, because %s", err.Error())
			return nil, nil, err
		}

		category, err := mcpTxnResolveCategory(allCategories, getMCPCategoryTypeByTransactionType(getMCPTransactionTypeName(newTransactionType)), "category_id", mcpTxnStringValue(request.CategoryId), "category_name", mcpTxnStringValue(request.CategoryName))

		if err != nil {
			return nil, nil, err
		}

		categoryId = category.CategoryId
	} else if newTransactionType == transaction.Type {
		categoryId = transaction.CategoryId
	} else {
		return nil, nil, NewMCPRequiredFieldError("category_id or category_name")
	}

	allTransactionTagIds, err := services.GetTransactionTagService().GetAllTagIdsOfTransactions(c, uid, []int64{transaction.TransactionId})

	if err != nil {
		log.Errorf(c, "[update_transaction_tool_handler.Handle] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionTagIds := allTransactionTagIds[transaction.TransactionId]

	if transactionTagIds == nil {
		transactionTagIds = make([]int64, 0)
	}

	newTagIds := transactionTagIds
	var allTags []*models.TransactionTag

	if request.ClearTags {
		newTagIds = make([]int64, 0)
	} else if request.TagIds != nil || request.TagNames != nil {
		allTags, err = services.GetTransactionTagService().GetAllTagsByUid(c, uid)

		if err != nil {
			log.Warnf(c, "[update_transaction_tool_handler.Handle] get transaction tag error, because %s", err.Error())
			return nil, nil, err
		}

		newTagIds, err = mcpTxnResolveTagIds(allTags, "tag_ids", request.TagIds, "tags", request.TagNames)

		if err != nil {
			return nil, nil, err
		}
	}

	if mcpTxnInt64SetEquals(newTagIds, transactionTagIds) {
		newTagIds = transactionTagIds
	}

	var addPictureIds []int64

	if len(request.AddAttachmentIds) > 0 {
		if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
			return nil, nil, err
		}

		existingPictureInfos, err := services.GetTransactionPictureService().GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[update_transaction_tool_handler.Handle] failed to get attachments of transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		addPictureIds, err = mcpAttachmentCheckNewAttachments(c, uid, "add_attachment_ids", request.AddAttachmentIds, len(existingPictureInfos), currentConfig, services)

		if err != nil {
			return nil, nil, err
		}
	}

	newTransaction, err := mcpTxnBuildUpdatedTransaction(transaction, request, newTransactionType, categoryId, sourceAccount, destinationAccount)

	if err != nil {
		return nil, nil, err
	}

	changedFields := mcpTxnGetChangedFields(transaction, newTransaction, transactionTagIds, newTagIds)

	if len(addPictureIds) > 0 {
		changedFields = append(changedFields, "attachments")
	}

	if len(changedFields) == 0 {
		return h.createResponse(c, uid, user, transaction, currentConfig, services, MCPWriteStatusUnchanged, changedFields, false, nil)
	}

	if err := mcpTxnValidateAccounts(newTransaction, sourceAccount, destinationAccount); err != nil {
		return nil, nil, err
	}

	transactionTimezone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
	newTransactionTimezone := time.FixedZone("Transaction Timezone", int(newTransaction.TimezoneUtcOffset)*60)
	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, transactionTimezone, accountMap[transaction.AccountId], accountMap[transaction.RelatedAccountId])
	newTransactionEditable := user.CanEditTransactionByTransactionTime(newTransaction.TransactionTime, newTransactionTimezone, sourceAccount, destinationAccount)

	if !transactionEditable || !newTransactionEditable {
		return nil, nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
	}

	if request.DryRun {
		if allTags == nil && len(newTagIds) > 0 {
			allTags, err = services.GetTransactionTagService().GetAllTagsByUid(c, uid)

			if err != nil {
				log.Warnf(c, "[update_transaction_tool_handler.Handle] get transaction tag error, because %s", err.Error())
				return nil, nil, err
			}
		}

		allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

		if err != nil {
			log.Warnf(c, "[update_transaction_tool_handler.Handle] get transaction category error, because %s", err.Error())
			return nil, nil, err
		}

		detailData := &mcpTxnDetailData{
			accounts:   accountMap,
			categories: services.GetTransactionCategoryService().GetCategoryMapByList(allCategories),
			tags:       services.GetTransactionTagService().GetTagMapByList(allTags),
			tagIds: map[int64][]int64{
				newTransaction.TransactionId: newTagIds,
			},
		}

		response := MCPUpdateTransactionResponse{
			Success:       true,
			DryRun:        true,
			Committed:     false,
			Status:        MCPWriteStatusWouldBeUpdated,
			ChangedFields: changedFields,
			Transaction:   mcpTxnBuildDetail(newTransaction, detailData, nil, false),
		}

		return newMCPJSONTextResult(response)
	}

	var addTransactionTagIds []int64
	var removeTransactionTagIds []int64

	if !utils.Int64SliceEquals(newTagIds, transactionTagIds) {
		removeTransactionTagIds = transactionTagIds
		addTransactionTagIds = newTagIds
	}

	changeToTransfer := newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	err = services.GetTransactionService().ModifyTransaction(c, newTransaction, changeToTransfer, len(transactionTagIds), addTransactionTagIds, removeTransactionTagIds, addPictureIds, nil)

	if err != nil {
		if err == errs.ErrNothingWillBeUpdated {
			return h.createResponse(c, uid, user, transaction, currentConfig, services, MCPWriteStatusUnchanged, make([]string, 0), false, nil)
		}

		log.Errorf(c, "[update_transaction_tool_handler.Handle] failed to update transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[update_transaction_tool_handler.Handle] user \"uid:%d\" has updated transaction \"id:%d\" successfully", uid, transactionId)

	affectedAccountIds := utils.ToUniqueInt64Slice([]int64{transaction.AccountId, transaction.RelatedAccountId, newTransaction.AccountId, newTransaction.RelatedAccountId})
	savedTransaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, transactionId)

	if err != nil {
		log.Warnf(c, "[update_transaction_tool_handler.Handle] failed to get transaction \"id:%d\" after updated, because %s", transactionId, err.Error())
		savedTransaction = newTransaction
	}

	return h.createResponse(c, uid, user, savedTransaction, currentConfig, services, MCPWriteStatusUpdated, changedFields, true, affectedAccountIds)
}

func (h *mcpUpdateTransactionToolHandler) createResponse(c *core.WebContext, uid int64, user *models.User, transaction *models.Transaction, currentConfig *settings.Config, services MCPAvailableServices, status string, changedFields []string, committed bool, affectedAccountIds []int64) (any, []*MCPTextContent, error) {
	response := MCPUpdateTransactionResponse{
		Success:       true,
		Committed:     committed,
		Status:        status,
		ChangedFields: changedFields,
	}

	detailData, err := mcpTxnLoadDetailData(c, uid, []*models.Transaction{transaction}, currentConfig, services)

	if err != nil {
		log.Warnf(c, "[update_transaction_tool_handler.createResponse] failed to get related data of transaction \"id:%d\", because %s", transaction.TransactionId, err.Error())
	} else {
		response.Transaction = mcpTxnBuildDetail(transaction, detailData, user, true)

		for i := 0; i < len(affectedAccountIds); i++ {
			account, exists := detailData.accounts[affectedAccountIds[i]]

			if !exists || account == nil {
				continue
			}

			balance, signedBalance := mcpTxnGetAccountBalances(account)
			response.AccountBalances = append(response.AccountBalances, &MCPTransactionDetailAccountBalance{
				AccountId:     utils.Int64ToString(account.AccountId),
				AccountName:   account.Name,
				Currency:      account.Currency,
				Balance:       balance,
				SignedBalance: signedBalance,
			})
		}
	}

	return newMCPJSONTextResult(response)
}

// mcpTxnValidateUpdateRequest validates all fields of the update request which do not depend on stored data, so that dry run and real write use the same rules
func mcpTxnValidateUpdateRequest(request *MCPUpdateTransactionRequest) error {
	if request.Comment != nil {
		if err := checkMCPStringLength("comment", *request.Comment, false, mcpTransactionCommentMaxLength); err != nil {
			return err
		}
	}

	if request.ClearTags && (request.TagIds != nil || request.TagNames != nil) {
		return NewMCPInvalidFieldError("clear_tags", "true", "\"clear_tags\" cannot be used together with \"tag_ids\" or \"tags\"")
	}

	if count := len(request.TagIds) + len(request.TagNames); count > models.MaximumTagsCountOfTransaction {
		return NewMCPTooManyItemsError("tags", models.MaximumTagsCountOfTransaction, count)
	}

	if count := len(request.AddAttachmentIds); count > models.MaximumPicturesCountOfTransaction {
		return NewMCPTooManyItemsError("add_attachment_ids", models.MaximumPicturesCountOfTransaction, count)
	}

	if request.ClearGeoLocation && request.GeoLocation != nil {
		return NewMCPInvalidFieldError("clear_geo_location", "true", "\"clear_geo_location\" cannot be used together with \"geo_location\"")
	}

	if _, _, err := mcpTxnParseGeoLocation("geo_location", request.GeoLocation); err != nil {
		return err
	}

	if request.Time != nil {
		if _, err := parseMCPTime("time", *request.Time, true); err != nil {
			return err
		}
	}

	if request.Amount != nil {
		if _, err := parseMCPAmount("amount", *request.Amount, true); err != nil {
			return err
		}
	}

	if request.DestinationAmount != nil {
		if _, err := parseMCPAmount("destination_amount", *request.DestinationAmount, true); err != nil {
			return err
		}
	}

	return nil
}

// mcpTxnGetUpdatedTransactionType returns the new transaction type, changing between balance modification and other types is not allowed (same as the web interface)
func mcpTxnGetUpdatedTransactionType(oldType models.TransactionDbType, newType string) (models.TransactionDbType, error) {
	var newTransactionType models.TransactionDbType

	switch newType {
	case transactionTypeIncome:
		newTransactionType = models.TRANSACTION_DB_TYPE_INCOME
	case transactionTypeExpense:
		newTransactionType = models.TRANSACTION_DB_TYPE_EXPENSE
	case transactionTypeTransfer:
		newTransactionType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	case transactionTypeModifyBalance:
		newTransactionType = models.TRANSACTION_DB_TYPE_MODIFY_BALANCE
	default:
		return 0, NewMCPInvalidFieldError("type", newType, "\"type\" must be one of income, expense, transfer")
	}

	if (oldType == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE) != (newTransactionType == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE) {
		return 0, NewMCPNotSupportedError("type", "the type of a balance modification transaction cannot be changed, and other transactions cannot be changed to balance modification")
	}

	return newTransactionType, nil
}

// mcpTxnBuildUpdatedTransaction returns the new transaction model (same as the web interface builds) by applying the specified fields to the old transaction
func mcpTxnBuildUpdatedTransaction(transaction *models.Transaction, request *MCPUpdateTransactionRequest, newTransactionType models.TransactionDbType, categoryId int64, sourceAccount *models.Account, destinationAccount *models.Account) (*models.Transaction, error) {
	newTransaction := &models.Transaction{
		TransactionId:     transaction.TransactionId,
		Uid:               transaction.Uid,
		Type:              newTransactionType,
		CategoryId:        categoryId,
		TransactionTime:   transaction.TransactionTime,
		TimezoneUtcOffset: transaction.TimezoneUtcOffset,
		AccountId:         sourceAccount.AccountId,
		Amount:            transaction.Amount,
		HideAmount:        transaction.HideAmount,
		Comment:           transaction.Comment,
		GeoLongitude:      transaction.GeoLongitude,
		GeoLatitude:       transaction.GeoLatitude,
	}

	if request.Time != nil {
		transactionTime, err := parseMCPTime("time", *request.Time, true)

		if err != nil {
			return nil, err
		}

		if transactionTime.Unix() != utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) {
			newTransaction.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(transactionTime.Unix())
		}

		newTransaction.TimezoneUtcOffset = utils.GetTimezoneOffsetMinutes(transactionTime.Unix(), transactionTime.Location())
	}

	amountChanged := false

	if request.Amount != nil {
		amount, err := parseMCPAmount("amount", *request.Amount, true)

		if err != nil {
			return nil, err
		}

		amountChanged = amount != transaction.Amount
		newTransaction.Amount = amount
	}

	if newTransactionType == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if destinationAccount == nil {
			return nil, NewMCPRequiredFieldError("destination_account_id or destination_account_name")
		}

		newTransaction.RelatedAccountId = destinationAccount.AccountId

		if request.DestinationAmount != nil {
			destinationAmount, err := parseMCPAmount("destination_amount", *request.DestinationAmount, true)

			if err != nil {
				return nil, err
			}

			newTransaction.RelatedAccountAmount = destinationAmount
		} else if sourceAccount.Currency == destinationAccount.Currency && (amountChanged || transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT) {
			newTransaction.RelatedAccountAmount = newTransaction.Amount
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			newTransaction.RelatedAccountAmount = transaction.RelatedAccountAmount
		} else {
			return nil, NewMCPRequiredFieldError("destination_amount")
		}
	}

	if request.Comment != nil {
		newTransaction.Comment = *request.Comment
	}

	if request.ClearGeoLocation {
		newTransaction.GeoLatitude = 0
		newTransaction.GeoLongitude = 0
	} else if request.GeoLocation != nil {
		latitude, longitude, err := mcpTxnParseGeoLocation("geo_location", request.GeoLocation)

		if err != nil {
			return nil, err
		}

		newTransaction.GeoLatitude = latitude
		newTransaction.GeoLongitude = longitude
	}

	return newTransaction, nil
}

// mcpTxnGetChangedFields returns the fields which are different between the old and new transaction (same comparison as the web interface uses)
func mcpTxnGetChangedFields(oldTransaction *models.Transaction, newTransaction *models.Transaction, oldTagIds []int64, newTagIds []int64) []string {
	changedFields := make([]string, 0)

	if newTransaction.Type != oldTransaction.Type {
		changedFields = append(changedFields, "type")
	}

	if newTransaction.CategoryId != oldTransaction.CategoryId {
		changedFields = append(changedFields, "category_id")
	}

	if utils.GetUnixTimeFromTransactionTime(newTransaction.TransactionTime) != utils.GetUnixTimeFromTransactionTime(oldTransaction.TransactionTime) {
		changedFields = append(changedFields, "time")
	}

	if newTransaction.TimezoneUtcOffset != oldTransaction.TimezoneUtcOffset {
		changedFields = append(changedFields, "utc_offset_minutes")
	}

	if newTransaction.AccountId != oldTransaction.AccountId {
		changedFields = append(changedFields, "account_id")
	}

	if newTransaction.Amount != oldTransaction.Amount {
		changedFields = append(changedFields, "amount")
	}

	if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && newTransaction.RelatedAccountId != oldTransaction.RelatedAccountId {
		changedFields = append(changedFields, "destination_account_id")
	}

	if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && newTransaction.RelatedAccountAmount != oldTransaction.RelatedAccountAmount {
		changedFields = append(changedFields, "destination_amount")
	}

	if newTransaction.HideAmount != oldTransaction.HideAmount {
		changedFields = append(changedFields, "hide_amount")
	}

	if newTransaction.Comment != oldTransaction.Comment {
		changedFields = append(changedFields, "comment")
	}

	if newTransaction.GeoLongitude != oldTransaction.GeoLongitude || newTransaction.GeoLatitude != oldTransaction.GeoLatitude {
		changedFields = append(changedFields, "geo_location")
	}

	if !mcpTxnInt64SetEquals(oldTagIds, newTagIds) {
		changedFields = append(changedFields, "tags")
	}

	return changedFields
}

// mcpTxnInt64SetEquals returns whether the two slices contain the same items regardless of order and duplicates
func mcpTxnInt64SetEquals(s1 []int64, s2 []int64) bool {
	set1 := make(map[int64]bool, len(s1))
	set2 := make(map[int64]bool, len(s2))

	for i := 0; i < len(s1); i++ {
		set1[s1[i]] = true
	}

	for i := 0; i < len(s2); i++ {
		set2[s2[i]] = true
	}

	if len(set1) != len(set2) {
		return false
	}

	for item := range set1 {
		if !set2[item] {
			return false
		}
	}

	return true
}

// mcpTxnStringValue returns the value of the string pointer, or empty string if nil
func mcpTxnStringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

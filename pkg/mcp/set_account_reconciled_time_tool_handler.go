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

// MCPSetAccountReconciledTimeRequest represents all parameters of the set account reconciled time request
type MCPSetAccountReconciledTimeRequest struct {
	AccountId      string `json:"account_id,omitempty" jsonschema_description:"Account ID (preferred, either account_id or account_name is required, parent accounts with sub-accounts are not allowed)"`
	AccountName    string `json:"account_name,omitempty" jsonschema_description:"Account name (used if account_id is not set, returns an ambiguous_name error with candidates if more than one account has this name)"`
	ReconciledTime string `json:"reconciled_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Time up to which the account has been reconciled in RFC 3339 format (optional, default: now). It cannot be earlier than the current last reconciled time of the account"`
	DryRun         bool   `json:"dry_run,omitempty" jsonschema_description:"If true, only validate and report the change without saving (optional)"`
}

// MCPSetAccountReconciledTimeResponse represents the response structure for set account reconciled time
type MCPSetAccountReconciledTimeResponse struct {
	Success                    bool   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun                     bool   `json:"dry_run,omitzero" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Committed                  bool   `json:"committed" jsonschema_description:"Indicates whether the last reconciled time has been saved"`
	Status                     string `json:"status" jsonschema:"enum=updated,enum=would_be_updated,enum=unchanged" jsonschema_description:"Result status"`
	AccountId                  string `json:"account_id" jsonschema_description:"Account ID"`
	AccountName                string `json:"account_name" jsonschema_description:"Account name"`
	PreviousLastReconciledTime string `json:"previous_last_reconciled_time,omitempty" jsonschema_description:"Last reconciled time before this operation in RFC 3339 format (UTC), empty if never reconciled"`
	LastReconciledTime         string `json:"last_reconciled_time" jsonschema_description:"Last reconciled time after this operation in RFC 3339 format (UTC)"`
	EditScopeNote              string `json:"edit_scope_note,omitempty" jsonschema_description:"How the last reconciled time affects editing transactions of this account"`
}

type mcpSetAccountReconciledTimeToolHandler struct{}

var MCPSetAccountReconciledTimeToolHandler = &mcpSetAccountReconciledTimeToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpSetAccountReconciledTimeToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_SET_ACCOUNT_RECONCILED_TIME
}

// Name returns the name of the MCP tool
func (h *mcpSetAccountReconciledTimeToolHandler) Name() string {
	return "set_account_reconciled_time"
}

// Description returns the description of the MCP tool
func (h *mcpSetAccountReconciledTimeToolHandler) Description() string {
	return "Mark an account as reconciled up to a point in time (the same as \"Mark as reconciled\" in the web interface), by setting the last reconciled time of the account. Requires the user setting \"use last reconciled time\" (see current_user.use_last_reconciled_time of get_server_capabilities). The time cannot move backwards. Use get_account_reconciliation_statement to review the account before marking it as reconciled."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpSetAccountReconciledTimeToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpSetAccountReconciledTimeToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpSetAccountReconciledTimeToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpSetAccountReconciledTimeToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPSetAccountReconciledTimeRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpSetAccountReconciledTimeToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPSetAccountReconciledTimeResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpSetAccountReconciledTimeToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPSetAccountReconciledTimeRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if !user.UseLastReconciledTime {
		return nil, nil, NewMCPNotSupportedError("", "last reconciled time is not enabled for the current user, enable \"use last reconciled time\" in the user settings of the web interface first")
	}

	reconciledTime := time.Now()

	if request.ReconciledTime != "" {
		parsedTime, err := parseMCPTime("reconciled_time", request.ReconciledTime, true)

		if err != nil {
			return nil, nil, err
		}

		reconciledTime = *parsedTime
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[set_account_reconciled_time_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	account, err := mcpAccountNewResolver().Resolve(allAccounts, "account_id", request.AccountId, "account_name", request.AccountName)

	if err != nil {
		return nil, nil, err
	}

	if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "account_id", "account \""+account.Name+"\" has sub-accounts, the last reconciled time can only be set for its sub-accounts")
	}

	newLastReconciledTime := reconciledTime.Unix()
	previousLastReconciledTime := account.GetLastReconciledTime()

	response := &MCPSetAccountReconciledTimeResponse{
		Success:            true,
		AccountId:          utils.Int64ToString(account.AccountId),
		AccountName:        account.Name,
		LastReconciledTime: formatMCPUnixTime(newLastReconciledTime),
	}

	if previousLastReconciledTime > 0 {
		response.PreviousLastReconciledTime = formatMCPUnixTime(previousLastReconciledTime)
	}

	if user.TransactionEditScope == models.TRANSACTION_EDIT_SCOPE_LAST_RECONCILED_TIME_OR_LATER {
		response.EditScopeNote = "Transactions of this account at or before the last reconciled time can no longer be created, modified or deleted (transaction edit scope: last_reconciled_time_or_later)."
	}

	if account.Extend != nil && account.Extend.LastReconciledTime != nil {
		if newLastReconciledTime < *account.Extend.LastReconciledTime {
			err := NewMCPToolError(MCPToolErrorCodeNotAllowed, "reconciled_time", "\"reconciled_time\" cannot be earlier than the current last reconciled time of the account ("+formatMCPUnixTime(*account.Extend.LastReconciledTime)+")")
			err.Value = request.ReconciledTime
			return nil, nil, err
		} else if newLastReconciledTime == *account.Extend.LastReconciledTime {
			response.Status = MCPWriteStatusUnchanged
			return newMCPJSONTextResult(response)
		}
	}

	if request.DryRun {
		response.DryRun = true
		response.Status = MCPWriteStatusWouldBeUpdated
		return newMCPJSONTextResult(response)
	}

	if account.Extend == nil {
		account.Extend = &models.AccountExtend{}
	}

	account.Extend.LastReconciledTime = &newLastReconciledTime
	err = services.GetAccountService().UpdateAccountExtend(c, uid, account)

	if err != nil {
		log.Errorf(c, "[set_account_reconciled_time_tool_handler.Handle] failed to update last reconciled time for account \"id:%d\" of user \"uid:%d\", because %s", account.AccountId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[set_account_reconciled_time_tool_handler.Handle] user \"uid:%d\" has updated last reconciled time \"%d\" for account \"id:%d\"", uid, newLastReconciledTime, account.AccountId)

	response.Committed = true
	response.Status = MCPWriteStatusUpdated

	return newMCPJSONTextResult(response)
}

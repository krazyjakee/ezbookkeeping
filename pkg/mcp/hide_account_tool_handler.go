package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPHideAccountRequest represents all parameters of the hide account request
type MCPHideAccountRequest struct {
	AccountId   string `json:"account_id,omitempty" jsonschema_description:"Account ID (preferred, either account_id or account_name is required)"`
	AccountName string `json:"account_name,omitempty" jsonschema_description:"Account name (used if account_id is not set, returns an ambiguous_name error with candidates if more than one account has this name)"`
	Hidden      bool   `json:"hidden" jsonschema_description:"true to hide (archive) the account, false to unhide it"`
}

// MCPHideAccountResponse represents the response structure for hide account
type MCPHideAccountResponse struct {
	Success bool            `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Status  string          `json:"status" jsonschema:"enum=updated,enum=unchanged" jsonschema_description:"Result status: updated (committed) or unchanged (the account already had the requested visibility)"`
	Account *MCPAccountInfo `json:"account" jsonschema_description:"The account after the operation"`
}

type mcpHideAccountToolHandler struct{}

var MCPHideAccountToolHandler = &mcpHideAccountToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpHideAccountToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_HIDE_ACCOUNT
}

// Name returns the name of the MCP tool
func (h *mcpHideAccountToolHandler) Name() string {
	return "hide_account"
}

// Description returns the description of the MCP tool
func (h *mcpHideAccountToolHandler) Description() string {
	return "Hide (archive) or unhide an account or sub-account. Hidden accounts keep their transactions and balances but are not shown or offered by default. Hiding a parent account does not change the visibility flag of its sub-accounts."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpHideAccountToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpHideAccountToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpHideAccountToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpHideAccountToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPHideAccountRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpHideAccountToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPHideAccountResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpHideAccountToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	hideAccountRequest, err := parseMCPToolArguments[MCPHideAccountRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[hide_account_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	account, err := mcpAccountNewResolver().Resolve(allAccounts, "account_id", hideAccountRequest.AccountId, "account_name", hideAccountRequest.AccountName)

	if err != nil {
		return nil, nil, err
	}

	if account.Hidden == hideAccountRequest.Hidden {
		return newMCPJSONTextResult(&MCPHideAccountResponse{
			Success: true,
			Status:  MCPWriteStatusUnchanged,
			Account: mcpAccountBuildInfo(account, allAccounts, true),
		})
	}

	err = services.GetAccountService().HideAccount(c, uid, []int64{account.AccountId}, hideAccountRequest.Hidden)

	if err != nil {
		log.Errorf(c, "[hide_account_tool_handler.Handle] failed to hide account \"id:%d\" for user \"uid:%d\", because %s", account.AccountId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[hide_account_tool_handler.Handle] user \"uid:%d\" has set hidden of account \"id:%d\" to %t", uid, account.AccountId, hideAccountRequest.Hidden)

	account.Hidden = hideAccountRequest.Hidden
	accountInfo := mcpAccountBuildInfo(account, allAccounts, true)
	accountAndSubAccounts, err := services.GetAccountService().GetAccountAndSubAccountsByAccountId(c, uid, account.AccountId)

	if err != nil {
		log.Warnf(c, "[hide_account_tool_handler.Handle] failed to get latest account \"id:%d\" info after account updated, because %s", account.AccountId, err.Error())
	} else if latestAccountInfo, err := mcpAccountBuildInfoFromList(accountAndSubAccounts, account.AccountId); err == nil {
		accountInfo = latestAccountInfo
	}

	return newMCPJSONTextResult(&MCPHideAccountResponse{
		Success: true,
		Status:  MCPWriteStatusUpdated,
		Account: accountInfo,
	})
}

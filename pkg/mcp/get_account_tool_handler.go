package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPGetAccountRequest represents all parameters of the get account request
type MCPGetAccountRequest struct {
	AccountId   string `json:"account_id,omitempty" jsonschema_description:"Account ID (preferred, either account_id or account_name is required)"`
	AccountName string `json:"account_name,omitempty" jsonschema_description:"Account name (used if account_id is not set, returns an ambiguous_name error with candidates if more than one account has this name)"`
}

// MCPGetAccountResponse represents the response structure for get account
type MCPGetAccountResponse struct {
	Account *MCPAccountInfo `json:"account" jsonschema_description:"The account (including hidden sub-accounts if it is a parent account)"`
}

type mcpGetAccountToolHandler struct{}

var MCPGetAccountToolHandler = &mcpGetAccountToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetAccountToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_ACCOUNT
}

// Name returns the name of the MCP tool
func (h *mcpGetAccountToolHandler) Name() string {
	return "get_account"
}

// Description returns the description of the MCP tool
func (h *mcpGetAccountToolHandler) Description() string {
	return "Get an account (or sub-account) by stable ID or by name, including category, currency, type, visibility, parent/sub-accounts and balances (signed balance, and outstanding balance for liability accounts)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetAccountToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetAccountToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetAccountToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetAccountToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetAccountRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetAccountToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetAccountResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetAccountToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	getAccountRequest, err := parseMCPToolArguments[MCPGetAccountRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[get_account_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	account, err := mcpAccountNewResolver().Resolve(allAccounts, "account_id", getAccountRequest.AccountId, "account_name", getAccountRequest.AccountName)

	if err != nil {
		return nil, nil, err
	}

	return newMCPJSONTextResult(&MCPGetAccountResponse{
		Account: mcpAccountBuildInfo(account, allAccounts, true),
	})
}

package mcp

import (
	"reflect"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPListAccountsRequest represents all parameters of the list accounts request
type MCPListAccountsRequest struct {
	IncludeHidden bool   `json:"include_hidden,omitempty" jsonschema_description:"If true, hidden accounts and hidden sub-accounts are also returned (optional, default: false)"`
	Category      string `json:"category,omitempty" jsonschema:"enum=cash,enum=checking_account,enum=savings_account,enum=credit_card,enum=virtual,enum=debt,enum=receivables,enum=certificate_of_deposit,enum=investment" jsonschema_description:"Only return accounts of this category (optional)"`
}

// MCPListAccountsResponse represents the response structure for list accounts
type MCPListAccountsResponse struct {
	Accounts []*MCPAccountInfo `json:"accounts" jsonschema_description:"Top-level accounts sorted by category and display order, sub-accounts are nested in sub_accounts of their parent account"`
}

type mcpListAccountsToolHandler struct{}

var MCPListAccountsToolHandler = &mcpListAccountsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpListAccountsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_LIST_ACCOUNTS
}

// Name returns the name of the MCP tool
func (h *mcpListAccountsToolHandler) Name() string {
	return "list_accounts"
}

// Description returns the description of the MCP tool
func (h *mcpListAccountsToolHandler) Description() string {
	return "List accounts with stable IDs, category, currency, asset/liability type, visibility, parent/sub-account relationships and balances. For liability accounts, balance is the signed accounting balance (negative means owed) and outstanding_balance is the amount owed as displayed in the web interface."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpListAccountsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpListAccountsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpListAccountsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpListAccountsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPListAccountsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpListAccountsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPListAccountsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpListAccountsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	listAccountsRequest, err := parseMCPToolArguments[MCPListAccountsRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	category := models.AccountCategory(0)

	if strings.TrimSpace(listAccountsRequest.Category) != "" {
		category, err = parseMCPAccountCategory("category", strings.TrimSpace(listAccountsRequest.Category))

		if err != nil {
			return nil, nil, err
		}
	}

	uid := user.Uid
	accounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[list_accounts_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	return newMCPJSONTextResult(&MCPListAccountsResponse{
		Accounts: mcpAccountBuildInfoList(accounts, listAccountsRequest.IncludeHidden, category),
	})
}

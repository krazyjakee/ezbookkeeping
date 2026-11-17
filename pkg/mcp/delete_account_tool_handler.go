package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPDeleteAccountRequest represents all parameters of the delete account request
type MCPDeleteAccountRequest struct {
	AccountId string `json:"account_id" jsonschema_description:"ID of the account or sub-account to delete (names are not accepted for this destructive operation)"`
	Confirm   bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion"`
}

// MCPDeleteAccountResponse represents the response structure for delete account
type MCPDeleteAccountResponse struct {
	Success              bool     `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Status               string   `json:"status" jsonschema:"enum=deleted" jsonschema_description:"Result status, always deleted (committed) on success"`
	DeletedAccountId     string   `json:"deleted_account_id" jsonschema_description:"ID of the deleted account"`
	DeletedSubAccountIds []string `json:"deleted_sub_account_ids,omitempty" jsonschema_description:"IDs of the sub-accounts deleted together with the parent account"`
	Permanent            bool     `json:"permanent" jsonschema_description:"Always true, deleted accounts cannot be restored through MCP"`
}

type mcpDeleteAccountToolHandler struct{}

var MCPDeleteAccountToolHandler = &mcpDeleteAccountToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteAccountToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_ACCOUNT
}

// Name returns the name of the MCP tool
func (h *mcpDeleteAccountToolHandler) Name() string {
	return "delete_account"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteAccountToolHandler) Description() string {
	return "Permanently delete an account (together with its sub-accounts) or a single sub-account by ID. There is no restore through MCP, consider hide_account instead. " +
		"The same restrictions as the web interface apply: the account and its sub-accounts must have no transactions other than their opening balance (which is deleted together), " +
		"must not be used by transaction templates or active scheduled transactions, and the last sub-account of a parent account cannot be deleted. Requires confirm=true."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteAccountToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteAccountToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteAccountToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteAccountToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteAccountRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteAccountToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteAccountResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteAccountToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	deleteAccountRequest, err := parseMCPToolArguments[MCPDeleteAccountRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	accountId, err := parseMCPRequiredId("account_id", deleteAccountRequest.AccountId)

	if err != nil {
		return nil, nil, err
	}

	if !deleteAccountRequest.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting an account is permanent, set \"confirm\" to true to proceed")
	}

	uid := user.Uid
	accountAndSubAccounts, err := services.GetAccountService().GetAccountAndSubAccountsByAccountId(c, uid, accountId)

	if err != nil {
		log.Errorf(c, "[delete_account_tool_handler.Handle] failed to get account \"id:%d\" for user \"uid:%d\", because %s", accountId, uid, err.Error())
		return nil, nil, err
	}

	var account *models.Account
	subAccountIds := make([]string, 0)

	for i := 0; i < len(accountAndSubAccounts); i++ {
		if accountAndSubAccounts[i].AccountId == accountId {
			account = accountAndSubAccounts[i]
		} else if accountAndSubAccounts[i].ParentAccountId == accountId {
			subAccountIds = append(subAccountIds, utils.Int64ToString(accountAndSubAccounts[i].AccountId))
		}
	}

	if account == nil {
		return nil, nil, NewMCPNotFoundError("account_id", "account", deleteAccountRequest.AccountId)
	}

	if account.ParentAccountId == models.LevelOneAccountParentId {
		err = services.GetAccountService().DeleteAccount(c, uid, accountId)
	} else {
		subAccountIds = nil
		err = services.GetAccountService().DeleteSubAccount(c, uid, accountId)
	}

	if err != nil {
		log.Errorf(c, "[delete_account_tool_handler.Handle] failed to delete account \"id:%d\" for user \"uid:%d\", because %s", accountId, uid, err.Error())
		return nil, nil, mcpAccountConvertServiceError(err)
	}

	log.Infof(c, "[delete_account_tool_handler.Handle] user \"uid:%d\" has deleted account \"id:%d\"", uid, accountId)

	return newMCPJSONTextResult(&MCPDeleteAccountResponse{
		Success:              true,
		Status:               MCPWriteStatusDeleted,
		DeletedAccountId:     utils.Int64ToString(accountId),
		DeletedSubAccountIds: subAccountIds,
		Permanent:            true,
	})
}

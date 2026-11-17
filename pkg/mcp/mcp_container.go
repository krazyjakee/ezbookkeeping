package mcp

import (
	"encoding/json/v2"
	"errors"

	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPContainer contains the all mcp handlers
type MCPContainer struct {
	mcpTextContentTools      *orderedmap.OrderedMap[string, MCPToolHandler[MCPTextContent]]
	mcpImageContentTools     *orderedmap.OrderedMap[string, MCPToolHandler[MCPImageContent]]
	mcpAudioContentTools     *orderedmap.OrderedMap[string, MCPToolHandler[MCPAudioContent]]
	mcpResourceLinkTools     *orderedmap.OrderedMap[string, MCPToolHandler[MCPResourceLink]]
	mcpEmbeddedResourceTools *orderedmap.OrderedMap[string, MCPToolHandler[MCPEmbeddedResource]]
	mcpTools                 []*MCPTool
}

// Initialize a mcp handler container singleton instance
var (
	Container = &MCPContainer{}
)

// GetMCPTools returns the registered MCP tools
func (c *MCPContainer) GetMCPTools() []*MCPTool {
	if len(c.mcpTools) == 0 {
		return nil
	}

	return c.mcpTools
}

// HandleTool returns the result of the MCP tool handler based on the tool name
func (c *MCPContainer) HandleTool(ctx *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, error) {
	if handler, exists := c.mcpTextContentTools.Get(callToolReq.Name); exists {
		return handleTool(ctx, handler, currentConfig, services, callToolReq, user)
	}

	if handler, exists := c.mcpImageContentTools.Get(callToolReq.Name); exists {
		return handleTool(ctx, handler, currentConfig, services, callToolReq, user)
	}

	if handler, exists := c.mcpAudioContentTools.Get(callToolReq.Name); exists {
		return handleTool(ctx, handler, currentConfig, services, callToolReq, user)
	}

	if handler, exists := c.mcpResourceLinkTools.Get(callToolReq.Name); exists {
		return handleTool(ctx, handler, currentConfig, services, callToolReq, user)
	}

	if handler, exists := c.mcpEmbeddedResourceTools.Get(callToolReq.Name); exists {
		return handleTool(ctx, handler, currentConfig, services, callToolReq, user)
	}

	return nil, errs.ErrApiNotFound
}

// InitializeMCPHandlers initializes the all mcp handlers according to the config
func InitializeMCPHandlers(config *settings.Config) error {
	container := &MCPContainer{
		mcpTextContentTools:      orderedmap.New[string, MCPToolHandler[MCPTextContent]](),
		mcpImageContentTools:     orderedmap.New[string, MCPToolHandler[MCPImageContent]](),
		mcpAudioContentTools:     orderedmap.New[string, MCPToolHandler[MCPAudioContent]](),
		mcpResourceLinkTools:     orderedmap.New[string, MCPToolHandler[MCPResourceLink]](),
		mcpEmbeddedResourceTools: orderedmap.New[string, MCPToolHandler[MCPEmbeddedResource]](),
		mcpTools:                 make([]*MCPTool, 0),
	}

	registerMCPTextContentToolHandler(container, MCPQueryAllAccountsToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryAllAccountsBalanceToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryAllTransactionCategoriesToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryAllTransactionTagsToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryLatestExchangeRatesToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryTransactionsToolHandler)
	registerMCPTextContentToolHandler(container, MCPAddTransactionToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetServerCapabilitiesToolHandler)
	registerMCPTextContentToolHandler(container, MCPListAccountsToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetAccountToolHandler)
	registerMCPTextContentToolHandler(container, MCPCreateAccountToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateAccountToolHandler)
	registerMCPTextContentToolHandler(container, MCPHideAccountToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteAccountToolHandler)
	registerMCPTextContentToolHandler(container, MCPListTransactionCategoriesToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetTransactionCategoryToolHandler)
	registerMCPTextContentToolHandler(container, MCPCreateTransactionCategoryToolHandler)
	registerMCPTextContentToolHandler(container, MCPBatchCreateTransactionCategoriesToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateTransactionCategoryToolHandler)
	registerMCPTextContentToolHandler(container, MCPHideTransactionCategoryToolHandler)
	registerMCPTextContentToolHandler(container, MCPReorderTransactionCategoriesToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionCategoryToolHandler)
	registerMCPTextContentToolHandler(container, MCPListTransactionTagsToolHandler)
	registerMCPTextContentToolHandler(container, MCPCreateTransactionTagsToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateTransactionTagToolHandler)
	registerMCPTextContentToolHandler(container, MCPReorderTransactionTagsToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionTagToolHandler)
	registerMCPTextContentToolHandler(container, MCPCreateTransactionTagGroupToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateTransactionTagGroupToolHandler)
	registerMCPTextContentToolHandler(container, MCPReorderTransactionTagGroupsToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionTagGroupToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetTransactionToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateTransactionToolHandler)
	registerMCPTextContentToolHandler(container, MCPListTransactionTemplatesToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetTransactionTemplateToolHandler)
	registerMCPTextContentToolHandler(container, MCPCreateTransactionTemplateToolHandler)
	registerMCPTextContentToolHandler(container, MCPUpdateTransactionTemplateToolHandler)
	registerMCPTextContentToolHandler(container, MCPHideTransactionTemplateToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionTemplateToolHandler)
	registerMCPTextContentToolHandler(container, MCPUploadTransactionAttachmentToolHandler)
	registerMCPTextContentToolHandler(container, MCPListTransactionAttachmentsToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetTransactionAttachmentToolHandler)
	registerMCPEmbeddedResourceToolHandler(container, MCPDownloadTransactionAttachmentToolHandler)
	registerMCPTextContentToolHandler(container, MCPDetachTransactionAttachmentToolHandler)
	registerMCPTextContentToolHandler(container, MCPDeleteTransactionAttachmentToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryAccountBalancesAsOfToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryTransactionTotalsToolHandler)
	registerMCPTextContentToolHandler(container, MCPGetReconciliationStatementToolHandler)
	registerMCPTextContentToolHandler(container, MCPQueryAuditLogToolHandler)
	registerMCPTextContentToolHandler(container, MCPImportTransactionsToolHandler)
	registerMCPTextContentToolHandler(container, MCPSetAccountReconciledTimeToolHandler)
	registerMCPTextContentToolHandler(container, MCPExportTransactionsToolHandler)

	Container = container
	return nil
}

func registerMCPTextContentToolHandler(c *MCPContainer, handler MCPToolHandler[MCPTextContent]) {
	registerMCPToolHandler(c, c.mcpTextContentTools, handler)
}

func registerMCPImageContentToolHandler(c *MCPContainer, handler MCPToolHandler[MCPImageContent]) {
	registerMCPToolHandler(c, c.mcpImageContentTools, handler)
}

func registerMCPAudioContentToolHandler(c *MCPContainer, handler MCPToolHandler[MCPAudioContent]) {
	registerMCPToolHandler(c, c.mcpAudioContentTools, handler)
}

func registerMCPResourceLinkToolHandler(c *MCPContainer, handler MCPToolHandler[MCPResourceLink]) {
	registerMCPToolHandler(c, c.mcpResourceLinkTools, handler)
}

func registerMCPEmbeddedResourceToolHandler(c *MCPContainer, handler MCPToolHandler[MCPEmbeddedResource]) {
	registerMCPToolHandler(c, c.mcpEmbeddedResourceTools, handler)
}

func registerMCPToolHandler[T MCPTextContent | MCPImageContent | MCPAudioContent | MCPResourceLink | MCPEmbeddedResource](c *MCPContainer, mcpToolHandlerMap *orderedmap.OrderedMap[string, MCPToolHandler[T]], handler MCPToolHandler[T]) {
	if _, exists := mcpToolHandlerMap.Get(handler.Name()); exists {
		return
	}

	mcpToolHandlerMap.Set(handler.Name(), handler)
	c.mcpTools = append(c.mcpTools, createNewMCPToolInfo(handler.Name(), handler))
}

func handleTool[T MCPTextContent | MCPImageContent | MCPAudioContent | MCPResourceLink | MCPEmbeddedResource](ctx *core.WebContext, handler MCPToolHandler[T], currentConfig *settings.Config, services MCPAvailableServices, callToolReq *MCPCallToolRequest, user *models.User) (any, error) {
	enabledTools := core.MCPAllTools

	if context := ctx.GetTokenContext(); context != "" {
		var tokenContext models.MCPTokenContext
		err := json.Unmarshal([]byte(context), &tokenContext)

		if err != nil {
			log.Errorf(ctx, "[mcp_container.handleTool] failed to parse token context \"%s\", because %s", context, err.Error())
			return nil, errs.ErrInvalidToken
		}

		enabledTools = tokenContext.EnabledTools
	}

	if !handler.ToolType().IsEnabled(enabledTools) {
		return nil, errs.ErrMCPToolNotEnabled
	}

	var writeCall *mcpWriteToolCall

	if handler.DataOperationType() != MCP_TOOL_DATA_OPERATION_READ_ONLY && user != nil {
		defaultDryRun := false

		if defaultDryRunHandler, ok := any(handler).(mcpDefaultDryRunToolHandler); ok {
			defaultDryRun = defaultDryRunHandler.DefaultDryRun()
		}

		var err error
		writeCall, err = mcpBeginWriteToolCall(ctx, handler.Name(), mcpToolInputSupportsDryRun(handler.InputType()), defaultDryRun, callToolReq, user, services)

		if err != nil {
			var toolErr *MCPToolError

			if errors.As(err, &toolErr) {
				return createMCPToolErrorResponse(toolErr)
			}

			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		if writeCall.replayResponse != nil {
			return createMCPToolReplayResponse(ctx, writeCall.replayResponse), nil
		}
	}

	structuredResponse, result, err := handler.Handle(ctx, callToolReq, user, currentConfig, services)

	if err != nil {
		writeCall.abort(ctx)

		var toolErr *MCPToolError

		if errors.As(err, &toolErr) {
			return createMCPToolErrorResponse(toolErr)
		}

		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	writeCall.finish(ctx, services, structuredResponse)

	callToolResp := MCPCallToolResponse[T]{
		Content: result,
		IsError: false,
	}

	if ctx.GetHeader(MCPProtocolVersionHeaderName) >= string(ToolResultStructuredContentMinVersion) {
		callToolResp.StructuredContent = structuredResponse
	}

	return callToolResp, nil
}

// createMCPToolErrorResponse returns the tool result with isError set to true for user-correctable errors,
// so that the client (and the model) can read the failing field, limit and candidates and correct the input
func createMCPToolErrorResponse(toolErr *MCPToolError) (any, error) {
	content, err := json.Marshal(&MCPToolErrorResult{
		Success: false,
		Error:   toolErr,
	})

	if err != nil {
		return nil, errs.ErrOperationFailed
	}

	return MCPCallToolResponse[MCPTextContent]{
		Content: []*MCPTextContent{
			NewMCPTextContent(string(content)),
		},
		IsError: true,
	}, nil
}

func createNewMCPToolInfo[T MCPTextContent | MCPImageContent | MCPAudioContent | MCPResourceLink | MCPEmbeddedResource](name string, handler MCPToolHandler[T]) *MCPTool {
	dataOperationType := handler.DataOperationType()
	mcpTool := &MCPTool{
		ToolType:    handler.ToolType(),
		Name:        name,
		Description: handler.Description(),
		Annotations: &MCPToolAnnotations{
			ReadOnlyHint:    dataOperationType == MCP_TOOL_DATA_OPERATION_READ_ONLY,
			DestructiveHint: dataOperationType != MCP_TOOL_DATA_OPERATION_READ_ONLY && dataOperationType != MCP_TOOL_DATA_OPERATION_ADD,
			IdempotentHint:  handler.IsIdempotent(),
			OpenWorldHint:   handler.DependsOnExternalData(),
		},
	}

	schemeGenerator := jsonschema.Reflector{
		Anonymous:      true,
		DoNotReference: true,
		ExpandedStruct: true,
	}

	if handler.InputType() != nil {
		schema := schemeGenerator.ReflectFromType(handler.InputType())
		schema.Version = ""

		if dataOperationType != MCP_TOOL_DATA_OPERATION_READ_ONLY {
			addMCPWriteToolControlArgumentsToSchema(schema)
		}

		mcpTool.InputSchema = schema
	} else {
		mcpTool.InputSchema = &jsonschema.Schema{
			Type: "object",
		}
	}

	if handler.OutputType() != nil {
		schema := schemeGenerator.ReflectFromType(handler.OutputType())
		schema.Version = ""

		if dataOperationType != MCP_TOOL_DATA_OPERATION_READ_ONLY {
			addMCPWriteToolReplayFieldToSchema(schema)
		}

		mcpTool.OutputSchema = schema
	} else {
		mcpTool.OutputSchema = &jsonschema.Schema{
			Type: "object",
		}
	}

	return mcpTool
}

// addMCPWriteToolControlArgumentsToSchema adds the control arguments accepted by all write tools (idempotency_key and operation_id) to the input schema
func addMCPWriteToolControlArgumentsToSchema(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}

	if _, exists := schema.Properties.Get(mcpIdempotencyKeyArgumentName); !exists {
		maxLength := uint64(models.MCPIdempotencyKeyMaxLength)
		schema.Properties.Set(mcpIdempotencyKeyArgumentName, &jsonschema.Schema{
			Type:        "string",
			MaxLength:   &maxLength,
			Description: "Client generated idempotency key (optional, maximum 128 characters, e.g. a UUID). Retrying the same request with the same key returns the stored original result (marked with idempotent_replay=true) without executing the write again; the same key with a different request fails with idempotency_conflict. Keys are kept for 30 days and are ignored for dry runs",
		})
	}

	if _, exists := schema.Properties.Get(mcpOperationIdArgumentName); !exists {
		maxLength := uint64(mcpOperationIdMaxLength)
		schema.Properties.Set(mcpOperationIdArgumentName, &jsonschema.Schema{
			Type:        "string",
			MaxLength:   &maxLength,
			Description: "Client provided operation / reference ID (optional, maximum 128 characters) which is recorded in the audit history of committed changes (see query_audit_log)",
		})
	}
}

// addMCPWriteToolReplayFieldToSchema adds the idempotent_replay field of replayed results to the output schema of write tools
func addMCPWriteToolReplayFieldToSchema(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}

	if _, exists := schema.Properties.Get(mcpIdempotentReplayFieldName); !exists {
		schema.Properties.Set(mcpIdempotentReplayFieldName, &jsonschema.Schema{
			Type:        "boolean",
			Description: "True if this is the stored result of an earlier call with the same idempotency key (the write was not executed again)",
		})
	}
}

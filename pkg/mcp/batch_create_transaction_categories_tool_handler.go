package mcp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Error codes used only in the results of batch creating transaction categories
const (
	mcpTransactionCategoryBatchErrorCodeWriteFailed  = "write_failed"
	mcpTransactionCategoryBatchErrorCodeNotCommitted = "not_committed"
)

// MCPBatchCreateTransactionCategoriesSubItem represents a sub-category to create in the batch create transaction categories request
type MCPBatchCreateTransactionCategoriesSubItem struct {
	Name    string `json:"name" jsonschema_description:"Sub-category name"`
	Icon    string `json:"icon,omitempty" jsonschema_description:"System icon ID (optional, default 1)"`
	Color   string `json:"color,omitempty" jsonschema_description:"Color in 6-digit hex RGB format (optional, default 000000)"`
	Comment string `json:"comment,omitempty" jsonschema_description:"Sub-category description (optional)"`
}

// MCPBatchCreateTransactionCategoriesItem represents a primary category (with its sub-categories) to create in the batch create transaction categories request
type MCPBatchCreateTransactionCategoriesItem struct {
	Name          string                                        `json:"name" jsonschema_description:"Primary category name"`
	Type          string                                        `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type (income, expense, transfer), sub-categories use the same type"`
	Icon          string                                        `json:"icon,omitempty" jsonschema_description:"System icon ID (optional, default 1)"`
	Color         string                                        `json:"color,omitempty" jsonschema_description:"Color in 6-digit hex RGB format (optional, default 000000)"`
	Comment       string                                        `json:"comment,omitempty" jsonschema_description:"Primary category description (optional)"`
	SubCategories []*MCPBatchCreateTransactionCategoriesSubItem `json:"sub_categories,omitempty" jsonschema_description:"Sub-categories of this primary category (optional)"`
}

// MCPBatchCreateTransactionCategoriesRequest represents all parameters of the batch create transaction categories request
type MCPBatchCreateTransactionCategoriesRequest struct {
	Categories   []*MCPBatchCreateTransactionCategoriesItem `json:"categories" jsonschema_description:"Primary categories to create, each with optional sub-categories"`
	SkipExisting *bool                                      `json:"skip_existing,omitempty" jsonschema_description:"If true (default), a category with the same name (exact, or case-insensitive if no exact match) under the same parent with the same type is reused instead of creating a duplicate, so the same batch can be rerun safely. Set to false to always create new categories"`
	DryRun       bool                                       `json:"dry_run,omitempty" jsonschema_description:"If true, only validate and preview the per-item results without saving (optional)"`
}

// MCPBatchCreateTransactionCategoriesResult represents the result of one item in the batch create transaction categories request
type MCPBatchCreateTransactionCategoriesResult struct {
	Index      int           `json:"index" jsonschema_description:"Index of the primary category in the request"`
	SubIndex   *int          `json:"sub_index,omitempty" jsonschema_description:"Index of the sub-category in the primary category (only for sub-categories)"`
	Name       string        `json:"name" jsonschema_description:"Category name"`
	Type       string        `json:"type" jsonschema_description:"Transaction category type"`
	ParentId   string        `json:"parent_id,omitempty" jsonschema_description:"Parent category ID (only for sub-categories whose parent category is saved or reused)"`
	ParentName string        `json:"parent_name,omitempty" jsonschema_description:"Parent category name (only for sub-categories)"`
	Status     string        `json:"status" jsonschema:"enum=created,enum=existing,enum=would_be_created,enum=failed,enum=skipped" jsonschema_description:"created: saved; existing: an existing category is reused; would_be_created: will be created (dry run, or not committed because of validation failures of other items); failed: validation or write failure (see error); skipped: not processed because of another failure"`
	Id         string        `json:"id,omitempty" jsonschema_description:"Category ID (for created or reused categories)"`
	Error      *MCPToolError `json:"error,omitempty" jsonschema_description:"Error details (for failed or skipped items)"`
}

// MCPBatchCreateTransactionCategoriesResponse represents the response structure for batch creating transaction categories
type MCPBatchCreateTransactionCategoriesResponse struct {
	Success       bool                                         `json:"success" jsonschema_description:"Indicates whether all items are valid and (unless dry run) committed"`
	DryRun        bool                                         `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing is saved)"`
	Committed     bool                                         `json:"committed" jsonschema_description:"Indicates whether at least one category has been saved"`
	CreatedCount  int                                          `json:"created_count" jsonschema_description:"Count of created (or, in dry run, would be created) categories"`
	ExistingCount int                                          `json:"existing_count" jsonschema_description:"Count of reused existing categories"`
	FailedCount   int                                          `json:"failed_count" jsonschema_description:"Count of failed items"`
	SkippedCount  int                                          `json:"skipped_count" jsonschema_description:"Count of skipped items"`
	Results       []*MCPBatchCreateTransactionCategoriesResult `json:"results" jsonschema_description:"Per-item results in request order (each primary category followed by its sub-categories)"`
}

// mcpTransactionCategoryBatchPlanGroup represents the plan of one primary category and its sub-categories
type mcpTransactionCategoryBatchPlanGroup struct {
	categoryType    models.TransactionCategoryType
	primaryResult   *MCPBatchCreateTransactionCategoriesResult
	primaryExisting *models.TransactionCategory
	primaryNew      *models.TransactionCategory
	subResults      []*MCPBatchCreateTransactionCategoriesResult
	subNews         []*models.TransactionCategory
}

type mcpBatchCreateTransactionCategoriesToolHandler struct{}

var MCPBatchCreateTransactionCategoriesToolHandler = &mcpBatchCreateTransactionCategoriesToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpBatchCreateTransactionCategoriesToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_BATCH_CREATE_CATEGORIES
}

// Name returns the name of the MCP tool
func (h *mcpBatchCreateTransactionCategoriesToolHandler) Name() string {
	return "batch_create_transaction_categories"
}

// Description returns the description of the MCP tool
func (h *mcpBatchCreateTransactionCategoriesToolHandler) Description() string {
	return fmt.Sprintf("Create a hierarchy of transaction categories (primary categories with sub-categories) in one call, with per-item results. "+
		"By default existing categories with the same name under the same parent and type are reused (skip_existing), so rerunning the same setup does not create duplicates. "+
		"Validation is all-or-nothing: if any item is invalid (or a name is ambiguous), nothing is saved. "+
		"When committing, each primary category with its new sub-categories is saved in one database transaction, groups are saved in request order; "+
		"if a write fails, later groups are not saved and the results report the actual committed state. "+
		"Use dry_run to preview. At most %d categories (primary and sub-categories) per call.", mcpMaximumBatchItemsCount)
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpBatchCreateTransactionCategoriesToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpBatchCreateTransactionCategoriesToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpBatchCreateTransactionCategoriesToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpBatchCreateTransactionCategoriesToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPBatchCreateTransactionCategoriesRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpBatchCreateTransactionCategoriesToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPBatchCreateTransactionCategoriesResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpBatchCreateTransactionCategoriesToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPBatchCreateTransactionCategoriesRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if len(request.Categories) < 1 {
		return nil, nil, NewMCPRequiredFieldError("categories")
	}

	totalCount := 0

	for i := 0; i < len(request.Categories); i++ {
		totalCount++

		if request.Categories[i] != nil {
			totalCount += len(request.Categories[i].SubCategories)
		}
	}

	if totalCount > mcpMaximumBatchItemsCount {
		return nil, nil, NewMCPTooManyItemsError("categories", mcpMaximumBatchItemsCount, totalCount)
	}

	skipExisting := true

	if request.SkipExisting != nil {
		skipExisting = *request.SkipExisting
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[batch_create_transaction_categories_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	groups, results, valid := mcpTransactionCategoryPlanBatch(uid, allCategories, request.Categories, skipExisting)
	committed := false

	if valid && !request.DryRun {
		committed, valid = h.commit(c, uid, groups, services)
	}

	response := MCPBatchCreateTransactionCategoriesResponse{
		Success:   valid,
		DryRun:    request.DryRun,
		Committed: committed,
		Results:   results,
	}

	for i := 0; i < len(results); i++ {
		switch results[i].Status {
		case MCPWriteStatusCreated, MCPWriteStatusWouldBeCreated:
			response.CreatedCount++
		case MCPWriteStatusExisting:
			response.ExistingCount++
		case MCPWriteStatusFailed:
			response.FailedCount++
		case MCPWriteStatusSkipped:
			response.SkippedCount++
		}
	}

	return newMCPJSONTextResult(response)
}

// commit saves the planned categories group by group, returns whether anything has been saved and whether all groups have been saved
func (h *mcpBatchCreateTransactionCategoriesToolHandler) commit(c *core.WebContext, uid int64, groups []*mcpTransactionCategoryBatchPlanGroup, services MCPAvailableServices) (bool, bool) {
	categoryService := services.GetTransactionCategoryService()
	maxDisplayOrders := make(map[models.TransactionCategoryType]int32)
	committed := false

	for i := 0; i < len(groups); i++ {
		group := groups[i]
		var err error

		if group.primaryNew != nil {
			newSubCategories := make([]*models.TransactionCategory, 0, len(group.subNews))
			maxOrderId, exists := maxDisplayOrders[group.categoryType]

			if !exists {
				maxOrderId, err = categoryService.GetMaxDisplayOrder(c, uid, group.categoryType)
			}

			if err == nil {
				group.primaryNew.DisplayOrder = maxOrderId + 1

				for j := 0; j < len(group.subNews); j++ {
					if group.subNews[j] != nil {
						group.subNews[j].DisplayOrder = int32(len(newSubCategories) + 1)
						newSubCategories = append(newSubCategories, group.subNews[j])
					}
				}

				_, err = categoryService.CreateCategories(c, uid, map[*models.TransactionCategory][]*models.TransactionCategory{
					nil:              {group.primaryNew},
					group.primaryNew: newSubCategories,
				})
			}

			if err != nil {
				log.Errorf(c, "[batch_create_transaction_categories_tool_handler.commit] failed to create category group #%d for user \"uid:%d\", because %s", i, uid, err.Error())
				mcpTransactionCategoryBatchMarkWriteFailed(group.primaryResult, err)

				for j := 0; j < len(group.subResults); j++ {
					if group.subNews[j] != nil {
						mcpTransactionCategoryBatchMarkWriteFailed(group.subResults[j], err)
					}
				}

				mcpTransactionCategoryBatchMarkNotCommitted(groups[i+1:])
				return committed, false
			}

			committed = true
			maxDisplayOrders[group.categoryType] = group.primaryNew.DisplayOrder
			group.primaryResult.Status = MCPWriteStatusCreated
			group.primaryResult.Id = utils.Int64ToString(group.primaryNew.CategoryId)

			for j := 0; j < len(group.subResults); j++ {
				group.subResults[j].ParentId = group.primaryResult.Id

				if group.subNews[j] != nil {
					group.subResults[j].Status = MCPWriteStatusCreated
					group.subResults[j].Id = utils.Int64ToString(group.subNews[j].CategoryId)
				}
			}

			log.Infof(c, "[batch_create_transaction_categories_tool_handler.commit] user \"uid:%d\" has created category \"id:%d\" with %d sub-categories", uid, group.primaryNew.CategoryId, len(newSubCategories))
		} else if group.primaryExisting != nil {
			hasNewSubCategories := false

			for j := 0; j < len(group.subNews); j++ {
				if group.subNews[j] != nil {
					hasNewSubCategories = true
					break
				}
			}

			if !hasNewSubCategories {
				continue
			}

			var maxOrderId int32
			maxOrderId, err = categoryService.GetMaxSubCategoryDisplayOrder(c, uid, group.categoryType, group.primaryExisting.CategoryId)

			for j := 0; j < len(group.subNews); j++ {
				subCategory := group.subNews[j]

				if subCategory == nil {
					continue
				}

				if err == nil {
					maxOrderId++
					subCategory.DisplayOrder = maxOrderId
					err = categoryService.CreateCategory(c, subCategory)
				}

				if err != nil {
					log.Errorf(c, "[batch_create_transaction_categories_tool_handler.commit] failed to create sub-category of category \"id:%d\" for user \"uid:%d\", because %s", group.primaryExisting.CategoryId, uid, err.Error())
					mcpTransactionCategoryBatchMarkWriteFailed(group.subResults[j], err)

					for k := j + 1; k < len(group.subNews); k++ {
						if group.subNews[k] != nil {
							mcpTransactionCategoryBatchMarkSkipped(group.subResults[k])
						}
					}

					mcpTransactionCategoryBatchMarkNotCommitted(groups[i+1:])
					return committed, false
				}

				committed = true
				group.subResults[j].Status = MCPWriteStatusCreated
				group.subResults[j].Id = utils.Int64ToString(subCategory.CategoryId)
				log.Infof(c, "[batch_create_transaction_categories_tool_handler.commit] user \"uid:%d\" has created sub-category \"id:%d\" of category \"id:%d\"", uid, subCategory.CategoryId, group.primaryExisting.CategoryId)
			}
		}
	}

	return committed, true
}

// mcpTransactionCategoryPlanBatch validates the requested category hierarchy and matches it against the existing categories.
// It returns the plan groups, the per-item results (in request order) and whether all items are valid.
// No item will be saved if any item is invalid.
func mcpTransactionCategoryPlanBatch(uid int64, existingCategories []*models.TransactionCategory, items []*MCPBatchCreateTransactionCategoriesItem, skipExisting bool) ([]*mcpTransactionCategoryBatchPlanGroup, []*MCPBatchCreateTransactionCategoriesResult, bool) {
	categoryMap := make(map[int64]*models.TransactionCategory, len(existingCategories))

	for i := 0; i < len(existingCategories); i++ {
		categoryMap[existingCategories[i].CategoryId] = existingCategories[i]
	}

	resolver := mcpTransactionCategoryNewResolver(categoryMap)
	groups := make([]*mcpTransactionCategoryBatchPlanGroup, 0, len(items))
	results := make([]*MCPBatchCreateTransactionCategoriesResult, 0, len(items))
	requestedPrimaryIndexes := make(map[string]int)
	valid := true

	for i := 0; i < len(items); i++ {
		item := items[i]
		fieldPrefix := fmt.Sprintf("categories[%d].", i)
		group := &mcpTransactionCategoryBatchPlanGroup{
			primaryResult: &MCPBatchCreateTransactionCategoriesResult{
				Index: i,
			},
		}

		groups = append(groups, group)
		results = append(results, group.primaryResult)

		if item == nil {
			mcpTransactionCategoryBatchMarkFailed(group.primaryResult, NewMCPRequiredFieldError(fmt.Sprintf("categories[%d]", i)))
			valid = false
			continue
		}

		group.primaryResult.Name = item.Name
		group.primaryResult.Type = item.Type

		var icon int64
		var color string
		categoryType, err := parseMCPCategoryType(fieldPrefix+"type", item.Type)

		if err == nil {
			group.categoryType = categoryType
			err = mcpTransactionCategoryValidateFields(fieldPrefix, item.Name, item.Comment)
		}

		if err == nil {
			icon, err = mcpTransactionCategoryParseIcon(fieldPrefix+"icon", item.Icon)
		}

		if err == nil {
			color, err = mcpTransactionCategoryParseColor(fieldPrefix+"color", item.Color)
		}

		if err == nil {
			primaryKey := item.Type + "\x00" + strings.ToLower(strings.TrimSpace(item.Name))

			if duplicatedIndex, exists := requestedPrimaryIndexes[primaryKey]; exists {
				err = NewMCPInvalidFieldError(fieldPrefix+"name", item.Name, fmt.Sprintf("duplicate of categories[%d] (same name and type), merge the sub-categories into one item", duplicatedIndex))
			} else {
				requestedPrimaryIndexes[primaryKey] = i
			}
		}

		if err == nil && skipExisting {
			siblings := mcpTransactionCategoryFilter(existingCategories, categoryType, models.LevelOneTransactionCategoryParentId)
			matches := resolver.FindAllByName(siblings, item.Name)

			if len(matches) > 1 {
				err = NewMCPAmbiguousNameError(fieldPrefix+"name", "transaction category", item.Name, resolver.GetCandidates(matches))
			} else if len(matches) == 1 {
				group.primaryExisting = matches[0]
				group.primaryResult.Status = MCPWriteStatusExisting
				group.primaryResult.Id = utils.Int64ToString(matches[0].CategoryId)
			}
		}

		if err != nil {
			mcpTransactionCategoryBatchMarkFailed(group.primaryResult, err)
			valid = false
		} else if group.primaryExisting == nil {
			group.primaryNew = mcpTransactionCategoryNewModel(uid, item.Name, categoryType, models.LevelOneTransactionCategoryParentId, icon, color, item.Comment, 0)
			group.primaryResult.Status = MCPWriteStatusWouldBeCreated
		}

		primaryFailed := err != nil
		requestedSubIndexes := make(map[string]int)

		for j := 0; j < len(item.SubCategories); j++ {
			subItem := item.SubCategories[j]
			subIndex := j
			subFieldPrefix := fmt.Sprintf("%ssub_categories[%d].", fieldPrefix, j)
			subResult := &MCPBatchCreateTransactionCategoriesResult{
				Index:      i,
				SubIndex:   &subIndex,
				Type:       item.Type,
				ParentName: item.Name,
			}

			if group.primaryExisting != nil {
				subResult.ParentId = utils.Int64ToString(group.primaryExisting.CategoryId)
			}

			group.subResults = append(group.subResults, subResult)
			group.subNews = append(group.subNews, nil)
			results = append(results, subResult)

			if subItem == nil {
				mcpTransactionCategoryBatchMarkFailed(subResult, NewMCPRequiredFieldError(fmt.Sprintf("%ssub_categories[%d]", fieldPrefix, j)))
				valid = false
				continue
			}

			subResult.Name = subItem.Name

			var subIcon int64
			var subColor string
			subErr := mcpTransactionCategoryValidateFields(subFieldPrefix, subItem.Name, subItem.Comment)

			if subErr == nil {
				subIcon, subErr = mcpTransactionCategoryParseIcon(subFieldPrefix+"icon", subItem.Icon)
			}

			if subErr == nil {
				subColor, subErr = mcpTransactionCategoryParseColor(subFieldPrefix+"color", subItem.Color)
			}

			if subErr != nil {
				mcpTransactionCategoryBatchMarkFailed(subResult, subErr)
				valid = false
				continue
			}

			if primaryFailed {
				subResult.Status = MCPWriteStatusSkipped
				subResult.Error = NewMCPToolError(MCPToolErrorCodeNotAllowed, fieldPrefix+"name", fmt.Sprintf("skipped because the primary category categories[%d] is invalid", i))
				valid = false
				continue
			}

			subKey := strings.ToLower(strings.TrimSpace(subItem.Name))

			if duplicatedIndex, exists := requestedSubIndexes[subKey]; exists {
				mcpTransactionCategoryBatchMarkFailed(subResult, NewMCPInvalidFieldError(subFieldPrefix+"name", subItem.Name, fmt.Sprintf("duplicate of %ssub_categories[%d]", fieldPrefix, duplicatedIndex)))
				valid = false
				continue
			}

			requestedSubIndexes[subKey] = j

			if group.primaryExisting != nil && skipExisting {
				siblings := mcpTransactionCategoryFilter(existingCategories, categoryType, group.primaryExisting.CategoryId)
				matches := resolver.FindAllByName(siblings, subItem.Name)

				if len(matches) > 1 {
					mcpTransactionCategoryBatchMarkFailed(subResult, NewMCPAmbiguousNameError(subFieldPrefix+"name", "transaction category", subItem.Name, resolver.GetCandidates(matches)))
					valid = false
					continue
				} else if len(matches) == 1 {
					subResult.Status = MCPWriteStatusExisting
					subResult.Id = utils.Int64ToString(matches[0].CategoryId)
					continue
				}
			}

			parentCategoryId := int64(models.LevelOneTransactionCategoryParentId)

			if group.primaryExisting != nil {
				parentCategoryId = group.primaryExisting.CategoryId
			}

			group.subNews[j] = mcpTransactionCategoryNewModel(uid, subItem.Name, categoryType, parentCategoryId, subIcon, subColor, subItem.Comment, 0)
			subResult.Status = MCPWriteStatusWouldBeCreated
		}
	}

	return groups, results, valid
}

func mcpTransactionCategoryBatchMarkFailed(result *MCPBatchCreateTransactionCategoriesResult, err error) {
	result.Status = MCPWriteStatusFailed
	result.Id = ""

	if toolErr, ok := err.(*MCPToolError); ok {
		result.Error = toolErr
	} else {
		result.Error = NewMCPToolError(MCPToolErrorCodeInvalidValue, "", err.Error())
	}
}

func mcpTransactionCategoryBatchMarkWriteFailed(result *MCPBatchCreateTransactionCategoriesResult, err error) {
	result.Status = MCPWriteStatusFailed
	result.Id = ""
	result.Error = NewMCPToolError(mcpTransactionCategoryBatchErrorCodeWriteFailed, "", "failed to save: "+err.Error())
}

func mcpTransactionCategoryBatchMarkSkipped(result *MCPBatchCreateTransactionCategoriesResult) {
	result.Status = MCPWriteStatusSkipped
	result.Id = ""
	result.Error = NewMCPToolError(mcpTransactionCategoryBatchErrorCodeNotCommitted, "", "not saved because an earlier write in this batch failed")
}

func mcpTransactionCategoryBatchMarkNotCommitted(groups []*mcpTransactionCategoryBatchPlanGroup) {
	for i := 0; i < len(groups); i++ {
		group := groups[i]

		if group.primaryNew != nil {
			mcpTransactionCategoryBatchMarkSkipped(group.primaryResult)
		}

		for j := 0; j < len(group.subResults); j++ {
			if group.subNews[j] != nil {
				mcpTransactionCategoryBatchMarkSkipped(group.subResults[j])
			}
		}
	}
}

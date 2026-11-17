package mcp

import (
	"fmt"
)

// MCP tool error codes
const (
	MCPToolErrorCodeInvalidArguments = "invalid_arguments"
	MCPToolErrorCodeRequired         = "required"
	MCPToolErrorCodeInvalidValue     = "invalid_value"
	MCPToolErrorCodeTooLong          = "too_long"
	MCPToolErrorCodeTooMany          = "too_many"
	MCPToolErrorCodeTooLarge         = "too_large"
	MCPToolErrorCodeOutOfRange       = "out_of_range"
	MCPToolErrorCodeNotFound         = "not_found"
	MCPToolErrorCodeAmbiguousName    = "ambiguous_name"
	MCPToolErrorCodeNotSupported     = "not_supported"
	MCPToolErrorCodeNotAllowed       = "not_allowed"
)

// MCPEntityCandidate represents a candidate entity when a name cannot be resolved to exactly one entity
type MCPEntityCandidate struct {
	Id          string `json:"id" jsonschema_description:"Stable ID of the candidate"`
	Name        string `json:"name" jsonschema_description:"Name of the candidate"`
	Description string `json:"description,omitempty" jsonschema_description:"Additional information to tell the candidates apart (e.g. type, parent, hidden status)"`
}

// MCPToolError represents a user-correctable error of a MCP tool call, it is returned to the client as a tool result with isError set to true
type MCPToolError struct {
	Code       string                `json:"code"`
	Message    string                `json:"message"`
	Field      string                `json:"field,omitempty"`
	Limit      *int64                `json:"limit,omitempty"`
	Actual     *int64                `json:"actual,omitempty"`
	Value      string                `json:"value,omitempty"`
	Candidates []*MCPEntityCandidate `json:"candidates,omitempty"`
}

// MCPToolErrorResult represents the error result of a MCP tool call
type MCPToolErrorResult struct {
	Success bool          `json:"success"`
	Error   *MCPToolError `json:"error"`
}

// Error returns the error message
func (e *MCPToolError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}

	return e.Message
}

// NewMCPToolError returns a new MCP tool error
func NewMCPToolError(code string, field string, message string) *MCPToolError {
	return &MCPToolError{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

// NewMCPRequiredFieldError returns a new MCP tool error which indicates the field is required
func NewMCPRequiredFieldError(field string) *MCPToolError {
	return NewMCPToolError(MCPToolErrorCodeRequired, field, fmt.Sprintf("\"%s\" is required", field))
}

// NewMCPInvalidFieldError returns a new MCP tool error which indicates the field value is invalid
func NewMCPInvalidFieldError(field string, value string, message string) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeInvalidValue, field, message)
	err.Value = value
	return err
}

// NewMCPNotSupportedError returns a new MCP tool error which indicates the operation is not supported
func NewMCPNotSupportedError(field string, message string) *MCPToolError {
	return NewMCPToolError(MCPToolErrorCodeNotSupported, field, message)
}

// NewMCPFieldTooLongError returns a new MCP tool error which indicates the field value exceeds the maximum length
func NewMCPFieldTooLongError(field string, limit int, actual int) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeTooLong, field, fmt.Sprintf("\"%s\" is too long, the maximum length is %d characters but got %d characters", field, limit, actual))
	limitValue := int64(limit)
	actualValue := int64(actual)
	err.Limit = &limitValue
	err.Actual = &actualValue
	return err
}

// NewMCPTooManyItemsError returns a new MCP tool error which indicates the field has too many items
func NewMCPTooManyItemsError(field string, limit int, actual int) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeTooMany, field, fmt.Sprintf("\"%s\" has too many items, the maximum count is %d but got %d", field, limit, actual))
	limitValue := int64(limit)
	actualValue := int64(actual)
	err.Limit = &limitValue
	err.Actual = &actualValue
	return err
}

// NewMCPTooLargeError returns a new MCP tool error which indicates the size (in bytes) of the field content exceeds the maximum size
func NewMCPTooLargeError(field string, limit int64, actual int64) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeTooLarge, field, fmt.Sprintf("\"%s\" is too large, the maximum size is %d bytes but got %d bytes", field, limit, actual))
	err.Limit = &limit
	err.Actual = &actual
	return err
}

// NewMCPNotFoundError returns a new MCP tool error which indicates the entity is not found
func NewMCPNotFoundError(field string, entityName string, value string) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeNotFound, field, fmt.Sprintf("%s \"%s\" is not found", entityName, value))
	err.Value = value
	return err
}

// NewMCPAmbiguousNameError returns a new MCP tool error which indicates the name matches more than one entity
func NewMCPAmbiguousNameError(field string, entityName string, value string, candidates []*MCPEntityCandidate) *MCPToolError {
	err := NewMCPToolError(MCPToolErrorCodeAmbiguousName, field, fmt.Sprintf("%s name \"%s\" matches %d items, please specify the ID instead", entityName, value, len(candidates)))
	err.Value = value
	err.Candidates = candidates
	return err
}

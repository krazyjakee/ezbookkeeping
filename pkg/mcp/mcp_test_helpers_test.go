package mcp

import (
	"errors"
	"testing"
)

// mcpTestGetToolError returns the MCP tool error wrapped in err, and fails the test immediately if err is not an MCP tool error
func mcpTestGetToolError(t *testing.T, err error) *MCPToolError {
	t.Helper()
	var toolErr *MCPToolError

	if !errors.As(err, &toolErr) {
		t.Fatalf("expected MCPToolError, but got %v", err)
	}

	return toolErr
}

package mcp

import (
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// mcpTransactionReferenceServiceProvider is implemented by MCPAvailableServices which provide the transaction reference service
type mcpTransactionReferenceServiceProvider interface {
	GetTransactionReferenceService() *services.TransactionReferenceService
}

// mcpIdempotencyRecordServiceProvider is implemented by MCPAvailableServices which provide the mcp idempotency record service
type mcpIdempotencyRecordServiceProvider interface {
	GetMCPIdempotencyRecordService() *services.MCPIdempotencyRecordService
}

// mcpAuditLogServiceProvider is implemented by MCPAvailableServices which provide the mcp audit log service
type mcpAuditLogServiceProvider interface {
	GetMCPAuditLogService() *services.MCPAuditLogService
}

// mcpGetTransactionReferenceService returns the transaction reference service provided by the available services, or the default service instance
func mcpGetTransactionReferenceService(availableServices MCPAvailableServices) *services.TransactionReferenceService {
	if provider, ok := availableServices.(mcpTransactionReferenceServiceProvider); ok {
		if service := provider.GetTransactionReferenceService(); service != nil {
			return service
		}
	}

	return services.TransactionReferences
}

// mcpGetIdempotencyRecordService returns the mcp idempotency record service provided by the available services, or the default service instance
func mcpGetIdempotencyRecordService(availableServices MCPAvailableServices) *services.MCPIdempotencyRecordService {
	if provider, ok := availableServices.(mcpIdempotencyRecordServiceProvider); ok {
		if service := provider.GetMCPIdempotencyRecordService(); service != nil {
			return service
		}
	}

	return services.MCPIdempotencyRecords
}

// mcpGetAuditLogService returns the mcp audit log service provided by the available services, or the default service instance
func mcpGetAuditLogService(availableServices MCPAvailableServices) *services.MCPAuditLogService {
	if provider, ok := availableServices.(mcpAuditLogServiceProvider); ok {
		if service := provider.GetMCPAuditLogService(); service != nil {
			return service
		}
	}

	return services.MCPAuditLogs
}

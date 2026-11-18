import type { NameAndDisplayName } from '@/core/base.ts';

export const ALL_MCP_TOOLS: NameAndDisplayName[] = [
    {
        name: 'query_all_accounts',
        displayName:'Query All Accounts'
    },
    {
        name: 'query_all_accounts_balance',
        displayName:'Query All Account Balances'
    },
    {
        name: 'query_all_transaction_categories',
        displayName:'Query All Transaction Categories'
    },
    {
        name: 'query_all_transaction_tags',
        displayName:'Query All Transaction Tags'
    },
    {
        name: 'query_latest_exchange_rates',
        displayName:'Query Latest Exchange Rates'
    },
    {
        name: 'query_transactions',
        displayName:'Query Transactions'
    },
    {
        name: 'add_transaction',
        displayName:'Add Transaction'
    },
    {
        name: 'delete_transaction',
        displayName:'Delete Transaction'
    },
    {
        name: 'get_server_capabilities',
        displayName:'Get Server Capabilities'
    },
    {
        name: 'list_accounts',
        displayName:'List Accounts'
    },
    {
        name: 'get_account',
        displayName:'Get Account'
    },
    {
        name: 'create_account',
        displayName:'Create Account'
    },
    {
        name: 'update_account',
        displayName:'Update Account'
    },
    {
        name: 'hide_account',
        displayName:'Hide Account'
    },
    {
        name: 'delete_account',
        displayName:'Delete Account'
    },
    {
        name: 'list_transaction_categories',
        displayName:'List Transaction Categories'
    },
    {
        name: 'get_transaction_category',
        displayName:'Get Transaction Category'
    },
    {
        name: 'create_transaction_category',
        displayName:'Create Transaction Category'
    },
    {
        name: 'batch_create_transaction_categories',
        displayName:'Batch Create Transaction Categories'
    },
    {
        name: 'update_transaction_category',
        displayName:'Update Transaction Category'
    },
    {
        name: 'hide_transaction_category',
        displayName:'Hide Transaction Category'
    },
    {
        name: 'reorder_transaction_categories',
        displayName:'Reorder Transaction Categories'
    },
    {
        name: 'delete_transaction_category',
        displayName:'Delete Transaction Category'
    },
    {
        name: 'list_transaction_tags',
        displayName:'List Transaction Tags'
    },
    {
        name: 'create_transaction_tags',
        displayName:'Create Transaction Tags'
    },
    {
        name: 'update_transaction_tag',
        displayName:'Update Transaction Tag'
    },
    {
        name: 'reorder_transaction_tags',
        displayName:'Reorder Transaction Tags'
    },
    {
        name: 'delete_transaction_tag',
        displayName:'Delete Transaction Tag'
    },
    {
        name: 'create_transaction_tag_group',
        displayName:'Create Transaction Tag Group'
    },
    {
        name: 'update_transaction_tag_group',
        displayName:'Update Transaction Tag Group'
    },
    {
        name: 'reorder_transaction_tag_groups',
        displayName:'Reorder Transaction Tag Groups'
    },
    {
        name: 'delete_transaction_tag_group',
        displayName:'Delete Transaction Tag Group'
    },
    {
        name: 'get_transaction',
        displayName:'Get Transaction'
    },
    {
        name: 'update_transaction',
        displayName:'Update Transaction'
    },
    {
        name: 'list_transaction_templates',
        displayName:'List Transaction Templates'
    },
    {
        name: 'get_transaction_template',
        displayName:'Get Transaction Template'
    },
    {
        name: 'create_transaction_template',
        displayName:'Create Transaction Template'
    },
    {
        name: 'update_transaction_template',
        displayName:'Update Transaction Template'
    },
    {
        name: 'hide_transaction_template',
        displayName:'Hide Transaction Template'
    },
    {
        name: 'delete_transaction_template',
        displayName:'Delete Transaction Template'
    },
    {
        name: 'upload_transaction_attachment',
        displayName:'Upload Transaction Attachment'
    },
    {
        name: 'list_transaction_attachments',
        displayName:'List Transaction Attachments'
    },
    {
        name: 'get_transaction_attachment',
        displayName:'Get Transaction Attachment'
    },
    {
        name: 'download_transaction_attachment',
        displayName:'Download Transaction Attachment'
    },
    {
        name: 'detach_transaction_attachment',
        displayName:'Detach Transaction Attachment'
    },
    {
        name: 'delete_transaction_attachment',
        displayName:'Delete Transaction Attachment'
    },
    {
        name: 'query_account_balances_as_of',
        displayName:'Query Account Balances As Of Date'
    },
    {
        name: 'query_transaction_totals',
        displayName:'Query Transaction Totals'
    },
    {
        name: 'get_account_reconciliation_statement',
        displayName:'Get Account Reconciliation Statement'
    },
    {
        name: 'query_audit_log',
        displayName:'Query Audit Log'
    },
    {
        name: 'import_transactions',
        displayName:'Import Transactions'
    },
    {
        name: 'set_account_reconciled_time',
        displayName:'Set Account Reconciled Time'
    },
    {
        name: 'export_transactions',
        displayName:'Export Transactions'
    },
    {
        name: 'add_transaction_from_document',
        displayName:'Add Transaction From Document'
    }
];

export const ALL_MCP_TOOL_NAMES: string[] = ALL_MCP_TOOLS.map(tool => tool.name);

export const DEFAULT_ENABLED_MCP_TOOL_NAMES: string[] = [
    'query_all_accounts',
    'query_all_accounts_balance',
    'query_all_transaction_categories',
    'query_all_transaction_tags',
    'query_latest_exchange_rates',
    'query_transactions',
    'add_transaction',
    'add_transaction_from_document',
    'get_server_capabilities',
    'list_accounts',
    'get_account',
    'list_transaction_categories',
    'get_transaction_category',
    'list_transaction_tags',
    'get_transaction',
    'list_transaction_templates',
    'get_transaction_template',
    'list_transaction_attachments',
    'get_transaction_attachment',
    'download_transaction_attachment',
    'query_account_balances_as_of',
    'query_transaction_totals',
    'get_account_reconciliation_statement',
    'query_audit_log',
    'export_transactions'
]

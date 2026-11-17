---
title: MCP (Model Context Protocol)
---

# MCP (Model Context Protocol)

[Model Context Protocol](https://modelcontextprotocol.io) (MCP) is an open protocol developed by Anthropic that enables AI models to securely connect to external data sources and tools. It provides a standardized way for AI tools to access information and perform actions while maintaining security and user control.

With the MCP protocol, you can use your preferred AI tools to add transactions (e.g. create transactions using natural language or import multiple transactions from different formats), query transaction data (e.g. use your AI  tools to analyze historical transaction records), and more.

> [!CAUTION] CAUTION
> When using a third-party large language model service, be cautious with the MCP feature, because your private data may be sent to the third-party provider.

## Configuration

By default, ezBookkeeping does not enable the MCP server. You can enable it by updating the `enable_mcp` setting in the configuration file or by setting the environment variable. For details, see the [Configuration - MCP](/configuration/#mcp-model-context-protocol). After enabled, the "Generate Token" button will appear on "User Settings" -> "Security" page in the desktop version.

## Integration with AI Tools

The MCP service of ezBookkeeping requires a special token to access. You can go to "User Settings" -> "Security" and click the "Generate Token" button in desktop version, and then enter your password. Alternatively, you can generate the token by the `user-session-new` command in ezBookkeeping's CLI, for more details, see the [Command Line](/command_line/#userdata).

> [!NOTE] TIP
> If the IP address used to access the MCP server is fixed (e.g., within a home network), you can restrict specific IP addresses that can access the MCP server. For details, see the [Configuration - MCP](/configuration/#mcp-model-context-protocol).

Then, copy the following code into your AI tool's MCP server configuration to enable it to access and interact with ezBookkeeping data.

```json
{
    "mcpServers": {
        "ezbookkeeping-mcp": {
            "type": "streamable-http",
            "url": "http://{domain}/mcp",
            "headers": {
                "Authorization": "Bearer {token}"
            }
        }
    }
}
```

## Supported MCP Tool

### Conventions

The following conventions apply to all tools. Call `get_server_capabilities` to discover the limits and settings of your server.

* **Stable IDs**: all IDs are returned as strings and do not change. Tools accept IDs (e.g. `account_id`, `category_id`, `tag_ids`) as well as names (e.g. `account_name`, `category_name`, `tags`). IDs are preferred.
* **Name resolution**: an exact name match is preferred, otherwise a case-insensitive match is used. If a name matches more than one item, the tool fails with the `ambiguous_name` error and lists the candidates (ID, name and description); it never chooses one silently. Unknown names fail with `not_found` (unknown tags are no longer ignored silently).
* **Dry run**: write tools that support `dry_run` perform exactly the same validation as the actual write (including description length, edit scope and name resolution) but save nothing. Results contain `dry_run: true` and statuses such as `would_be_created` or `would_be_updated`; committed results contain the saved IDs and the persisted values re-read from the database.
* **Skip existing**: setup tools (`create_account`, `create_transaction_category`, `batch_create_transaction_categories`, `create_transaction_tags`, `create_transaction_tag_group`, `create_transaction_template`) support `skip_existing`. When enabled, an existing item with the same name (under the same parent / type / group) is reused and reported with status `existing`, so rerunning a setup does not create duplicates. It defaults to `true` for the batch tools (`batch_create_transaction_categories`, `create_transaction_tags`) and `create_transaction_tag_group`, and `false` otherwise.
* **Batch atomicity**: batch tools validate all items first. If any item is invalid, nothing is saved and every item reports its own result. When committing, results report the actual saved state of every item (`created`, `existing`, `failed`, ...), so a partially failed batch can be inspected and safely rerun with `skip_existing`.
* **Structured errors**: user-correctable errors are returned as a tool result with `isError: true` and the JSON body `{"success": false, "error": {"code", "message", "field", "limit", "actual", "value", "candidates"}}`. Error codes are `invalid_arguments`, `required`, `invalid_value`, `too_long`, `too_many`, `out_of_range`, `not_found`, `ambiguous_name`, `not_supported` and `not_allowed`. For example, an overlong description fails with `{"code": "too_long", "field": "comment", "limit": 1024, "actual": 1100}` during dry run as well.
* **Destructive operations**: delete tools only accept IDs and require `confirm: true`. Deletions are permanent and cannot be restored via MCP. They have the same restrictions as the web interface (e.g. an account or category in use cannot be deleted).
* **Liability balances**: for liability accounts (credit card and debt), `balance` is the signed accounting balance (negative means money owed) and `outstanding_balance` is the balance shown in the web interface (positive means money owed). For example, after a £120 personally funded expense recorded against a director's loan debt account, `balance` is `-120.00` and `outstanding_balance` is `120.00`.
* **Tool annotations**: every tool declares `readOnlyHint`, `destructiveHint`, `idempotentHint` and `openWorldHint`. Query tools are read-only; create tools are non-destructive; update, hide, reorder and delete tools are destructive.
* **Token permissions**: tokens generated with "Enable All MCP Tools" can use all tools, including the tools added in later versions. Tokens generated with a selected tool list can only use the selected tools.

### Get server capabilities

**MCP Tool Name**

`get_server_capabilities`

**Request Parameters**

None

**Response Parameters**

| Field | Type | Description |
| --- | --- | --- |
| `server` | `object` | Application name, version, commit hash and supported MCP protocol version |
| `limits` | `object` | Maximum lengths of transaction description, account / category / tag / tag group / template names and descriptions, maximum tags and attachments per transaction, transaction amount range, maximum items of one batch call |
| `attachments` | `object` | Whether attachments are enabled, maximum file size in bytes, maximum count per transaction, supported file types (extension and content type), and whether uploading via MCP is supported (currently `false`) |
| `scheduled_transactions` | `object` | Whether scheduled transactions are enabled on this server |
| `current_user` | `object` | Default currency, transaction edit scope (the range of transaction time which can be created, modified or deleted) and whether last reconciled time is used |
| `features` | `object` | Textual description of dry run, skip existing, name resolution, structured errors, destructive operations, batch atomicity, liability balances and query order semantics |
| `supported_currencies` | `string[]` | Supported currency codes (ISO 4217) |
| `values` | `object` | Accepted account categories, transaction types and category types |

### Add transaction

**MCP Tool Name**

`add_transaction`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `type` | `string` | Required | Transaction type (income, expense, transfer) |
| `time` | `string` | Required | Transaction time in RFC 3339 format (e.g. 2023-01-01T12:00:00Z). The UTC offset is used as the transaction timezone |
| `category_id` | `string` | Optional | Secondary category ID (either `category_id` or `category_name` is required) |
| `category_name` | `string` | Optional | Secondary category name |
| `account_id` | `string` | Optional | Account ID (either `account_id` or `account_name` is required) |
| `account_name` | `string` | Optional | Account name |
| `amount` | `string` | Required | Transaction amount (e.g. 12.34) |
| `destination_account_id` | `string` | Optional | Destination account ID for transfer transactions |
| `destination_account_name` | `string` | Optional | Destination account name for transfer transactions |
| `destination_amount` | `string` | Optional | Destination amount for transfer transactions |
| `tag_ids` | `string[]` | Optional | IDs of tags associated with the transaction |
| `tags` | `string[]` | Optional | Names of tags associated with the transaction (maximum 10 tags allowed in total) |
| `comment` | `string` | Optional | Transaction description (see `limits.transaction_description_max_length` of `get_server_capabilities`) |
| `geo_location` | `object` | Optional | Geographic location (`latitude`, `longitude`) |
| `dry_run` | `boolean` | Optional | If `true`, the transaction will not be saved, only validated |

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `success` | `boolean` | Required | Indicates whether this operation is successful |
| `dry_run` | `boolean` | Optional | Indicates whether this operation is a dry run (transaction not added actually) |
| `committed` | `boolean` | Optional | Indicates whether the transaction has been saved |
| `transaction_id` | `string` | Optional | ID of the saved transaction (only when committed) |
| `transaction` | `MCPTransactionDetail` | Optional | Persisted transaction (same structure as `get_transaction`) |
| `account_balance` | `string` | Optional | Account balance (or outstanding balance for debt accounts) after the transaction |
| `destination_account_balance` | `string` | Optional | Destination account balance (or outstanding balance for debt accounts) after the transaction (only for transfer transactions) |

The response also contains the signed accounting balances of the affected accounts. See the tool output schema for details.

### Query transactions

**MCP Tool Name**

`query_transactions`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `start_time` | `string` | Required | Start time for the query in RFC 3339 format (e.g. 2023-01-01T12:00:00Z) |
| `end_time` | `string` | Required | End time for the query in RFC 3339 format or (e.g. 2023-01-01T12:00:00Z) |
| `type` | `string` | Optional | Transaction type to filter by (income, expense, transfer, balance_modification) |
| `category_id` | `string` | Optional | Primary or secondary category ID to filter transactions by |
| `category_name` | `string` | Optional | Primary or secondary category name to filter transactions by |
| `account_id` | `string` | Optional | Account ID to filter transactions by (a parent account includes its sub-accounts) |
| `account_name` | `string` | Optional | Account name to filter transactions by |
| `tag_ids` | `string[]` | Optional | Tag IDs to filter transactions by |
| `tag_names` | `string[]` | Optional | Tag names to filter transactions by |
| `tag_filter_mode` | `string` | Optional | How the tag filter is applied (has_any, has_all, not_has_any, not_has_all) |
| `no_tags` | `boolean` | Optional | Only return transactions without tags |
| `amount_filter` | `string` | Optional | Amount filter (e.g. `gt:12.34`, `lt:12.34`, `eq:12.34`, `ne:12.34`, `bt:10:20`, `nb:10:20`) |
| `has_attachments` | `boolean` | Optional | Only return transactions which have attachments |
| `keyword` | `string` | Optional | Keyword to search in transaction description |
| `match_mode` | `string` | Optional | Match mode for keyword search (leave empty for database default setting, ignore_case for case-insensitive search) |
| `count` | `number` | Optional | Maximum number of results to return (default: 100) |
| `page` | `number` | Optional | Page number for pagination (default: 1) |
| `response_fields` | `string` | Optional | Comma-separated list of optional fields to include in the response (leave empty for all fields, available fields: time, currency, category_name, account_name, comment). The transaction `id` is always returned |

Transactions are always ordered by transaction time descending.

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `total_count` | `number` | Required | Total number of transactions matching the query |
| `current_page` | `number` | Required | Current page number of the results |
| `total_page` | `number` | Required | Total number of pages available for the query, calculated based on total_count and count |
| `transactions` | `MCPTransactionInfo[]` | Required | List of transactions matching the query |

**MCPTransactionInfo**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | `string` | Required | Transaction ID |
| `time` | `string` | Optional | Time of the transaction in RFC 3339 format (e.g. 2023-01-01T12:00:00Z) |
| `type` | `string` | Required | Transaction type (income, expense, transfer, balance_modification) |
| `amount` | `string` | Required | Amount of the transaction in the specified currency |
| `currency` | `string` | Optional | Currency code of the transaction (e.g. USD, EUR) |
| `category_name` | `string` | Optional | Secondary category name for the transaction |
| `account_name` | `string` | Optional | Account name for the transaction |
| `destination_amount` | `string` | Optional | Destination amount for transfer transactions |
| `destination_currency` | `string` | Optional | Currency code of the destination amount for transfer transactions |
| `destination_account_name` | `string` | Optional | Destination account name for transfer transactions |
| `comment` | `string` | Optional | Description of the transaction |
| `attachment_count` | `number` | Required | Count of attachments of the transaction |

Each transaction also contains `account_id`, `category_id`, `destination_account_id`, tags and geographic location. See the tool output schema for details.

### Get transaction

**MCP Tool Name**

`get_transaction`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `transaction_id` | `string` | Required | Transaction ID (the ID of either side of a transfer can be used) |

**Response Parameters**

Returns the full transaction (`MCPTransactionDetail`): ID, type, time and UTC offset, amount, currency, account / destination account (ID and name), category (ID, name and primary category), tags (ID and name), description, geographic location, attachments (ID, extension and content type) and their count, created / updated time, and whether the transaction can be edited under the current user's transaction edit scope.

### Update transaction

**MCP Tool Name**

`update_transaction`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `transaction_id` | `string` | Required | Transaction ID |
| `type` | `string` | Optional | New transaction type (income, expense, transfer) |
| `time` | `string` | Optional | New transaction time in RFC 3339 format, the UTC offset is used as the transaction timezone |
| `amount` | `string` | Optional | New amount |
| `account_id` / `account_name` | `string` | Optional | New account |
| `destination_account_id` / `destination_account_name` | `string` | Optional | New destination account for transfer transactions |
| `destination_amount` | `string` | Optional | New destination amount for transfer transactions |
| `category_id` / `category_name` | `string` | Optional | New secondary category |
| `tag_ids` / `tags` | `string[]` | Optional | Replace all tags of the transaction with these tags |
| `clear_tags` | `boolean` | Optional | Remove all tags of the transaction |
| `comment` | `string` | Optional | New description |
| `geo_location` | `object` | Optional | New geographic location (`latitude`, `longitude`) |
| `clear_geo_location` | `boolean` | Optional | Remove the geographic location |
| `dry_run` | `boolean` | Optional | If `true`, only validate the changes |

Only provided fields are changed. Both the original and the new transaction must be within the user's transaction edit scope. Attachments are not changed.

**Response Parameters**

Returns `status` (`updated`, `would_be_updated` or `unchanged`), `dry_run`, `changed_fields`, the persisted transaction and the balances of the affected accounts.

### Delete transaction

**MCP Tool Name**

`delete_transaction`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `transaction_id` | `string` | Required | Transaction ID |
| `confirm` | `boolean` | Required | Must be `true` to confirm the permanent deletion |

The transaction (both sides for transfers) and its attachments are deleted permanently, there is no restore. The transaction must be within the user's transaction edit scope.

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `success` | `boolean` | Required | Indicates whether this operation is successful |
| `deleted_transaction_id` | `string` | Required | ID of the deleted transaction |
| `permanent` | `boolean` | Required | Always `true` |

### Accounts

| MCP Tool Name | Request Parameters | Description |
| --- | --- | --- |
| `list_accounts` | `include_hidden`, `category` | List accounts with IDs, category, asset / liability type, currency, visibility, parent / sub-account hierarchy, display order and balances (`balance`, `display_balance`, `outstanding_balance` for liability accounts) |
| `get_account` | `account_id` or `account_name` | Get an account (main or sub-account) with its sub-accounts |
| `create_account` | `name`, `category`, `currency`, `comment`, `icon`, `color`, `opening_balance` or `outstanding_balance` (liability accounts only), `opening_balance_time`, `credit_card_statement_date`, `credit_card_limit`, `sub_accounts[]`, `skip_existing`, `dry_run` | Create an account. An account with `sub_accounts` is created as a parent account whose sub-accounts hold the balances. A non-zero opening balance requires `opening_balance_time` (RFC 3339), and creates a balance modification transaction at that time |
| `update_account` | `account_id` or `account_name`, `name`, `category`, `currency`, `comment`, `icon`, `color`, `credit_card_statement_date`, `credit_card_limit`, `hidden`, `add_sub_accounts[]`, `dry_run` | Update an account. Changing the balance or currency of an existing account is not supported, as in the web interface |
| `hide_account` | `account_id` or `account_name`, `hidden` | Hide (archive) or unhide an account |
| `delete_account` | `account_id`, `confirm` | Delete an account (or a sub-account) permanently, with the same restrictions as the web interface |

Account categories are `cash`, `checking_account`, `savings_account`, `credit_card`, `virtual`, `debt`, `receivables`, `certificate_of_deposit` and `investment`. `credit_card` and `debt` are liability accounts. For example, a director's loan account can be created with `{"name": "Director's Loan", "category": "debt", "currency": "GBP", "skip_existing": true}`.

### Transaction categories

| MCP Tool Name | Request Parameters | Description |
| --- | --- | --- |
| `list_transaction_categories` | `type`, `include_hidden` | List primary categories with their secondary categories, IDs, type, visibility and display order |
| `get_transaction_category` | `category_id` or `category_name`, `type`, `parent_category_id`, `parent_category_name` | Get a category, `type` and parent can be used to disambiguate names |
| `create_transaction_category` | `name`, `type`, `parent_category_id` or `parent_category_name`, `icon`, `color`, `comment`, `skip_existing`, `dry_run` | Create a primary category, or a secondary category when a parent is specified |
| `batch_create_transaction_categories` | `categories[]` (`name`, `type`, `icon`, `color`, `comment`, `sub_categories[]`), `skip_existing` (default `true`), `dry_run` | Create a category hierarchy in one call, with per-item results |
| `update_transaction_category` | `category_id` or `category_name`, `type`, `parent_category_name`, `name`, `icon`, `color`, `comment`, `hidden`, `new_parent_category_id` or `new_parent_category_name`, `dry_run` | Update a category, or move a secondary category to another primary category of the same type |
| `hide_transaction_category` | `category_id` or `category_name`, `type`, `parent_category_name`, `hidden` | Hide or unhide a category |
| `reorder_transaction_categories` | `category_ids` | Reorder sibling categories (same type and parent) in the given order |
| `delete_transaction_category` | `category_id`, `confirm` | Delete a category permanently, a category in use cannot be deleted |

### Transaction tags and tag groups

| MCP Tool Name | Request Parameters | Description |
| --- | --- | --- |
| `list_transaction_tags` | `include_hidden` | List tag groups and tags with IDs, group, visibility and display order |
| `create_transaction_tags` | `tags[]` (`name`, `tag_group_id` or `tag_group_name`), `skip_existing` (default `true`), `dry_run` | Create tags in one call, with per-item results |
| `update_transaction_tag` | `tag_id` or `tag_name`, `name`, `tag_group_id` (`"0"` for no group) or `tag_group_name`, `hidden`, `dry_run` | Rename a tag, move it to another group, or hide / unhide it |
| `reorder_transaction_tags` | `tag_ids`, `dry_run` | Reorder tags of the same tag group |
| `delete_transaction_tag` | `tag_id`, `confirm` | Delete a tag permanently |
| `create_transaction_tag_group` | `name`, `skip_existing` (default `true`), `dry_run` | Create a tag group |
| `update_transaction_tag_group` | `tag_group_id` or `tag_group_name`, `name`, `dry_run` | Rename a tag group |
| `reorder_transaction_tag_groups` | `tag_group_ids`, `dry_run` | Reorder tag groups |
| `delete_transaction_tag_group` | `tag_group_id`, `confirm` | Delete a tag group permanently, with the same restrictions as the web interface |

### Transaction templates and scheduled transactions

| MCP Tool Name | Request Parameters | Description |
| --- | --- | --- |
| `list_transaction_templates` | `template_type` (`normal` or `scheduled`), `include_hidden` | List templates or scheduled transactions |
| `get_transaction_template` | `template_id` or `template_name`, `template_type` | Get a template or scheduled transaction |
| `create_transaction_template` | `template_type`, `name`, `type`, `category_id` / `category_name`, `account_id` / `account_name`, `destination_account_id` / `destination_account_name`, `amount`, `destination_amount`, `hide_amount`, `tag_ids`, `tag_names`, `comment`, `frequency_type`, `frequency`, `start_date`, `end_date`, `timezone`, `skip_existing`, `dry_run` | Create a template or scheduled transaction |
| `update_transaction_template` | `template_id` or `template_name`, `template_type`, any field of `create_transaction_template`, `enabled`, `dry_run` | Update a template, or enable / disable a scheduled transaction |
| `hide_transaction_template` | `template_id` or `template_name`, `template_type`, `hidden` | Hide or unhide a template. Hiding does not stop a scheduled transaction, use `update_transaction_template` with `enabled: false` instead |
| `delete_transaction_template` | `template_id`, `confirm` | Delete a template or scheduled transaction permanently |

Scheduled transactions can only be created or modified when scheduled transactions are enabled on the server (`enable_scheduled_transaction`). `frequency_type` is one of `disabled`, `daily`, `weekly`, `monthly`, `yearly` and `every_n_days`; `start_date` and `end_date` use the `YYYY-MM-DD` format and `timezone` is an IANA time zone name. Scheduled transactions are returned with `status` (`disabled`, `not_started`, `active` or `ended`), `next_occurrence`, and `remaining_occurrences` when an end date is set.

### Query all account names

**MCP Tool Name**

`query_all_accounts`

**Request Parameters**

None

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `cashAccounts` | `string[]` | Optional | List of cash account names |
| `checkingAccounts` | `string[]` | Optional | List of checking account names |
| `savingsAccounts` | `string[]` | Optional | List of savings account names |
| `creditCardAccounts` | `string[]` | Optional | List of credit card account names |
| `virtualAccounts` | `string[]` | Optional | List of virtual account names |
| `debtAccounts` | `string[]` | Optional | List of debt account names |
| `receivableAccounts` | `string[]` | Optional | List of receivable account names |
| `certificateOfDepositAccounts` | `string[]` | Optional | List of certificate of deposit account names |
| `investmentAccounts` | `string[]` | Optional | List of investment account names |

### Query all account balances

**MCP Tool Name**

`query_all_accounts_balance`

**Request Parameters**

None

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `cashAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of cash account balances |
| `checkingAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of checking account balances |
| `savingsAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of savings account balances |
| `creditCardAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of credit card account outstanding balances |
| `virtualAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of virtual account balances |
| `debtAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of debt account outstanding balances |
| `receivableAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of receivable account balances |
| `certificateOfDepositAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of certificate of deposit account balances |
| `investmentAccounts` | `MCPAccountBalanceInfo[]` | Optional | List of investment account balances |

**MCPAccountBalanceInfo**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | `string` | Required | Account name |
| `type` | `string` | Required | Account type (asset or liability) |
| `balance` | `string` | Optional | Current balance of the account |
| `outstandingBalance` | `string` | Optional | Current outstanding balance of the account (positive value indicates amount owed) |
| `currency` | `string` | Required | Currency code of the account (e.g. USD, EUR) |

### Query all transaction category names

**MCP Tool Name**

`query_all_transaction_categories`

**Request Parameters**

None

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `incomeCategories` | `Map<string, string[]>` | Required | List of income categories, field key is the primary category name, field value is the list of secondary category names |
| `expenseCategories` | `Map<string, string[]>` | Required | List of expense categories, field key is the primary category name, field value is the list of secondary category names |
| `transferCategories` | `Map<string, string[]>` | Required | List of transfer categories, field key is the primary category name, field value is the list of secondary category names |

### Query all transaction tag names

**MCP Tool Name**

`query_all_transaction_tags`

**Request Parameters**

None

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `tags` | `string[]` | Required | List of transaction tags |

### Query latest exchange rates

**MCP Tool Name**

`query_latest_exchange_rates`

**Request Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currencies` | `string` | Required | Comma-separated list of currencies to query exchange rates for (e.g. USD,CNY,EUR) |

**Response Parameters**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `base_currency` | `string` | Required | Base currency code (e.g. USD) |
| `update_time` | `string` | Required | Last update time of the exchange rates in RFC 3339 format (e.g. "2023-01-01T12:00:00Z") |
| `rates` | `MCPQueryExchangeRateInfo[]` | Required | Exchange rates for the specified currencies |

**MCPQueryExchangeRateInfo**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | Required | Currency code (e.g. USD) |
| `rate_to_base` | `string` | Required | The amount of the base currency that can be obtained for 1 unit of this currency |

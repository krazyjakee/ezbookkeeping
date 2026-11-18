<template>
    <main-page-layout nav-items-class="account-category-tabs">
        <template #nav-items>
            <li class="ebk-mgmt-side-item">
                <section class="ebk-panel ebk-mgmt-side-panel" aria-labelledby="account-list-summary-title">
                    <header class="ebk-panel-header-strip">
                        <span id="account-list-summary-title">{{ tt('Asset Summary') }}</span>
                    </header>
                    <div class="ebk-mgmt-side-panel-body">
                        <div class="ebk-mgmt-kv">
                            <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Net assets') }}</span>
                            <span class="ebk-mgmt-kv-value" v-if="!loading || allAccountCount > 0">{{ netAssets }}</span>
                            <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="120" :loading="true" v-else></v-skeleton-loader>
                        </div>
                        <div class="ebk-mgmt-kv">
                            <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Total assets') }}</span>
                            <span class="ebk-mgmt-kv-value" v-if="!loading || allAccountCount > 0">{{ totalAssets }}</span>
                            <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="120" :loading="true" v-else></v-skeleton-loader>
                        </div>
                        <div class="ebk-mgmt-kv">
                            <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Total liabilities') }}</span>
                            <span class="ebk-mgmt-kv-value" v-if="!loading || allAccountCount > 0">{{ totalLiabilities }}</span>
                            <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="120" :loading="true" v-else></v-skeleton-loader>
                        </div>
                    </div>
                </section>
            </li>

            <li class="nav-section-title">
                <div class="title-wrapper">
                    <span class="title-text">{{ tt('Account Category') }}</span>
                </div>
            </li>

            <li class="nav-link"
                :key="accountCategory.type"
                v-for="accountCategory in AccountCategory.values(customAccountCategoryOrder)"
                v-show="!hideAccountCategoriesWithoutAccounts || (allCategorizedAccountsMap[accountCategory.type] && allCategorizedAccountsMap[accountCategory.type]!.accounts.length > 0)">
                <a class="d-flex align-center cursor-pointer" role="button" tabindex="0"
                   :class="{ 'router-link-active router-link-exact-active': activeAccountCategoryType === accountCategory.type, 'disabled': loading }"
                   :aria-current="activeAccountCategoryType === accountCategory.type ? 'true' : undefined"
                   @click="activeAccountCategoryType = accountCategory.type"
                   @keydown.enter.prevent="activeAccountCategoryType = accountCategory.type"
                   @keydown.space.prevent="activeAccountCategoryType = accountCategory.type">
                    <ItemIcon icon-type="account" :icon-id="accountCategory.defaultAccountIconId" />
                    <div class="nav-item-title account-category-nav-title">
                        <span class="text-truncate">{{ tt(accountCategory.name) }}</span>
                        <small class="account-category-nav-balance text-truncate" v-if="!loading || allAccountCount > 0">{{ accountCategoryTotalBalance(accountCategory, false) }}</small>
                        <v-skeleton-loader class="ebk-mgmt-skeleton" width="90" height="14" type="text" :loading="true" v-else></v-skeleton-loader>
                    </div>
                </a>
            </li>
        </template>

        <template #content>
            <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                <v-window-item value="accountPage">
                    <v-card class="ebk-mgmt-panel account-list-panel" :aria-busy="loading ? 'true' : 'false'">
                        <header class="ebk-panel-header-strip ebk-mgmt-header">
                            <h2 class="ebk-mgmt-header-label">{{ activeAccountCategory ? tt(activeAccountCategory.name) : tt('Account List') }}</h2>
                            <span class="ebk-mgmt-header-meta ebk-numeric" v-if="!loading && activeAccountCategoryVisibleAccountCount > 0">{{ formatNumberToLocalizedNumerals(activeAccountCategoryVisibleAccountCount) }}</span>
                            <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                                <v-btn color="primary" variant="tonal"
                                       :disabled="loading" @click="saveSortResult"
                                       v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                                <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                                       :disabled="loading" @click="add">{{ tt('Add') }}</v-btn>
                                <v-btn density="comfortable" color="default" variant="text"
                                       :aria-label="tt('Refresh')" :icon="true" :loading="loading" @click="reload(true)">
                                    <template #loader>
                                        <v-progress-circular indeterminate size="20"/>
                                    </template>
                                    <v-icon :icon="mdiRefresh" size="22" />
                                    <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                                </v-btn>
                                <v-btn density="comfortable" color="default" variant="text"
                                       :aria-label="tt('More')" :disabled="loading" :icon="true">
                                    <v-icon :icon="mdiDotsVertical" />
                                    <v-tooltip activator="parent">{{ tt('More') }}</v-tooltip>
                                    <v-menu activator="parent">
                                        <v-list>
                                            <v-list-item :prepend-icon="mdiEyeOutline"
                                                         :title="tt('Show Hidden Accounts')"
                                                         v-if="!showHidden" @click="showHidden = true"></v-list-item>
                                            <v-list-item :prepend-icon="mdiEyeOffOutline"
                                                         :title="tt('Hide Hidden Accounts')"
                                                         v-if="showHidden" @click="showHidden = false"></v-list-item>
                                            <v-divider class="my-2" v-if="hasAnyVisibleAccount"/>
                                            <v-list-item :prepend-icon="mdiCalculatorVariantOutline"
                                                         :title="tt('Set Accounts Included in Total')"
                                                         v-if="hasAnyVisibleAccount" @click="showAccountsIncludedInTotalDialog = true"></v-list-item>
                                        </v-list>
                                    </v-menu>
                                </v-btn>
                            </div>
                        </header>

                        <div class="ebk-mgmt-summary">
                            <span class="ebk-mgmt-summary-item">
                                <span class="ebk-mgmt-summary-label ebk-technical-heading">{{ activeAccountCategory?.isLiability ? tt('Outstanding Balance') : tt('Balance') }}</span>
                                <v-skeleton-loader class="ebk-mgmt-skeleton" width="120" type="text" :loading="true" v-if="loading && activeAccountCategory && !hasAccount(activeAccountCategory)"></v-skeleton-loader>
                                <span class="ebk-mgmt-summary-value" v-else>{{ activeAccountCategoryTotalBalance }}</span>
                            </span>
                            <template v-if="activeAccountCategory?.type === AccountCategory.CreditCard.type && activeCreditCardTotalAvailabeCredit && !(loading && !hasAccount(activeAccountCategory))">
                                <span class="ebk-mgmt-summary-separator" aria-hidden="true">·</span>
                                <span class="ebk-mgmt-summary-item">
                                    <span class="ebk-mgmt-summary-label ebk-technical-heading">{{ tt('Available Credit') }}</span>
                                    <span class="ebk-mgmt-summary-value">{{ activeCreditCardTotalAvailabeCredit }}</span>
                                </span>
                            </template>
                            <v-btn density="comfortable" color="default" variant="text" size="small"
                                   :aria-label="showAccountBalance ? tt('Hide Account Balance') : tt('Show Account Balance')"
                                   :aria-pressed="showAccountBalance ? 'false' : 'true'"
                                   :icon="true" :disabled="loading"
                                   @click="showAccountBalance = !showAccountBalance">
                                <v-icon :icon="showAccountBalance ? mdiEyeOffOutline : mdiEyeOutline" size="20" />
                                <v-tooltip activator="parent">{{ showAccountBalance ? tt('Hide Account Balance') : tt('Show Account Balance') }}</v-tooltip>
                            </v-btn>
                        </div>

                        <div class="account-list" role="table" :aria-label="activeAccountCategory ? tt(activeAccountCategory.name) : tt('Account List')">
                            <div class="account-list-head" role="row">
                                <span class="account-list-cell-name" role="columnheader">{{ tt('Account Name') }}</span>
                                <span class="account-list-cell-amount" role="columnheader">{{ activeAccountCategory?.isLiability ? tt('Outstanding Balance') : tt('Balance') }}</span>
                                <span class="account-list-cell-operation" role="columnheader"><span class="ebk-visually-hidden">{{ tt('Operation') }}</span></span>
                            </div>

                            <div class="account-list-skeleton" aria-hidden="true" v-if="loading && activeAccountCategory && !hasAccount(activeAccountCategory)">
                                <div class="account-list-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                                    <div class="account-list-cell-name">
                                        <span class="ebk-mgmt-drag disabled"></span>
                                        <span class="ebk-mgmt-icon-tile ebk-mgmt-icon-tile--skeleton"></span>
                                        <v-skeleton-loader class="ebk-mgmt-skeleton ms-3" type="text" width="160" :loading="true"></v-skeleton-loader>
                                    </div>
                                    <div class="account-list-cell-amount">
                                        <v-skeleton-loader class="ebk-mgmt-skeleton ms-auto" type="text" width="100" :loading="true"></v-skeleton-loader>
                                    </div>
                                    <div class="account-list-cell-operation"></div>
                                </div>
                            </div>

                            <div class="account-list-state" role="row" v-if="!loading && activeAccountCategory && !hasAccount(activeAccountCategory)">
                                <div role="cell">
                                    <state-panel type="error" :title="tt('Unable to retrieve account list')" v-if="loadError">
                                        <template #actions>
                                            <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload(true)">{{ tt('Retry') }}</v-btn>
                                        </template>
                                    </state-panel>
                                    <state-panel type="empty" :title="tt('No available account')" v-else>
                                        <template #actions>
                                            <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus" @click="add">{{ tt('Add') }}</v-btn>
                                        </template>
                                    </state-panel>
                                </div>
                            </div>

                            <draggable-list
                                class="account-list-body"
                                role="rowgroup"
                                item-key="id"
                                handle=".drag-handle"
                                ghost-class="dragging-item"
                                :disabled="loading || activeAccountCategoryVisibleAccountCount <= 1"
                                :list="allCategorizedAccountsMap[activeAccountCategory.type]!.accounts"
                                v-if="activeAccountCategory && allCategorizedAccountsMap[activeAccountCategory.type] && allCategorizedAccountsMap[activeAccountCategory.type]!.accounts && allCategorizedAccountsMap[activeAccountCategory.type]!.accounts.length"
                                @change="onMove"
                            >
                                <template #item="{ element }">
                                    <div class="account-list-group ebk-mgmt-card">
                                        <template v-if="showHidden || !element.hidden">
                                            <div class="account-list-row ebk-mgmt-row" role="row"
                                                 :class="{ 'ebk-mgmt-row--hidden': element.hidden }">
                                                <div class="account-list-cell-name" role="cell">
                                                    <span class="ebk-mgmt-drag" :class="{ 'drag-handle': !loading && activeAccountCategoryVisibleAccountCount > 1, 'disabled': loading || activeAccountCategoryVisibleAccountCount <= 1 }">
                                                        <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                                        <v-tooltip activator="parent" v-if="!loading && activeAccountCategoryVisibleAccountCount > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                                    </span>
                                                    <span class="ebk-mgmt-icon-tile ebk-mgmt-dimmable" aria-hidden="true">
                                                        <ItemIcon size="1.25rem" :icon-type="getAccountIconType(element.iconType)" :icon-id="element.icon" :color="element.color" />
                                                    </span>
                                                    <div class="ebk-mgmt-name">
                                                        <div class="ebk-mgmt-name-line">
                                                            <span class="ebk-mgmt-name-text ebk-mgmt-dimmable">{{ element.name }}</span>
                                                            <span class="ebk-mgmt-hidden-chip" v-if="element.hidden">
                                                                <v-icon :icon="mdiEyeOffOutline" size="12" />
                                                                {{ tt('Hidden') }}
                                                            </span>
                                                        </div>
                                                        <span class="ebk-mgmt-meta ebk-mgmt-dimmable">
                                                            <span>{{ accountCurrency(element) }}</span>
                                                            <template v-if="element.comment">
                                                                <span aria-hidden="true"> · </span>
                                                                <span>{{ element.comment }}</span>
                                                            </template>
                                                        </span>
                                                        <v-tooltip activator="parent" location="bottom start" open-delay="600" v-if="element.comment">{{ element.comment }}</v-tooltip>
                                                    </div>
                                                    <v-btn class="ms-2" density="comfortable" size="small" color="default" variant="text"
                                                           :aria-label="isSubAccountsExpanded(element) ? tt('Collapse Sub-accounts') : tt('Expand Sub-accounts')"
                                                           :aria-expanded="isSubAccountsExpanded(element) ? 'true' : 'false'"
                                                           :icon="true" :disabled="loading"
                                                           v-if="element.type === AccountType.MultiSubAccounts.type"
                                                           @click="toggleSubAccounts(element)">
                                                        <v-icon :icon="isSubAccountsExpanded(element) ? mdiChevronUp : mdiChevronDown" size="20" />
                                                        <v-tooltip activator="parent">{{ isSubAccountsExpanded(element) ? tt('Collapse Sub-accounts') : tt('Expand Sub-accounts') }}</v-tooltip>
                                                    </v-btn>
                                                </div>
                                                <div class="account-list-cell-amount" role="cell">
                                                    <span class="ebk-mgmt-amount account-balance">{{ accountBalance(element, '', showAccountBalance) }}</span>
                                                    <span class="ebk-mgmt-amount-meta" v-if="canShowAvailableCredit(element) && accountAvailableCredit(element, showAccountBalance)">
                                                        {{ tt('pageSpecific.accountListPage.availableBalance') }} {{ accountAvailableCredit(element, showAccountBalance) }}
                                                    </span>
                                                </div>
                                                <div class="account-list-cell-operation" role="cell">
                                                    <div class="ebk-mgmt-actions">
                                                        <template v-if="element.type === AccountType.SingleAccount.type">
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Transaction List')" :disabled="loading"
                                                                   :to="`/transaction/list?accountIds=${element.id}`">
                                                                <v-icon :icon="mdiListBoxOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Transaction List') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Reconciliation Statement')" :disabled="loading"
                                                                   @click="showReconciliationStatementDialog(element)">
                                                                <v-icon :icon="mdiInvoiceListOutline" size="20" />
                                                                <v-tooltip activator="parent" location="top">{{ tt('Reconciliation Statement') }}</v-tooltip>
                                                                <v-menu activator="parent" :open-on-hover="true" :open-on-click="false">
                                                                    <v-list>
                                                                        <v-list-subheader>{{ tt('Reconciliation Statement') }}</v-list-subheader>
                                                                        <v-list-item class="text-body-medium" density="compact"
                                                                                     :key="dateRange.type" :value="dateRange.type"
                                                                                     :title="dateRange.displayName"
                                                                                     v-for="dateRange in accountReconciliationStatementDateRanges(element)"
                                                                                     @click="showReconciliationStatementDialog(element, dateRange.type)"></v-list-item>
                                                                    </v-list>
                                                                </v-menu>
                                                            </v-btn>
                                                        </template>
                                                        <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="element.hidden ? tt('Show') : tt('Hide')" :disabled="loading"
                                                               @click="hide(element, !element.hidden)">
                                                            <v-icon :icon="element.hidden ? mdiEyeOutline : mdiEyeOffOutline" size="20" />
                                                            <v-tooltip activator="parent">{{ element.hidden ? tt('Show') : tt('Hide') }}</v-tooltip>
                                                        </v-btn>
                                                        <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="tt('Edit')" :disabled="loading"
                                                               @click="edit(element)">
                                                            <v-icon :icon="mdiPencilOutline" size="20" />
                                                            <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                        </v-btn>
                                                        <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="tt('More')" :disabled="loading"
                                                               v-if="element.type === AccountType.SingleAccount.type">
                                                            <v-icon :icon="mdiDotsHorizontalCircleOutline" size="20" />
                                                            <v-tooltip activator="parent">{{ tt('More') }}</v-tooltip>
                                                            <v-menu activator="parent">
                                                                <v-list>
                                                                    <v-list-item class="text-body-medium" density="compact"
                                                                                 :title="tt('Mark as Reconciled')"
                                                                                 :prepend-icon="mdiReceiptTextCheckOutline"
                                                                                 @click="updateLastReconciledTime(element)"
                                                                                 v-if="useLastReconciledTime"></v-list-item>
                                                                    <v-divider class="my-2" v-if="useLastReconciledTime" />
                                                                    <v-list-item class="text-body-medium" density="compact"
                                                                                 :title="tt('Move All Transactions')"
                                                                                 :prepend-icon="mdiSwapHorizontal"
                                                                                 @click="moveAllTransactions(element)"></v-list-item>
                                                                    <v-list-item class="text-body-medium text-error" density="compact"
                                                                                 :title="tt('Clear All Transactions')"
                                                                                 :prepend-icon="mdiDeleteSweepOutline"
                                                                                 @click="clearAllTransactions(element)"></v-list-item>
                                                                </v-list>
                                                            </v-menu>
                                                        </v-btn>
                                                        <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="tt('Delete')" :disabled="loading"
                                                               @click="remove(element)">
                                                            <v-icon :icon="mdiDeleteOutline" size="20" />
                                                            <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                                        </v-btn>
                                                        <span class="ebk-mgmt-actions-divider" aria-hidden="true"></span>
                                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="tt('Move Up')"
                                                               :disabled="loading || !canMoveAccount(element, -1)"
                                                               @click="moveAccount(element, -1)">
                                                            <v-icon :icon="mdiArrowUp" size="18" />
                                                            <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                                        </v-btn>
                                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                                               :aria-label="tt('Move Down')"
                                                               :disabled="loading || !canMoveAccount(element, 1)"
                                                               @click="moveAccount(element, 1)">
                                                            <v-icon :icon="mdiArrowDown" size="18" />
                                                            <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                                        </v-btn>
                                                    </div>
                                                </div>
                                            </div>

                                            <template v-if="element.type === AccountType.MultiSubAccounts.type && isSubAccountsExpanded(element)">
                                                <div class="account-list-row account-list-row--child ebk-mgmt-row" role="row"
                                                     :class="{ 'ebk-mgmt-row--hidden': subAccount.hidden || element.hidden, 'account-list-row--child-last': isLastVisibleSubAccount(element, subAccount) }"
                                                     :key="subAccount.id"
                                                     v-for="subAccount in getVisibleSubAccounts(element)">
                                                    <div class="account-list-cell-name" role="cell">
                                                        <span class="account-list-guide" aria-hidden="true"></span>
                                                        <span class="ebk-mgmt-icon-tile ebk-mgmt-dimmable" aria-hidden="true">
                                                            <ItemIcon size="1.25rem" :icon-type="getAccountIconType(subAccount.iconType)" :icon-id="subAccount.icon" :color="subAccount.color" />
                                                        </span>
                                                        <div class="ebk-mgmt-name">
                                                            <div class="ebk-mgmt-name-line">
                                                                <span class="ebk-mgmt-name-text ebk-mgmt-dimmable">{{ subAccount.name }}</span>
                                                                <span class="ebk-mgmt-hidden-chip" v-if="subAccount.hidden">
                                                                    <v-icon :icon="mdiEyeOffOutline" size="12" />
                                                                    {{ tt('Hidden') }}
                                                                </span>
                                                            </div>
                                                            <span class="ebk-mgmt-meta ebk-mgmt-dimmable">
                                                                <span>{{ getCurrencyName(subAccount.currency) }}</span>
                                                                <template v-if="subAccount.comment">
                                                                    <span aria-hidden="true"> · </span>
                                                                    <span>{{ subAccount.comment }}</span>
                                                                </template>
                                                            </span>
                                                            <v-tooltip activator="parent" location="bottom start" open-delay="600" v-if="subAccount.comment">{{ subAccount.comment }}</v-tooltip>
                                                        </div>
                                                    </div>
                                                    <div class="account-list-cell-amount" role="cell">
                                                        <span class="ebk-mgmt-amount account-balance">{{ accountBalance(element, subAccount.id, showAccountBalance) }}</span>
                                                    </div>
                                                    <div class="account-list-cell-operation" role="cell">
                                                        <div class="ebk-mgmt-actions">
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Transaction List')" :disabled="loading"
                                                                   :to="`/transaction/list?accountIds=${subAccount.id}`">
                                                                <v-icon :icon="mdiListBoxOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Transaction List') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Reconciliation Statement')" :disabled="loading"
                                                                   @click="showReconciliationStatementDialog(subAccount)">
                                                                <v-icon :icon="mdiInvoiceListOutline" size="20" />
                                                                <v-tooltip activator="parent" location="top">{{ tt('Reconciliation Statement') }}</v-tooltip>
                                                                <v-menu activator="parent" :open-on-hover="true" :open-on-click="false">
                                                                    <v-list>
                                                                        <v-list-subheader>{{ tt('Reconciliation Statement') }}</v-list-subheader>
                                                                        <v-list-item class="text-body-medium" density="compact"
                                                                                     :key="dateRange.type" :value="dateRange.type"
                                                                                     :title="dateRange.displayName"
                                                                                     v-for="dateRange in accountReconciliationStatementDateRanges(subAccount)"
                                                                                     @click="showReconciliationStatementDialog(subAccount, dateRange.type)"></v-list-item>
                                                                    </v-list>
                                                                </v-menu>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="subAccount.hidden ? tt('Show') : tt('Hide')" :disabled="loading"
                                                                   @click="hide(subAccount, !subAccount.hidden)">
                                                                <v-icon :icon="subAccount.hidden ? mdiEyeOutline : mdiEyeOffOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ subAccount.hidden ? tt('Show') : tt('Hide') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Edit')" :disabled="loading"
                                                                   @click="edit(element)">
                                                                <v-icon :icon="mdiPencilOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('More')" :disabled="loading">
                                                                <v-icon :icon="mdiDotsHorizontalCircleOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('More') }}</v-tooltip>
                                                                <v-menu activator="parent">
                                                                    <v-list>
                                                                        <v-list-item class="text-body-medium" density="compact"
                                                                                     :title="tt('Mark as Reconciled')"
                                                                                     :prepend-icon="mdiReceiptTextCheckOutline"
                                                                                     @click="updateLastReconciledTime(subAccount)"
                                                                                     v-if="useLastReconciledTime"></v-list-item>
                                                                        <v-divider class="my-2" v-if="useLastReconciledTime" />
                                                                        <v-list-item class="text-body-medium" density="compact"
                                                                                     :title="tt('Move All Transactions')"
                                                                                     :prepend-icon="mdiSwapHorizontal"
                                                                                     @click="moveAllTransactions(subAccount)"></v-list-item>
                                                                        <v-list-item class="text-body-medium text-error" density="compact"
                                                                                     :title="tt('Clear All Transactions')"
                                                                                     :prepend-icon="mdiDeleteSweepOutline"
                                                                                     @click="clearAllTransactions(subAccount)"></v-list-item>
                                                                    </v-list>
                                                                </v-menu>
                                                            </v-btn>
                                                            <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Delete')" :disabled="loading"
                                                                   @click="remove(element, subAccount)">
                                                                <v-icon :icon="mdiDeleteOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                                            </v-btn>
                                                            <span class="account-list-move-placeholder" aria-hidden="true"></span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </template>
                                        </template>
                                    </div>
                                </template>
                            </draggable-list>
                        </div>
                    </v-card>
                </v-window-item>
            </v-window>
        </template>
    </main-page-layout>

    <account-filter-settings-dialog type="accountListTotalAmount"
                                    v-model:show="showAccountsIncludedInTotalDialog"
                                    @settings:change="showAccountsIncludedInTotalDialog = false" />

    <edit-dialog ref="editDialog" />
    <reconciliation-statement-dialog ref="reconciliationStatementDialog"
                                     @error="onShowDateRangeError" />
    <move-all-transactions-dialog ref="moveAllTransactionsDialog" />
    <clear-all-transactions-dialog ref="clearAllTransactionsDialog" />

    <date-range-selection-dialog :title="tt('Custom Date Range')"
                                 v-model:show="showCustomDateRangeDialog"
                                 @dateRange:change="onCustomDateRangeChanged"
                                 @error="onShowDateRangeError" />

    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import EditDialog from './list/dialogs/EditDialog.vue';
import ReconciliationStatementDialog from './list/dialogs/ReconciliationStatementDialog.vue';
import MoveAllTransactionsDialog from '@/views/desktop/accounts/list/dialogs/MoveAllTransactionsDialog.vue';
import ClearAllTransactionsDialog from '@/views/desktop/accounts/list/dialogs/ClearAllTransactionsDialog.vue';
import AccountFilterSettingsDialog from '@/views/desktop/common/dialogs/AccountFilterSettingsDialog.vue';

import { ref, computed, useTemplateRef, watch } from 'vue';
import { useDisplay } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { useAccountListPageBase } from '@/views/base/accounts/AccountListPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { DateRange, DateRangeScene, type LocalizedDateRange, type TimeRangeAndDateType } from '@/core/datetime.ts';
import { AccountType, AccountCategory } from '@/core/account.ts';
import { DEFAULT_RECONCILIATION_STATEMENT_DATE_RANGE_IN_DESKTOP } from '@/core/statistics.ts';
import type { Account } from '@/models/account.ts';

import { isNumber } from '@/lib/common.ts';
import {
    getCurrentUnixTime,
    getDateRangeByDateType,
    getDateRangeByBillingCycleDateType,
    getDateRangeByLastReconciledTimeRangeDateType
} from '@/lib/datetime.ts';
import { getAccountIconType } from '@/lib/icon.ts';

import {
    mdiEyeOutline,
    mdiEyeOffOutline,
    mdiCalculatorVariantOutline,
    mdiRefresh,
    mdiPlus,
    mdiArrowUp,
    mdiArrowDown,
    mdiPencilOutline,
    mdiDotsHorizontalCircleOutline,
    mdiReceiptTextCheckOutline,
    mdiSwapHorizontal,
    mdiDeleteSweepOutline,
    mdiDeleteOutline,
    mdiListBoxOutline,
    mdiInvoiceListOutline,
    mdiDrag,
    mdiDotsVertical,
    mdiChevronUp,
    mdiChevronDown
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type EditDialogType = InstanceType<typeof EditDialog>;
type ReconciliationStatementDialogType = InstanceType<typeof ReconciliationStatementDialog>;
type MoveAllTransactionsDialogType = InstanceType<typeof MoveAllTransactionsDialog>;
type ClearAllTransactionsDialogType = InstanceType<typeof ClearAllTransactionsDialog>;

const { lgAndUp } = useDisplay();

const { tt, getAllDateRanges, getCurrencyName, joinMultiText, formatNumberToLocalizedNumerals } = useI18n();

const {
    loading,
    showHidden,
    displayOrderModified,
    showAccountBalance,
    customAccountCategoryOrder,
    defaultAccountCategory,
    firstDayOfWeek,
    fiscalYearStart,
    useLastReconciledTime,
    allCategorizedAccountsMap,
    allAccountCount,
    netAssets,
    totalAssets,
    totalLiabilities,
    canShowAvailableCredit,
    accountCategoryTotalBalance,
    accountBalance,
    accountAvailableCredit
} = useAccountListPageBase();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const editDialog = useTemplateRef<EditDialogType>('editDialog');
const reconciliationStatementDialog = useTemplateRef<ReconciliationStatementDialogType>('reconciliationStatementDialog');
const moveAllTransactionsDialog = useTemplateRef<MoveAllTransactionsDialogType>('moveAllTransactionsDialog');
const clearAllTransactionsDialog = useTemplateRef<ClearAllTransactionsDialogType>('clearAllTransactionsDialog');

const activeAccountCategoryType = ref<number>(defaultAccountCategory.value.type);
const activeTab = ref<string>('accountPage');
const loadError = ref<boolean>(false);
const subAccountsExpandedStates = ref<Record<string, boolean>>({});
const accountToShowReconciliationStatement = ref<Account | null>(null);
const alwaysShowNav = ref<boolean>(lgAndUp.value);
const showNav = ref<boolean>(lgAndUp.value);
const showAccountsIncludedInTotalDialog = ref<boolean>(false);
const showCustomDateRangeDialog = ref<boolean>(false);

const hideAccountCategoriesWithoutAccounts = computed<boolean>(() => settingsStore.appSettings.hideCategoriesWithoutAccounts);
const hasAnyVisibleAccount = computed<boolean>(() => accountsStore.allVisibleAccountsCount > 0);
const activeAccountCategory = computed<AccountCategory | undefined>(() => AccountCategory.valueOf(activeAccountCategoryType.value));
const activeAccountCategoryTotalBalance = computed<string>(() => accountCategoryTotalBalance(activeAccountCategory.value, false));
const activeCreditCardTotalAvailabeCredit = computed<string>(() => accountCategoryTotalBalance(activeAccountCategory.value, true));

const activeAccountCategoryVisibleAccountCount = computed<number>(() => {
    if (!activeAccountCategory.value) {
        return 0;
    }

    const categorizedAccounts = allCategorizedAccountsMap.value[activeAccountCategory.value.type];

    if (!categorizedAccounts || !categorizedAccounts.accounts || !categorizedAccounts.accounts.length) {
        return 0;
    }

    if (showHidden.value) {
        return categorizedAccounts.accounts.length;
    }

    let visibleCount = 0;

    for (const account of categorizedAccounts.accounts) {
        if (!account.hidden) {
            visibleCount++;
        }
    }

    return visibleCount;
});

function reload(force: boolean): void {
    loading.value = true;

    accountsStore.loadAllAccounts({
        force: force
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        if (force) {
            desktopPageStore.addToast('Account list has been updated');
        }
    }).catch(error => {
        loading.value = false;

        if (error && error.isUpToDate) {
            loadError.value = false;
            displayOrderModified.value = false;
        } else if (!allAccountCount.value) {
            loadError.value = true;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function hasAccount(accountCategory: AccountCategory): boolean {
    return accountsStore.hasAccount(accountCategory, !showHidden.value);
}

function isSubAccountsExpanded(account: Account): boolean {
    return subAccountsExpandedStates.value[account.id] ?? !settingsStore.appSettings.collapseSubAccountsByDefault;
}

function toggleSubAccounts(account: Account): void {
    subAccountsExpandedStates.value[account.id] = !isSubAccountsExpanded(account);
}

function getVisibleSubAccounts(account: Account): Account[] {
    if (!account.subAccounts) {
        return [];
    }

    return account.subAccounts.filter(subAccount => showHidden.value || !subAccount.hidden);
}

function isLastVisibleSubAccount(account: Account, subAccount: Account): boolean {
    const visibleSubAccounts = getVisibleSubAccounts(account);
    return visibleSubAccounts.length > 0 && visibleSubAccounts[visibleSubAccounts.length - 1]!.id === subAccount.id;
}

function getAdjacentVisibleAccountIndex(account: Account, direction: number): { from: number, to: number } | null {
    if (!activeAccountCategory.value) {
        return null;
    }

    const accounts = allCategorizedAccountsMap.value[activeAccountCategory.value.type]?.accounts;

    if (!accounts) {
        return null;
    }

    const from = accounts.findIndex(item => item.id === account.id);

    if (from < 0) {
        return null;
    }

    for (let to = from + direction; to >= 0 && to < accounts.length; to += direction) {
        if (showHidden.value || !accounts[to]!.hidden) {
            return { from, to };
        }
    }

    return null;
}

function canMoveAccount(account: Account, direction: number): boolean {
    return activeAccountCategoryVisibleAccountCount.value > 1 && !!getAdjacentVisibleAccountIndex(account, direction);
}

// non-drag alternative to the drag handle (guidelines §6.4), uses the same display order update as dragging
function moveAccount(account: Account, direction: number): void {
    const position = getAdjacentVisibleAccountIndex(account, direction);

    if (!position) {
        return;
    }

    accountsStore.changeAccountDisplayOrder({
        accountId: account.id,
        from: position.from,
        to: position.to,
        updateListOrder: true,
        updateGlobalListOrder: true
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function accountCurrency(account: Account): string | null {
    if (account.type === AccountType.SingleAccount.type) {
        return getCurrencyName(account.currency);
    } else if (account.type === AccountType.MultiSubAccounts.type) {
        const subAccountCurrencies = account.getSubAccountCurrencies(showHidden.value, undefined)
            .map(currencyCode => getCurrencyName(currencyCode));
        return joinMultiText(subAccountCurrencies);
    } else {
        return null;
    }
}

function accountReconciliationStatementDateRanges(account: Account): LocalizedDateRange[] {
    return getAllDateRanges(DateRangeScene.Normal, {
        includeCustom: true,
        includeBillingCycle: !!accountsStore.getAccountStatementDate(account.id),
        includeLastReconciledTimeRange: userStore.currentUserUseLastReconciledTime && !!account.lastReconciledTime
    });
}

function add(): void {
    editDialog.value?.open({
        category: activeAccountCategoryType.value
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function edit(account: Account): void {
    editDialog.value?.open({
        id: account.id,
        currentAccount: account
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }

        if (accountsStore.accountListStateInvalid && !loading.value) {
            reload(false);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function showReconciliationStatementDialog(account: Account, dateRangeType?: number): void {
    if (!isNumber(dateRangeType)) {
        const defualtDateRange = DateRange.valueOf(settingsStore.appSettings.reconciliationStatementButtonDefaultDateRangeTypeInDesktop);

        if (!defualtDateRange) {
            dateRangeType = DEFAULT_RECONCILIATION_STATEMENT_DATE_RANGE_IN_DESKTOP.type;
        } else if (defualtDateRange.isBillingCycle && !accountsStore.getAccountStatementDate(account.id)) {
            dateRangeType = DEFAULT_RECONCILIATION_STATEMENT_DATE_RANGE_IN_DESKTOP.type;
        } else if (defualtDateRange.isLastReconciledTimeRange && (!userStore.currentUserUseLastReconciledTime || !account.lastReconciledTime)) {
            dateRangeType = DEFAULT_RECONCILIATION_STATEMENT_DATE_RANGE_IN_DESKTOP.type;
        } else {
            dateRangeType = defualtDateRange.type;
        }
    }

    if (!isNumber(dateRangeType) || dateRangeType === DateRange.Custom.type) {
        accountToShowReconciliationStatement.value = account;
        showCustomDateRangeDialog.value = true;
        return;
    }

    let dateRange: TimeRangeAndDateType | null = null;

    if (DateRange.isBillingCycle(dateRangeType)) {
        dateRange = getDateRangeByBillingCycleDateType(dateRangeType, firstDayOfWeek.value, fiscalYearStart.value, accountsStore.getAccountStatementDate(account.id));
    } else if (DateRange.isLastReconciledTimeRange(dateRangeType)) {
        dateRange = getDateRangeByLastReconciledTimeRangeDateType(dateRangeType, account.lastReconciledTime);
    } else {
        dateRange = getDateRangeByDateType(dateRangeType, firstDayOfWeek.value, fiscalYearStart.value);
    }

    if (!dateRange) {
        return;
    }

    reconciliationStatementDialog.value?.open({
        accountId: account.id,
        startTime: dateRange.minTime,
        endTime: dateRange.maxTime
    });
}

function updateLastReconciledTime(account: Account): void {
    confirmDialog.value?.open('Are you sure you want to update the last reconciled time of this account to the current time?').then(() => {
        loading.value = true;

        accountsStore.updateAccountLastReconciledTime(account.id, getCurrentUnixTime()).then(() => {
            loading.value = false;
            desktopPageStore.addToast('Last reconciled time have been updated');

            if (accountsStore.accountListStateInvalid && !loading.value) {
                reload(false);
            }

        }).catch(error => {
            loading.value = false;

            if (error) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function moveAllTransactions(account: Account): void {
    moveAllTransactionsDialog.value?.open(account).then(() => {
        desktopPageStore.addToast('All transactions in this account have been moved.');

        if (accountsStore.accountListStateInvalid && !loading.value) {
            reload(false);
        }
    });
}

function clearAllTransactions(account: Account): void {
    clearAllTransactionsDialog.value?.open(account).then(() => {
        desktopPageStore.addToast('All transactions in this account have been cleared');

        if (accountsStore.accountListStateInvalid && !loading.value) {
            reload(false);
        }
    });
}

function hide(targetAccount: Account, hidden: boolean): void {
    loading.value = true;

    accountsStore.hideAccount({
        account: targetAccount,
        hidden: hidden
    }).then(() => {
        loading.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(account: Account, targetSubAccount?: Account): void {
    if (targetSubAccount) {
        const subAccount: Account | null = account.getSubAccount(targetSubAccount.id);

        if (!subAccount) {
            desktopPageStore.addToast('Unable to delete this sub-account');
            return;
        }

        confirmDialog.value?.open('Are you sure you want to delete this sub-account?', { color: 'error' }).then(() => {
            loading.value = true;

            accountsStore.deleteSubAccount({
                subAccount: subAccount
            }).then(() => {
                loading.value = false;
            }).catch(error => {
                loading.value = false;

                if (!error.processed) {
                    desktopPageStore.addErrorToast(error);
                }
            });
        });
    } else {
        confirmDialog.value?.open('Are you sure you want to delete this account?', { color: 'error' }).then(() => {
            loading.value = true;

            accountsStore.deleteAccount({
                account: account
            }).then(() => {
                loading.value = false;
            }).catch(error => {
                loading.value = false;

                if (!error.processed) {
                    desktopPageStore.addErrorToast(error);
                }
            });
        });
    }
}

function saveSortResult(): void {
    if (!displayOrderModified.value) {
        return;
    }

    loading.value = true;

    accountsStore.updateAccountDisplayOrders().then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function onMove(event: { moved: { element: { id: string }, oldIndex: number, newIndex: number } }): void {
    if (!event || !event.moved) {
        return;
    }

    const moveEvent = event.moved;

    if (!moveEvent.element || !moveEvent.element.id) {
        desktopPageStore.addToast('Unable to move account');
        return;
    }

    accountsStore.changeAccountDisplayOrder({
        accountId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex,
        updateListOrder: false,
        updateGlobalListOrder: true
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function onCustomDateRangeChanged(minUnixTime: number, maxUnixTime: number): void {
    if (!accountToShowReconciliationStatement.value) {
        desktopPageStore.addToast('An error occurred');
        return;
    }

    showCustomDateRangeDialog.value = false;

    reconciliationStatementDialog.value?.open({
        accountId: accountToShowReconciliationStatement.value.id,
        startTime: minUnixTime,
        endTime: maxUnixTime
    });

    accountToShowReconciliationStatement.value = null;
}

function onShowDateRangeError(message: string): void {
    desktopPageStore.addErrorToast(message);
}

watch(lgAndUp, (newValue) => {
    alwaysShowNav.value = newValue;

    if (!showNav.value) {
        showNav.value = newValue;
    }
});

reload(false);
</script>

<style>
.account-category-tabs .nav-link > a {
    min-height: 52px;
}

.account-category-nav-title {
    display: flex;
    flex-direction: column;
    min-width: 0;
    line-height: 1.2;
}

.account-category-nav-balance {
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
    font-variant-numeric: var(--ebk-font-numeric);
}

.account-list-panel {
    min-height: min(780px, calc(100vh - var(--ebk-shell-header-height, 64px) - 2rem));
}

/* Account table: grid rows so each multi-account can carry its indented sub-account rows while dragging as one item */
.account-list {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(8rem, auto) auto;
}

/* every wrapper level shares the table columns so amounts and actions line up across rows */
.account-list > *,
.account-list-body > *,
.account-list-group > * {
    grid-column: 1 / -1;
}

.account-list-body,
.account-list-group,
.account-list-skeleton,
.account-list-head,
.account-list-row {
    display: grid;
    grid-template-columns: subgrid;
}

.account-list-head,
.account-list-row {
    align-items: center;
    column-gap: var(--ebk-space-3);
    padding-inline: var(--ebk-space-4) var(--ebk-space-2);
}

.account-list-group:empty {
    display: none;
}

.account-list-head {
    min-height: 40px;
    border-block-end: 1px solid var(--ebk-color-hairline);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--ebk-color-text-secondary);
    font-size: var(--ebk-font-technical-size);
    font-weight: var(--ebk-font-technical-weight);
    letter-spacing: var(--ebk-font-technical-letter-spacing);

    &:lang(en), &:lang(de), &:lang(es), &:lang(fr), &:lang(it), &:lang(nl), &:lang(pl), &:lang(pt), &:lang(ro), &:lang(sk), &:lang(sl), &:lang(tr), &:lang(vi) {
        text-transform: uppercase;
    }
}

.account-list-row {
    min-height: 56px;
    padding-block: var(--ebk-space-1);
    border-block-end: 1px solid var(--ebk-color-hairline);
    transition: background-color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);

    &:hover {
        background-color: rgba(var(--v-theme-on-surface), 0.04);
    }
}

.account-list-cell-name {
    display: flex;
    align-items: center;
    min-width: 0;
}

.account-list-cell-amount {
    min-width: 0;
    text-align: end;
    white-space: nowrap;
}

.account-list-cell-operation {
    display: flex;
    justify-content: flex-end;
}

.account-list-row--child {
    position: relative;
    min-height: 48px;
    background-color: rgba(var(--ebk-color-surface-sunken-rgb), 0.45);

    .account-list-cell-name {
        padding-inline-start: 3.75rem;
    }

    .ebk-mgmt-icon-tile {
        width: 28px;
        height: 28px;
    }
}

/* nested child guide: vertical hairline from the parent icon plus a short branch */
.account-list-guide {
    position: absolute;
    inset-block: 0;
    inset-inline-start: calc(var(--ebk-space-4) + 3rem);
    width: 0.75rem;
    pointer-events: none;

    &::before,
    &::after {
        content: "";
        position: absolute;
        inset-inline-start: 0;
    }

    &::before {
        inset-block: 0;
        border-inline-start: 1px solid var(--ebk-color-hairline);
    }

    &::after {
        inset-block-start: 50%;
        width: 0.75rem;
        border-block-start: 1px solid var(--ebk-color-hairline);
    }
}

.account-list-row--child-last .account-list-guide::before {
    inset-block-end: 50%;
}

/* keeps sub-account actions aligned with the parent row, which ends with the divider and the two move buttons */
.account-list-move-placeholder {
    width: calc(var(--ebk-space-1) * 2 + 1px + 2px + 72px);
}

.account-list-state {
    padding: var(--ebk-space-4);
}

.account-list-group.dragging-item {
    opacity: 0.6;
}

@media (forced-colors: active) {
    .account-list-guide::before,
    .account-list-guide::after {
        border-color: CanvasText;
    }
}
</style>

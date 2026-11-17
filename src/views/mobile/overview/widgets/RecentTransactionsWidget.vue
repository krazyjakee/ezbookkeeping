<template>
    <overview-widget-panel flush :title="title || tt('Recent Transactions')" :show-title="showTitle"
                           :loading="loading" :empty="!loading && !transactions.length">
        <div class="mobile-overview-rows" v-if="transactions.length">
            <a class="mobile-overview-row mobile-recent-transaction-row"
               :href="`/transaction/detail?id=${transaction.id}&type=${transaction.type}`"
               :key="transaction.id" v-for="transaction in transactions">
                <span class="mobile-overview-row-icon">
                    <ItemIcon :icon-type="getCategoryIconType(getTransactionCategory(transaction)?.iconType)"
                              :icon-id="getTransactionCategory(transaction)?.icon ?? ''"
                              :color="getTransactionCategory(transaction)?.color"
                              v-if="getTransactionCategory(transaction) && getTransactionCategory(transaction)?.color"></ItemIcon>
                    <f7-icon f7="pencil_ellipsis_rectangle" v-else></f7-icon>
                </span>
                <span class="mobile-overview-row-main">
                    <span class="mobile-overview-row-title">{{ getTransactionCategoryName(transaction) }}</span>
                    <span class="mobile-overview-row-meta mobile-recent-transaction-meta" v-if="getSourceAccount(transaction)">
                        <span>{{ getSourceAccount(transaction)?.name }}</span>
                        <template v-if="isTransferBetweenAccounts(transaction)">
                            <span aria-hidden="true" class="mobile-recent-transaction-arrow">→</span>
                            <span>{{ getDestinationAccount(transaction)?.name }}</span>
                        </template>
                    </span>
                    <span class="mobile-overview-row-meta mobile-recent-transaction-meta">
                        <span class="ebk-numeric">{{ getDisplayDayOfWeek(transaction) }} {{ getDisplayDayOfMonth(transaction) }} · {{ getDisplayTime(transaction) }}</span>
                        <template v-if="transaction.comment">
                            <span>·</span>
                            <span>{{ transaction.comment }}</span>
                        </template>
                    </span>
                </span>
                <span class="mobile-overview-row-end">
                    <signed-amount show-type-label :type="getAmountType(transaction)" :text="getDisplayAmount(transaction)"
                                   :hide-sign="isAmountHidden(transaction)" />
                </span>
                <f7-icon class="mobile-overview-row-chevron icon-with-direction" f7="chevron_right" aria-hidden="true"></f7-icon>
            </a>
        </div>
        <div class="mobile-overview-rows skeleton-text" v-else-if="loading">
            <div class="mobile-overview-row mobile-recent-transaction-row" :key="idx" v-for="idx in itemCount">
                <span class="mobile-overview-row-icon"><f7-icon f7="app_fill"></f7-icon></span>
                <span class="mobile-overview-row-main">
                    <span class="mobile-overview-row-title">Category</span>
                    <span class="mobile-overview-row-meta">Source Account</span>
                    <span class="mobile-overview-row-meta">Sun DD · HH:mm</span>
                </span>
                <span class="mobile-overview-row-end">
                    <span>0.00 GBP</span>
                    <span class="mobile-recent-transaction-type">Expense</span>
                </span>
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useOverviewStore } from '@/stores/overview.ts';

import { TransactionType } from '@/core/transaction.ts';
import type { OverviewRecentTransactionsQuery } from '@/core/overview_layout.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';

import type { Account } from '@/models/account.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type { TransactionInfoResponse } from '@/models/transaction.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import { parseDateTimeFromUnixTimeWithTimezoneOffset } from '@/lib/datetime.ts';
import { getCategoryIconType } from '@/lib/icon.ts';
import { getOverviewRecentTransactionsQuery } from '@/lib/overview_layout.ts';

const props = defineProps<{
    loading: boolean;
    title?: string;
    showTitle: boolean;
    editing?: boolean;
    itemCount: number;
    accountIds?: string[];
    categoryIds?: string[];
    tagFilter?: string;
    amountFilter?: string;
    keyword?: string;
}>();

const {
    tt,
    getWeekdayShortName,
    getCalendarDisplayDayOfMonthFromDateTime,
    formatDateTimeToShortTime,
    formatAmountToLocalizedNumeralsWithCurrency
} = useI18n();

const settingsStore = useSettingsStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const overviewStore = useOverviewStore();

const showAmountInHomePage = computed<boolean>(() => settingsStore.appSettings.showAmountInHomePage);
const recentTransactionsQuery = computed<OverviewRecentTransactionsQuery>(() => getOverviewRecentTransactionsQuery({
    itemCount: props.itemCount,
    accountIds: props.accountIds ?? [],
    categoryIds: props.categoryIds ?? [],
    tagFilter: props.tagFilter ?? '',
    amountFilter: props.amountFilter ?? '',
    keyword: props.keyword ?? ''
}));
const transactions = computed<TransactionInfoResponse[]>(() => overviewStore.getRecentTransactions(recentTransactionsQuery.value).slice(0, props.itemCount));

function getTransactionDateTime(transaction: TransactionInfoResponse) {
    return parseDateTimeFromUnixTimeWithTimezoneOffset(transaction.time, transaction.utcOffset);
}

function getDisplayDayOfMonth(transaction: TransactionInfoResponse): string {
    return getCalendarDisplayDayOfMonthFromDateTime(getTransactionDateTime(transaction));
}

function getDisplayDayOfWeek(transaction: TransactionInfoResponse): string {
    return getWeekdayShortName(getTransactionDateTime(transaction).getWeekDay());
}

function getDisplayTime(transaction: TransactionInfoResponse): string {
    return formatDateTimeToShortTime(getTransactionDateTime(transaction));
}

function getDisplayAmount(transaction: TransactionInfoResponse): string {
    const account = getSourceAccount(transaction);
    const amount = !showAmountInHomePage.value || transaction.hideAmount ? DISPLAY_HIDDEN_AMOUNT : parseBigDecimal(transaction.sourceAmount);
    return formatAmountToLocalizedNumeralsWithCurrency(amount, account?.currency ?? '');
}

function isAmountHidden(transaction: TransactionInfoResponse): boolean {
    return !showAmountInHomePage.value || !!transaction.hideAmount || parseBigDecimal(transaction.sourceAmount).isZero();
}

function getAmountType(transaction: TransactionInfoResponse): 'income' | 'expense' | 'transfer' | 'none' {
    if (transaction.type === TransactionType.Income) {
        return 'income';
    } else if (transaction.type === TransactionType.Expense) {
        return 'expense';
    } else if (transaction.type === TransactionType.Transfer) {
        return 'transfer';
    } else {
        return 'none';
    }
}

function isTransferBetweenAccounts(transaction: TransactionInfoResponse): boolean {
    const sourceAccount = getSourceAccount(transaction);
    const destinationAccount = getDestinationAccount(transaction);
    return transaction.type === TransactionType.Transfer && !!sourceAccount && !!destinationAccount && sourceAccount.id !== destinationAccount.id;
}

function getSourceAccount(transaction: TransactionInfoResponse): Account | undefined {
    return accountsStore.allAccountsMap[transaction.sourceAccountId];
}

function getDestinationAccount(transaction: TransactionInfoResponse): Account | undefined {
    return accountsStore.allAccountsMap[transaction.destinationAccountId];
}

function getTransactionCategory(transaction: TransactionInfoResponse): TransactionCategory | undefined {
    if (transaction.type === TransactionType.ModifyBalance) {
        return undefined;
    }

    return transactionCategoriesStore.allTransactionCategoriesMap[transaction.categoryId];
}

function getTransactionCategoryName(transaction: TransactionInfoResponse): string {
    if (transaction.type === TransactionType.ModifyBalance) {
        return tt('Modify Balance');
    }

    const category = getTransactionCategory(transaction);

    if (category) {
        return category.name;
    } else if (transaction.type === TransactionType.Income) {
        return tt('Income');
    } else if (transaction.type === TransactionType.Expense) {
        return tt('Expense');
    } else if (transaction.type === TransactionType.Transfer) {
        return tt('Transfer');
    } else {
        return tt('Transaction');
    }
}
</script>

<style>
.mobile-recent-transaction-row {
    min-height: 60px;
}

.mobile-recent-transaction-meta > span + span {
    margin-inline-start: 4px;
}

.mobile-recent-transaction-arrow {
    display: inline-block;
}

html[dir="rtl"] .mobile-recent-transaction-arrow {
    transform: scaleX(-1);
}

.mobile-recent-transaction-type {
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.75rem;
    font-weight: 400;
}

.mobile-recent-transaction-row .ebk-signed-amount-type {
    font-weight: 400;
}
</style>

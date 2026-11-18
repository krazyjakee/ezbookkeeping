<template>
    <overview-widget-panel class="overview-widget--list recent-transactions-widget"
                           :title="title || tt('Recent Transactions')"
                           :loading="loading" :empty="!transactions.length">
        <v-list class="overview-widget__list py-0" density="compact" v-if="transactions.length">
            <v-list-item class="overview-widget__list-item" :key="transaction.id"
                         :title="getTransactionCategoryName(transaction)"
                         v-for="transaction in transactions"
                         @click="showTransaction(transaction)">
                <template #prepend>
                    <span class="overview-widget__item-icon">
                        <ItemIcon size="22px" :icon-type="getCategoryIconType(getTransactionCategory(transaction)?.iconType)"
                                  :icon-id="getTransactionCategory(transaction)?.icon ?? ''"
                                  :color="getTransactionCategory(transaction)?.color"
                                  v-if="getTransactionCategory(transaction) && getTransactionCategory(transaction)?.color" />
                        <v-icon size="22" :icon="mdiPencilBoxOutline" v-else-if="!getTransactionCategory(transaction) || !getTransactionCategory(transaction)?.color" />
                    </span>
                </template>

                <template #append>
                    <signed-amount class="overview-widget__list-amount" show-type-label
                                   :type="getAmountType(transaction)"
                                   :text="getDisplayTransactionAmount(transaction)"
                                   :hide-sign="isAmountHidden(transaction)" />
                </template>

                <div class="text-truncate overview-widget__caption mt-h1" :title="getDisplayDescription(transaction)">{{ getDisplayDescription(transaction) }}</div>
            </v-list-item>
        </v-list>
        <div class="overview-widget__list" v-else-if="loading">
            <div class="overview-widget__list-skeleton" :key="idx" v-for="idx in props.itemCount">
                <v-skeleton-loader class="skeleton-no-margin" type="avatar" width="32px" height="32px" />
                <div class="flex-grow-1">
                    <v-skeleton-loader class="skeleton-no-margin" type="text" />
                    <v-skeleton-loader class="skeleton-no-margin" type="text" width="60%" />
                </div>
            </div>
        </div>

        <edit-dialog ref="editDialog" :type="TransactionEditPageType.Transaction" />
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import EditDialog from '@/views/desktop/transactions/list/dialogs/EditDialog.vue';

import { computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useOverviewStore } from '@/stores/overview.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { BigDecimal, HiddenAmount } from '@/core/numeral.ts';
import { TransactionType } from '@/core/transaction.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';
import type { OverviewRecentTransactionsQuery } from '@/core/overview_layout.ts';

import type { TransactionCategory } from '@/models/transaction_category.ts';
import { type TransactionInfoResponse, Transaction } from '@/models/transaction.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { getCategoryIconType } from '@/lib/icon.ts';
import { getOverviewRecentTransactionsQuery } from '@/lib/overview_layout.ts';

import {
    mdiPencilBoxOutline
} from '@mdi/js';

type EditDialogType = InstanceType<typeof EditDialog>;

const props = defineProps<{
    loading: boolean;
    title?: string;
    editing?: boolean;
    itemCount: number;
    accountIds?: string[];
    categoryIds?: string[];
    tagFilter?: string;
    amountFilter?: string;
    keyword?: string;
}>();

const emit = defineEmits<{
    (e: 'refresh'): void
}>();

const { tt, formatDateTimeToShortDateTime, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const settingsStore = useSettingsStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const overviewStore = useOverviewStore();
const desktopPageStore = useDesktopPageStore();

const editDialog = useTemplateRef<EditDialogType>('editDialog');

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

function getDisplayDescription(transaction: TransactionInfoResponse): string {
    const sourceAccountName = accountsStore.allAccountsMap[transaction.sourceAccountId]?.name ?? '';
    const destinationAccountName = transaction.type === TransactionType.Transfer && transaction.destinationAccountId !== transaction.sourceAccountId ? (accountsStore.allAccountsMap[transaction.destinationAccountId]?.name ?? '') : '';
    const accountName = destinationAccountName ? `${sourceAccountName} → ${destinationAccountName}` : sourceAccountName;
    return `${formatDateTimeToShortDateTime(parseDateTimeFromUnixTime(transaction.time))}${accountName ? ` · ${accountName}` : ''}${transaction.comment ? ` · ${transaction.comment}` : ''}`;
}

function isAmountHidden(transaction: TransactionInfoResponse): boolean {
    return !showAmountInHomePage.value || transaction.hideAmount;
}

function getDisplayTransactionAmount(transaction: TransactionInfoResponse): string {
    let amount: BigDecimal | HiddenAmount = DISPLAY_HIDDEN_AMOUNT;

    if (!isAmountHidden(transaction)) {
        // the sign comes from the transaction type (design guidelines D1), balance modifications keep their natural sign
        amount = transaction.type === TransactionType.ModifyBalance ? parseBigDecimal(transaction.sourceAmount) : parseBigDecimal(transaction.sourceAmount).abs();
    }

    return formatAmountToLocalizedNumeralsWithCurrency(amount, accountsStore.allAccountsMap[transaction.sourceAccountId]?.currency ?? '');
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

function showTransaction(transaction: TransactionInfoResponse): void {
    if (props.editing) {
        return;
    }

    editDialog.value?.open({
        id: transaction.id,
        currentTransaction: Transaction.of(transaction)
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }

        emit('refresh');
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}
</script>

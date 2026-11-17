<template>
    <overview-widget-panel class="overview-widget--list"
                           :title="title || tt('Account Unreconciled Transactions')"
                           :loading="loading" :empty="!displayAccounts.length" :empty-text="noDataText">
        <v-list class="overview-widget__list py-0" density="compact" v-if="displayAccounts.length">
            <v-list-item class="overview-widget__list-item" :key="item.account.id" :title="item.account.name"
                         @click="showReconciliationStatementDialog(item.account.id)"
                         v-for="item in displayAccounts">
                <template #prepend>
                    <span class="overview-widget__item-icon">
                        <ItemIcon size="22px" :icon-type="getAccountIconType(item.account.iconType)" :icon-id="item.account.icon" :color="item.account.color" />
                    </span>
                </template>
                <template #append>
                    <span class="overview-widget__count-badge ebk-numeric">{{ formatNumberToLocalizedNumerals(item.count) }}</span>
                </template>
            </v-list-item>
        </v-list>
        <div class="overview-widget__list" v-else-if="loading">
            <div class="overview-widget__list-skeleton" :key="idx" v-for="idx in props.itemCount">
                <v-skeleton-loader class="skeleton-no-margin" type="avatar" width="32px" height="32px" />
                <v-skeleton-loader class="skeleton-no-margin flex-grow-1" type="text" />
            </div>
        </div>

        <reconciliation-statement-dialog ref="reconciliationStatementDialog"
                                         @update:last-reconciled-time="onUpdateLastReconciledTime"
                                         @error="onShowReconciliationStatementError" />
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import ReconciliationStatementDialog from '@/views/desktop/accounts/list/dialogs/ReconciliationStatementDialog.vue';

import { useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import {
    type CommonAccountUnreconciledTransactionsWidgetProps,
    useAccountUnreconciledTransactionsWidgetBase
} from '@/views/base/overview/AccountUnreconciledTransactionsWidgetBase.ts';

import { getAccountIconType } from '@/lib/icon.ts';

type ReconciliationStatementDialogType = InstanceType<typeof ReconciliationStatementDialog>;

const emit = defineEmits<{
    (e: 'refresh'): void
}>();

interface DesktopAccountUnreconciledTransactionsWidgetProps extends CommonAccountUnreconciledTransactionsWidgetProps {
    editing?: boolean
}

const props = defineProps<DesktopAccountUnreconciledTransactionsWidgetProps>();

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const { noDataText, displayAccounts, getTransactionListDateRange } = useAccountUnreconciledTransactionsWidgetBase(props);

const desktopPageStore = useDesktopPageStore();

const reconciliationStatementDialog = useTemplateRef<ReconciliationStatementDialogType>('reconciliationStatementDialog');

function showReconciliationStatementDialog(accountId: string): void {
    if (props.editing || props.loading) {
        return;
    }

    const dateRange = getTransactionListDateRange(accountId);

    reconciliationStatementDialog.value?.open({
        accountId: accountId,
        startTime: dateRange?.minTime ?? 0,
        endTime: dateRange?.maxTime ?? 0
    });
}

function onUpdateLastReconciledTime(): void {
    emit('refresh');
}

function onShowReconciliationStatementError(message: string): void {
    desktopPageStore.addErrorToast(message);
}
</script>

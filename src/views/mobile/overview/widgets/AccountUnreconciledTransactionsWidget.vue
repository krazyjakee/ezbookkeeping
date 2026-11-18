<template>
    <overview-widget-panel flush :title="title || tt('Account Unreconciled Transactions')" :show-title="showTitle"
                           :loading="loading" :empty="!loading && !displayAccounts.length" :empty-text="noDataText"
                           v-if="loading || editing || !hideWhenEmpty || displayAccounts.length">
        <div class="mobile-overview-rows" v-if="displayAccounts.length">
            <a class="mobile-overview-row" :key="item.account.id"
               :href="getReconciliationStatementUrl(item.account.id)"
               v-for="item in displayAccounts">
                <span class="mobile-overview-row-icon">
                    <ItemIcon :icon-type="getAccountIconType(item.account.iconType)" :icon-id="item.account.icon" :color="item.account.color" />
                </span>
                <span class="mobile-overview-row-main">
                    <span class="mobile-overview-row-title">{{ item.account.name }}</span>
                </span>
                <span class="mobile-overview-row-end ebk-numeric">{{ formatNumberToLocalizedNumerals(item.count) }}</span>
                <f7-icon class="mobile-overview-row-chevron icon-with-direction" f7="chevron_right" aria-hidden="true"></f7-icon>
            </a>
        </div>
        <div class="mobile-overview-rows skeleton-text" v-else-if="loading">
            <div class="mobile-overview-row" :key="idx" v-for="idx in itemCount">
                <span class="mobile-overview-row-icon"><f7-icon f7="app_fill"></f7-icon></span>
                <span class="mobile-overview-row-main"><span class="mobile-overview-row-title">Account Name</span></span>
                <span class="mobile-overview-row-end">0</span>
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';

import { useI18n } from '@/locales/helpers.ts';

import {
    type CommonAccountUnreconciledTransactionsWidgetProps,
    useAccountUnreconciledTransactionsWidgetBase
} from '@/views/base/overview/AccountUnreconciledTransactionsWidgetBase.ts';

import { DateRange } from '@/core/datetime.ts';

import { getAccountIconType } from '@/lib/icon.ts';

interface MobileAccountUnreconciledTransactionsWidgetProps extends CommonAccountUnreconciledTransactionsWidgetProps {
    editing?: boolean;
    showTitle: boolean;
    hideWhenEmpty?: boolean;
}

const props = defineProps<MobileAccountUnreconciledTransactionsWidgetProps>();

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const { noDataText, displayAccounts, getTransactionListDateRange } = useAccountUnreconciledTransactionsWidgetBase(props);

function getReconciliationStatementUrl(accountId: string): string {
    const dateRange = getTransactionListDateRange(accountId);

    if (!dateRange) {
        return `/account/reconciliation_statements?accountId=${accountId}&dateType=${DateRange.All.type}`;
    }

    return `/account/reconciliation_statements?accountId=${accountId}&dateType=${dateRange.dateType}`;
}
</script>

<template>
    <overview-widget-panel flush class="mobile-period-summary" :title="tt('Period Summary')" :loading="loading">
        <div class="mobile-period-summary-table" role="table" :aria-label="tt('Period Summary')">
            <div class="mobile-period-summary-head ebk-technical-heading" role="row">
                <span role="columnheader">{{ tt('Date Range') }}</span>
                <span role="columnheader">{{ tt('Income') }} <span aria-hidden="true">↑</span></span>
                <span role="columnheader">{{ tt('Expense') }} <span aria-hidden="true">↓</span></span>
                <span class="mobile-period-summary-chevron-col" aria-hidden="true"></span>
            </div>
            <a class="mobile-period-summary-row" role="row" :key="period.dateRange.type"
               :href="`/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: period.dateRange.type })}`"
               v-for="period in displayedPeriods">
                <span class="mobile-period-summary-period" role="rowheader">
                    <span class="mobile-period-summary-name" :class="{ 'skeleton-text': loading }">{{ loading ? period.skeletonTitle : tt(period.dateRange.name) }}</span>
                    <span class="mobile-period-summary-range ebk-numeric" :class="{ 'skeleton-text': loading }">{{ loading ? period.skeletonTime : getDisplayDateTime(period.dateRange.type) }}</span>
                </span>
                <span class="mobile-period-summary-amount" role="cell" :class="{ 'skeleton-text': loading }">
                    <span v-if="loading">0.00 GBP</span>
                    <signed-amount type="income" :text="getDisplayIncomeAmount(getOverviewItem(period.dateRange.type)!)"
                                   :hide-sign="!showAmountInHomePage || !!getOverviewItem(period.dateRange.type)?.incomeAmount.isZero()"
                                   v-else-if="getOverviewItem(period.dateRange.type)?.valid" />
                    <span v-else>-</span>
                </span>
                <span class="mobile-period-summary-amount" role="cell" :class="{ 'skeleton-text': loading }">
                    <span v-if="loading">0.00 GBP</span>
                    <signed-amount type="expense" :text="getDisplayExpenseAmount(getOverviewItem(period.dateRange.type)!)"
                                   :hide-sign="!showAmountInHomePage || !!getOverviewItem(period.dateRange.type)?.expenseAmount.isZero()"
                                   v-else-if="getOverviewItem(period.dateRange.type)?.valid" />
                    <span v-else>-</span>
                </span>
                <f7-icon class="mobile-period-summary-chevron-col mobile-overview-row-chevron icon-with-direction" f7="chevron_right" aria-hidden="true"></f7-icon>
            </a>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import { useOverviewStore } from '@/stores/overview.ts';

import { DateRange } from '@/core/datetime.ts';

const props = defineProps<{
    loading: boolean;
    dateRanges: number[];
}>();

const { tt } = useI18n();
const overviewStore = useOverviewStore();

const {
    showAmountInHomePage,
    getOverviewItem,
    getDisplayDateTime,
    getDisplayIncomeAmount,
    getDisplayExpenseAmount
} = usePeriodStatisticsWidgetBase({});

const periods = [
    { dateRange: DateRange.Today, skeletonTitle: 'Today', skeletonTime: 'MM/DD/YYYY' },
    { dateRange: DateRange.ThisWeek, skeletonTitle: 'This Week', skeletonTime: 'MM/DD - MM/DD' },
    { dateRange: DateRange.ThisMonth, skeletonTitle: 'This Month', skeletonTime: 'MM/DD - MM/DD' },
    { dateRange: DateRange.ThisYear, skeletonTitle: 'This Year', skeletonTime: 'YYYY' }
];

const displayedPeriods = computed(() => periods.filter(period => !props.dateRanges || !props.dateRanges.length || props.dateRanges.includes(period.dateRange.type)));
</script>

<style>
.mobile-period-summary-table {
    --mobile-period-summary-columns: minmax(0, 1fr) minmax(0, 6.75rem) minmax(0, 6.75rem) 14px;
    display: flex;
    flex-direction: column;
}

.mobile-period-summary-head,
.mobile-period-summary-row {
    display: grid;
    grid-template-columns: var(--mobile-period-summary-columns);
    align-items: center;
    column-gap: var(--ebk-space-2);
    padding-inline: var(--ebk-space-3) var(--ebk-space-2);
}

.mobile-period-summary-head {
    min-height: 32px;
    border-block-end: 1px solid var(--mobile-overview-panel-hairline);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--mobile-overview-panel-text-secondary);
}

.mobile-period-summary-head > span:nth-child(2),
.mobile-period-summary-head > span:nth-child(3) {
    text-align: end;
    white-space: nowrap;
}

.mobile-period-summary-row {
    min-height: 56px;
    padding-block: var(--ebk-space-2);
    color: inherit;
    text-decoration: none;
}

.mobile-period-summary-row + .mobile-period-summary-row {
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
}

.mobile-period-summary-row:active {
    background-color: var(--ebk-color-selected-background);
}

.mobile-period-summary-period {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
}

.mobile-period-summary-name {
    font-weight: 600;
}

.mobile-period-summary-range {
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.75rem;
    line-height: 1.3;
}

.mobile-period-summary-amount {
    min-width: 0;
    font-size: 0.9375rem;
    font-weight: 600;
    text-align: end;
    font-variant-numeric: var(--ebk-font-numeric);
}

.mobile-period-summary-amount .ebk-signed-amount {
    white-space: normal;
    overflow-wrap: anywhere;
}

.mobile-period-summary-chevron-col {
    justify-self: end;
    margin-inline-start: 0;
}
</style>

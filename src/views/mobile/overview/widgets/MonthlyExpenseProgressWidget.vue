<template>
    <overview-widget-panel :title="title || tt('Monthly Expense Progress')" :show-title="showTitle"
                           :meta="displayDateRange?.thisMonth?.displayTime" :loading="loading">
        <div class="mobile-overview-hero">
            <span class="mobile-overview-label ebk-technical-heading">
                <span class="mobile-overview-label-arrow" aria-hidden="true">↓</span>{{ tt('Expense') }}
            </span>
            <span class="mobile-overview-hero-amount" :class="{ 'skeleton-text': loading }">
                <span v-if="loading">0.00 GBP</span>
                <signed-amount type="expense" :text="currentDisplayExpenseAmount" :hide-sign="isHiddenOrZero(currentExpenseAmount)" v-else-if="currentDisplayExpenseAmount" />
                <span class="mobile-overview-no-data" v-else>{{ tt('No data') }}</span>
            </span>
        </div>

        <div class="mobile-expense-progress-elapsed">
            <div class="mobile-overview-detail-row">
                <span>{{ tt('Month elapsed') }}</span>
                <span>{{ displayElapsedPercent }}</span>
            </div>
            <f7-progressbar :progress="currentMonthElapsedPercent * 100" :aria-label="tt('Month elapsed')"></f7-progressbar>
        </div>

        <div class="mobile-overview-net-line">
            <div class="mobile-overview-detail-row">
                <span>{{ tt('Estimated month-end expense') }}</span>
                <span :class="{ 'skeleton-text': loading }">
                    <signed-amount type="expense" no-color :text="displayEstimatedExpense" :hide-sign="isHiddenOrZero(estimatedExpense)" />
                </span>
            </div>
            <div class="mobile-overview-detail-row margin-top-half">
                <span>{{ tt('Last month total') }}</span>
                <span :class="{ 'skeleton-text': loading }">
                    <signed-amount type="expense" no-color :text="displayLastMonthExpense" :hide-sign="isHiddenOrZero(lastMonthExpense)" />
                </span>
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { type DateTime, DateRange } from '@/core/datetime.ts';

import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';
import { getCurrentDateTime } from '@/lib/datetime.ts';

defineProps<{
    loading: boolean;
    title?: string;
    showTitle: boolean;
}>();

const { tt, formatPercentToLocalizedNumerals } = useI18n();
const {
    showAmountInHomePage,
    displayDateRange,
    transactionOverview,
    currentExpenseAmount,
    currentDisplayExpenseAmount,
    getDisplayAmount
} = usePeriodStatisticsWidgetBase({
    dateType: DateRange.ThisMonth.type
});

const currentDateTime = ref<DateTime>(getCurrentDateTime());

const currentMonthElapsedPercent = computed<number>(() => currentDateTime.value.getGregorianCalendarDay() / currentDateTime.value.getMaxDayOfGregorianCalendarMonth());
const estimatedExpense = computed<BigDecimal>(() => currentExpenseAmount.value.divide(currentMonthElapsedPercent.value));
const lastMonthExpense = computed<BigDecimal>(() => transactionOverview.value.lastMonth?.expenseAmount ?? BIG_DECIMAL_ZERO);

const displayElapsedPercent = formatPercentToLocalizedNumerals(currentMonthElapsedPercent.value * 100.0, 0, '<1');
const displayEstimatedExpense = computed<string>(() => getDisplayAmount(estimatedExpense.value, !!transactionOverview.value.thisMonth?.incompleteExpenseAmount));
const displayLastMonthExpense = computed<string>(() => getDisplayAmount(lastMonthExpense.value, !!transactionOverview.value.lastMonth?.incompleteExpenseAmount));

function isHiddenOrZero(amount: BigDecimal): boolean {
    return !showAmountInHomePage.value || amount.isZero();
}
</script>

<style>
.mobile-expense-progress-elapsed {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-2);
    margin-block-start: var(--ebk-space-3);
}

.mobile-overview-no-data {
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 1rem;
    font-weight: 400;
}
</style>

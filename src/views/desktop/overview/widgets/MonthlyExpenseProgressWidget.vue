<template>
    <overview-widget-panel class="expense-progress-widget"
                           :title="title || tt('Monthly Expense Progress')"
                           :meta="displayDateRange?.thisMonth?.displayTime"
                           :loading="loading" :empty="!hasData">
        <div class="overview-widget__kpi">
            <span class="overview-widget__label ebk-technical-heading">
                <v-icon class="overview-widget__label-icon" :icon="mdiArrowDown" size="14" />{{ tt('Expense') }}
            </span>
            <signed-amount class="overview-widget__hero-amount" type="expense"
                           :text="currentSignedDisplayExpenseAmount.text" :hide-sign="currentSignedDisplayExpenseAmount.hideSign"
                           v-if="hasData" />
            <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-hero" type="heading" width="160px" v-else />
        </div>

        <div class="expense-progress-widget__elapsed">
            <div class="overview-widget__detail-row">
                <span class="overview-widget__label ebk-technical-heading" :id="progressLabelId">{{ tt('Month elapsed') }}</span>
                <span class="overview-widget__meta-value ebk-numeric">{{ displayElapsedPercent }}</span>
            </div>
            <v-progress-linear color="primary" rounded height="4" bg-opacity="0.16"
                               :aria-labelledby="progressLabelId"
                               :model-value="currentMonthElapsedPercent * 100" />
        </div>

        <div class="expense-progress-widget__projection overview-widget__kpi-pair">
            <div class="overview-widget__kpi">
                <span class="overview-widget__label">{{ tt('Estimated month-end expense') }}</span>
                <signed-amount class="overview-widget__row-amount" type="expense"
                               :text="displayEstimatedExpense.text" :hide-sign="displayEstimatedExpense.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin" type="text" width="100px" v-else />
            </div>
            <div class="overview-widget__kpi">
                <span class="overview-widget__label">{{ tt('Last month total') }}</span>
                <signed-amount class="overview-widget__row-amount" type="expense" no-color
                               :text="displayLastMonthExpense.text" :hide-sign="displayLastMonthExpense.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin" type="text" width="100px" v-else />
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { ref, computed, useId } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type SignedDisplayAmount,
    usePeriodStatisticsWidgetBase
} from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { type DateTime, DateRange } from '@/core/datetime.ts';

import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';
import { getCurrentDateTime } from '@/lib/datetime.ts';

import {
    mdiArrowDown
} from '@mdi/js';

defineProps<{
    loading?: boolean;
    editing?: boolean;
    title?: string;
}>();

const { tt, formatPercentToLocalizedNumerals } = useI18n();
const {
    showAmountInHomePage,
    displayDateRange,
    transactionOverview,
    currentOverviewItem,
    currentExpenseAmount,
    currentSignedDisplayExpenseAmount,
    getDisplayAmount
} = usePeriodStatisticsWidgetBase({
    dateType: DateRange.ThisMonth.type
});

const progressLabelId = `expense-progress-${useId()}`;
const currentDateTime = ref<DateTime>(getCurrentDateTime());

const hasData = computed<boolean>(() => !!currentOverviewItem.value?.valid);
const currentMonthElapsedPercent = computed<number>(() => currentDateTime.value.getGregorianCalendarDay() / currentDateTime.value.getMaxDayOfGregorianCalendarMonth());
const estimatedExpense = computed<BigDecimal>(() => currentExpenseAmount.value.divide(currentMonthElapsedPercent.value));
const lastMonthExpense = computed<BigDecimal>(() => transactionOverview.value.lastMonth?.expenseAmount ?? BIG_DECIMAL_ZERO);

const displayElapsedPercent = formatPercentToLocalizedNumerals(currentMonthElapsedPercent.value * 100.0, 0, '<1');
const displayEstimatedExpense = computed<SignedDisplayAmount>(() => toSignedExpense(estimatedExpense.value, !!transactionOverview.value.thisMonth?.incompleteExpenseAmount));
const displayLastMonthExpense = computed<SignedDisplayAmount>(() => toSignedExpense(lastMonthExpense.value, !!transactionOverview.value.lastMonth?.incompleteExpenseAmount));

function toSignedExpense(amount: BigDecimal, incomplete: boolean): SignedDisplayAmount {
    return {
        text: getDisplayAmount(amount, incomplete),
        type: 'expense',
        hideSign: !showAmountInHomePage.value || amount.isZero()
    };
}
</script>

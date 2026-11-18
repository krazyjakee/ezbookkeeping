<template>
    <overview-widget-panel :title="title || currentPeriodTitle" :show-title="showTitle"
                           :meta="currentDisplayDateTime" :loading="loading">
        <div class="mobile-overview-hero">
            <span class="mobile-overview-label ebk-technical-heading">{{ tt('Net Income') }}</span>
            <span class="mobile-overview-hero-amount" :class="{ 'skeleton-text': loading }">
                <span v-if="loading">0.00 GBP</span>
                <signed-amount :type="netType" :text="displayNet" :hide-sign="isHiddenOrZero(currentNetIncomeAmount)" v-else-if="displayNet" />
                <span class="mobile-overview-no-data" v-else>{{ tt('No data') }}</span>
            </span>
        </div>

        <div class="mobile-overview-detail-row margin-top">
            <span>{{ tt('Savings Rate') }}</span>
            <span class="mobile-savings-rate-value" :class="{ 'skeleton-text': loading }">{{ loading ? '00.00%' : (currentDisplaySavingsRate || '-') }}</span>
        </div>

        <div class="mobile-overview-pair">
            <div>
                <span class="mobile-overview-label ebk-technical-heading">
                    <span class="mobile-overview-label-arrow" aria-hidden="true">↑</span>{{ tt('Income') }}
                </span>
                <span class="mobile-overview-pair-amount" :class="{ 'skeleton-text': loading }">
                    <signed-amount type="income" :text="currentDisplayIncomeAmount" :hide-sign="isHiddenOrZero(currentIncomeAmount)" v-if="currentDisplayIncomeAmount" />
                    <span v-else>-</span>
                </span>
            </div>
            <div>
                <span class="mobile-overview-label ebk-technical-heading">
                    <span class="mobile-overview-label-arrow" aria-hidden="true">↓</span>{{ tt('Expense') }}
                </span>
                <span class="mobile-overview-pair-amount" :class="{ 'skeleton-text': loading }">
                    <signed-amount type="expense" :text="currentDisplayExpenseAmount" :hide-sign="isHiddenOrZero(currentExpenseAmount)" v-if="currentDisplayExpenseAmount" />
                    <span v-else>-</span>
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
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import type { BigDecimal } from '@/core/numeral.ts';

const props = defineProps<{
    loading: boolean;
    title?: string;
    showTitle: boolean;
    dateType: number;
}>();

const { tt } = useI18n();
const {
    showAmountInHomePage,
    currentOverviewItem,
    currentPeriodTitle,
    currentDisplayDateTime,
    currentIncomeAmount,
    currentExpenseAmount,
    currentNetIncomeAmount,
    currentDisplayIncomeAmount,
    currentDisplayExpenseAmount,
    currentDisplaySavingsRate,
    getDisplayAmount
} = usePeriodStatisticsWidgetBase(props);

const netType = computed<'income' | 'expense'>(() => currentNetIncomeAmount.value.isNegative() ? 'expense' : 'income');
const displayNet = computed<string>(() => {
    const item = currentOverviewItem.value;

    if (!item?.valid) {
        return '';
    }

    return getDisplayAmount(currentNetIncomeAmount.value.abs(), item.incompleteIncomeAmount || item.incompleteExpenseAmount);
});

function isHiddenOrZero(amount: BigDecimal): boolean {
    return !showAmountInHomePage.value || amount.isZero();
}
</script>

<style>
.mobile-savings-rate-value {
    color: var(--ebk-color-primary-text);
}
</style>

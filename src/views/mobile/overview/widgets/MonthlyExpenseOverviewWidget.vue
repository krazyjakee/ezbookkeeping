<template>
    <overview-widget-panel bracketed class="mobile-monthly-position"
                           :class="`mobile-monthly-position--height-${height || 3}`"
                           :title="tt('Monthly Position')"
                           :meta="displayDateRange?.thisMonth?.displayTime"
                           :loading="loading"
                           :custom-background-color="customBackgroundColor"
                           :custom-text-color="customTextColor">
        <template #actions>
            <f7-link :aria-label="showAmountInHomePage ? tt('Hide Amount') : tt('Show Amount')"
                     @click="showAmountInHomePage = !showAmountInHomePage">
                <f7-icon :f7="showAmountInHomePage ? 'eye_slash' : 'eye'"></f7-icon>
            </f7-link>
        </template>

        <div class="mobile-overview-hero">
            <span class="mobile-overview-label ebk-technical-heading">{{ tt('Net assets') }}</span>
            <span class="mobile-overview-hero-amount ebk-numeric" :class="{ 'skeleton-text': loading }">{{ netAssets }}</span>
        </div>

        <div class="mobile-overview-pair">
            <div>
                <span class="mobile-overview-label ebk-technical-heading">
                    <span class="mobile-overview-label-arrow" aria-hidden="true">↑</span>{{ tt('Income') }}
                </span>
                <span class="mobile-overview-pair-amount" :class="{ 'skeleton-text': loading }">
                    <signed-amount type="income" :text="displayIncome" :hide-sign="isHiddenOrZero(currentIncomeAmount)" v-if="displayIncome" />
                    <span v-else>-</span>
                </span>
            </div>
            <div>
                <span class="mobile-overview-label ebk-technical-heading">
                    <span class="mobile-overview-label-arrow" aria-hidden="true">↓</span>{{ tt('Expense') }}
                </span>
                <span class="mobile-overview-pair-amount" :class="{ 'skeleton-text': loading }">
                    <signed-amount type="expense" :text="displayExpense" :hide-sign="isHiddenOrZero(currentExpenseAmount)" v-if="displayExpense" />
                    <span v-else>-</span>
                </span>
            </div>
        </div>

        <div class="mobile-overview-detail-row mobile-overview-net-line">
            <span class="ebk-technical-heading">{{ tt('Net Cash Flow') }}</span>
            <span :class="{ 'skeleton-text': loading }">
                <signed-amount :type="netType" :text="displayNet" :hide-sign="isHiddenOrZero(currentNetIncomeAmount)" v-if="displayNet" />
                <span v-else>-</span>
            </span>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';
import { useAssetSummaryWidgetBase } from '@/views/base/overview/AssetSummaryWidgetBase.ts';

import { useEnvironmentsStore } from '@/stores/environment.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import type { ColorValue } from '@/core/color.ts';
import { DateRange } from '@/core/datetime.ts';
import {
    DEFAULT_MOBILE_OVERVIEW_WIDGET_LIGHT_BACKGROUND_COLOR,
    DEFAULT_MOBILE_OVERVIEW_WIDGET_DARK_BACKGROUND_COLOR
} from '@/consts/color.ts';

import { getDisplayColor, getContrastTextColor } from '@/lib/color.ts';

const props = defineProps<{
    loading: boolean;
    height: number;
    lightBackgroundColor?: ColorValue;
    darkBackgroundColor?: ColorValue;
}>();

const { tt } = useI18n();

const environmentsStore = useEnvironmentsStore();

const {
    showAmountInHomePage,
    displayDateRange,
    currentOverviewItem,
    currentIncomeAmount,
    currentExpenseAmount,
    currentNetIncomeAmount,
    currentDisplayIncomeAmount,
    currentDisplayExpenseAmount,
    getDisplayAmount
} = usePeriodStatisticsWidgetBase({
    dateType: DateRange.ThisMonth.type
});

const { netAssets } = useAssetSummaryWidgetBase('overview');

const isDarkMode = computed<boolean>(() => environmentsStore.framework7DarkMode || false);
const customBackground = computed<ColorValue | undefined>(() => {
    if (isDarkMode.value) {
        return props.darkBackgroundColor && props.darkBackgroundColor !== DEFAULT_MOBILE_OVERVIEW_WIDGET_DARK_BACKGROUND_COLOR ? props.darkBackgroundColor : undefined;
    } else {
        return props.lightBackgroundColor && props.lightBackgroundColor !== DEFAULT_MOBILE_OVERVIEW_WIDGET_LIGHT_BACKGROUND_COLOR ? props.lightBackgroundColor : undefined;
    }
});
const customBackgroundColor = computed<string | undefined>(() => customBackground.value ? getDisplayColor(customBackground.value) : undefined);
const customTextColor = computed<string | undefined>(() => customBackground.value ? getDisplayColor(getContrastTextColor(customBackground.value)) : undefined);

const displayIncome = computed<string>(() => currentDisplayIncomeAmount.value);
const displayExpense = computed<string>(() => currentDisplayExpenseAmount.value);
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
.mobile-monthly-position--height-1 {
    --mobile-overview-panel-padding: var(--ebk-space-2) var(--ebk-space-3);
}

.mobile-monthly-position--height-3 .mobile-overview-panel-body {
    padding-block: var(--ebk-space-4);
}
</style>

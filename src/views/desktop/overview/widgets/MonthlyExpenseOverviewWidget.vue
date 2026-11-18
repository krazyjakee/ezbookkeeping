<template>
    <overview-widget-panel class="monthly-position-widget" bracketed
                           :title="tt('Monthly Position')"
                           :meta="displayDateRange?.thisMonth?.displayTime"
                           :loading="loading" :empty="!hasData">
        <template #actions>
            <v-btn density="comfortable" size="small" color="default" variant="text"
                   :aria-label="showAmountInHomePage ? tt('Hide Amount') : tt('Show Amount')"
                   :aria-pressed="!showAmountInHomePage ? 'true' : 'false'"
                   :icon="true" @click="showAmountInHomePage = !showAmountInHomePage">
                <v-icon :icon="showAmountInHomePage ? mdiEyeOffOutline : mdiEyeOutline" size="20" />
                <v-tooltip activator="parent">{{ showAmountInHomePage ? tt('Hide Amount') : tt('Show Amount') }}</v-tooltip>
            </v-btn>
            <v-btn density="comfortable" size="small" color="default" variant="text"
                   :aria-label="tt('Refresh')" :icon="true" :disabled="loading" @click="emit('refresh')">
                <v-icon :icon="mdiRefresh" size="20" />
                <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
            </v-btn>
            <v-btn density="comfortable" size="small" color="default" variant="text"
                   :aria-label="tt('View Details')" :icon="true" :to="currentDetailsUrl">
                <v-icon :icon="mdiChevronRight" class="icon-with-direction" size="20" />
                <v-tooltip activator="parent">{{ tt('View Details') }}</v-tooltip>
            </v-btn>
        </template>

        <div class="monthly-position-widget__content">
            <div class="monthly-position-widget__hero">
                <span class="overview-widget__label ebk-technical-heading">
                    <v-icon class="overview-widget__label-icon" :icon="mdiArrowDown" size="14" />{{ tt('Expense') }}
                </span>
                <signed-amount class="overview-widget__hero-amount" type="expense"
                               :text="displayExpense.text" :hide-sign="displayExpense.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-hero" type="heading" width="220px" v-else />
            </div>

            <div class="monthly-position-widget__pair">
                <div class="overview-widget__kpi">
                    <span class="overview-widget__label ebk-technical-heading">
                        <v-icon class="overview-widget__label-icon" :icon="mdiArrowUp" size="14" />{{ tt('Income') }}
                    </span>
                    <signed-amount class="overview-widget__kpi-amount" type="income"
                                   :text="displayIncome.text" :hide-sign="displayIncome.hideSign"
                                   v-if="hasData" />
                    <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-amount" type="text" width="120px" v-else />
                </div>
                <div class="overview-widget__kpi">
                    <span class="overview-widget__label ebk-technical-heading">{{ tt('Net Cash Flow') }}</span>
                    <signed-amount class="overview-widget__kpi-amount" :type="displayNetCashFlow.type"
                                   :text="displayNetCashFlow.text" :hide-sign="displayNetCashFlow.hideSign"
                                   v-if="hasData" />
                    <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-amount" type="text" width="120px" v-else />
                </div>
            </div>
        </div>

    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type SignedDisplayAmount,
    usePeriodStatisticsWidgetBase
} from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import { DateRange } from '@/core/datetime.ts';

import {
    mdiArrowDown,
    mdiArrowUp,
    mdiChevronRight,
    mdiRefresh,
    mdiEyeOutline,
    mdiEyeOffOutline
} from '@mdi/js';

defineProps<{
    loading: boolean;
}>();

const emit = defineEmits<{
    (e: 'refresh'): void
}>();

const { tt } = useI18n();

const {
    showAmountInHomePage,
    displayDateRange,
    currentOverviewItem,
    currentDetailsUrl,
    currentSignedDisplayIncomeAmount,
    currentSignedDisplayExpenseAmount,
    currentSignedDisplayNetIncomeAmount
} = usePeriodStatisticsWidgetBase({
    dateType: DateRange.ThisMonth.type
});

const hasData = computed<boolean>(() => !!currentOverviewItem.value?.valid);
const displayIncome = computed<SignedDisplayAmount>(() => currentSignedDisplayIncomeAmount.value);
const displayExpense = computed<SignedDisplayAmount>(() => currentSignedDisplayExpenseAmount.value);
const displayNetCashFlow = computed<SignedDisplayAmount>(() => currentSignedDisplayNetIncomeAmount.value);
</script>

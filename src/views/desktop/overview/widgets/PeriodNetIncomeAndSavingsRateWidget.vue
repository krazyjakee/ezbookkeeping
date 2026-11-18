<template>
    <overview-widget-panel class="savings-rate-widget"
                           :title="title || currentPeriodTitle"
                           :meta="currentDisplayDateTime"
                           :loading="loading" :empty="!hasData">
        <template #actions v-if="!editing">
            <v-btn density="comfortable" size="small" color="default" variant="text"
                   :aria-label="tt('View Details')" :icon="true" :to="currentDetailsUrl">
                <v-icon :icon="mdiChevronRight" class="icon-with-direction" size="20" />
                <v-tooltip activator="parent">{{ tt('View Details') }}</v-tooltip>
            </v-btn>
        </template>

        <div class="overview-widget__kpi">
            <span class="overview-widget__label ebk-technical-heading">{{ tt('Net Income') }}</span>
            <signed-amount class="overview-widget__hero-amount" :type="currentSignedDisplayNetIncomeAmount.type"
                           :text="currentSignedDisplayNetIncomeAmount.text" :hide-sign="currentSignedDisplayNetIncomeAmount.hideSign"
                           v-if="hasData" />
            <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-hero" type="heading" width="160px" v-else />
        </div>

        <div class="overview-widget__detail-row savings-rate-widget__rate">
            <span class="overview-widget__label ebk-technical-heading">{{ tt('Savings Rate') }}</span>
            <span class="savings-rate-widget__rate-value ebk-numeric" v-if="hasData">{{ currentDisplaySavingsRate || '-' }}</span>
            <v-skeleton-loader class="skeleton-no-margin" type="text" width="64px" v-else />
        </div>

        <div class="overview-widget__kpi-pair overview-widget__kpi-pair--compact">
            <div class="overview-widget__kpi">
                <span class="overview-widget__label ebk-technical-heading">
                    <v-icon class="overview-widget__label-icon" :icon="mdiArrowUp" size="14" />{{ tt('Income') }}
                </span>
                <signed-amount class="overview-widget__row-amount" type="income"
                               :text="currentSignedDisplayIncomeAmount.text" :hide-sign="currentSignedDisplayIncomeAmount.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin" type="text" width="80px" v-else />
            </div>
            <div class="overview-widget__kpi">
                <span class="overview-widget__label ebk-technical-heading">
                    <v-icon class="overview-widget__label-icon" :icon="mdiArrowDown" size="14" />{{ tt('Expense') }}
                </span>
                <signed-amount class="overview-widget__row-amount" type="expense"
                               :text="currentSignedDisplayExpenseAmount.text" :hide-sign="currentSignedDisplayExpenseAmount.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin" type="text" width="80px" v-else />
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { type CommonPeriodStatisticsWidgetProps, usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import {
    mdiArrowDown,
    mdiArrowUp,
    mdiChevronRight
} from '@mdi/js';

interface PeriodNetIncomeAndSavingsRateWidgetProps extends CommonPeriodStatisticsWidgetProps {
    loading?: boolean;
    editing?: boolean;
    title?: string;
}

const props = defineProps<PeriodNetIncomeAndSavingsRateWidgetProps>();

const { tt } = useI18n();
const {
    currentOverviewItem,
    currentPeriodTitle,
    currentDisplayDateTime,
    currentSignedDisplayIncomeAmount,
    currentSignedDisplayExpenseAmount,
    currentSignedDisplayNetIncomeAmount,
    currentDisplaySavingsRate,
    currentDetailsUrl
} = usePeriodStatisticsWidgetBase(props);

const hasData = computed<boolean>(() => !!currentOverviewItem.value?.valid);
</script>

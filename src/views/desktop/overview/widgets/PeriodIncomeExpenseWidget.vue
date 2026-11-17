<template>
    <overview-widget-panel class="period-income-expense-widget"
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

        <div class="overview-widget__kpi-pair">
            <div class="overview-widget__kpi">
                <span class="overview-widget__label ebk-technical-heading">
                    <v-icon class="overview-widget__label-icon" :icon="mdiArrowUp" size="14" />{{ tt('Income') }}
                </span>
                <signed-amount class="overview-widget__kpi-amount" type="income"
                               :text="currentSignedDisplayIncomeAmount.text" :hide-sign="currentSignedDisplayIncomeAmount.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-amount" type="text" width="120px" v-else />
            </div>
            <div class="overview-widget__kpi">
                <span class="overview-widget__label ebk-technical-heading">
                    <v-icon class="overview-widget__label-icon" :icon="mdiArrowDown" size="14" />{{ tt('Expense') }}
                </span>
                <signed-amount class="overview-widget__kpi-amount" type="expense"
                               :text="currentSignedDisplayExpenseAmount.text" :hide-sign="currentSignedDisplayExpenseAmount.hideSign"
                               v-if="hasData" />
                <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-amount" type="text" width="120px" v-else />
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

interface PeriodIncomeExpenseWidgetProps extends CommonPeriodStatisticsWidgetProps {
    loading?: boolean;
    editing?: boolean;
    title?: string;
}

const props = defineProps<PeriodIncomeExpenseWidgetProps>();

const { tt } = useI18n();
const {
    currentOverviewItem,
    currentPeriodTitle,
    currentDisplayDateTime,
    currentSignedDisplayIncomeAmount,
    currentSignedDisplayExpenseAmount,
    currentDetailsUrl
} = usePeriodStatisticsWidgetBase(props);

const hasData = computed<boolean>(() => !!currentOverviewItem.value?.valid);
</script>

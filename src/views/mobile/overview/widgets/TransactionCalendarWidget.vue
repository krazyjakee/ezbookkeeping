<template>
    <overview-widget-panel flush class="mobile-transaction-calendar-panel"
                           :title="title || tt('Transaction Calendar')"
                           :meta="displayDateRange?.thisMonth?.displayTime"
                           :loading="loading">
        <div class="transaction-calendar-container" :class="{ 'mobile-transaction-calendar-busy': loading }">
            <transaction-calendar calendar-class="justify-content-center" week-day-name-type="short"
                                  :readonly="loading || editing" :is-dark-mode="isDarkMode"
                                  :default-currency="false"
                                  :min-date="transactionCalendarMinDate" :max-date="transactionCalendarMaxDate"
                                  :daily-total-amounts="dailyTotalAmounts"
                                  :show-amount="showAmountInCalendar"
                                  :show-income-amount="showIncome"
                                  :show-expense-amount="showExpense"
                                  :show-alternate-date="showAlternateDate"
                                  :model-value="currentCalendarDate"
                                  @update:model-value="selectDate" />
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import { useTransactionCalendarWidgetBase } from '@/views/base/overview/TransactionCalendarWidgetBase.ts';

import { useEnvironmentsStore } from '@/stores/environment.ts';

import type { TextualYearMonthDay } from '@/core/datetime.ts';
import { MobileOverviewWidgetNavigationType } from '@/core/overview_layout.ts';

const props = defineProps<{
    loading: boolean;
    editing?: boolean;
    title?: string;
    totalAmountType?: number;
    transactionTypes: number[];
    showAlternateDate: boolean;
    showAmount: boolean;
}>();

const emit = defineEmits<{
    (e: 'navigate', type: MobileOverviewWidgetNavigationType, path?: string): void;
}>();

const { tt } = useI18n();

const environmentsStore = useEnvironmentsStore();

const { displayDateRange } = usePeriodStatisticsWidgetBase({});

const {
    currentCalendarDate,
    showAmountInCalendar,
    showIncome,
    showExpense,
    dailyTotalAmounts,
    transactionCalendarMinDate,
    transactionCalendarMaxDate,
    getTransactionListUrl
} = useTransactionCalendarWidgetBase(props);

const isDarkMode = computed<boolean>(() => environmentsStore.framework7DarkMode || false);

function selectDate(date: TextualYearMonthDay): void {
    currentCalendarDate.value = date;
    const url = getTransactionListUrl(date);

    if (url) {
        emit('navigate', MobileOverviewWidgetNavigationType.Url, url);
    }
}
</script>

<style>
.mobile-transaction-calendar-panel .transaction-calendar-container {
    padding: var(--ebk-space-1) var(--ebk-space-1) var(--ebk-space-2);
    transition: opacity var(--ebk-motion-duration-sm) var(--ebk-motion-ease-standard);
}

.mobile-transaction-calendar-panel .mobile-transaction-calendar-busy {
    opacity: 0.6;
}

.mobile-transaction-calendar-panel .transaction-calendar-daily-amount.text-income::before {
    content: "↑";
    margin-inline-end: 1px;
}

.mobile-transaction-calendar-panel .transaction-calendar-daily-amount.text-expense::before {
    content: "↓";
    margin-inline-end: 1px;
}

.mobile-transaction-calendar-panel .transaction-calendar-daily-amount {
    font-variant-numeric: var(--ebk-font-numeric);
}
</style>

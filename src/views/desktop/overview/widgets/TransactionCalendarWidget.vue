<template>
    <overview-widget-panel class="transaction-calendar-widget"
                           :title="title || tt('Transaction Calendar')"
                           :meta="displayDateRange?.thisMonth?.displayTime"
                           :loading="loading"
                           body-class="overview-widget__calendar-body transaction-calendar-container">
        <transaction-calendar calendar-class="auto-height-calendar justify-content-center"
                              day-has-transaction-class="font-weight-bold"
                              week-day-name-type="short"
                              :readonly="loading || editing" :is-dark-mode="isDarkMode"
                              :default-currency="defaultCurrency"
                              :min-date="transactionCalendarMinDate" :max-date="transactionCalendarMaxDate"
                              :daily-total-amounts="dailyTotalAmounts"
                              :show-amount="showAmountInCalendar"
                              :show-income-amount="showIncome"
                              :show-expense-amount="showExpense"
                              :show-alternate-date="showAlternateDate"
                              :model-value="currentCalendarDate"
                              @update:model-value="selectDate" />

        <template #footer>
            <span class="overview-widget__summary-label ebk-technical-heading">{{ displaySelectedDate }}</span>
            <template v-if="selectedDayTotal && selectedDayTotal.transactionCount > 0">
                <span class="overview-widget__summary-item" v-if="showIncome">
                    <span class="ebk-visually-hidden">{{ tt('Income') }}</span>
                    <signed-amount type="income" :text="getDisplayDayAmount(selectedDayTotal.income, selectedDayTotal.incompleteIncome)"
                                   :hide-sign="!showAmountInHomePage || selectedDayTotal.income.isZero()" />
                </span>
                <span class="overview-widget__summary-item" v-if="showExpense">
                    <span class="ebk-visually-hidden">{{ tt('Expense') }}</span>
                    <signed-amount type="expense" :text="getDisplayDayAmount(selectedDayTotal.expense, selectedDayTotal.incompleteExpense)"
                                   :hide-sign="!showAmountInHomePage || selectedDayTotal.expense.isZero()" />
                </span>
            </template>
            <span class="overview-widget__summary-item overview-widget__summary-item--muted" v-else-if="!loading">{{ tt('No data') }}</span>
        </template>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useTheme } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { useTransactionCalendarWidgetBase } from '@/views/base/overview/TransactionCalendarWidgetBase.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import type { TransactionTotalAmount } from '@/stores/transaction.ts';

import { ThemeType } from '@/core/theme.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import { type TextualYearMonthDay, KnownDateTimeFormat } from '@/core/datetime.ts';

import { parseDateTimeFromKnownDateTimeFormat } from '@/lib/datetime.ts';

const props = defineProps<{
    loading: boolean;
    editing?: boolean;
    title?: string;
    totalAmountType?: number;
    transactionTypes: number[];
    showAlternateDate: boolean;
    showAmount: boolean;
}>();

const router = useRouter();
const theme = useTheme();

const { tt, formatDateTimeToLongDate } = useI18n();

const {
    currentCalendarDate,
    showAmountInCalendar,
    showIncome,
    showExpense,
    defaultCurrency,
    dailyTotalAmounts,
    transactionCalendarMinDate,
    transactionCalendarMaxDate,
    getTransactionListUrl
} = useTransactionCalendarWidgetBase(props);

const {
    showAmountInHomePage,
    displayDateRange,
    getDisplayAmount
} = usePeriodStatisticsWidgetBase({});

const isDarkMode = computed<boolean>(() => theme.global.name.value === ThemeType.Dark);

// summary of the selected day under the calendar (design guidelines §6.1)
const selectedDayTotal = computed<TransactionTotalAmount | undefined>(() => {
    const day = parseInt(currentCalendarDate.value.split('-')[2] ?? '', 10);
    return Number.isNaN(day) ? undefined : dailyTotalAmounts.value[day];
});

const displaySelectedDate = computed<string>(() => {
    const dateTime = parseDateTimeFromKnownDateTimeFormat(currentCalendarDate.value, KnownDateTimeFormat.DefaultDate);
    return dateTime ? formatDateTimeToLongDate(dateTime) : '';
});

function getDisplayDayAmount(amount: BigDecimal, incomplete: boolean): string {
    return getDisplayAmount(amount, incomplete);
}

function selectDate(date: TextualYearMonthDay): void {
    currentCalendarDate.value = date;
    const url = getTransactionListUrl(date);

    if (url) {
        router.push(url);
    }
}
</script>

<template>
    <overview-widget-panel chart class="overview-widget--chart"
                           :title="displayTitle"
                           :meta="displayMonthCount"
                           :loading="loading" :empty="!monthlyIncomeAndExpenseData.length"
                           :data-table="dataTable"
                           body-class="overview-widget__chart-body">
        <monthly-income-and-expense-chart :data="monthlyIncomeAndExpenseData" :is-dark-mode="isDarkMode"
                                          :loading="loading" :disabled="loading" :enable-click-item="true"
                                          :chart-type="chartType"
                                          :transaction-types="transactionTypes"
                                          :smooth-curve="smoothCurve"
                                          :hide-x-axis-labels="!showXAxisLabels" :hide-legend="!showLegend"
                                          :no-margin="!showLegend && !showXAxisLabels"
                                          @click="clickMonthlyIncomeOrExpense" />

        <template #footer v-if="latestMonth">
            <span class="overview-widget__summary-label ebk-technical-heading">{{ latestMonth.name }}</span>
            <span class="overview-widget__summary-item" v-if="showIncome">
                <span class="ebk-visually-hidden">{{ tt('Income') }}</span>
                <signed-amount type="income" :text="latestMonth.income.text" :hide-sign="latestMonth.income.hideSign" />
            </span>
            <span class="overview-widget__summary-item" v-if="showExpense">
                <span class="ebk-visually-hidden">{{ tt('Expense') }}</span>
                <signed-amount type="expense" :text="latestMonth.expense.text" :hide-sign="latestMonth.expense.hideSign" />
            </span>
        </template>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';
import { type MonthlyIncomeAndExpenseCardClickEvent } from '@/components/desktop/MonthlyIncomeAndExpenseChart.vue';
import type { ChartDataTable, ChartDataTableColumn } from '@/components/base/ChartDataTableBase.ts';

import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useTheme } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useOverviewStore } from '@/stores/overview.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { DateRange } from '@/core/datetime.ts';
import { DISPLAY_HIDDEN_AMOUNT, INCOMPLETE_AMOUNT_SUFFIX } from '@/consts/numeral.ts';
import { ThemeType } from '@/core/theme.ts';
import { TransactionType } from '@/core/transaction.ts';
import {
    type TransactionOverviewData,
    type TransactionMonthlyIncomeAndExpenseData,
    LATEST_12MONTHS_TRANSACTION_AMOUNTS_REQUEST_TYPES
} from '@/models/transaction.ts';

import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';
import { getUnixTimeAfterUnixTime, getUnixTimeBeforeUnixTime, parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

interface SummaryAmount {
    text: string;
    hideSign: boolean;
}

interface LatestMonthSummary {
    name: string;
    income: SummaryAmount;
    expense: SummaryAmount;
}

const props = defineProps<{
    loading: boolean;
    title?: string;
    chartType: number;
    transactionTypes: number[];
    months: number;
    smoothCurve: boolean;
    showXAxisLabels: boolean;
    showLegend: boolean;
}>();

const router = useRouter();
const theme = useTheme();

const { tt, formatAmountToLocalizedNumeralsWithCurrency, formatDateTimeToGregorianLikeLongYearMonth } = useI18n();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const overviewStore = useOverviewStore();

const isDarkMode = computed<boolean>(() => theme.global.name.value === ThemeType.Dark);
const transactionOverview = computed<TransactionOverviewData>(() => overviewStore.transactionOverview);

const showIncome = computed<boolean>(() => props.transactionTypes.includes(TransactionType.Income));
const showExpense = computed<boolean>(() => props.transactionTypes.includes(TransactionType.Expense));
const displayMonthCount = computed<string>(() => props.months > 0 ? tt('format.misc.nMonths', { n: props.months }) : '');

const displayTitle = computed<string>(() => {
    if (props.title) {
        return props.title;
    }

    if (showIncome.value && !showExpense.value) {
        return tt('Income Trends');
    } else if (!showIncome.value && showExpense.value) {
        return tt('Expense Trends');
    } else {
        return tt('Income and Expense Trends');
    }
});

const monthlyIncomeAndExpenseData = computed<TransactionMonthlyIncomeAndExpenseData[]>(() => {
    const data: TransactionMonthlyIncomeAndExpenseData[] = [];

    if (!transactionOverview.value || !transactionOverview.value.thisMonth || !transactionOverview.value.thisMonth.valid) {
        return data;
    }

    for (const amountRequestType of LATEST_12MONTHS_TRANSACTION_AMOUNTS_REQUEST_TYPES) {
        const dateRange = overviewStore.transactionDataRange[amountRequestType];

        if (!dateRange) {
            continue;
        }

        const item = transactionOverview.value[amountRequestType];

        data.push({
            monthStartTime: dateRange.startTime,
            incomeAmount: item?.incomeAmount || BIG_DECIMAL_ZERO,
            expenseAmount: item?.expenseAmount || BIG_DECIMAL_ZERO,
            incompleteIncomeAmount: item ? item.incompleteIncomeAmount : true,
            incompleteExpenseAmount: item ? item.incompleteExpenseAmount : true
        });
    }

    if (props.months <= 0) {
        return data;
    }

    return data.slice(-props.months);
});

// text summary with the exact values of the latest month (design guidelines D7)
const latestMonth = computed<LatestMonthSummary | null>(() => {
    const latest = monthlyIncomeAndExpenseData.value.at(-1);

    if (!latest) {
        return null;
    }

    return {
        name: formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(latest.monthStartTime)),
        income: getSummaryAmount(latest.incomeAmount, latest.incompleteIncomeAmount),
        expense: getSummaryAmount(latest.expenseAmount, latest.incompleteExpenseAmount)
    };
});

// data table with the exact monthly values drawn by the chart (design guidelines D7)
const dataTable = computed<ChartDataTable | null>(() => {
    if (!monthlyIncomeAndExpenseData.value.length) {
        return null;
    }

    const columns: ChartDataTableColumn[] = [
        { key: 'date', name: tt('Date'), numeric: false }
    ];

    if (showIncome.value) {
        columns.push({ key: 'income', name: tt('Income'), numeric: true });
    }

    if (showExpense.value) {
        columns.push({ key: 'expense', name: tt('Expense'), numeric: true });
    }

    return {
        columns: columns,
        rows: monthlyIncomeAndExpenseData.value.map(item => {
            const income = getSummaryAmount(item.incomeAmount, item.incompleteIncomeAmount);
            const expense = getSummaryAmount(item.expenseAmount, item.incompleteExpenseAmount);

            return {
                key: `month-${item.monthStartTime}`,
                name: formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(item.monthStartTime)),
                cells: [
                    ...(showIncome.value ? [{ text: income.text, type: 'income' as const, hideSign: income.hideSign }] : []),
                    ...(showExpense.value ? [{ text: expense.text, type: 'expense' as const, hideSign: expense.hideSign }] : [])
                ]
            };
        })
    };
});

function getSummaryAmount(amount: BigDecimal, incomplete: boolean): SummaryAmount {
    const showAmount = settingsStore.appSettings.showAmountInHomePage;

    return {
        text: showAmount ? formatAmountToLocalizedNumeralsWithCurrency(amount, userStore.currentUserDefaultCurrency) + (incomplete ? INCOMPLETE_AMOUNT_SUFFIX : '') : formatAmountToLocalizedNumeralsWithCurrency(DISPLAY_HIDDEN_AMOUNT, userStore.currentUserDefaultCurrency),
        hideSign: !showAmount || amount.isZero()
    };
}

function clickMonthlyIncomeOrExpense(e: MonthlyIncomeAndExpenseCardClickEvent): void {
    const minTime = e.monthStartTime;
    const maxTime = getUnixTimeBeforeUnixTime(getUnixTimeAfterUnixTime(minTime, 1, 'months'), 1, 'seconds');
    const type = e.transactionType;

    router.push(`/transaction/list?${overviewStore.getTransactionListPageParams({
        type: type,
        dateType: DateRange.Custom.type,
        minTime: minTime,
        maxTime: maxTime
    })}`);
}
</script>

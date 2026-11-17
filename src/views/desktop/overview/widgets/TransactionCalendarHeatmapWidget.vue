<template>
    <overview-widget-panel chart class="overview-widget--chart transaction-calendar-heatmap-card"
                           :title="title || tt('Transaction Calendar Heatmap')"
                           :meta="`${tt(transactionTypeName)} · ${tt('format.misc.nMonths', { n: months })}`"
                           :loading="loading" :empty="!hasData"
                           body-class="overview-widget__chart-body transaction-calendar-heatmap-body">
        <date-range-calendar-heat-map-chart :start-time="startTime" :end-time="endTime"
                                            :items="items" :value-type="ChartValueType.Amount"
                                            :value-type-name="tt(transactionTypeName)" :default-currency="defaultCurrency"
                                            :outlier-color-mode="outlierColorMode"
                                            :outlier-top-count="outlierTopCount"
                                            :outlier-amount-threshold="outlierAmountThreshold"
                                            :show-value="showAmountInHomePage" :enable-click-item="!editing"
                                            @click="clickDate" />

        <template #footer v-if="items.length">
            <span class="overview-widget__summary-label ebk-technical-heading">{{ transactionType === TransactionType.Income ? tt('Total Income') : tt('Total Expense') }}</span>
            <signed-amount class="overview-widget__summary-item"
                           :type="transactionType === TransactionType.Income ? 'income' : 'expense'"
                           :text="displayTotal" :hide-sign="!showAmountInHomePage" />
        </template>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useOverviewStore } from '@/stores/overview.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { DateRange, KnownDateTimeFormat } from '@/core/datetime.ts';
import {
    type CalendarChartSourceDataItem,
    ChartValueType,
    TransactionCalendarHeatmapOutlierColorMode
} from '@/core/chart.ts';
import { TransactionType } from '@/core/transaction.ts';

import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import {
    parseDateTimeFromKnownDateTimeFormat,
    getUnixTimeBeforeUnixTime,
    getUnixTimeAfterUnixTime
} from '@/lib/datetime.ts';

import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';

const props = defineProps<{
    loading: boolean;
    months: number;
    transactionType: TransactionType;
    outlierColorMode?: TransactionCalendarHeatmapOutlierColorMode;
    outlierTopCount?: number;
    outlierAmountThreshold?: number;
    editing?: boolean;
    title?: string
}>();

const router = useRouter();

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const overviewStore = useOverviewStore();
const exchangeRatesStore = useExchangeRatesStore();

const showAmountInHomePage = computed<boolean>(() => settingsStore.appSettings.showAmountInHomePage);
const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
const startTime = computed<number>(() => getUnixTimeAfterUnixTime(getUnixTimeBeforeUnixTime(overviewStore.transactionDataRange.today.startTime, props.months, 'months'), 1, 'days'));
const endTime = computed<number>(() => overviewStore.transactionDataRange.today.endTime);
const transactionTypeName = computed<string>(() => props.transactionType === TransactionType.Income ? 'Income' : 'Expense');

const items = computed<CalendarChartSourceDataItem[]>(() => {
    const result: CalendarChartSourceDataItem[] = [];

    for (const dailyItem of overviewStore.transactionDailyAmountsData) {
        const dateTime = parseDateTimeFromKnownDateTimeFormat(dailyItem.date, KnownDateTimeFormat.DefaultDate);

        if (!dateTime || dateTime.getUnixTime() < startTime.value) {
            continue;
        }

        let total: BigDecimal = BIG_DECIMAL_ZERO;

        for (const amountItem of dailyItem.amounts) {
            let amount: BigDecimal = parseBigDecimal(props.transactionType === TransactionType.Income ? amountItem.incomeAmount : amountItem.expenseAmount);

            if (amountItem.currency !== defaultCurrency.value) {
                const exchangedAmount = exchangeRatesStore.getExchangedAmount(amount, amountItem.currency, defaultCurrency.value);

                if (!exchangedAmount) {
                    continue;
                }

                amount = exchangedAmount.truncate();
            }

            total = total.add(amount);
        }

        if (!total.isZero()) {
            result.push({
                id: dailyItem.date,
                value: total
            });
        }
    }

    return result;
});

const hasData = computed<boolean>(() => overviewStore.transactionDailyAmountsData.length > 0);

// text summary with the exact total of the visible range (design guidelines D7)
const displayTotal = computed<string>(() => {
    let total: BigDecimal = BIG_DECIMAL_ZERO;

    for (const item of items.value) {
        total = total.add(item.value);
    }

    return formatAmountToLocalizedNumeralsWithCurrency(showAmountInHomePage.value ? total.abs() : DISPLAY_HIDDEN_AMOUNT, defaultCurrency.value);
});

function clickDate(date: string): void {
    const dateTime = parseDateTimeFromKnownDateTimeFormat(date, KnownDateTimeFormat.DefaultDate);

    if (!dateTime) {
        return;
    }

    const minTime = dateTime.getUnixTime();
    const maxTime = getUnixTimeBeforeUnixTime(getUnixTimeAfterUnixTime(minTime, 1, 'days'), 1, 'seconds');
    router.push(`/transaction/list?${overviewStore.getTransactionListPageParams({ type: props.transactionType, dateType: DateRange.Custom.type, minTime, maxTime })}`);
}
</script>

<template>
    <overview-widget-panel class="overview-widget--chart net-assets-trend-card"
                           :title="title || tt('Net Assets Trends')"
                           :meta="tt('format.misc.nMonths', { n: months })"
                           :loading="loading" :empty="!hasData"
                           body-class="overview-widget__chart-body">
        <trends-chart hide-x-axis-line hide-y-axis-labels hide-horizontal-grid-lines hide-line-symbols no-animation
                      chart-mode="daily" :type="TrendChartType.Area.type"
                      :start-time="startTime" :end-time="endTime"
                      :start-year-month="undefined" :end-year-month="undefined"
                      :sorting-type="ChartSortingType.Amount.type"
                      :data-aggregation-type="ChartDataAggregationType.Last"
                      :date-aggregation-type="ChartDateAggregationType.Month.type"
                      :fiscal-year-start="fiscalYearStart" :items="items"
                      :value-type="ChartValueType.Amount" :show-value="showAmountInHomePage"
                      :default-currency="defaultCurrency"
                      :smooth-curve="smoothCurve"
                      :hide-legend="!showLegend" legend-position="bottom"
                      :hide-x-axis-labels="!showXAxisLabels"
                      :no-margin="!showLegend && !showXAxisLabels"
                      :enable-click-item="true"
                      @click="clickMonth" />

        <template #footer v-if="hasData">
            <span class="overview-widget__summary-label ebk-technical-heading">{{ tt('Net assets') }}</span>
            <span class="overview-widget__summary-item ebk-numeric">{{ displayLatestNetAssets }}</span>
        </template>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import type { TrendsBarChartClickEvent } from '@/components/base/TrendsChartBase.ts';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useOverviewStore } from '@/stores/overview.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import { entries } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import { ChartValueType } from '@/core/chart.ts';
import { ChartSortingType, ChartDataAggregationType, ChartDateAggregationType, TrendChartType } from '@/core/statistics.ts';
import type { TransactionAssetTrendsAnalysisDataItem, TransactionAssetTrendsAnalysisDataAmount } from '@/models/transaction.ts';

import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import { getUnixTimeBeforeUnixTime } from '@/lib/datetime.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';

const props = defineProps<{
    loading: boolean;
    title?: string;
    months: number;
    smoothCurve: boolean;
    showLegend: boolean;
    showXAxisLabels: boolean;
}>();

const router = useRouter();

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const overviewStore = useOverviewStore();
const exchangeRatesStore = useExchangeRatesStore();

const showAmountInHomePage = computed<boolean>(() => settingsStore.appSettings.showAmountInHomePage);
const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);

const startTime = computed<number>(() => getUnixTimeBeforeUnixTime(overviewStore.transactionDataRange.thisMonth.startTime, props.months - 1, 'months'));
const endTime = computed<number>(() => overviewStore.transactionDataRange.thisMonth.endTime);
const fiscalYearStart = computed<number>(() => userStore.currentUserFiscalYearStart);

const items = computed<TransactionAssetTrendsAnalysisDataItem[]>(() => {
    const lastBalances: Record<string, string> = {};
    const amounts: TransactionAssetTrendsAnalysisDataAmount[] = [];

    for (const dailyItem of overviewStore.transactionAssetTrendsData) {
        for (const accountItem of dailyItem.items) {
            lastBalances[accountItem.accountId] = accountItem.accountClosingBalance;
        }

        let total: BigDecimal = BIG_DECIMAL_ZERO;

        for (const [accountId, balanceText] of entries(lastBalances)) {
            const account = accountsStore.allAccountsMap[accountId];

            if (!account || settingsStore.appSettings.overviewAccountFilterInHomePage[account.id]) {
                continue;
            }

            let balance: BigDecimal = parseBigDecimal(balanceText);

            if (account.currency !== defaultCurrency.value) {
                const exchangedBalance = exchangeRatesStore.getExchangedAmount(balance, account.currency, defaultCurrency.value);

                if (!exchangedBalance) {
                    continue;
                }

                balance = exchangedBalance.truncate();
            }

            if (account.isAsset) {
                total = total.add(balance);
            } else if (account.isLiability) {
                total = total.add(balance);
            }
        }

        const dailyAmount: TransactionAssetTrendsAnalysisDataAmount = {
            year: dailyItem.year,
            month: dailyItem.month,
            day: dailyItem.day,
            value: total
        };

        amounts.push(dailyAmount);
    }

    return [{
        name: tt('Net assets'),
        type: 'total',
        id: 'total',
        icon: '',
        iconType: 0,
        color: '',
        hidden: false,
        displayOrders: [1],
        value: amounts.at(-1)?.value ?? BIG_DECIMAL_ZERO,
        items: amounts
    }];
});

const hasData = computed<boolean>(() => overviewStore.transactionAssetTrendsData.length > 0);

// text summary with the exact latest value of the chart (design guidelines D7)
const displayLatestNetAssets = computed<string>(() => {
    const latest = items.value[0]?.value ?? BIG_DECIMAL_ZERO;
    return formatAmountToLocalizedNumeralsWithCurrency(showAmountInHomePage.value ? latest : DISPLAY_HIDDEN_AMOUNT, defaultCurrency.value);
});

function clickMonth(e: TrendsBarChartClickEvent): void {
    router.push(`/transaction/list?${overviewStore.getTransactionListPageParams({
        dateType: e.dateRange.dateType,
        minTime: e.dateRange.minTime,
        maxTime: e.dateRange.maxTime
    })}`);
}
</script>

<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title>
                <f7-link popover-open=".chart-data-type-popover-menu" :class="{ 'disabled': loading }">
                    <span class="statistics-page-title">{{ queryChartDataTypeName }}</span>
                    <f7-icon class="page-title-bar-icon" color="gray" f7="chevron_down_circle_fill"></f7-icon>
                </f7-link>
            </f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="ellipsis" :aria-label="tt('More')" @click="showMoreActionSheet = true"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-popover class="chart-data-type-popover-menu"
                    @popover:open="scrollPopoverToSelectedItem">
            <f7-list dividers>
                <f7-list-item group-title>
                    <small>{{ tt('Categorical Analysis') }}</small>
                </f7-list-item>
                <f7-list-item link="#" no-chevron popover-close
                              :title="tt(dataType.name)"
                              :class="{ 'list-item-selected': analysisType === StatisticsAnalysisType.CategoricalAnalysis && query.chartDataType === dataType.type }"
                              :key="dataType.type"
                              v-for="dataType in ChartDataType.values(StatisticsAnalysisType.CategoricalAnalysis)"
                              @click="setChartDataType(StatisticsAnalysisType.CategoricalAnalysis, dataType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="analysisType === StatisticsAnalysisType.CategoricalAnalysis && query.chartDataType === dataType.type"></f7-icon>
                    </template>
                </f7-list-item>
                <f7-list-item group-title>
                    <small>{{ tt('Trend Analysis') }}</small>
                </f7-list-item>
                <f7-list-item link="#" no-chevron popover-close
                              :title="tt(dataType.name)"
                              :class="{ 'list-item-selected': analysisType === StatisticsAnalysisType.TrendAnalysis && query.chartDataType === dataType.type }"
                              :key="dataType.type"
                              v-for="dataType in ChartDataType.values(StatisticsAnalysisType.TrendAnalysis)"
                              @click="setChartDataType(StatisticsAnalysisType.TrendAnalysis, dataType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="analysisType === StatisticsAnalysisType.TrendAnalysis && query.chartDataType === dataType.type"></f7-icon>
                    </template>
                </f7-list-item>
                <f7-list-item group-title>
                    <small>{{ tt('Asset Trends') }}</small>
                </f7-list-item>
                <f7-list-item link="#" no-chevron popover-close
                              :title="tt(dataType.name)"
                              :class="{ 'list-item-selected': analysisType === StatisticsAnalysisType.AssetTrends && query.chartDataType === dataType.type }"
                              :key="dataType.type"
                              v-for="dataType in ChartDataType.values(StatisticsAnalysisType.AssetTrends)"
                              @click="setChartDataType(StatisticsAnalysisType.AssetTrends, dataType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="analysisType === StatisticsAnalysisType.AssetTrends && query.chartDataType === dataType.type"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>

        <section class="ebk-panel statistics-period-panel" :class="{ 'disabled': loading }" :aria-label="tt('Date Range')">
            <f7-link class="statistics-period-step" :class="{ 'disabled': reloading || !canShiftDateRange }" :aria-label="tt('Previous Period')" @click="shiftDateRange(-1)">
                <f7-icon class="icon-with-direction" f7="chevron_left"></f7-icon>
            </f7-link>
            <f7-link class="statistics-period-name" :class="{ 'disabled': reloading || !canChangeDateRange }" popover-open=".date-popover-menu">
                <span :class="{ 'tabbar-item-changed': isQueryDateRangeChanged }">{{ queryDateRangeName }}</span>
            </f7-link>
            <f7-link class="statistics-period-step" :class="{ 'disabled': reloading || !canShiftDateRange }" :aria-label="tt('Next Period')" @click="shiftDateRange(1)">
                <f7-icon class="icon-with-direction" f7="chevron_right"></f7-icon>
            </f7-link>
            <f7-link class="statistics-period-aggregation" :class="{ 'disabled': reloading }" popover-open=".date-aggregation-popover-menu"
                     v-if="analysisType === StatisticsAnalysisType.TrendAnalysis">
                <span :class="{ 'tabbar-item-changed': trendDateAggregationType !== ChartDateAggregationType.Default.type }">{{ queryTrendDateAggregationTypeName }}</span>
            </f7-link>
            <f7-link class="statistics-period-aggregation" :class="{ 'disabled': reloading }" popover-open=".date-aggregation-popover-menu"
                     v-if="analysisType === StatisticsAnalysisType.AssetTrends">
                <span :class="{ 'tabbar-item-changed': assetTrendsDateAggregationType !== ChartDateAggregationType.Default.type }">{{ queryAssetTrendsDateAggregationTypeName }}</span>
            </f7-link>
        </section>

        <f7-block class="statistics-analysis-segmented" :class="{ 'disabled': loading || reloading }">
            <f7-segmented strong round>
                <f7-button round :key="item.type" :text="item.name" :active="analysisType === item.type"
                           v-for="item in allAnalysisTypeItems"
                           @click="setAnalysisType(item.type)"></f7-button>
            </f7-segmented>
        </f7-block>

        <section class="ebk-panel statistics-panel" aria-labelledby="statistics-chart-panel-title">
            <header class="ebk-panel-header-strip">
                <span id="statistics-chart-panel-title" class="statistics-panel-title">{{ queryChartDataTypeName }}</span>
                <div class="ebk-panel-header-actions statistics-panel-actions">
                    <f7-segmented class="statistics-chart-type-segmented" v-if="allChartTypes.length">
                        <f7-button small :key="chartType.type" :text="chartType.displayName"
                                   :active="queryChartType === chartType.type"
                                   :class="{ 'disabled': loading }"
                                   v-for="chartType in allChartTypes"
                                   @click="setChartType(chartType.type)"></f7-button>
                    </f7-segmented>
                    <f7-link class="statistics-sort-link" href="#" popover-open=".sorting-type-popover-menu" :class="{ 'disabled': loading }">
                        <span>{{ tt('Sort by') }}</span>&nbsp;<span>{{ querySortingTypeName }}</span>
                    </f7-link>
                </div>
            </header>

            <state-panel type="error" class="statistics-state-panel" :title="tt('Unable to retrieve transaction statistics')"
                         v-if="loadingError">
                <template #actions>
                    <f7-button small fill-false outline @click="retryLoading">{{ tt('Refresh') }}</f7-button>
                </template>
            </state-panel>

            <template v-else-if="analysisType === StatisticsAnalysisType.CategoricalAnalysis && query.categoricalChartType === CategoricalChartType.Pie.type">
                <div class="pie-chart-container">
                    <pie-chart
                        :items="[
                            { name: '---', value: parseBigDecimal(60), color: '7c7c7f' },
                            { name: '---', value: parseBigDecimal(20), color: 'a5a5aa' },
                            { name: '---', value: parseBigDecimal(20), color: 'c5c5c9' }
                        ]"
                        :value-type="ChartValueType.Amount"
                        :skeleton="true"
                        :show-center-text="true"
                        :show-selected-item-info="true"
                        :use-custom-color="true"
                        class="statistics-pie-chart"
                        center-text-background="#cccccc"
                        v-if="loading"
                    ></pie-chart>
                    <pie-chart
                        :items="categoricalAnalysisData.items"
                        :value-type="ChartValueType.Amount"
                        :show-value="showAmountInChart"
                        :show-percent="showPercentInCategoricalChart"
                        :show-center-text="true"
                        :show-selected-item-info="true"
                        :enable-click-item="true"
                        :default-currency="defaultCurrency"
                        class="statistics-pie-chart"
                        v-else-if="!loading && hasCategoricalData"
                        @click="onClickPieChartItem"
                    >
                        <text class="statistics-pie-chart-total-amount-title">
                            {{ totalAmountName }}
                        </text>
                        <text class="statistics-pie-chart-total-amount-value">
                            {{ getDisplayAmount(categoricalAnalysisData.value, defaultCurrency, 16) }}
                        </text>
                    </pie-chart>
                    <state-panel type="empty" class="statistics-state-panel" :title="tt('No transaction data')"
                                 v-else-if="!loading && !hasCategoricalData" />
                </div>
            </template>

            <template v-else-if="analysisType === StatisticsAnalysisType.CategoricalAnalysis && query.categoricalChartType === CategoricalChartType.Bar.type">
                <div class="statistics-panel-total">
                    <span class="ebk-technical-heading">{{ totalAmountName }}</span>
                    <span class="statistics-panel-total-amount" :class="{ 'skeleton-text': loading }" v-if="loading">***.**</span>
                    <signed-amount class="statistics-panel-total-amount"
                                   :text="getSignedAmount(categoricalAnalysisData.value).text"
                                   :type="getSignedAmount(categoricalAnalysisData.value).type"
                                   v-else-if="hasCategoricalData" />
                </div>
                <f7-list dividers class="statistics-ranked-list skeleton-text" v-if="loading">
                    <f7-list-item class="statistics-list-item" link="#" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                        <template #media>
                            <div class="display-flex align-items-center statistics-icon">
                                <f7-icon f7="app_fill"></f7-icon>
                            </div>
                        </template>
                        <template #title>
                            <div class="statistics-list-item-text">
                                <span>Category Name</span>
                                <small class="statistics-percent">33.33</small>
                            </div>
                        </template>
                        <template #after>
                            <span>0.00 USD</span>
                        </template>
                        <template #inner-end>
                            <div class="statistics-item-end">
                                <div class="statistics-percent-line">
                                    <f7-progressbar></f7-progressbar>
                                </div>
                            </div>
                        </template>
                    </f7-list-item>
                </f7-list>

                <state-panel type="empty" class="statistics-state-panel" :title="tt('No transaction data')"
                             v-else-if="!loading && !hasCategoricalData" />

                <f7-list dividers class="statistics-ranked-list" v-else-if="!loading && hasCategoricalData">
                    <template :key="idx" v-for="(item, idx) in categoricalAnalysisData.items">
                        <f7-list-item class="statistics-list-item"
                                      :link="getTransactionItemLinkUrl(item.id)"
                                      v-if="!item.hidden">
                            <template #media>
                                <div class="display-flex align-items-center statistics-icon">
                                    <ItemIcon :icon-type="getIconType(queryChartDataCategory, item.iconType)" :icon-id="item.icon" :color="item.color" v-if="item.icon"></ItemIcon>
                                    <f7-icon f7="pencil_ellipsis_rectangle" v-else-if="!item.icon"></f7-icon>
                                </div>
                            </template>

                            <template #title>
                                <div class="statistics-list-item-text">
                                    <span>{{ item.name }}</span>
                                    <small class="statistics-percent" v-if="showPercentInCategoricalChart && item.percent >= 0 && item.value.isPositiveOrZero()">{{ formatPercentToLocalizedNumerals(item.percent, 2, '<0.01') }}</small>
                                </div>
                            </template>

                            <template #after>
                                <signed-amount :text="getSignedAmount(item.value).text" :type="getSignedAmount(item.value).type" />
                            </template>

                            <template #inner-end>
                                <div class="statistics-item-end">
                                    <div class="statistics-percent-line">
                                        <f7-progressbar :progress="item.percent >= 0 ? item.percent : 0" :style="{ '--f7-progressbar-progress-color': (item.color ? getTransactionCategoricalAnalysisDataItemDisplayColor(item) : '') } "></f7-progressbar>
                                    </div>
                                </div>
                            </template>
                        </f7-list-item>
                    </template>
                </f7-list>
            </template>

            <div class="statistics-trends-chart" v-else-if="analysisType === StatisticsAnalysisType.TrendAnalysis">
                <trends-bar-chart
                    chart-mode="monthly"
                    :loading="loading || reloading"
                    :start-time="undefined"
                    :end-time="undefined"
                    :start-year-month="query.trendChartStartYearMonth"
                    :end-year-month="query.trendChartEndYearMonth"
                    :sorting-type="query.sortingType"
                    :data-aggregation-type="ChartDataAggregationType.Sum"
                    :date-aggregation-type="trendDateAggregationType"
                    :fiscal-year-start="fiscalYearStart"
                    :items="trendsAnalysisData && trendsAnalysisData.items && trendsAnalysisData.items.length ? trendsAnalysisData.items : []"
                    :value-type="ChartValueType.Amount"
                    :stacked="showStackedInTrendsChart"
                    :translate-name="translateNameInTrendsChart"
                    :default-currency="defaultCurrency"
                    @click="onClickTrendChartItem"
                />
            </div>

            <div class="statistics-trends-chart" v-else-if="analysisType === StatisticsAnalysisType.AssetTrends">
                <trends-bar-chart
                    chart-mode="daily"
                    :loading="loading || reloading"
                    :start-time="query.assetTrendsChartStartTime"
                    :end-time="query.assetTrendsChartEndTime"
                    :start-year-month="undefined"
                    :end-year-month="undefined"
                    :sorting-type="query.sortingType"
                    :data-aggregation-type="ChartDataAggregationType.Last"
                    :date-aggregation-type="assetTrendsDateAggregationType"
                    :fiscal-year-start="fiscalYearStart"
                    :items="assetTrendsData && assetTrendsData.items && assetTrendsData.items.length ? assetTrendsData.items : []"
                    :value-type="ChartValueType.Amount"
                    :stacked="showStackedInTrendsChart"
                    :translate-name="translateNameInTrendsChart"
                    :default-currency="defaultCurrency"
                    @click="onClickTrendChartItem"
                />
            </div>
        </section>

        <section class="ebk-panel statistics-panel" aria-labelledby="statistics-ranked-list-title"
                 v-if="!loading && !loadingError && rankedListItems.length">
            <header class="ebk-panel-header-strip">
                <span id="statistics-ranked-list-title">{{ tt('Breakdown') }}</span>
                <span class="statistics-panel-meta">{{ rankedListMeta }}</span>
            </header>
            <ol class="statistics-ranked-rows">
                <li class="statistics-ranked-row" :key="item.key" v-for="(item, idx) in rankedListItems">
                    <span class="statistics-ranked-row-index ebk-numeric">{{ idx + 1 }}</span>
                    <span class="statistics-ranked-row-name">{{ item.name }}</span>
                    <span class="statistics-ranked-row-percent ebk-numeric" v-if="item.percent">{{ item.percent }}</span>
                    <signed-amount class="statistics-ranked-row-amount" :text="item.amount.text" :type="item.amount.type" />
                </li>
            </ol>
        </section>

        <section class="ebk-panel statistics-panel statistics-summary-panel" aria-labelledby="statistics-summary-title"
                 v-if="!loading && !loadingError && (incomeExpenseTotals || summarySentences.length)">
            <header class="ebk-panel-header-strip">
                <span id="statistics-summary-title">{{ tt('Report Summary') }}</span>
                <span class="statistics-panel-meta">{{ queryDateRangeName }}</span>
            </header>
            <div class="statistics-summary-row" v-if="incomeExpenseTotals">
                <div class="statistics-summary-cell">
                    <span class="ebk-technical-heading"><span aria-hidden="true">↑ </span>{{ tt('Income') }}</span>
                    <signed-amount :text="getSignedAmount(incomeExpenseTotals.income, 'income').text" :type="getSignedAmount(incomeExpenseTotals.income, 'income').type" />
                </div>
                <div class="statistics-summary-cell">
                    <span class="ebk-technical-heading"><span aria-hidden="true">↓ </span>{{ tt('Expense') }}</span>
                    <signed-amount :text="getSignedAmount(incomeExpenseTotals.expense, 'expense').text" :type="getSignedAmount(incomeExpenseTotals.expense, 'expense').type" />
                </div>
                <div class="statistics-summary-cell">
                    <span class="ebk-technical-heading">{{ tt('Net Movement') }}</span>
                    <signed-amount :text="getNetAmount(incomeExpenseTotals.net).text" :type="getNetAmount(incomeExpenseTotals.net).type" />
                </div>
            </div>
            <div class="statistics-summary-text" aria-live="polite" v-if="summarySentences.length">
                <p :key="idx" v-for="(sentence, idx) in summarySentences">{{ sentence }}</p>
            </div>
        </section>

        <f7-popover class="sorting-type-popover-menu">
            <f7-list dividers>
                <f7-list-item link="#" no-chevron popover-close
                              :title="sortingType.displayName"
                              :class="{ 'list-item-selected': query.sortingType === sortingType.type }"
                              :key="sortingType.type"
                              v-for="sortingType in allSortingTypes"
                              @click="setSortingType(sortingType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="query.sortingType === sortingType.type"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>


        <f7-popover class="date-popover-menu"
                    @popover:open="scrollPopoverToSelectedItem">
            <f7-list dividers>
                <f7-list-item link="#" no-chevron popover-close
                              :title="dateRange.displayName"
                              :class="{ 'list-item-selected': queryDateType === dateRange.type }"
                              :key="dateRange.type"
                              v-for="dateRange in allDateRanges"
                              @click="setDateFilter(dateRange.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="queryDateType === dateRange.type"></f7-icon>
                    </template>
                    <template #footer>
                        <div v-if="dateRange.isUserCustomRange && canShowCustomDateRange(dateRange.type)">
                            <span>{{ queryStartTime }}</span>
                            <span>&nbsp;-&nbsp;</span>
                            <br/>
                            <span>{{ queryEndTime }}</span>
                        </div>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>

        <f7-popover class="date-aggregation-popover-menu"
                    @popover:open="scrollPopoverToSelectedItem">
            <f7-list dividers v-if="analysisType === StatisticsAnalysisType.TrendAnalysis">
                <f7-list-item link="#" no-chevron popover-close
                              :title="aggregationType.displayName"
                              :class="{ 'list-item-selected': trendDateAggregationType === aggregationType.type }"
                              :key="aggregationType.type"
                              v-for="aggregationType in allTrendAnalysisDateAggregationTypes"
                              @click="setTrendDateAggregationType(aggregationType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="trendDateAggregationType === aggregationType.type"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
            <f7-list dividers v-else-if="analysisType === StatisticsAnalysisType.AssetTrends">
                <f7-list-item link="#" no-chevron popover-close
                              :title="aggregationType.displayName"
                              :class="{ 'list-item-selected': assetTrendsDateAggregationType === aggregationType.type }"
                              :key="aggregationType.type"
                              v-for="aggregationType in allAssetTrendsDateAggregationTypes"
                              @click="setAssetTrendsDateAggregationType(aggregationType.type)">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="assetTrendsDateAggregationType === aggregationType.type"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>

        <date-range-selection-sheet :title="tt('Custom Date Range')"
                                    :min-time="query.categoricalChartStartTime"
                                    :max-time="query.categoricalChartEndTime"
                                    v-model:show="showCustomDateRangeSheet"
                                    @dateRange:change="setCustomDateFilter">
        </date-range-selection-sheet>

        <month-range-selection-sheet :title="tt('Custom Date Range')"
                                     :min-time="query.trendChartStartYearMonth"
                                     :max-time="query.trendChartEndYearMonth"
                                     v-model:show="showCustomMonthRangeSheet"
                                     @dateRange:change="setCustomDateFilter">
        </month-range-selection-sheet>

        <f7-actions close-by-outside-click close-on-escape :opened="showMoreActionSheet" @actions:closed="showMoreActionSheet = false">
            <f7-actions-group>
                <f7-actions-button :class="{ 'disabled': reloading }" @click="filterAccounts">{{ tt('Filter Accounts') }}</f7-actions-button>
                <f7-actions-button :class="{ 'disabled': reloading }" @click="filterCategories" v-if="canUseCategoryFilter">{{ tt('Filter Transaction Categories') }}</f7-actions-button>
                <f7-actions-button :class="{ 'disabled': reloading }" @click="filterTags" v-if="canUseTagFilter">{{ tt('Filter Transaction Tags') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group v-if="canUseKeywordFilter">
                <f7-actions-label v-if="query.keyword">{{ query.keyword }}</f7-actions-label>
                <f7-actions-button :class="{ 'disabled': reloading }" @click="filterDescription">{{ tt('Filter transaction description') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button @click="settings">{{ tt('Settings') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>
    </f7-page>
</template>

<script setup lang="ts">
import SignedAmount from '@/components/common/SignedAmount.vue';
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { type StatisticsAmountKind, useStatisticsTransactionPageBase } from '@/views/base/statistics/StatisticsTransactionPageBase.ts';
import { useChartTextSummary } from '@/components/base/ChartSummaryBase.ts';

import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useStatisticsStore } from '@/stores/statistics.ts';

import type { TypeAndDisplayName } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import { type TextualYearMonth, type TimeRangeAndDateType, DateRangeScene, DateRange } from '@/core/datetime.ts';
import { type CategoricalChartSourceDataItem, ChartValueType } from '@/core/chart.ts';
import {
    ChartDataAggregationType,
    StatisticsAnalysisType,
    CategoricalChartType,
    ChartDataType,
    ChartSortingType,
    ChartDateAggregationType
} from '@/core/statistics.ts';

import { isString, isNumber } from '@/lib/common.ts';
import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import {
    getYearMonthDayDateTime,
    parseDateTimeFromUnixTime,
    getGregorianCalendarYearAndMonthFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    getShiftedDateRangeAndDateType,
    getDateTypeByDateRange,
    getDateRangeByDateType
} from '@/lib/datetime.ts';
import { getIconType } from '@/lib/icon.ts';
import { scrollToSelectedItem } from '@/lib/ui/common.ts';
import { type Framework7Dom, useI18nUIComponents } from '@/lib/ui/mobile.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

interface StatisticsSignedAmount {
    text: string;
    type: 'income' | 'expense' | 'none';
}

interface StatisticsRankedListItem {
    key: string;
    name: string;
    percent: string;
    amount: StatisticsSignedAmount;
}

const {
    tt,
    getAllCategoricalChartTypes,
    formatDateTimeToLongDate,
    formatDateTimeToGregorianLikeShortYearMonth,
    formatPercentToLocalizedNumerals
} = useI18n();

const { getRankedItemsSummary, getSeriesSummary, getIncomeExpenseSummary } = useChartTextSummary();

const { showPrompt, showToast } = useI18nUIComponents();

const {
    loading,
    analysisType,
    trendDateAggregationType,
    assetTrendsDateAggregationType,
    defaultCurrency,
    firstDayOfWeek,
    fiscalYearStart,
    allDateRanges,
    allSortingTypes,
    allTrendAnalysisDateAggregationTypes,
    allAssetTrendsDateAggregationTypes,
    query,
    queryChartDataCategory,
    queryDateType,
    queryStartTime,
    queryEndTime,
    queryDateRangeName,
    queryChartDataTypeName,
    querySortingTypeName,
    queryTrendDateAggregationTypeName,
    queryAssetTrendsDateAggregationTypeName,
    isQueryDateRangeChanged,
    canChangeDateRange,
    canShiftDateRange,
    canUseCategoryFilter,
    canUseTagFilter,
    canUseKeywordFilter,
    showAmountInChart,
    totalAmountName,
    showPercentInCategoricalChart,
    showStackedInTrendsChart,
    translateNameInTrendsChart,
    categoricalAnalysisData,
    trendsAnalysisData,
    assetTrendsData,
    chartDataAmountKind,
    incomeExpenseTotals,
    canShowCustomDateRange,
    getTransactionCategoricalAnalysisDataItemDisplayColor,
    getDisplayAmount
} = useStatisticsTransactionPageBase();

const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const statisticsStore = useStatisticsStore();

const loadingError = ref<unknown | null>(null);
const reloading = ref<boolean>(false);
const showCustomDateRangeSheet = ref<boolean>(false);
const showCustomMonthRangeSheet = ref<boolean>(false);
const showMoreActionSheet = ref<boolean>(false);

const allAnalysisTypeItems = computed<{ type: StatisticsAnalysisType, name: string }[]>(() => [
    { type: StatisticsAnalysisType.CategoricalAnalysis, name: tt('Categories') },
    { type: StatisticsAnalysisType.TrendAnalysis, name: tt('Trends') },
    { type: StatisticsAnalysisType.AssetTrends, name: tt('Asset Trends') }
]);

const hasCategoricalData = computed<boolean>(() => !!categoricalAnalysisData.value && !!categoricalAnalysisData.value.items && categoricalAnalysisData.value.items.length > 0);

// ranked list with exact values that accompanies the chart (design guideline D7)
const rankedListItems = computed<StatisticsRankedListItem[]>(() => {
    const result: StatisticsRankedListItem[] = [];

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        // the bar chart type is already a ranked list with exact values
        if (query.value.categoricalChartType !== CategoricalChartType.Pie.type || !hasCategoricalData.value) {
            return result;
        }

        for (const item of categoricalAnalysisData.value.items) {
            if (item.hidden) {
                continue;
            }

            result.push({
                key: item.id,
                name: item.name,
                percent: showPercentInCategoricalChart.value && item.percent >= 0 && item.value.isPositiveOrZero() ? formatPercentToLocalizedNumerals(item.percent, 2, '<0.01') : '',
                amount: getSignedAmount(item.value)
            });
        }
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && trendsAnalysisData.value) {
        // trends are flows: each series is ranked by its total of the selected period
        const items = trendsAnalysisData.value.items.filter(item => !item.hidden);
        let positiveTotal: BigDecimal = BIG_DECIMAL_ZERO;

        for (const item of items) {
            if (item.value.isPositive()) {
                positiveTotal = positiveTotal.add(item.value);
            }
        }

        for (const item of items) {
            const percent = positiveTotal.isPositive() && item.value.isPositive() ? item.value.divide(positiveTotal).multiply(100).toDoubleNumber() : -1;

            result.push({
                key: item.id,
                name: translateNameInTrendsChart.value ? tt(item.name) : item.name,
                percent: percent >= 0 && items.length > 1 ? formatPercentToLocalizedNumerals(percent, 2, '<0.01') : '',
                amount: getSignedAmount(item.value)
            });
        }
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && assetTrendsData.value) {
        // balances are stocks: each series shows its latest value, never a sum of the period
        for (const item of assetTrendsData.value.items) {
            if (item.hidden || !item.items.length) {
                continue;
            }

            const latestValue = item.items[item.items.length - 1]!.value;

            result.push({
                key: item.id,
                name: translateNameInTrendsChart.value ? tt(item.name) : item.name,
                percent: '',
                amount: { text: getDisplayAmount(latestValue, defaultCurrency.value), type: 'none' }
            });
        }
    }

    return result;
});

const rankedListMeta = computed<string>(() => {
    if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        const latestDates: string[] = [];

        for (const item of assetTrendsData.value?.items ?? []) {
            const latest = item.items[item.items.length - 1];

            if (!item.hidden && latest) {
                latestDates.push(formatDateTimeToLongDate(getYearMonthDayDateTime(latest.year, latest.month, latest.day)));
                break;
            }
        }

        return latestDates[0] ?? '';
    }

    return queryDateRangeName.value;
});

const summarySentences = computed<string[]>(() => {
    const sentences: string[] = [];
    const formatValue = (value: BigDecimal) => getDisplayAmount(value, defaultCurrency.value);

    if (incomeExpenseTotals.value) {
        sentences.push(getIncomeExpenseSummary(
            getSignedAmountText(getSignedAmount(incomeExpenseTotals.value.income, 'income')),
            getSignedAmountText(getSignedAmount(incomeExpenseTotals.value.expense, 'expense')),
            getSignedAmountText(getNetAmount(incomeExpenseTotals.value.net))
        ));
    }

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis && hasCategoricalData.value) {
        sentences.push(getRankedItemsSummary(categoricalAnalysisData.value.items.filter(item => !item.hidden), {
            totalName: totalAmountName.value,
            formatValue: formatValue,
            showPercent: showPercentInCategoricalChart.value
        }));
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && trendsAnalysisData.value) {
        const items = trendsAnalysisData.value.items.filter(item => !item.hidden);

        if (items.length === 1 && items[0]) {
            sentences.push(getSeriesSummary(translateNameInTrendsChart.value ? tt(items[0].name) : items[0].name,
                items[0].items.map(dataItem => ({ label: formatDateTimeToGregorianLikeShortYearMonth(parseDateTimeFromUnixTime(getYearMonthFirstUnixTime(dataItem))), value: dataItem.value })),
                formatValue));
        } else if (items.length > 1) {
            sentences.push(getRankedItemsSummary(items.map(item => ({ name: translateNameInTrendsChart.value ? tt(item.name) : item.name, value: item.value })), {
                totalName: totalAmountName.value,
                formatValue: formatValue,
                showPercent: false
            }));
        }
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && assetTrendsData.value) {
        const items = assetTrendsData.value.items.filter(item => !item.hidden);

        if (items.length === 1 && items[0] && items[0].items.length) {
            sentences.push(getSeriesSummary(translateNameInTrendsChart.value ? tt(items[0].name) : items[0].name,
                [items[0].items[items[0].items.length - 1]!].map(dataItem => ({ label: formatDateTimeToLongDate(getYearMonthDayDateTime(dataItem.year, dataItem.month, dataItem.day)), value: dataItem.value })),
                formatValue, 'date'));
        }
    }

    return sentences.filter(sentence => !!sentence);
});

const allChartTypes = computed<TypeAndDisplayName[]>(() => {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        return getAllCategoricalChartTypes();
    } else {
        return [];
    }
});

const queryChartType = computed<number | undefined>({
    get: () => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return query.value.categoricalChartType;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return query.value.trendChartType;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return query.value.assetTrendsChartType;
        } else {
            return undefined;
        }
    },
    set: (value: number | undefined) => {
        setChartType(value);
    }
});

function getSignedAmount(value: BigDecimal, kind?: StatisticsAmountKind): StatisticsSignedAmount {
    const amountKind = kind ?? chartDataAmountKind.value;

    if (amountKind === 'none' || value.isZero()) {
        return { text: getDisplayAmount(value, defaultCurrency.value), type: 'none' };
    }

    if (value.isNegative()) {
        return { text: getDisplayAmount(value.negate(), defaultCurrency.value), type: amountKind === 'income' ? 'expense' : 'income' };
    }

    return { text: getDisplayAmount(value, defaultCurrency.value), type: amountKind };
}

function getNetAmount(value: BigDecimal): StatisticsSignedAmount {
    return getSignedAmount(value, value.isZero() ? 'none' : 'income');
}

function getSignedAmountText(amount: StatisticsSignedAmount): string {
    if (amount.type === 'income') {
        return `+${amount.text}`;
    } else if (amount.type === 'expense') {
        return `\u2212${amount.text}`;
    }

    return amount.text;
}

function setAnalysisType(type: StatisticsAnalysisType): void {
    if (analysisType.value === type) {
        return;
    }

    let chartDataType = query.value.chartDataType;

    if (!ChartDataType.isAvailableForAnalysisType(chartDataType, type)) {
        chartDataType = type === StatisticsAnalysisType.AssetTrends ? ChartDataType.DefaultForAssetTrends.type : ChartDataType.Default.type;
    }

    setChartDataType(type, chartDataType);
}

function retryLoading(): void {
    loadingError.value = null;
    loading.value = true;
    init();
}

function getTransactionItemLinkUrl(itemId: string, dateRange?: TimeRangeAndDateType): string {
    return `/transaction/list?${statisticsStore.getTransactionListPageParams(analysisType.value, itemId, dateRange)}`;
}

function init(): void {
    statisticsStore.initTransactionStatisticsFilter(analysisType.value);

    Promise.all([
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false })
    ]).then(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return statisticsStore.loadCategoricalAnalysis({
                force: false
            }) as Promise<unknown>;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return statisticsStore.loadTrendAnalysis({
                force: false
            }) as Promise<unknown>;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return statisticsStore.loadAssetTrends({
                force: false
            }) as Promise<unknown>;
        } else {
            return Promise.reject('An error occurred');
        }
    }).then(() => {
        loading.value = false;
    }).catch(error => {
        if (error.processed) {
            loading.value = false;
        } else {
            loadingError.value = error;
            showToast(error.message || error);
        }
    });
}

function reload(done?: () => void): void {
    const force = !!done;
    let dispatchPromise: Promise<unknown> | null = null;

    reloading.value = true;

    if (query.value.chartDataType === ChartDataType.OutflowsByAccount.type ||
        query.value.chartDataType === ChartDataType.ExpenseByAccount.type ||
        query.value.chartDataType === ChartDataType.ExpenseByPrimaryCategory.type ||
        query.value.chartDataType === ChartDataType.ExpenseBySecondaryCategory.type ||
        query.value.chartDataType === ChartDataType.InflowsByAccount.type ||
        query.value.chartDataType === ChartDataType.IncomeByAccount.type ||
        query.value.chartDataType === ChartDataType.IncomeByPrimaryCategory.type ||
        query.value.chartDataType === ChartDataType.IncomeBySecondaryCategory.type ||
        query.value.chartDataType === ChartDataType.TotalOutflows.type ||
        query.value.chartDataType === ChartDataType.TotalExpense.type ||
        query.value.chartDataType === ChartDataType.TotalInflows.type ||
        query.value.chartDataType === ChartDataType.TotalIncome.type ||
        query.value.chartDataType === ChartDataType.NetCashFlow.type ||
        query.value.chartDataType === ChartDataType.NetIncome.type ||
        query.value.chartDataType === ChartDataType.NetWorth.type) {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            dispatchPromise = statisticsStore.loadCategoricalAnalysis({
                force: force
            });
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            dispatchPromise = statisticsStore.loadTrendAnalysis({
                force: force
            });
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            dispatchPromise = statisticsStore.loadAssetTrends({
                force: force
            });
        }
    } else if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
        query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
        query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
        query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            dispatchPromise = accountsStore.loadAllAccounts({
                force: force
            });
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            dispatchPromise = statisticsStore.loadAssetTrends({
                force: force
            });
        }
    }

    if (dispatchPromise) {
        dispatchPromise.then(() => {
            reloading.value = false;

            done?.();

            if (force) {
                showToast('Data has been updated');
            }
        }).catch(error => {
            reloading.value = false;

            done?.();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    } else {
        reloading.value = false;
    }
}

function setChartType(type?: number): void {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        statisticsStore.updateTransactionStatisticsFilter({
            categoricalChartType: type
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        statisticsStore.updateTransactionStatisticsFilter({
            trendChartType: type
        });
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartType: type
        });
    }
}

function setChartDataType(type: number, chartDataType: number): void {
    let analysisTypeChanged = false;

    if (analysisType.value !== type) {
        if (!ChartDataType.isAvailableForAnalysisType(query.value.chartDataType, type)) {
            statisticsStore.updateTransactionStatisticsFilter({
                chartDataType: ChartDataType.Default.type
            });
        }

        analysisType.value = type;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        analysisTypeChanged = true;
    }

    statisticsStore.updateTransactionStatisticsFilter({
        chartDataType: chartDataType
    });

    if (analysisTypeChanged) {
        reload();
    }
}

function setSortingType(type: number): void {
    if (type < ChartSortingType.Amount.type || type > ChartSortingType.Name.type) {
        return;
    }

    statisticsStore.updateTransactionStatisticsFilter({
        sortingType: type
    });
}

function setTrendDateAggregationType(type: number): void {
    trendDateAggregationType.value = type;
}

function setAssetTrendsDateAggregationType(type: number): void {
    assetTrendsDateAggregationType.value = type;
}

function setDateFilter(dateType: number): void {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomDateRangeSheet.value = true;
            return;
        } else if (query.value.categoricalChartDateType === dateType) {
            return;
        }
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomMonthRangeSheet.value = true;
            return;
        } else if (query.value.trendChartDateType === dateType) {
            return;
        }
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomDateRangeSheet.value = true;
            return;
        } else if (query.value.assetTrendsChartDateType === dateType) {
            return;
        }
    }

    const dateRange = getDateRangeByDateType(dateType, firstDayOfWeek.value, fiscalYearStart.value);

    if (!dateRange) {
        return;
    }

    let changed = false;

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            categoricalChartDateType: dateRange.dateType,
            categoricalChartStartTime: dateRange.minTime,
            categoricalChartEndTime: dateRange.maxTime
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            trendChartDateType: dateRange.dateType,
            trendChartStartYearMonth: getGregorianCalendarYearAndMonthFromUnixTime(dateRange.minTime),
            trendChartEndYearMonth: getGregorianCalendarYearAndMonthFromUnixTime(dateRange.maxTime)
        });
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartDateType: dateRange.dateType,
            assetTrendsChartStartTime: dateRange.minTime,
            assetTrendsChartEndTime: dateRange.maxTime
        });
    }

    if (changed) {
        reload();
    }
}

function setCustomDateFilter(startTime: number | TextualYearMonth, endTime: number | TextualYearMonth): void {
    if (!startTime || !endTime) {
        return;
    }

    let changed = false;

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis && isNumber(startTime) && isNumber(endTime)) {
        const chartDateType = getDateTypeByDateRange(startTime, endTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            categoricalChartDateType: chartDateType,
            categoricalChartStartTime: startTime,
            categoricalChartEndTime: endTime
        });

        showCustomDateRangeSheet.value = false;
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && isString(startTime) && isString(endTime)) {
        const chartDateType = getDateTypeByDateRange(getYearMonthFirstUnixTime(startTime), getYearMonthLastUnixTime(endTime), firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.TrendAnalysis);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            trendChartDateType: chartDateType,
            trendChartStartYearMonth: startTime,
            trendChartEndYearMonth: endTime
        });

        showCustomMonthRangeSheet.value = false;
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && isNumber(startTime) && isNumber(endTime)) {
        const chartDateType = getDateTypeByDateRange(startTime, endTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.AssetTrends);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartDateType: chartDateType,
            assetTrendsChartStartTime: startTime,
            assetTrendsChartEndTime: endTime
        });

        showCustomDateRangeSheet.value = false;
    }

    if (changed) {
        reload();
    }
}

function shiftDateRange(scale: number): void {
    let changed = false;

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        if (query.value.categoricalChartDateType === DateRange.All.type) {
            return;
        }

        const newDateRange = getShiftedDateRangeAndDateType(query.value.categoricalChartStartTime, query.value.categoricalChartEndTime, scale, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            categoricalChartDateType: newDateRange.dateType,
            categoricalChartStartTime: newDateRange.minTime,
            categoricalChartEndTime: newDateRange.maxTime
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        const newDateRange = getShiftedDateRangeAndDateType(getYearMonthFirstUnixTime(query.value.trendChartStartYearMonth), getYearMonthLastUnixTime(query.value.trendChartEndYearMonth), scale, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.TrendAnalysis);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            trendChartDateType: newDateRange.dateType,
            trendChartStartYearMonth: getGregorianCalendarYearAndMonthFromUnixTime(newDateRange.minTime),
            trendChartEndYearMonth: getGregorianCalendarYearAndMonthFromUnixTime(newDateRange.maxTime)
        });
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        if (query.value.assetTrendsChartDateType === DateRange.All.type) {
            return;
        }

        const newDateRange = getShiftedDateRangeAndDateType(query.value.assetTrendsChartStartTime, query.value.assetTrendsChartEndTime, scale, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.AssetTrends);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartDateType: newDateRange.dateType,
            assetTrendsChartStartTime: newDateRange.minTime,
            assetTrendsChartEndTime: newDateRange.maxTime
        });
    }

    if (changed) {
        reload();
    }
}

function filterAccounts(): void {
    props.f7router.navigate('/settings/filter/account?type=statisticsCurrent');
}

function filterCategories(): void {
    props.f7router.navigate('/settings/filter/category?type=statisticsCurrent');
}

function filterTags(): void {
    props.f7router.navigate('/settings/filter/tag?type=statisticsCurrent');
}

function filterDescription(): void {
    if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        return;
    }

    showPrompt('Filter transaction description', query.value.keyword, value => {
        if (query.value.keyword === value) {
            return;
        }

        let changed = false;

        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            changed = statisticsStore.updateTransactionStatisticsFilter({
                keyword: value
            });
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            changed = statisticsStore.updateTransactionStatisticsFilter({
                keyword: value
            });
        }

        if (changed) {
            reload();
        }
    });
}

function settings(): void {
    props.f7router.navigate('/statistic/settings');
}

function scrollPopoverToSelectedItem(event: { $el: Framework7Dom }): void {
    scrollToSelectedItem(event.$el[0], '.popover-inner', '.popover-inner', 'li.list-item-selected');
}

function onClickPieChartItem(item: CategoricalChartSourceDataItem): void {
    if (item.id) {
        props.f7router.navigate(getTransactionItemLinkUrl(item.id));
    }
}

function onClickTrendChartItem(item: { itemId: string, dateRange: TimeRangeAndDateType }): void {
    props.f7router.navigate(getTransactionItemLinkUrl(item.itemId, item.dateRange));
}

function onPageAfterIn(): void {
    if (statisticsStore.transactionStatisticsStateInvalid && !loading.value) {
        reload();
    }

    // a failed load stays on this page and shows the error state panel with a retry action (guideline 4.6)
}

init();
</script>

<style>
.statistics-page-title {
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--f7-text-color);
}

.card-header.no-border:after {
    display: none;
}

.statistics-chart-header {
    font-size: var(--f7-list-item-header-font-size);
}

.statistics-pie-chart .pie-chart-text-group {
    fill: var(--ebk-color-text-primary);
    font-variant-numeric: tabular-nums;
    text-anchor: middle;
}

.statistics-pie-chart-total-amount-title {
    -moz-transform: translateY(0.5em);
    -ms-transform: translateY(0.5em);
    -webkit-transform: translateY(0.5em);
    transform: translateY(0.5em);
}

.statistics-pie-chart-total-amount-value {
    -moz-transform: translateY(2em);
    -ms-transform: translateY(2em);
    -webkit-transform: translateY(2em);
    transform: translateY(2em);
}

.statistics-pie-chart-total-no-data {
    -moz-transform: translateY(1.5em);
    -ms-transform: translateY(1.5em);
    -webkit-transform: translateY(1.5em);
    transform: translateY(1.5em);
}

.chart-data-type-popover-menu .popover-inner {
    max-height: 440px;
    overflow-y: auto;
}

/* Instrument panel layout (#32) */
.statistics-period-panel {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-1);
    margin: var(--ebk-space-3) var(--ebk-space-3) 0;
    padding: var(--ebk-space-1) var(--ebk-space-2);
    min-height: 52px;
}

.statistics-period-panel .statistics-period-step {
    flex: 0 0 auto;
    min-width: 44px;
    min-height: 44px;
    justify-content: center;
}

.statistics-period-panel .statistics-period-name {
    flex: 1 1 auto;
    min-width: 0;
    justify-content: center;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
}

.statistics-period-panel .statistics-period-name > span,
.statistics-period-panel .statistics-period-aggregation > span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.statistics-period-panel .statistics-period-aggregation {
    flex: 0 1 auto;
    min-width: 0;
    max-width: 40%;
    min-height: 44px;
    padding-inline: var(--ebk-space-2);
    border-inline-start: 1px solid var(--ebk-color-hairline);
    font-size: 0.875rem;
}

.statistics-analysis-segmented.block {
    margin: var(--ebk-space-3) var(--ebk-space-3) 0;
    padding: 0;
}

.statistics-panel {
    margin: var(--ebk-space-3) var(--ebk-space-3) 0;
    overflow: hidden;
}

.statistics-panel:last-of-type {
    margin-block-end: var(--ebk-space-4);
}

.statistics-panel .ebk-panel-header-strip {
    flex-wrap: wrap;
    padding-block: var(--ebk-space-1);
}

.statistics-panel-title {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
}

.statistics-panel-meta {
    color: var(--ebk-color-text-secondary);
    font-weight: 400;
    letter-spacing: normal !important;
    text-transform: none !important;
    font-variant-numeric: tabular-nums;
}

.statistics-panel-actions {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-2);
    font-size: 0.75rem;
    font-weight: 400;
}

.statistics-chart-type-segmented.segmented {
    width: auto;
}

.statistics-chart-type-segmented .button {
    min-height: 32px;
    padding-inline: var(--ebk-space-2);
    font-size: 0.75rem;
}

.statistics-sort-link {
    min-height: 32px;
    white-space: nowrap;
}

.statistics-state-panel {
    margin: var(--ebk-space-3);
}

.statistics-panel .pie-chart-container {
    padding-block: var(--ebk-space-2);
}

.statistics-panel-total {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--ebk-space-2);
    padding: var(--ebk-space-3) var(--ebk-space-3) 0;
    color: var(--ebk-color-text-secondary);
}

.statistics-panel-total-amount {
    color: var(--ebk-color-text-primary);
    font-size: 1.25rem;
    font-weight: 600;
}

.statistics-panel .list.statistics-ranked-list,
.statistics-trends-chart .list {
    margin: 0;
    --f7-list-inset-side-margin: 0;
    --f7-list-strong-bg-color: transparent;
    --f7-list-outline-border-color: transparent;
}

.statistics-ranked-rows {
    margin: 0;
    padding: 0;
    list-style: none;
}

.statistics-ranked-row {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-2);
    min-height: 52px;
    padding: var(--ebk-space-2) var(--ebk-space-3);
    border-block-end: 1px solid var(--ebk-color-hairline);
}

.statistics-ranked-row:last-child {
    border-block-end: 0;
}

.statistics-ranked-row-index {
    flex: 0 0 1.5rem;
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
}

.statistics-ranked-row-name {
    flex: 1 1 auto;
    min-width: 0;
    overflow-wrap: anywhere;
}

.statistics-ranked-row-percent {
    flex: 0 0 auto;
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
}

.statistics-ranked-row-amount {
    flex: 0 0 auto;
    font-weight: 600;
}

.statistics-summary-row {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
}

.statistics-summary-cell {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-1);
    min-width: 0;
    padding: var(--ebk-space-3);
    color: var(--ebk-color-text-secondary);
}

.statistics-summary-cell + .statistics-summary-cell {
    border-inline-start: 1px solid var(--ebk-color-hairline);
}

.statistics-summary-cell > .ebk-signed-amount {
    font-size: 1rem;
    font-weight: 600;
    white-space: normal;
    overflow-wrap: anywhere;
}

.statistics-summary-text {
    padding: var(--ebk-space-3);
    border-block-start: 1px solid var(--ebk-color-hairline);
    line-height: 1.5;
}

.statistics-summary-text > p {
    margin: 0;
}

.statistics-summary-text > p + p {
    margin-block-start: var(--ebk-space-2);
}
</style>

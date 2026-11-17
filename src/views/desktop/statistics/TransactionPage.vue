<template>
    <main-page-layout>
        <template #nav-items>
            <li class="nav-section-title">
                <div class="title-wrapper">
                    <span class="title-text" id="statistics-report-view-title">{{ tt('Report View') }}</span>
                </div>
            </li>
            <li class="nav-link" :key="analysisTypeItem.type" v-for="analysisTypeItem in allAnalysisTypeItems">
                <a class="d-flex align-center cursor-pointer" role="button" tabindex="0"
                   :class="{ 'router-link-active router-link-exact-active': queryAnalysisType === analysisTypeItem.type, 'disabled': loading }"
                   :aria-current="queryAnalysisType === analysisTypeItem.type ? 'page' : undefined"
                   :aria-disabled="loading ? 'true' : undefined"
                   @click="!loading && (queryAnalysisType = analysisTypeItem.type)"
                   @keydown.enter.prevent="!loading && (queryAnalysisType = analysisTypeItem.type)"
                   @keydown.space.prevent="!loading && (queryAnalysisType = analysisTypeItem.type)">
                    <v-icon class="nav-item-icon" :icon="analysisTypeItem.icon" size="18" />
                    <span class="nav-item-title">{{ analysisTypeItem.name }}</span>
                </a>
            </li>
            <li class="statistics-report-options-item">
                <section class="ebk-panel statistics-report-options" aria-labelledby="statistics-report-options-title">
                    <header class="ebk-panel-header-strip">
                        <span id="statistics-report-options-title">{{ tt('Report Options') }}</span>
                    </header>
                    <div class="statistics-report-options-body">
                        <label class="statistics-report-option-label ebk-technical-heading" for="statistics-chart-type">{{ tt('Chart Type') }}</label>
                        <v-select
                            id="statistics-chart-type"
                            item-title="displayName"
                            item-value="type"
                            density="compact"
                            hide-details
                            :disabled="loading"
                            :items="allChartTypes"
                            v-model="queryChartType"
                            v-if="!isQuerySpecialChartType"
                        />
                        <v-select
                            id="statistics-chart-type"
                            item-title="displayName"
                            item-value="type"
                            density="compact"
                            hide-details
                            :disabled="true"
                            :items="[{ displayName: tt('Sankey Chart'), type: 0 }]"
                            :model-value="0"
                            v-else-if="queryChartDataType === ChartDataType.Overview.type"
                        />
                        <label class="statistics-report-option-label ebk-technical-heading" for="statistics-sort-order">{{ tt('Sort Order') }}</label>
                        <v-select
                            id="statistics-sort-order"
                            item-title="displayName"
                            item-value="type"
                            density="compact"
                            hide-details
                            :disabled="loading"
                            :items="allSortingTypes"
                            v-model="querySortingType"
                        />
                        <span class="statistics-report-option-label ebk-technical-heading">{{ tt('Chart Data Type') }}</span>
                        <ul class="statistics-report-data-types">
                            <li class="nav-link" :key="dataType.type"
                                v-for="dataType in ChartDataType.values(undefined, true)"
                                v-show="dataType.isAvailableAnalysisType(queryAnalysisType)">
                                <a class="d-flex align-center cursor-pointer" role="button" tabindex="0"
                                   :class="{ 'router-link-active router-link-exact-active': queryChartDataType === dataType.type, 'disabled': loading }"
                                   :aria-current="queryChartDataType === dataType.type ? 'true' : undefined"
                                   :aria-disabled="loading ? 'true' : undefined"
                                   @click="!loading && (queryChartDataType = dataType.type)"
                                   @keydown.enter.prevent="!loading && (queryChartDataType = dataType.type)"
                                   @keydown.space.prevent="!loading && (queryChartDataType = dataType.type)">
                                    <span class="nav-item-title text-truncate">{{ tt(dataType.name) }}</span>
                                    <v-tooltip activator="parent" location="right">{{ tt(dataType.name) }}</v-tooltip>
                                </a>
                            </li>
                        </ul>
                    </div>
                </section>
            </li>
        </template>

        <template #content>
            <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                <v-window-item value="statisticsPage">
                    <div class="statistics-report">
                        <section class="ebk-panel statistics-filter-bar" :aria-label="tt('Filter')">
                            <v-btn-group class="statistics-period-stepper" color="default" density="comfortable" variant="outlined" divided>
                                <v-btn class="button-icon-with-direction" :icon="mdiArrowLeft"
                                       :aria-label="tt('Previous Period')" :disabled="loading || !canShiftDateRange"
                                       @click="shiftDateRange(-1)"/>
                                <v-menu location="bottom" max-height="500">
                                    <template #activator="{ props }">
                                        <v-btn class="statistics-period-name" :disabled="loading || !canChangeDateRange"
                                               v-bind="props">{{ queryDateRangeName }}</v-btn>
                                    </template>
                                    <v-list :selected="[queryDateType]">
                                        <v-list-item :key="dateRange.type" :value="dateRange.type"
                                                     :append-icon="(queryDateType === dateRange.type ? mdiCheck : undefined)"
                                                     v-for="dateRange in allDateRanges">
                                            <v-list-item-title class="cursor-pointer"
                                                               @click="setDateFilter(dateRange.type)">
                                                <div class="d-flex align-center">
                                                    <span>{{ dateRange.displayName }}</span>
                                                </div>
                                                <div class="statistics-custom-datetime-range smaller" v-if="dateRange.isUserCustomRange && canShowCustomDateRange(dateRange.type)">
                                                    <span>{{ queryStartTime }}</span>
                                                    <span>&nbsp;-&nbsp;</span>
                                                    <br/>
                                                    <span>{{ queryEndTime }}</span>
                                                </div>
                                            </v-list-item-title>
                                        </v-list-item>
                                    </v-list>
                                </v-menu>
                                <v-btn class="button-icon-with-direction" :icon="mdiArrowRight"
                                       :aria-label="tt('Next Period')" :disabled="loading || !canShiftDateRange"
                                       @click="shiftDateRange(1)"/>
                            </v-btn-group>

                            <v-menu location="bottom" max-height="500" v-if="queryAnalysisType === StatisticsAnalysisType.TrendAnalysis">
                                <template #activator="{ props }">
                                    <v-btn color="default" variant="outlined"
                                           :prepend-icon="mdiCalendarRangeOutline" :disabled="loading"
                                           v-bind="props">{{ queryTrendDateAggregationTypeName }}</v-btn>
                                </template>
                                <v-list>
                                    <v-list-item class="cursor-pointer" :key="aggregationType.type" :value="aggregationType.type"
                                                 :append-icon="(trendDateAggregationType === aggregationType.type ? mdiCheck : undefined)"
                                                 :title="aggregationType.displayName"
                                                 v-for="aggregationType in allTrendAnalysisDateAggregationTypes"
                                                 @click="setTrendDateAggregationType(aggregationType.type)">
                                    </v-list-item>
                                </v-list>
                            </v-menu>

                            <v-menu location="bottom" max-height="500" v-if="queryAnalysisType === StatisticsAnalysisType.AssetTrends">
                                <template #activator="{ props }">
                                    <v-btn color="default" variant="outlined"
                                           :prepend-icon="mdiCalendarRangeOutline" :disabled="loading"
                                           v-bind="props">{{ queryAssetTrendsDateAggregationTypeName }}</v-btn>
                                </template>
                                <v-list>
                                    <v-list-item class="cursor-pointer" :key="aggregationType.type" :value="aggregationType.type"
                                                 :append-icon="(assetTrendsDateAggregationType === aggregationType.type ? mdiCheck : undefined)"
                                                 :title="aggregationType.displayName"
                                                 v-for="aggregationType in allAssetTrendsDateAggregationTypes"
                                                 @click="setAssetTrendsDateAggregationType(aggregationType.type)">
                                    </v-list-item>
                                </v-list>
                            </v-menu>

                            <v-btn color="default" variant="outlined" :prepend-icon="mdiFilterOutline"
                                   :disabled="loading" @click="showFilterAccountDialog = true">
                                {{ tt('Accounts') }}
                                <v-tooltip activator="parent">{{ tt('Filter Accounts') }}</v-tooltip>
                            </v-btn>
                            <v-btn color="default" variant="outlined" :prepend-icon="mdiFilterOutline"
                                   :disabled="loading" @click="showFilterCategoryDialog = true"
                                   v-if="canUseCategoryFilter">
                                {{ tt('Transaction Categories') }}
                                <v-tooltip activator="parent">{{ tt('Filter Transaction Categories') }}</v-tooltip>
                            </v-btn>
                            <v-btn color="default" variant="outlined" :prepend-icon="mdiFilterOutline"
                                   :disabled="loading" @click="showFilterTagDialog = true"
                                   v-if="canUseTagFilter">
                                {{ tt('Tags') }}
                                <v-tooltip activator="parent">{{ tt('Filter Transaction Tags') }}</v-tooltip>
                            </v-btn>

                            <v-spacer/>

                            <div class="transaction-keyword-filter statistics-keyword-filter" v-if="canUseKeywordFilter">
                                <v-text-field autocomplete="off" density="compact" hide-details :disabled="loading"
                                              :prepend-inner-icon="mdiMagnify"
                                              :aria-label="tt('Filter transaction description')"
                                              :placeholder="tt('Filter transaction description')"
                                              v-model="filterKeyword"
                                              @keyup.enter="setKeywordFilter(filterKeyword)"
                                >
                                    <template #append-inner>
                                        <v-tooltip :text="tt('Filter')" v-if="filterKeyword !== query.keyword">
                                            <template v-slot:activator="{ props }">
                                                <v-icon :icon="mdiCheck"
                                                        :aria-label="tt('Filter')"
                                                        :disabled="loading"
                                                        @click="setKeywordFilter(filterKeyword)"
                                                        @keydown.enter.stop.prevent="setKeywordFilter(filterKeyword)"
                                                        @keydown.space.stop.prevent="setKeywordFilter(filterKeyword)"
                                                        v-bind="props" />
                                            </template>
                                        </v-tooltip>
                                    </template>
                                </v-text-field>
                            </div>

                            <v-btn density="comfortable" color="default" variant="text"
                                   :aria-label="tt('Refresh')" :icon="true" :loading="loading" @click="reload(true)">
                                <template #loader>
                                    <v-progress-circular indeterminate size="20"/>
                                </template>
                                <v-icon :icon="mdiRefresh" size="22" />
                                <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                            </v-btn>
                            <v-btn density="comfortable" color="default" variant="text"
                                   :aria-label="tt('More')" :disabled="loading" :icon="true">
                                <v-icon :icon="mdiDotsVertical" />
                                <v-menu activator="parent">
                                    <v-list>
                                        <v-list-item :prepend-icon="mdiExport"
                                                     :title="tt('Export Results')"
                                                     :disabled="!statisticsDataHasData"
                                                     @click="exportResults"
                                                     v-if="!isQuerySpecialChartType"></v-list-item>
                                        <v-divider class="my-2" v-if="!isQuerySpecialChartType"/>
                                        <v-list-item to="/settings/statistics"
                                                     :prepend-icon="mdiFilterCogOutline"
                                                     :title="tt('Settings')"></v-list-item>
                                    </v-list>
                                </v-menu>
                            </v-btn>
                        </section>

                        <div class="statistics-kpi-row" :class="{ 'statistics-kpi-row-single': !incomeExpenseKpis }"
                             v-if="initing || incomeExpenseKpis || balanceKpi">
                            <template v-if="incomeExpenseKpis">
                                <section class="ebk-panel statistics-kpi-tile" :key="kpi.key"
                                         :aria-labelledby="`statistics-kpi-${kpi.key}`"
                                         v-for="kpi in incomeExpenseKpis">
                                    <header class="ebk-panel-header-strip">
                                        <span :id="`statistics-kpi-${kpi.key}`">{{ kpi.label }}</span>
                                    </header>
                                    <div class="statistics-kpi-body" :class="{ 'disabled': loading }">
                                        <div class="statistics-kpi-amount">
                                            <v-icon class="statistics-kpi-arrow" :class="kpi.amount.type === 'income' ? 'text-income' : 'text-expense'"
                                                    size="22" aria-hidden="true"
                                                    :icon="kpi.amount.type === 'income' ? mdiArrowUp : mdiArrowDown"
                                                    v-if="kpi.amount.type !== 'none'" />
                                            <signed-amount :text="kpi.amount.text" :type="kpi.amount.type" />
                                        </div>
                                        <div class="statistics-kpi-meta">{{ queryDateRangeName }} · {{ defaultCurrency }}</div>
                                    </div>
                                </section>
                            </template>
                            <section class="ebk-panel statistics-kpi-tile" aria-labelledby="statistics-kpi-balance"
                                     v-else-if="balanceKpi">
                                <header class="ebk-panel-header-strip">
                                    <span id="statistics-kpi-balance">{{ balanceKpi.label }}</span>
                                </header>
                                <div class="statistics-kpi-body" :class="{ 'disabled': loading }">
                                    <div class="statistics-kpi-amount">
                                        <signed-amount :text="balanceKpi.text" type="none" />
                                    </div>
                                    <div class="statistics-kpi-meta">{{ defaultCurrency }}</div>
                                </div>
                            </section>
                            <template v-else-if="initing">
                                <section class="ebk-panel statistics-kpi-tile" :key="idx" v-for="idx in [1, 2, 3]">
                                    <header class="ebk-panel-header-strip">
                                        <v-skeleton-loader class="skeleton-no-margin" width="96px" type="text" :loading="true" />
                                    </header>
                                    <div class="statistics-kpi-body">
                                        <v-skeleton-loader class="skeleton-no-margin" width="160px" type="heading" :loading="true" />
                                    </div>
                                </section>
                            </template>
                        </div>

                        <section class="ebk-panel statistics-chart-panel" aria-labelledby="statistics-chart-panel-title">
                            <header class="ebk-panel-header-strip">
                                <span id="statistics-chart-panel-title">{{ queryChartDataTypeName }}</span>
                                <span class="statistics-panel-meta">{{ queryDateRangeName }}</span>
                                <div class="ebk-panel-header-actions statistics-panel-meta" v-if="!initing && hasStatisticsData">
                                    <span>{{ defaultCurrency }}</span>
                                </div>
                            </header>

                            <div class="statistics-chart-panel-body">
                                <state-panel type="error" :title="tt('Unable to retrieve transaction statistics')"
                                             v-if="!loading && !initing && loadingFailed">
                                    <template #actions>
                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload(false)">{{ tt('Refresh') }}</v-btn>
                                    </template>
                                </state-panel>

                                <state-panel type="empty" :title="tt('No transaction data')"
                                             v-else-if="!loading && !initing && !hasStatisticsData" />

                                <figure class="statistics-chart-figure" :class="{ 'readonly': loading }"
                                        aria-describedby="statistics-report-summary-text"
                                        v-if="!loadingFailed || loading">
                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.CategoricalAnalysis && queryChartDataType === ChartDataType.Overview.type">
                                        <account-and-category-sankey-chart
                                            :items="[]"
                                            :sorting-type="querySortingType"
                                            :skeleton="true"
                                            v-if="initing"
                                        />
                                        <account-and-category-sankey-chart
                                            :items="categoricalOverviewAnalysisData && categoricalOverviewAnalysisData.items && categoricalOverviewAnalysisData.items.length ? categoricalOverviewAnalysisData.items : []"
                                            :enable-click-item="true"
                                            :default-currency="defaultCurrency"
                                            v-else-if="!initing && hasStatisticsData"
                                            @click="onClickSankeyChartItem"
                                        />
                                    </div>

                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.CategoricalAnalysis && !isQuerySpecialChartType && query.categoricalChartType === CategoricalChartType.Pie.type">
                                        <pie-chart
                                            :items="[
                                                { id: '1', name: '---', value: parseBigDecimal(60), color: '7c7c7f' },
                                                { id: '2', name: '---', value: parseBigDecimal(20), color: 'a5a5aa' },
                                                { id: '3', name: '---', value: parseBigDecimal(20), color: 'c5c5c9' }
                                            ]"
                                            style-type="donut"
                                            :value-type="ChartValueType.Amount"
                                            :skeleton="true"
                                            :use-custom-color="true"
                                            v-if="initing"
                                        />
                                        <pie-chart
                                            :items="categoricalAnalysisData && categoricalAnalysisData.items && categoricalAnalysisData.items.length ? categoricalAnalysisData.items : []"
                                            style-type="donut"
                                            :value-type="ChartValueType.Amount"
                                            :show-value="showAmountInChart"
                                            :show-percent="showPercentInCategoricalChart"
                                            :enable-click-item="true"
                                            :default-currency="defaultCurrency"
                                            :center-title="totalAmountName"
                                            :center-value="getDisplayAmount(categoricalAnalysisData.value, defaultCurrency)"
                                            v-else-if="!initing && hasStatisticsData"
                                            @click="onClickPieChartItem"
                                        />
                                    </div>

                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.CategoricalAnalysis && !isQuerySpecialChartType && query.categoricalChartType === CategoricalChartType.Bar.type">
                                        <ol class="statistics-ranked-list" aria-hidden="true" v-if="initing">
                                            <li class="statistics-ranked-item" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                                                <v-icon class="disabled" size="32" :icon="mdiSquareRounded" />
                                                <div class="statistics-ranked-item-main">
                                                    <v-skeleton-loader class="skeleton-no-margin my-1" width="120px" type="text" :loading="true"></v-skeleton-loader>
                                                    <v-progress-linear :model-value="0" :height="4"></v-progress-linear>
                                                </div>
                                            </li>
                                        </ol>
                                        <ol class="statistics-ranked-list" v-else-if="!initing && hasStatisticsData">
                                            <template :key="idx" v-for="(item, idx) in categoricalAnalysisData.items">
                                                <li class="statistics-ranked-item" v-if="!item.hidden">
                                                    <router-link class="statistics-ranked-item-link" :to="getTransactionItemLinkUrl(item.id)">
                                                        <ItemIcon size="32px" :icon-type="getIconType(queryChartDataCategory, item.iconType)"
                                                                  :icon-id="item.icon" :color="item.color"></ItemIcon>
                                                        <div class="statistics-ranked-item-main">
                                                            <div class="statistics-ranked-item-row">
                                                                <span class="statistics-ranked-item-name">{{ item.name }}</span>
                                                                <span class="statistics-ranked-item-percent ebk-numeric" v-if="showPercentInCategoricalChart && item.percent >= 0 && item.value.isPositiveOrZero()">{{ formatPercentToLocalizedNumerals(item.percent, 2, '<0.01') }}</span>
                                                                <signed-amount class="statistics-ranked-item-amount"
                                                                               :text="getSignedAmount(item.value).text"
                                                                               :type="getSignedAmount(item.value).type" />
                                                            </div>
                                                            <v-progress-linear :color="item.color ? getTransactionCategoricalAnalysisDataItemDisplayColor(item) : 'primary'"
                                                                               bg-color="surface-sunken" :bg-opacity="1"
                                                                               :model-value="item.percent >= 0 ? item.percent : 0"
                                                                               :height="6" rounded aria-hidden="true"></v-progress-linear>
                                                        </div>
                                                    </router-link>
                                                </li>
                                            </template>
                                        </ol>
                                    </div>

                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.CategoricalAnalysis && !isQuerySpecialChartType && query.categoricalChartType === CategoricalChartType.Radar.type">
                                        <radar-chart
                                            :items="[
                                                {
                                                    name: '---',
                                                    values: Array.from({ length: 6 }, () => parseBigDecimal(10)),
                                                    displayOrders: [ 0, 1, 2, 3, 4, 5 ]
                                                }
                                            ]"
                                            :all-category-names="[ '---', '---', '---', '---', '---', '---' ]"
                                            :value-type="ChartValueType.Amount"
                                            :skeleton="true"
                                            :hide-legend="true"
                                            category-type-name=""
                                            v-if="initing"
                                        />
                                        <radar-chart
                                            :items="radarChartData"
                                            :all-category-names="radarChartCategoryNames"
                                            :value-type="ChartValueType.Amount"
                                            :show-value="showAmountInChart"
                                            :show-percent="showPercentInCategoricalChart"
                                            :default-currency="defaultCurrency"
                                            :category-type-name="tt('Name')"
                                            :hide-legend="true"
                                            v-else-if="!initing && hasStatisticsData"
                                        />
                                    </div>

                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.TrendAnalysis">
                                        <trends-chart
                                            chart-mode="monthly"
                                            :type="queryChartType"
                                            :start-time="undefined"
                                            :end-time="undefined"
                                            :start-year-month="query.trendChartStartYearMonth"
                                            :end-year-month="query.trendChartEndYearMonth"
                                            :sorting-type="querySortingType"
                                            :data-aggregation-type="ChartDataAggregationType.Sum"
                                            :date-aggregation-type="trendDateAggregationType"
                                            :fiscal-year-start="fiscalYearStart"
                                            :items="[]"
                                            :value-type="ChartValueType.Amount"
                                            :skeleton="true"
                                            v-if="initing"
                                        />
                                        <trends-chart
                                            chart-mode="monthly"
                                            :type="queryChartType"
                                            :start-time="undefined"
                                            :end-time="undefined"
                                            :start-year-month="query.trendChartStartYearMonth"
                                            :end-year-month="query.trendChartEndYearMonth"
                                            :sorting-type="querySortingType"
                                            :data-aggregation-type="ChartDataAggregationType.Sum"
                                            :date-aggregation-type="trendDateAggregationType"
                                            :fiscal-year-start="fiscalYearStart"
                                            :items="trendsAnalysisData && trendsAnalysisData.items && trendsAnalysisData.items.length ? trendsAnalysisData.items : []"
                                            :value-type="ChartValueType.Amount"
                                            :translate-name="translateNameInTrendsChart"
                                            :show-value="showAmountInChart"
                                            :enable-click-item="true"
                                            :default-currency="defaultCurrency"
                                            :stacked="showStackedInTrendsChart"
                                            :show-total-amount-in-tooltip="showTotalAmountInTrendsChart"
                                            :show-year-over-year="true"
                                            :show-period-over-period="trendDateAggregationType === ChartDateAggregationType.Month.type || trendDateAggregationType === ChartDateAggregationType.Quarter.type"
                                            :dashed-lines="chartDataAmountKind === 'expense'"
                                            legend-value-mode="sum"
                                            ref="monthlyTrendsChart"
                                            v-else-if="!initing && trendsAnalysisData && trendsAnalysisData.items && trendsAnalysisData.items.length"
                                            @click="onClickTrendChartItem"
                                        />
                                    </div>

                                    <div v-if="queryAnalysisType === StatisticsAnalysisType.AssetTrends">
                                        <trends-chart
                                            chart-mode="daily"
                                            :type="queryChartType"
                                            :start-time="query.assetTrendsChartStartTime"
                                            :end-time="query.assetTrendsChartEndTime"
                                            :start-year-month="undefined"
                                            :end-year-month="undefined"
                                            :sorting-type="querySortingType"
                                            :data-aggregation-type="ChartDataAggregationType.Last"
                                            :date-aggregation-type="assetTrendsDateAggregationType"
                                            :fiscal-year-start="fiscalYearStart"
                                            :items="[]"
                                            :value-type="ChartValueType.Amount"
                                            :skeleton="true"
                                            v-if="initing"
                                        />
                                        <trends-chart
                                            chart-mode="daily"
                                            :type="queryChartType"
                                            :start-time="query.assetTrendsChartStartTime"
                                            :end-time="query.assetTrendsChartEndTime"
                                            :start-year-month="undefined"
                                            :end-year-month="undefined"
                                            :sorting-type="querySortingType"
                                            :data-aggregation-type="ChartDataAggregationType.Last"
                                            :date-aggregation-type="assetTrendsDateAggregationType"
                                            :fiscal-year-start="fiscalYearStart"
                                            :items="assetTrendsData && assetTrendsData.items && assetTrendsData.items.length ? assetTrendsData.items : []"
                                            :value-type="ChartValueType.Amount"
                                            :translate-name="translateNameInTrendsChart"
                                            :show-value="showAmountInChart"
                                            :enable-click-item="true"
                                            :default-currency="defaultCurrency"
                                            :stacked="showStackedInTrendsChart"
                                            :show-total-amount-in-tooltip="showTotalAmountInTrendsChart"
                                            :show-year-over-year="true"
                                            :show-period-over-period="assetTrendsDateAggregationType === ChartDateAggregationType.Day.type || assetTrendsDateAggregationType === ChartDateAggregationType.Month.type || assetTrendsDateAggregationType === ChartDateAggregationType.Quarter.type"
                                            legend-value-mode="last"
                                            ref="dailyTrendsChart"
                                            v-else-if="!initing && assetTrendsData && assetTrendsData.items && assetTrendsData.items.length"
                                            @click="onClickTrendChartItem"
                                        />
                                    </div>
                                </figure>
                            </div>
                        </section>

                        <div class="statistics-detail-row" v-if="!initing && !loadingFailed && hasStatisticsData">
                            <section class="ebk-panel statistics-breakdown-panel" aria-labelledby="statistics-breakdown-title">
                                <header class="ebk-panel-header-strip">
                                    <span id="statistics-breakdown-title">{{ tt('Breakdown') }}</span>
                                    <span class="statistics-panel-meta ebk-numeric" v-if="breakdownTable">{{ breakdownTable.rows.length }}</span>
                                </header>
                                <div class="statistics-breakdown-scroll" role="region"
                                     aria-labelledby="statistics-breakdown-title" v-if="breakdownTable">
                                    <v-table class="statistics-breakdown-table" density="compact" fixed-header fixed-footer
                                             :height="breakdownTable.rows.length > 12 ? '32rem' : undefined"
                                             :class="{ 'disabled': loading }">
                                        <thead>
                                            <tr>
                                                <th scope="col" :key="column.key"
                                                    :class="{ 'text-end': column.numeric }"
                                                    v-for="column in breakdownTable.columns">{{ column.name }}</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            <tr :key="row.key" v-for="row in breakdownTable.rows">
                                                <td class="statistics-breakdown-name">
                                                    <div class="d-flex align-center">
                                                        <ItemIcon class="statistics-breakdown-icon" size="24px" :icon-type="row.icon.iconType"
                                                                  :icon-id="row.icon.iconId" :color="row.icon.color" v-if="row.icon"></ItemIcon>
                                                        <router-link class="statistics-breakdown-link" :to="row.link" v-if="row.link">{{ row.name }}</router-link>
                                                        <span class="statistics-breakdown-text" v-else>{{ row.name }}</span>
                                                    </div>
                                                </td>
                                                <td class="text-end" :key="cellIdx" v-for="(cell, cellIdx) in row.cells">
                                                    <signed-amount :text="cell.text" :type="cell.type" v-if="cell.type" />
                                                    <span class="ebk-numeric" v-else>{{ cell.text }}</span>
                                                </td>
                                            </tr>
                                        </tbody>
                                        <tfoot v-if="breakdownTable.totalRows.length">
                                            <tr class="statistics-breakdown-total" :key="row.key" v-for="row in breakdownTable.totalRows">
                                                <td>{{ row.name }}</td>
                                                <td class="text-end" :key="cellIdx" v-for="(cell, cellIdx) in row.cells">
                                                    <signed-amount :text="cell.text" :type="cell.type" v-if="cell.type" />
                                                    <span class="ebk-numeric" v-else>{{ cell.text }}</span>
                                                </td>
                                            </tr>
                                        </tfoot>
                                    </v-table>
                                </div>
                            </section>

                            <section class="ebk-panel statistics-summary-panel" aria-labelledby="statistics-report-summary-title">
                                <header class="ebk-panel-header-strip">
                                    <span id="statistics-report-summary-title">{{ tt('Report Summary') }}</span>
                                    <span class="statistics-panel-meta">{{ queryDateRangeName }}</span>
                                </header>
                                <div class="statistics-summary-body" id="statistics-report-summary-text" aria-live="polite">
                                    <p :key="idx" v-for="(sentence, idx) in reportSummarySentences">{{ sentence }}</p>
                                </div>
                            </section>
                        </div>
                    </div>
                </v-window-item>
            </v-window>
        </template>
    </main-page-layout>

    <date-range-selection-dialog :title="tt('Custom Date Range')"
                                  :min-time="query.categoricalChartStartTime"
                                  :max-time="query.categoricalChartEndTime"
                                  v-model:show="showCustomDateRangeDialog"
                                  @dateRange:change="setCustomDateFilter"
                                  @error="onShowDateRangeError" />

    <month-range-selection-dialog :title="tt('Custom Date Range')"
                                  :min-time="query.trendChartStartYearMonth"
                                  :max-time="query.trendChartEndYearMonth"
                                  v-model:show="showCustomMonthRangeDialog"
                                  @dateRange:change="setCustomDateFilter"
                                  @error="onShowDateRangeError" />

    <account-filter-settings-dialog type="statisticsCurrent"
                                    v-model:show="showFilterAccountDialog"
                                    @settings:change="setAccountFilter" />

    <category-filter-settings-dialog type="statisticsCurrent"
                                     v-model:show="showFilterCategoryDialog"
                                     @settings:change="setCategoryFilter" />

    <transaction-tag-filter-settings-dialog type="statisticsCurrent"
                                            v-model:show="showFilterTagDialog"
                                            @settings:change="setTagFilter" />

    <data-export-dialog ref="dataExportDialog" />
</template>

<script setup lang="ts">
import TrendsChart from '@/components/desktop/TrendsChart.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import DataExportDialog from '@/components/desktop/DataExportDialog.vue';
import AccountFilterSettingsDialog from '@/views/desktop/common/dialogs/AccountFilterSettingsDialog.vue';
import CategoryFilterSettingsDialog from '@/views/desktop/common/dialogs/CategoryFilterSettingsDialog.vue';
import TransactionTagFilterSettingsDialog from '@/views/desktop/common/dialogs/TransactionTagFilterSettingsDialog.vue';

import { ref, computed, useTemplateRef } from 'vue';
import { useRouter, onBeforeRouteUpdate } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';
import { type StatisticsAmountKind, useStatisticsTransactionPageBase } from '@/views/base/statistics/StatisticsTransactionPageBase.ts';
import { useChartTextSummary } from '@/components/base/ChartSummaryBase.ts';

import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { type TransactionStatisticsPartialFilter, useStatisticsStore } from '@/stores/statistics.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { type TypeAndDisplayName, itemAndIndex } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import type { ColorValue } from '@/core/color.ts';
import type { CommonIconItemType } from '@/core/icon.ts';
import { type TextualYearMonth, type TimeRangeAndDateType, DateRangeScene, DateRange } from '@/core/datetime.ts';
import { ChartValueType } from '@/core/chart.ts';
import {
    ChartDataAggregationType,
    StatisticsAnalysisType,
    CategoricalChartType,
    TrendChartType,
    ChartDataType,
    ChartSortingType,
    ChartDateAggregationType,
    ExportMermaidChartType
} from '@/core/statistics.ts';

import {
    isDefined,
    isString,
    isNumber,
    arrayItemToObjectField
} from '@/lib/common.ts';
import {
    BIG_DECIMAL_ZERO,
    parseBigDecimal
} from '@/lib/numeral.ts';
import {
    getGregorianCalendarYearAndMonthFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    getShiftedDateRangeAndDateType,
    getDateTypeByDateRange,
    getDateRangeByDateType
} from '@/lib/datetime.ts';
import {
    getIconType
} from '@/lib/icon.ts';
import {
    TransactionCategoricalOverviewAnalysisDataItemType
} from '@/models/transaction.ts';

import {
    mdiCheck,
    mdiArrowLeft,
    mdiArrowRight,
    mdiArrowUp,
    mdiArrowDown,
    mdiChartDonut,
    mdiChartBar,
    mdiChartAreaspline,
    mdiCalendarRangeOutline,
    mdiRefresh,
    mdiSquareRounded,
    mdiMagnify,
    mdiFilterOutline,
    mdiFilterCogOutline,
    mdiExport,
    mdiDotsVertical
} from '@mdi/js';

type TrendsChartType = InstanceType<typeof TrendsChart>;
type DataExportDialogType = InstanceType<typeof DataExportDialog>;

interface TransactionStatisticsProps {
    initAnalysisType?: string,
    initChartDataType?: string,
    initChartType?: string,
    initChartDateType?: string,
    initStartTime?: TextualYearMonth | '',
    initEndTime?: TextualYearMonth | '',
    initFilterAccountIds?: string,
    initFilterCategoryIds?: string,
    initTagFilter?: string,
    initKeyword?: string;
    initMatchMode?: string;
    initSortingType?: string,
    initTrendDateAggregationType?: string
    initAssetTrendsDateAggregationType?: string
}

const props = defineProps<TransactionStatisticsProps>();

interface StatisticsSignedAmount {
    text: string;
    type: 'income' | 'expense' | 'none';
}

interface StatisticsKpiItem {
    key: string;
    label: string;
    amount: StatisticsSignedAmount;
}

interface StatisticsBreakdownCell {
    text: string;
    type?: 'income' | 'expense' | 'none';
}

interface StatisticsBreakdownRow {
    key: string;
    name: string;
    icon?: { iconType: CommonIconItemType, iconId: string, color?: ColorValue };
    link?: string;
    cells: StatisticsBreakdownCell[];
}

interface StatisticsBreakdownTable {
    columns: { key: string, name: string, numeric: boolean }[];
    rows: StatisticsBreakdownRow[];
    totalRows: StatisticsBreakdownRow[];
}

const router = useRouter();

const {
    tt,
    getAllCategoricalChartTypes,
    getAllTrendChartTypes,
    formatAmountToWesternArabicNumeralsWithoutDigitGrouping,
    formatBigDecimalToWesternArabicNumeralsWithoutDigitGrouping,
    formatPercentToLocalizedNumerals
} = useI18n();

const {
    getRankedItemsSummary,
    getSeriesSummary,
    getIncomeExpenseSummary
} = useChartTextSummary();

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
    queryTrendDateAggregationTypeName,
    queryAssetTrendsDateAggregationTypeName,
    canChangeDateRange,
    canShiftDateRange,
    canUseCategoryFilter,
    canUseTagFilter,
    canUseKeywordFilter,
    showAmountInChart,
    totalAmountName,
    showPercentInCategoricalChart,
    showTotalAmountInTrendsChart,
    showStackedInTrendsChart,
    translateNameInTrendsChart,
    categoricalOverviewAnalysisData,
    categoricalAnalysisData,
    trendsAnalysisData,
    radarChartCategoryNames,
    radarChartData,
    assetTrendsData,
    chartDataAmountKind,
    incomeExpenseTotals,
    isAccountBalanceChartDataType,
    canShowCustomDateRange,
    getTransactionCategoricalAnalysisDataItemDisplayColor,
    getDisplayAmount
} = useStatisticsTransactionPageBase();

const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const statisticsStore = useStatisticsStore();
const desktopPageStore = useDesktopPageStore();

const monthlyTrendsChart = useTemplateRef<TrendsChartType>('monthlyTrendsChart');
const dailyTrendsChart = useTemplateRef<TrendsChartType>('dailyTrendsChart');
const dataExportDialog = useTemplateRef<DataExportDialogType>('dataExportDialog');

const activeTab = ref<string>('statisticsPage');
const initing = ref<boolean>(true);
const loadingFailed = ref<boolean>(false);
const filterKeyword = ref<string>('');
const showCustomDateRangeDialog = ref<boolean>(false);
const showCustomMonthRangeDialog = ref<boolean>(false);
const showFilterAccountDialog = ref<boolean>(false);
const showFilterCategoryDialog = ref<boolean>(false);
const showFilterTagDialog = ref<boolean>(false);

const allAnalysisTypeItems = computed<{ type: StatisticsAnalysisType, name: string, icon: string }[]>(() => [
    { type: StatisticsAnalysisType.CategoricalAnalysis, name: tt('Categorical Analysis'), icon: mdiChartDonut },
    { type: StatisticsAnalysisType.TrendAnalysis, name: tt('Trend Analysis'), icon: mdiChartBar },
    { type: StatisticsAnalysisType.AssetTrends, name: tt('Asset Trends'), icon: mdiChartAreaspline }
]);

const statisticsDataHasData = computed<boolean>(() => {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        return !!categoricalAnalysisData.value && !!categoricalAnalysisData.value.items && categoricalAnalysisData.value.items.length > 0;
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        return !!trendsAnalysisData.value && !!trendsAnalysisData.value.items && trendsAnalysisData.value.items.length > 0 && !!monthlyTrendsChart.value;
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        return !!assetTrendsData.value && !!assetTrendsData.value.items && assetTrendsData.value.items.length > 0 && !!dailyTrendsChart.value;
    }

    return false;
});

const allChartTypes = computed<TypeAndDisplayName[]>(() => {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        return getAllCategoricalChartTypes(true);
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        return getAllTrendChartTypes();
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        return getAllTrendChartTypes();
    } else {
        return [];
    }
});

const queryAnalysisType = computed<StatisticsAnalysisType>({
    get: () => analysisType.value,
    set: (value: number) => {
        setAnalysisType(value);
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

const queryChartDataType = computed<number>({
    get: () => query.value.chartDataType,
    set: (value: number) => {
        setChartDataType(value);
    }
});

const querySortingType = computed<number>({
    get: () => query.value.sortingType,
    set: (value: number) => {
        setSortingType(value);
    }
});

const isQuerySpecialChartType = computed<boolean>(() => {
    return ChartDataType.valueOf(queryChartDataType.value)?.specialChart ?? false;
});

const hasStatisticsData = computed<boolean>(() => {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        if (queryChartDataType.value === ChartDataType.Overview.type) {
            return !!categoricalOverviewAnalysisData.value && !!categoricalOverviewAnalysisData.value.items && categoricalOverviewAnalysisData.value.items.length > 0;
        }

        return !!categoricalAnalysisData.value && !!categoricalAnalysisData.value.items && categoricalAnalysisData.value.items.length > 0;
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        return !!trendsAnalysisData.value && !!trendsAnalysisData.value.items && trendsAnalysisData.value.items.length > 0;
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        return !!assetTrendsData.value && !!assetTrendsData.value.items && assetTrendsData.value.items.length > 0;
    }

    return false;
});

const incomeExpenseKpis = computed<StatisticsKpiItem[] | null>(() => {
    if (initing.value || loadingFailed.value || !incomeExpenseTotals.value) {
        return null;
    }

    return [
        { key: 'income', label: tt('Total Income'), amount: getSignedAmount(incomeExpenseTotals.value.income, 'income') },
        { key: 'expense', label: tt('Total Expense'), amount: getSignedAmount(incomeExpenseTotals.value.expense, 'expense') },
        { key: 'net', label: tt('Net Movement'), amount: getNetAmount(incomeExpenseTotals.value.net) }
    ];
});

const balanceKpi = computed<{ label: string, text: string } | null>(() => {
    if (initing.value || loadingFailed.value || analysisType.value !== StatisticsAnalysisType.CategoricalAnalysis || !isAccountBalanceChartDataType.value || !hasStatisticsData.value) {
        return null;
    }

    return {
        label: totalAmountName.value,
        text: getDisplayAmount(categoricalAnalysisData.value.value, defaultCurrency.value)
    };
});

const trendsDisplayData = computed<{ periods: string[], lastDataIndex: number, series: { id: string, name: string, values: BigDecimal[] }[] } | null>(() => {
    if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && monthlyTrendsChart.value) {
        return monthlyTrendsChart.value.getDisplayData();
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && dailyTrendsChart.value) {
        return dailyTrendsChart.value.getDisplayData();
    }

    return null;
});

const overviewPrimaryCategoryItems = computed<{ income: { id: string, name: string, value: BigDecimal }[], expense: { id: string, name: string, value: BigDecimal }[] }>(() => {
    const income: { id: string, name: string, value: BigDecimal }[] = [];
    const expense: { id: string, name: string, value: BigDecimal }[] = [];

    for (const item of categoricalOverviewAnalysisData.value?.items ?? []) {
        if (item.hidden) {
            continue;
        }

        if (item.type === TransactionCategoricalOverviewAnalysisDataItemType.IncomeByPrimaryCategory) {
            income.push({ id: item.id, name: item.name, value: item.value });
        } else if (item.type === TransactionCategoricalOverviewAnalysisDataItemType.ExpenseByPrimaryCategory) {
            expense.push({ id: item.id, name: item.name, value: item.value });
        }
    }

    return { income, expense };
});

const breakdownTable = computed<StatisticsBreakdownTable | null>(() => {
    if (!hasStatisticsData.value) {
        return null;
    }

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis && queryChartDataType.value === ChartDataType.Overview.type) {
        const rows: StatisticsBreakdownRow[] = [];

        for (const item of overviewPrimaryCategoryItems.value.income) {
            rows.push({ key: `income-${item.id}`, name: item.name, link: getTransactionItemLinkUrl(`category:${item.id}`), cells: [{ text: tt('Income') }, getSignedAmount(item.value, 'income')] });
        }

        for (const item of overviewPrimaryCategoryItems.value.expense) {
            rows.push({ key: `expense-${item.id}`, name: item.name, link: getTransactionItemLinkUrl(`category:${item.id}`), cells: [{ text: tt('Expense') }, getSignedAmount(item.value, 'expense')] });
        }

        const totalIncome = categoricalOverviewAnalysisData.value?.totalIncome ?? BIG_DECIMAL_ZERO;
        const totalExpense = categoricalOverviewAnalysisData.value?.totalExpense ?? BIG_DECIMAL_ZERO;

        return {
            columns: [
                { key: 'name', name: tt('Name'), numeric: false },
                { key: 'type', name: tt('Type'), numeric: true },
                { key: 'amount', name: tt('Amount'), numeric: true }
            ],
            rows: rows,
            totalRows: [
                { key: 'total-income', name: tt('Total Income'), cells: [{ text: '' }, getSignedAmount(totalIncome, 'income')] },
                { key: 'total-expense', name: tt('Total Expense'), cells: [{ text: '' }, getSignedAmount(totalExpense, 'expense')] },
                { key: 'net', name: tt('Net Movement'), cells: [{ text: '' }, getNetAmount(totalIncome.subtract(totalExpense))] }
            ]
        };
    } else if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        const rows: StatisticsBreakdownRow[] = [];
        const canLink = queryChartDataCategory.value === 'account' || queryChartDataCategory.value === 'category';

        for (const item of categoricalAnalysisData.value.items) {
            if (item.hidden) {
                continue;
            }

            const cells: StatisticsBreakdownCell[] = [];

            if (showPercentInCategoricalChart.value) {
                cells.push({ text: item.percent >= 0 && item.value.isPositiveOrZero() ? formatPercentToLocalizedNumerals(item.percent, 2, '<0.01') : '—' });
            }

            cells.push(getSignedAmount(item.value));

            rows.push({
                key: item.id,
                name: item.name,
                icon: { iconType: getIconType(queryChartDataCategory.value, item.iconType), iconId: item.icon, color: item.color },
                link: canLink ? getTransactionItemLinkUrl(item.id) : undefined,
                cells: cells
            });
        }

        const totalCells: StatisticsBreakdownCell[] = [];

        if (showPercentInCategoricalChart.value) {
            totalCells.push({ text: '' });
        }

        totalCells.push(getSignedAmount(categoricalAnalysisData.value.value));

        return {
            columns: [
                { key: 'name', name: tt('Name'), numeric: false },
                ...(showPercentInCategoricalChart.value ? [{ key: 'percent', name: tt('Proportion (%)'), numeric: true }] : []),
                { key: 'amount', name: tt('Amount'), numeric: true }
            ],
            rows: rows,
            totalRows: [
                { key: 'total', name: totalAmountName.value, cells: totalCells }
            ]
        };
    } else if (trendsDisplayData.value) {
        const series = trendsDisplayData.value.series;
        const isSum = analysisType.value === StatisticsAnalysisType.TrendAnalysis;
        const showTotalColumn = series.length > 1 && (!isSum || showTotalAmountInTrendsChart.value);
        const rows: StatisticsBreakdownRow[] = [];
        const seriesTotals: BigDecimal[] = series.map(() => BIG_DECIMAL_ZERO);
        let grandTotal: BigDecimal = BIG_DECIMAL_ZERO;

        for (const [period, periodIndex] of itemAndIndex(trendsDisplayData.value.periods)) {
            const cells: StatisticsBreakdownCell[] = [];
            let periodTotal: BigDecimal = BIG_DECIMAL_ZERO;

            for (const [seriesItem, seriesIndex] of itemAndIndex(series)) {
                const value = seriesItem.values[periodIndex] ?? BIG_DECIMAL_ZERO;
                periodTotal = periodTotal.add(value);
                seriesTotals[seriesIndex] = (seriesTotals[seriesIndex] ?? BIG_DECIMAL_ZERO).add(value);
                cells.push(getTrendCellAmount(value));
            }

            if (showTotalColumn) {
                cells.push(getTrendCellAmount(periodTotal));
            }

            grandTotal = grandTotal.add(periodTotal);
            rows.push({ key: `period-${periodIndex}`, name: period, cells: cells });
        }

        const totalCells: StatisticsBreakdownCell[] = seriesTotals.map(total => getTrendCellAmount(total));

        if (showTotalColumn) {
            totalCells.push(getTrendCellAmount(grandTotal));
        }

        return {
            columns: [
                { key: 'date', name: tt('Date'), numeric: false },
                ...series.map(item => ({ key: `series-${item.id}`, name: item.name, numeric: true })),
                ...(showTotalColumn ? [{ key: 'total', name: tt('Total Amount'), numeric: true }] : [])
            ],
            rows: rows,
            totalRows: isSum ? [{ key: 'total', name: tt('Total'), cells: totalCells }] : []
        };
    }

    return null;
});

const reportSummarySentences = computed<string[]>(() => {
    const sentences: string[] = [];

    if (!hasStatisticsData.value) {
        return sentences;
    }

    const formatValue = (value: BigDecimal) => getDisplayAmount(value, defaultCurrency.value);

    if (incomeExpenseKpis.value && incomeExpenseKpis.value.length === 3) {
        const [income, expense, net] = incomeExpenseKpis.value;
        sentences.push(getIncomeExpenseSummary(getSignedAmountText(income!.amount), getSignedAmountText(expense!.amount), getSignedAmountText(net!.amount)));
    }

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis && queryChartDataType.value === ChartDataType.Overview.type) {
        const totalExpense = categoricalOverviewAnalysisData.value?.totalExpense ?? BIG_DECIMAL_ZERO;
        const totalIncome = categoricalOverviewAnalysisData.value?.totalIncome ?? BIG_DECIMAL_ZERO;

        const expenseSummary = getRankedItemsSummary(overviewPrimaryCategoryItems.value.expense.map(item => ({
            name: item.name,
            value: item.value,
            percent: totalExpense.isPositive() ? item.value.divide(totalExpense).multiply(100).toDoubleNumber() : -1
        })), { totalName: tt('Total Expense'), formatValue: formatValue, showPercent: true });

        const incomeSummary = getRankedItemsSummary(overviewPrimaryCategoryItems.value.income.map(item => ({
            name: item.name,
            value: item.value,
            percent: totalIncome.isPositive() ? item.value.divide(totalIncome).multiply(100).toDoubleNumber() : -1
        })), { totalName: tt('Total Income'), formatValue: formatValue, showPercent: true });

        if (expenseSummary) {
            sentences.push(expenseSummary);
        }

        if (incomeSummary) {
            sentences.push(incomeSummary);
        }
    } else if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        const summary = getRankedItemsSummary(categoricalAnalysisData.value.items.filter(item => !item.hidden), {
            totalName: totalAmountName.value,
            formatValue: formatValue,
            showPercent: showPercentInCategoricalChart.value
        });

        if (summary) {
            sentences.push(summary);
        }
    } else if (trendsDisplayData.value && trendsDisplayData.value.series.length) {
        const isBalance = analysisType.value === StatisticsAnalysisType.AssetTrends;
        const lastDataIndex = trendsDisplayData.value.lastDataIndex;
        // balances are stocks: periods after the latest data (e.g. future months) are not part of the summary
        const periods = isBalance && lastDataIndex >= 0 ? trendsDisplayData.value.periods.slice(0, lastDataIndex + 1) : trendsDisplayData.value.periods;
        const series = trendsDisplayData.value.series;
        const seriesName = series.length === 1 && series[0] ? series[0].name : totalAmountName.value;
        const points = periods.map((period, index) => {
            let total: BigDecimal = BIG_DECIMAL_ZERO;

            for (const item of series) {
                total = total.add(item.values[index] ?? BIG_DECIMAL_ZERO);
            }

            return { label: period, value: total };
        });

        if (isBalance && points.length > 1) {
            const latestPoint = points[points.length - 1]!;
            sentences.push(getSeriesSummary(seriesName, [latestPoint], formatValue));
        }

        sentences.push(getSeriesSummary(seriesName, points, formatValue));

        if (series.length > 1) {
            if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
                const seriesTotals = series.map(item => {
                    let total: BigDecimal = BIG_DECIMAL_ZERO;

                    for (const value of item.values) {
                        total = total.add(value);
                    }

                    return { name: item.name, value: total };
                });

                let positiveTotal: BigDecimal = BIG_DECIMAL_ZERO;

                for (const item of seriesTotals) {
                    if (item.value.isPositive()) {
                        positiveTotal = positiveTotal.add(item.value);
                    }
                }

                const rankedSummary = getRankedItemsSummary(seriesTotals.map(item => ({
                    name: item.name,
                    value: item.value,
                    percent: positiveTotal.isPositive() && item.value.isPositive() ? item.value.divide(positiveTotal).multiply(100).toDoubleNumber() : -1
                })), { totalName: totalAmountName.value, formatValue: formatValue, showPercent: showTotalAmountInTrendsChart.value });

                if (rankedSummary) {
                    sentences.push(rankedSummary);
                }
            }
        }
    }

    return sentences.filter(sentence => !!sentence);
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
    if (value.isZero()) {
        return { text: getDisplayAmount(value, defaultCurrency.value), type: 'none' };
    }

    return getSignedAmount(value, 'income');
}

function getTrendCellAmount(value: BigDecimal): StatisticsBreakdownCell {
    if (analysisType.value === StatisticsAnalysisType.AssetTrends || chartDataAmountKind.value === 'none') {
        return { text: getDisplayAmount(value, defaultCurrency.value), type: 'none' };
    }

    return getSignedAmount(value);
}

function getSignedAmountText(amount: StatisticsSignedAmount): string {
    if (amount.type === 'income') {
        return `+${amount.text}`;
    } else if (amount.type === 'expense') {
        return `−${amount.text}`;
    }

    return amount.text;
}

function getFilterLinkUrl(): string {
    return `/statistics/transaction?${statisticsStore.getTransactionStatisticsPageParams(analysisType.value, trendDateAggregationType.value, assetTrendsDateAggregationType.value)}`;
}

function getTransactionItemLinkUrl(itemId: string, dateRange?: TimeRangeAndDateType): string {
    return `/transaction/list?${statisticsStore.getTransactionListPageParams(analysisType.value, itemId, dateRange)}`;
}

function init(initProps: TransactionStatisticsProps): void {
    let needReload = !isDefined(initProps.initAnalysisType);

    const filter: TransactionStatisticsPartialFilter = {
        chartDataType: initProps.initChartDataType ? parseInt(initProps.initChartDataType) : undefined,
        filterAccountIds: initProps.initFilterAccountIds ? arrayItemToObjectField(initProps.initFilterAccountIds.split(','), true) : {},
        filterCategoryIds: initProps.initFilterCategoryIds ? arrayItemToObjectField(initProps.initFilterCategoryIds.split(','), true) : {},
        tagFilter: initProps.initTagFilter,
        keyword: initProps.initKeyword,
        matchMode: initProps.initMatchMode ? parseInt(initProps.initMatchMode) : undefined,
        sortingType: initProps.initSortingType ? parseInt(initProps.initSortingType) : undefined
    };

    filterKeyword.value = filter.keyword || '';

    if (initProps.initAnalysisType === StatisticsAnalysisType.CategoricalAnalysis.toString()) {
        filter.categoricalChartType = initProps.initChartType ? parseInt(initProps.initChartType) : undefined;
        filter.categoricalChartDateType = initProps.initChartDateType ? parseInt(initProps.initChartDateType) : undefined;
        filter.categoricalChartStartTime = initProps.initStartTime ? parseInt(initProps.initStartTime) : undefined;
        filter.categoricalChartEndTime = initProps.initEndTime ? parseInt(initProps.initEndTime) : undefined;

        if (filter.categoricalChartDateType !== query.value.categoricalChartDateType) {
            needReload = true;
        } else if (filter.categoricalChartDateType === DateRange.Custom.type) {
            if (filter.categoricalChartStartTime !== query.value.categoricalChartStartTime
                || filter.categoricalChartEndTime !== query.value.categoricalChartEndTime) {
                needReload = true;
            }
        }

        if (initProps.initAnalysisType !== analysisType.value.toString()) {
            analysisType.value = StatisticsAnalysisType.CategoricalAnalysis;
            needReload = true;
        }
    } else if (initProps.initAnalysisType === StatisticsAnalysisType.TrendAnalysis.toString()) {
        filter.trendChartType = initProps.initChartType ? parseInt(initProps.initChartType) : undefined;
        filter.trendChartDateType = initProps.initChartDateType ? parseInt(initProps.initChartDateType) : undefined;
        filter.trendChartStartYearMonth = initProps.initStartTime;
        filter.trendChartEndYearMonth = initProps.initEndTime;

        if (filter.trendChartDateType !== query.value.trendChartDateType) {
            needReload = true;
        } else if (filter.trendChartDateType === DateRange.Custom.type) {
            if (filter.trendChartStartYearMonth !== query.value.trendChartStartYearMonth
                || filter.trendChartEndYearMonth !== query.value.trendChartEndYearMonth) {
                needReload = true;
            }
        }

        if (initProps.initAnalysisType !== analysisType.value.toString()) {
            analysisType.value = StatisticsAnalysisType.TrendAnalysis;
            needReload = true;
        }

        if (initProps.initTrendDateAggregationType) {
            trendDateAggregationType.value = parseInt(initProps.initTrendDateAggregationType);
        }
    } else if (initProps.initAnalysisType === StatisticsAnalysisType.AssetTrends.toString()) {
        filter.assetTrendsChartType = initProps.initChartType ? parseInt(initProps.initChartType) : undefined;
        filter.assetTrendsChartDateType = initProps.initChartDateType ? parseInt(initProps.initChartDateType) : undefined;
        filter.assetTrendsChartStartTime = initProps.initStartTime ? parseInt(initProps.initStartTime) : undefined;
        filter.assetTrendsChartEndTime = initProps.initEndTime ? parseInt(initProps.initEndTime) : undefined;

        if (filter.assetTrendsChartDateType !== query.value.assetTrendsChartDateType) {
            needReload = true;
        } else if (filter.assetTrendsChartDateType === DateRange.Custom.type) {
            if (filter.assetTrendsChartStartTime !== query.value.assetTrendsChartStartTime
                || filter.assetTrendsChartEndTime !== query.value.assetTrendsChartEndTime) {
                needReload = true;
            }
        }

        if (initProps.initAnalysisType !== analysisType.value.toString()) {
            analysisType.value = StatisticsAnalysisType.AssetTrends;
            needReload = true;
        }

        if (initProps.initAssetTrendsDateAggregationType) {
            assetTrendsDateAggregationType.value = parseInt(initProps.initAssetTrendsDateAggregationType);
        }
    }

    if (!isDefined(initProps.initAnalysisType)) {
        analysisType.value = StatisticsAnalysisType.CategoricalAnalysis;
        statisticsStore.initTransactionStatisticsFilter(analysisType.value);
    } else {
        statisticsStore.initTransactionStatisticsFilter(analysisType.value, filter);
    }

    if (!needReload && !statisticsStore.transactionStatisticsStateInvalid) {
        loading.value = false;
        initing.value = false;
        return;
    }

    Promise.all([
        accountsStore.loadAllAccounts({force: false}),
        transactionCategoriesStore.loadAllCategories({force: false})
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
        initing.value = false;
        loadingFailed.value = false;
    }).catch(error => {
        loading.value = false;
        initing.value = false;
        loadingFailed.value = true;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function reload(force: boolean): Promise<unknown> | null {
    let dispatchPromise: Promise<unknown> | null = null;

    loading.value = true;

    if (query.value.chartDataType === ChartDataType.Overview.type ||
        query.value.chartDataType === ChartDataType.OutflowsByAccount.type ||
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
            loading.value = false;
            loadingFailed.value = false;

            if (force) {
                desktopPageStore.addToast('Data has been updated');
            }
        }).catch(error => {
            loading.value = false;
            loadingFailed.value = true;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    }

    return dispatchPromise;
}

function setAnalysisType(type: StatisticsAnalysisType): void {
    if (analysisType.value === type) {
        return;
    }

    if (!ChartDataType.isAvailableForAnalysisType(query.value.chartDataType, type)) {
        let defaultChartDataType: ChartDataType = ChartDataType.Default;

        if (type === StatisticsAnalysisType.AssetTrends) {
            defaultChartDataType = ChartDataType.DefaultForAssetTrends;
        }

        statisticsStore.updateTransactionStatisticsFilter({
            chartDataType: defaultChartDataType.type
        });
    }

    if (analysisType.value !== StatisticsAnalysisType.TrendAnalysis && type === StatisticsAnalysisType.TrendAnalysis) {
        trendDateAggregationType.value = ChartDateAggregationType.Default.type;
    } else if (analysisType.value !== StatisticsAnalysisType.AssetTrends && type === StatisticsAnalysisType.AssetTrends) {
        assetTrendsDateAggregationType.value = ChartDateAggregationType.Default.type;
    }

    analysisType.value = type;
    loading.value = true;
    statisticsStore.updateTransactionStatisticsInvalidState(true);
    router.push(getFilterLinkUrl());
}

function setChartType(type?: number): void {
    let changed = false;

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            categoricalChartType: type
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            trendChartType: type
        });
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartType: type
        });
    }

    if (changed) {
        router.push(getFilterLinkUrl());
    }
}

function setChartDataType(type: number): void {
    const changed = statisticsStore.updateTransactionStatisticsFilter({
        chartDataType: type
    });

    if (changed) {
        router.push(getFilterLinkUrl());
    }
}

function setSortingType(type: number): void {
    if (type < ChartSortingType.Amount.type || type > ChartSortingType.Name.type) {
        return;
    }

    const changed = statisticsStore.updateTransactionStatisticsFilter({
        sortingType: type
    });

    if (changed) {
        router.push(getFilterLinkUrl());
    }
}

function setTrendDateAggregationType(type: number): void {
    const changed = trendDateAggregationType.value !== type;
    trendDateAggregationType.value = type;

    if (changed) {
        router.push(getFilterLinkUrl());
    }
}

function setAssetTrendsDateAggregationType(type: number): void {
    const changed = assetTrendsDateAggregationType.value !== type;
    assetTrendsDateAggregationType.value = type;

    if (changed) {
        router.push(getFilterLinkUrl());
    }
}

function setDateFilter(dateType: number): void {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomDateRangeDialog.value = true;
            return;
        } else if (query.value.categoricalChartDateType === dateType) {
            return;
        }
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomMonthRangeDialog.value = true;
            return;
        } else if (query.value.trendChartDateType === dateType) {
            return;
        }
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        if (dateType === DateRange.Custom.type) { // Custom
            showCustomDateRangeDialog.value = true;
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
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
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

        showCustomDateRangeDialog.value = false;
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && isString(startTime) && isString(endTime)) {
        const chartDateType = getDateTypeByDateRange(getYearMonthFirstUnixTime(startTime), getYearMonthLastUnixTime(endTime), firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.TrendAnalysis);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            trendChartDateType: chartDateType,
            trendChartStartYearMonth: startTime,
            trendChartEndYearMonth: endTime
        });

        showCustomMonthRangeDialog.value = false;
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && isNumber(startTime) && isNumber(endTime)) {
        const chartDateType = getDateTypeByDateRange(startTime, endTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.AssetTrends);

        changed = statisticsStore.updateTransactionStatisticsFilter({
            assetTrendsChartDateType: chartDateType,
            assetTrendsChartStartTime: startTime,
            assetTrendsChartEndTime: endTime
        });

        showCustomDateRangeDialog.value = false;
    }

    if (changed) {
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
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
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
    }
}

function setAccountFilter(changed: boolean): void {
    showFilterAccountDialog.value = false;

    if (changed) {
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
    }
}

function setCategoryFilter(changed: boolean): void {
    showFilterCategoryDialog.value = false;

    if (changed) {
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
    }
}

function setTagFilter(changed: boolean): void {
    showFilterTagDialog.value = false;

    if (changed) {
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
    }
}

function setKeywordFilter(keyword: string): void {
    if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
        return;
    }

    if (query.value.keyword === keyword) {
        return;
    }

    let changed = false;

    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            keyword: keyword
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
        changed = statisticsStore.updateTransactionStatisticsFilter({
            keyword: keyword
        });
    }

    if (changed) {
        loading.value = true;
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        router.push(getFilterLinkUrl());
    }
}

function exportResults(): void {
    if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis && categoricalAnalysisData.value && categoricalAnalysisData.value.items) {
        let supportedMermaidCharts: ExportMermaidChartType[] | undefined = undefined;

        if (query.value.categoricalChartType === CategoricalChartType.Pie.type) {
            supportedMermaidCharts = [ ExportMermaidChartType.PieChart ];
        }

        dataExportDialog.value?.open({
            headers: [
                tt('Name'),
                tt('Amount') + ` (${defaultCurrency.value})`,
                tt('Proportion (%)')
            ],
            data: categoricalAnalysisData.value.items
                .filter(item => !item.hidden)
                .map(item => [
                    item.name,
                    formatAmountToWesternArabicNumeralsWithoutDigitGrouping(item.value, defaultCurrency.value),
                    formatBigDecimalToWesternArabicNumeralsWithoutDigitGrouping(parseBigDecimal(item.percent.toFixed(4)))
                ]),
            supportedMermaidCharts: supportedMermaidCharts
        });
    } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis && trendsAnalysisData.value && trendsAnalysisData.value.items && monthlyTrendsChart.value) {
        const exportData = monthlyTrendsChart.value.exportData();
        let supportedMermaidCharts: ExportMermaidChartType[] | undefined = undefined;

        if (exportData.headers.length === 2 && query.value.trendChartType === TrendChartType.Column.type) {
            supportedMermaidCharts = [ ExportMermaidChartType.XYChartBar ];
        } else if (exportData.headers.length === 2 && query.value.trendChartType === TrendChartType.Area.type) {
            supportedMermaidCharts = [ ExportMermaidChartType.XYChartLine ];
        }

        dataExportDialog.value?.open({
            headers: exportData.headers || [],
            data: exportData.data || [],
            supportedMermaidCharts: supportedMermaidCharts
        });
    } else if (analysisType.value === StatisticsAnalysisType.AssetTrends && assetTrendsData.value && assetTrendsData.value.items && dailyTrendsChart.value) {
        const exportData = dailyTrendsChart.value.exportData();
        let supportedMermaidCharts: ExportMermaidChartType[] | undefined = undefined;

        if (exportData.headers.length === 2 && query.value.assetTrendsChartType === TrendChartType.Column.type) {
            supportedMermaidCharts = [ ExportMermaidChartType.XYChartBar ];
        } else if (exportData.headers.length === 2 && query.value.assetTrendsChartType === TrendChartType.Area.type) {
            supportedMermaidCharts = [ ExportMermaidChartType.XYChartLine ];
        }

        dataExportDialog.value?.open({
            headers: exportData.headers || [],
            data: exportData.data || [],
            supportedMermaidCharts: supportedMermaidCharts
        });
    }
}

function onClickSankeyChartItem(sourceItemType: 'account' | 'category', sourceItemId: string, targetItemType?: 'account' | 'category', targetItemId?: string): void {
    if (sourceItemType === 'category' && targetItemType === 'category' && sourceItemId && targetItemId) {
        const sourceCategory = transactionCategoriesStore.allTransactionCategoriesMap[sourceItemId];
        const targetCategory = transactionCategoriesStore.allTransactionCategoriesMap[targetItemId];

        if (sourceCategory?.parentId === targetCategory?.id) {
            router.push(getTransactionItemLinkUrl(`${sourceItemType}:${sourceItemId}`));
            return;
        } else if (targetCategory?.parentId === sourceCategory?.id) {
            router.push(getTransactionItemLinkUrl(`${targetItemType}:${targetItemId}`));
            return;
        }
    }

    router.push(getTransactionItemLinkUrl(`${sourceItemType}:${sourceItemId}` + (targetItemType && targetItemId ? `-${targetItemType}:${targetItemId}` : '')));
}

function onClickPieChartItem(item: Record<string, unknown>): void {
    router.push(getTransactionItemLinkUrl(item['id'] as string));
}

function onClickTrendChartItem(item: { itemId: string, dateRange: TimeRangeAndDateType }): void {
    router.push(getTransactionItemLinkUrl(item.itemId, item.dateRange));
}

function onShowDateRangeError(message: string): void {
    desktopPageStore.addErrorToast(message);
}

onBeforeRouteUpdate((to) => {
    if (to.query) {
        init({
            initAnalysisType: (to.query['analysisType'] as string | null) || undefined,
            initChartDataType: (to.query['chartDataType'] as string | null) || undefined,
            initChartType: (to.query['chartType'] as string | null) || undefined,
            initChartDateType: (to.query['chartDateType'] as string | null) || undefined,
            initStartTime: (to.query['startTime'] as TextualYearMonth | null) || undefined,
            initEndTime: (to.query['endTime'] as TextualYearMonth | null) || undefined,
            initFilterAccountIds: (to.query['filterAccountIds'] as string | null) || undefined,
            initFilterCategoryIds: (to.query['filterCategoryIds'] as string | null) || undefined,
            initTagFilter: (to.query['tagFilter'] as string | null) || undefined,
            initKeyword: (to.query['keyword'] as string | null) || undefined,
            initMatchMode: (to.query['matchMode'] as string | null) || undefined,
            initSortingType: (to.query['sortingType'] as string | null) || undefined,
            initTrendDateAggregationType: (to.query['trendDateAggregationType'] as string | null) || undefined,
            initAssetTrendsDateAggregationType: (to.query['assetTrendsDateAggregationType'] as string | null) || undefined
        });
    } else {
        init({});
    }
});

init(props);
</script>

<style lang="scss">
.statistics-custom-datetime-range {
    line-height: 1rem;
}

/* REPORT OPTIONS side panel */
.statistics-report-options-item {
    margin-block-start: var(--ebk-space-4);
    list-style: none;
}

.statistics-report-options {
    box-shadow: none;
}

.statistics-report-options-body {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-2);
    padding: var(--ebk-space-3);
}

.statistics-report-option-label {
    margin-block-start: var(--ebk-space-2);
    color: var(--ebk-color-text-secondary);

    &:first-child {
        margin-block-start: 0;
    }
}

.statistics-report-data-types {
    margin: 0 calc(-1 * var(--ebk-space-2));
    padding: 0;
    list-style: none;
}

/* Report content */
.statistics-report {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-4);
    width: 100%;
}

.statistics-filter-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-2);
    padding: var(--ebk-space-3) var(--ebk-space-4);
}

.statistics-period-stepper .statistics-period-name {
    min-width: 9rem;
    font-variant-numeric: tabular-nums;
}

.statistics-keyword-filter {
    flex: 0 1 18rem;
    min-width: 12rem;
}

.statistics-panel-meta {
    color: var(--ebk-color-text-secondary);
    font-weight: 400;
    letter-spacing: normal !important;
    text-transform: none !important;
    font-variant-numeric: tabular-nums;
}

.ebk-panel-header-strip > .statistics-panel-meta:not(.ebk-panel-header-actions) {
    margin-inline-start: var(--ebk-space-2);
}

/* KPI tiles */
.statistics-kpi-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: var(--ebk-space-4);
}

.statistics-kpi-row-single {
    grid-template-columns: minmax(14rem, 24rem);
}

.statistics-kpi-body {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-1);
    padding: var(--ebk-space-3) var(--ebk-space-4) var(--ebk-space-4);
}

.statistics-kpi-amount {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-2);
    min-width: 0;
    font-size: 2rem;
    font-weight: 600;
    line-height: 1.2;

    > .ebk-signed-amount {
        overflow: hidden;
        text-overflow: ellipsis;
    }
}

.statistics-kpi-meta {
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
}

/* Chart panel */
.statistics-chart-panel-body {
    padding: var(--ebk-space-4);
}

.statistics-chart-figure {
    margin: 0;
    min-width: 0;
}

/* Ranked list (bar chart type) */
.statistics-ranked-list {
    margin: 0;
    padding: 0;
    list-style: none;
}

.statistics-ranked-item {
    border-block-end: 1px solid var(--ebk-color-hairline);

    &:last-child {
        border-block-end: 0;
    }

    > .v-icon {
        margin-inline-end: var(--ebk-space-3);
    }
}

.statistics-ranked-item,
.statistics-ranked-item-link {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-3);
    min-height: 56px;
    padding-block: var(--ebk-space-2);
}

.statistics-ranked-item-link {
    flex: 1 1 auto;
    min-width: 0;
    padding-inline: var(--ebk-space-2);
    border-radius: var(--ebk-radius-control);
    color: inherit;
    text-decoration: none;

    &:hover {
        background-color: rgba(var(--ebk-color-hairline-rgb), 0.06);
    }
}

.statistics-ranked-item-main {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    gap: var(--ebk-space-2);
    min-width: 0;
}

.statistics-ranked-item-row {
    display: flex;
    align-items: baseline;
    gap: var(--ebk-space-2);
    min-width: 0;
}

.statistics-ranked-item-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.statistics-ranked-item-percent {
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
}

.statistics-ranked-item-amount {
    margin-inline-start: auto;
    font-weight: 600;
}

/* Breakdown and summary */
.statistics-detail-row {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(16rem, 1fr);
    align-items: start;
    gap: var(--ebk-space-4);
}

@media (max-width: 1279px) {
    .statistics-detail-row {
        grid-template-columns: minmax(0, 1fr);
    }
}

.statistics-breakdown-scroll {
    overflow: hidden;
    border-end-start-radius: var(--ebk-radius-panel);
    border-end-end-radius: var(--ebk-radius-panel);
}

.statistics-breakdown-table.v-table {
    background: transparent;

    th, td {
        white-space: nowrap;
    }

    td {
        font-variant-numeric: tabular-nums;
    }

    tfoot tr.statistics-breakdown-total td {
        border-block-start: 1px solid var(--ebk-color-hairline);
        background-color: var(--ebk-color-surface-sunken);
        font-weight: 600;
    }
}

.statistics-breakdown-name {
    max-width: 22rem;
}

.statistics-breakdown-icon {
    flex: 0 0 auto;
    margin-inline-end: var(--ebk-space-2);
}

.statistics-breakdown-link,
.statistics-breakdown-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.statistics-breakdown-link {
    color: inherit;
    text-decoration: none;

    &:hover {
        text-decoration: underline;
    }
}

.statistics-summary-body {
    padding: var(--ebk-space-4);
    line-height: 1.6;

    > p + p {
        margin-block-start: var(--ebk-space-2);
    }
}
</style>

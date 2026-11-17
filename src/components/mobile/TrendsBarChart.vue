<template>
    <f7-list strong inset dividers class="skeleton-text trends-bar-chart" :style="chartThemeStyle" v-if="loading">
        <f7-list-item class="statistics-list-item item-no-divider" link="#" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12 ]">
            <template #media>
                <div class="display-flex no-padding-horizontal">
                    <div class="display-flex align-items-center statistics-icon">
                        <f7-icon f7="app_fill"></f7-icon>
                    </div>
                </div>
            </template>
            <template #title>
                <div class="statistics-list-item-text">
                    <span>Date Range</span>
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

    <f7-list strong inset dividers v-else-if="!loading && (!allDisplayDataItems || !allDisplayDataItems.data || !allDisplayDataItems.data.length)">
        <f7-list-item :title="tt('No transaction data')"></f7-list-item>
    </f7-list>

    <f7-list strong inset dividers class="trends-bar-chart" :style="chartThemeStyle" v-if="!loading && allDisplayDataItems && allDisplayDataItems.data && allDisplayDataItems.data.length">
        <f7-list-item class="item-no-divider display-none"></f7-list-item>
        <f7-list-item class="item-no-divider" v-if="allDisplayDataItems.legends && allDisplayDataItems.legends.length > 1">
            <div class="trends-bar-chart-legends">
                <div class="trends-bar-chart-legend display-flex align-items-center"
                     role="button" tabindex="0"
                     :class="{ 'trends-bar-chart-legend-unselected': !!unselectedLegends[legend.id] }"
                     :aria-pressed="unselectedLegends[legend.id] ? 'false' : 'true'"
                     :key="idx"
                     v-for="(legend, idx) in allDisplayDataItems.legends"
                     @click="toggleLegend(legend)"
                     @keydown.enter.prevent="toggleLegend(legend)"
                     @keydown.space.prevent="toggleLegend(legend)">
                    <span class="trends-bar-chart-symbol" aria-hidden="true"
                          :class="`trends-bar-chart-symbol-${legend.symbol}`"
                          :style="{ 'color': legend.color }"></span>
                    <span class="trends-bar-chart-legend-text">{{ legend.name }}</span>
                </div>
            </div>
        </f7-list-item>
    </f7-list>

    <f7-list strong inset dividers class="trends-bar-chart" :style="chartThemeStyle" :key="`trends-bar-chart-${allDisplayDataItemsVersion}`"
             :virtual-list="useVirtualList"
             :virtual-list-params="useVirtualList ? { items: allDisplayDataItems.data, renderExternal, height: 'auto' } : undefined"
             v-if="!loading && allDisplayDataItems && allDisplayDataItems.data && allDisplayDataItems.data.length">
        <f7-list-item class="item-no-divider display-none"></f7-list-item>
        <f7-list-item class="item-no-divider" link="#"
                      :key="item.index"
                      :class="{ 'statistics-list-item': true, 'statistics-list-item-stacked': stacked, 'statistics-list-item-non-stacked': !stacked }"
                      :style="useVirtualList ? `top: ${virtualDataItems.topPosition}px` : undefined"
                      :virtual-list-index="item.index"
                      v-for="item in (useVirtualList ? virtualDataItems.items : allDisplayDataItems.data)"
                      @click="clickItem(item)"
        >
            <template #media>
                <div class="display-flex no-padding-horizontal">
                    <div class="display-flex align-items-center statistics-icon trends-bar-chart-icon">
                        <f7-icon f7="calendar"></f7-icon>
                    </div>
                </div>
            </template>

            <template #title>
                <div class="statistics-list-item-text">
                    <span>{{ item.displayDateRange }}</span>
                </div>
                <div class="width-100 statistics-percent-line statistics-multi-percent-line display-flex flex-direction-column" v-if="!stacked && item.items.length > 1">
                    <div class="display-flex flex-direction-column"
                         style="margin-top: 4px"
                         :key="dataIdx"
                         v-for="(data, dataIdx) in item.items"
                         v-show="data.value.isPositive()">
                        <div class="width-100 display-flex flex-direction-row align-items-center">
                            <span class="trends-bar-chart-symbol trends-bar-chart-bar-symbol" aria-hidden="true"
                                  :class="`trends-bar-chart-symbol-${data.symbol}`"
                                  :style="{ 'color': data.color }"></span>
                            <div class="display-inline-flex" :style="{ 'width': data.value.divide(item.maxAmount).multiply(item.percent).toDoubleNumber() + '%' }">
                                <f7-progressbar :progress="100" :style="{ '--f7-progressbar-progress-color': (data.color ? data.color : '') } "></f7-progressbar>
                            </div>
                            <div class="display-inline-flex" :style="{ 'width': (100.0 - data.value.divide(item.maxAmount).multiply(item.percent).toDoubleNumber()) + '%' }"
                                 v-if="data.value.divide(item.maxAmount).multiply(item.percent).lessThan(100)">
                                <f7-progressbar :progress="0"></f7-progressbar>
                            </div>
                        </div>
                    </div>
                </div>
            </template>

            <template #after>
                <span class="ebk-numeric trends-bar-chart-amount" v-if="stacked || item.items.length <= 1">{{ formatAmountToLocalizedNumeralsWithCurrency(item.totalAmount, defaultCurrency) }}</span>
            </template>

            <template #inner-end>
                <div class="statistics-item-end" v-if="stacked || item.items.length <= 1">
                    <div class="statistics-percent-line statistics-multi-percent-line display-flex">
                        <div class="display-inline-flex" :style="{ 'width': (data.value.divide(item.totalPositiveAmount).multiply(item.percent).toDoubleNumber()) + '%' }"
                             :key="dataIdx"
                             v-for="(data, dataIdx) in item.items"
                             v-show="data.value.isPositive()">
                            <f7-progressbar :progress="100" :style="{ '--f7-progressbar-progress-color': (data.color ? data.color : '') } "></f7-progressbar>
                        </div>
                        <div class="display-inline-flex" :style="{ 'width': (100.0 - item.percent) + '%' }"
                             v-if="item.percent < 100.0">
                            <f7-progressbar :progress="0"></f7-progressbar>
                        </div>
                    </div>
                </div>
            </template>
        </f7-list-item>
    </f7-list>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type TrendsChartDateType,
    type CommonTrendsChartProps,
    type TrendsBarChartClickEvent,
    useTrendsChartBase
} from '@/components/base/TrendsChartBase.ts'

import {
    getChartThemePalette,
    getChartSeriesSymbol
} from '@/components/base/ChartThemeBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useEnvironmentsStore } from '@/stores/environment.ts';

import { itemAndIndex } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import {
    type UnixTimeRange,
    DateRangeScene
} from '@/core/datetime.ts';
import type { ColorValue, ColorStyleValue } from '@/core/color.ts';
import {
    ChartDataAggregationType,
    ChartDateAggregationType
} from '@/core/statistics.ts';

import type { SortableTransactionStatisticDataItem } from '@/models/transaction.ts';

import {
    BIG_DECIMAL_ZERO,
    isBigDecimal
} from '@/lib/numeral.ts';
import {
    parseDateTimeFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    getDateTypeByDateRange,
    getFiscalYearFromUnixTime
} from '@/lib/datetime.ts';
import {
    getDisplayColor
} from '@/lib/color.ts';
import {
    sortStatisticsItems
} from '@/lib/statistics.ts';

interface TrendsBarChartLegend {
    readonly id: string;
    readonly name: string;
    readonly color: ColorStyleValue;
    readonly symbol: string;
    readonly displayOrders: number[];
}

interface TrendsBarChartDataAmount extends SortableTransactionStatisticDataItem, TrendsBarChartLegend {
    value: BigDecimal;
}

interface TrendsBarChartDataItem {
    index: number;
    dateRange: UnixTimeRange;
    displayDateRange: string;
    items: TrendsBarChartDataAmount[];
    totalAmount: BigDecimal;
    totalPositiveAmount: BigDecimal;
    maxAmount: BigDecimal;
    percent: number;
}

interface TrendsBarChartVirtualListData {
    items: TrendsBarChartDataItem[],
    topPosition: number
}

interface TrendsBarChartData {
    readonly data: TrendsBarChartDataItem[];
    readonly legends: TrendsBarChartLegend[];
}

interface MobileTrendsChartProps<T extends TrendsChartDateType> extends CommonTrendsChartProps<T> {
    loading?: boolean;
}

const props = defineProps<MobileTrendsChartProps<TrendsChartDateType>>();

const emit = defineEmits<{
    (e: 'click', value: TrendsBarChartClickEvent): void;
}>();

const {
    tt,
    formatDateTimeToShortDate,
    formatDateTimeToGregorianLikeShortYear,
    formatDateTimeToGregorianLikeShortYearMonth,
    formatYearQuarterToGregorianLikeYearQuarter,
    formatDateTimeToGregorianLikeFiscalYear,
    formatAmountToLocalizedNumeralsWithCurrency
} = useI18n();

const { allDateRanges, getItemName } = useTrendsChartBase(props);

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const environmentsStore = useEnvironmentsStore();

const allDisplayDataItemsVersion = ref<number>(0);
const unselectedLegends = ref<Record<string, boolean>>({});

const virtualDataItems = ref<TrendsBarChartVirtualListData>({
    items: [],
    topPosition: 0
});

const chartColors = computed<ColorValue[]>(() => settingsStore.chartColorList);
const useVirtualList = computed<boolean>(() => allDisplayDataItems.value.legends.length <= 1 || props.stacked);

// the same panel palette as the desktop ECharts charts (track at hairline strength, text and inactive legend colours)
const chartThemeStyle = computed<Record<string, string>>(() => {
    const palette = getChartThemePalette(environmentsStore.framework7DarkMode);

    return {
        '--ebk-chart-track-color': palette.hairline,
        '--ebk-chart-text-primary': palette.textPrimary,
        '--ebk-chart-text-secondary': palette.textSecondary,
        '--ebk-chart-inactive-color': palette.axisLine
    };
});

const allDisplayDataItems = computed<TrendsBarChartData>(() => {
    const allDateRangeItemsMap: Record<string, TrendsBarChartDataAmount[]> = {};
    const legends: TrendsBarChartLegend[] = [];

    for (const [item, index] of itemAndIndex(props.items)) {
        if (item.hidden) {
            continue;
        }

        const id = item.id ?? getItemName(item.name);

        const legend: TrendsBarChartLegend = {
            id: id,
            name: item.name ? getItemName(item.name) : id,
            color: getDisplayColor(props.useCustomColor && item.color ? item.color : chartColors.value[index % chartColors.value.length]),
            symbol: getChartSeriesSymbol(legends.length), // a shape per series so series are distinguishable without colour (D7)
            displayOrders: item.displayOrders ? item.displayOrders : [0]
        };

        legends.push(legend);

        if (unselectedLegends.value[id]) {
            continue;
        }

        const dateRangeItemMap: Record<string, TrendsBarChartDataAmount> = {};

        for (const dataItem of item.items) {
            let dateRangeKey = '';

            if (props.chartMode === 'daily' && 'month' in dataItem) {
                if (props.dateAggregationType === ChartDateAggregationType.Year.type) {
                    dateRangeKey = dataItem.year.toString();
                } else if (props.dateAggregationType === ChartDateAggregationType.FiscalYear.type) {
                    const fiscalYear = getFiscalYearFromUnixTime(
                        getYearMonthFirstUnixTime({ year: dataItem.year, month1base: dataItem.month }),
                        props.fiscalYearStart
                    );
                    dateRangeKey = fiscalYear.toString();
                } else if (props.dateAggregationType === ChartDateAggregationType.Quarter.type) {
                    dateRangeKey = `${dataItem.year}-${Math.floor((dataItem.month - 1) / 3) + 1}`;
                } else if (props.dateAggregationType === ChartDateAggregationType.Month.type) {
                    dateRangeKey = `${dataItem.year}-${dataItem.month}`;
                } else { // if (props.dateAggregationType === ChartDateAggregationType.Day.type) {
                    dateRangeKey = `${dataItem.year}-${dataItem.month}-${dataItem.day}`;
                }
            } else if (props.chartMode === 'monthly' && 'month1base' in dataItem) {
                if (props.dateAggregationType === ChartDateAggregationType.Year.type) {
                    dateRangeKey = dataItem.year.toString();
                } else if (props.dateAggregationType === ChartDateAggregationType.FiscalYear.type) {
                    const fiscalYear = getFiscalYearFromUnixTime(
                        getYearMonthFirstUnixTime({ year: dataItem.year, month1base: dataItem.month1base }),
                        props.fiscalYearStart
                    );
                    dateRangeKey = fiscalYear.toString();
                } else if (props.dateAggregationType === ChartDateAggregationType.Quarter.type) {
                    dateRangeKey = `${dataItem.year}-${Math.floor((dataItem.month1base - 1) / 3) + 1}`;
                } else { // if (props.dateAggregationType === ChartDateAggregationType.Month.type) {
                    dateRangeKey = `${dataItem.year}-${dataItem.month1base}`;
                }
            }

            const value = dataItem.value;

            if (dateRangeItemMap[dateRangeKey]) {
                if (isBigDecimal(value)) {
                    if (props.dataAggregationType === ChartDataAggregationType.Sum) {
                        dateRangeItemMap[dateRangeKey]!.value = dateRangeItemMap[dateRangeKey]!.value.add(value);
                    } else if (props.dataAggregationType === ChartDataAggregationType.Last) {
                        dateRangeItemMap[dateRangeKey]!.value = value;
                    }
                }
            } else {
                const allDataItems: TrendsBarChartDataAmount[] = allDateRangeItemsMap[dateRangeKey] || [];
                const finalDataItem: TrendsBarChartDataAmount = Object.assign({}, legend, {
                    value: isBigDecimal(value) ? value : BIG_DECIMAL_ZERO
                });

                allDataItems.push(finalDataItem);
                dateRangeItemMap[dateRangeKey] = finalDataItem;
                allDateRangeItemsMap[dateRangeKey] = allDataItems;
            }
        }
    }

    const finalDataItems: TrendsBarChartDataItem[] = [];
    let maxTotalAmount: BigDecimal = BIG_DECIMAL_ZERO;

    for (const dateRange of allDateRanges.value) {
        let dateRangeKey = '';

        if (props.dateAggregationType === ChartDateAggregationType.Year.type) {
            dateRangeKey = dateRange.year.toString();
        } else if (props.dateAggregationType === ChartDateAggregationType.FiscalYear.type) {
            dateRangeKey = dateRange.year.toString();
        } else if (props.dateAggregationType === ChartDateAggregationType.Quarter.type && 'quarter' in dateRange) {
            dateRangeKey = `${dateRange.year}-${dateRange.quarter}`;
        } else if (props.dateAggregationType === ChartDateAggregationType.Month.type && 'month0base' in dateRange) {
            dateRangeKey = `${dateRange.year}-${dateRange.month0base + 1}`;
        } else if (props.dateAggregationType === ChartDateAggregationType.Day.type && 'day' in dateRange && props.chartMode === 'daily') {
            dateRangeKey = `${dateRange.year}-${dateRange.month}-${dateRange.day}`;
        }

        const minDateTime = parseDateTimeFromUnixTime(dateRange.minUnixTime);
        let displayDateRange = '';

        if (props.dateAggregationType === ChartDateAggregationType.Year.type) {
            displayDateRange = formatDateTimeToGregorianLikeShortYear(minDateTime);
        } else if (props.dateAggregationType === ChartDateAggregationType.FiscalYear.type) {
            displayDateRange = formatDateTimeToGregorianLikeFiscalYear(minDateTime);
        } else if (props.dateAggregationType === ChartDateAggregationType.Quarter.type && 'quarter' in dateRange) {
            displayDateRange = formatYearQuarterToGregorianLikeYearQuarter(dateRange.year, dateRange.quarter);
        } else if (props.dateAggregationType === ChartDateAggregationType.Month.type) {
            displayDateRange = formatDateTimeToGregorianLikeShortYearMonth(minDateTime);
        } else if (props.dateAggregationType === ChartDateAggregationType.Day.type && props.chartMode === 'daily') {
            displayDateRange = formatDateTimeToShortDate(minDateTime);
        }

        const dataItems = allDateRangeItemsMap[dateRangeKey] || [];
        let totalAmount: BigDecimal = BIG_DECIMAL_ZERO;
        let totalPositiveAmount = BIG_DECIMAL_ZERO;
        let maxAmount = BIG_DECIMAL_ZERO;

        sortStatisticsItems(dataItems, props.sortingType);

        for (const dataItem of dataItems) {
            if (dataItem.value.isPositive()) {
                totalPositiveAmount = totalPositiveAmount.add(dataItem.value);
            }

            totalAmount = totalAmount.add(dataItem.value);

            if (dataItem.value.greaterThan(maxAmount)) {
                maxAmount = dataItem.value;
            }
        }

        if (totalAmount.greaterThan(maxTotalAmount)) {
            maxTotalAmount = totalAmount;
        }

        const finalDataItem: TrendsBarChartDataItem = {
            index: finalDataItems.length,
            dateRange: dateRange,
            displayDateRange: displayDateRange,
            items: dataItems,
            totalAmount: totalAmount,
            totalPositiveAmount: totalPositiveAmount,
            maxAmount: maxAmount,
            percent: 0.0
        };

        finalDataItems.push(finalDataItem);
    }

    for (const finalDataItem of finalDataItems) {
        if (maxTotalAmount.isPositive() && finalDataItem.totalAmount.isPositive()) {
            finalDataItem.percent = finalDataItem.totalAmount.divide(maxTotalAmount).multiply(100).toDoubleNumber();
        } else {
            finalDataItem.percent = 0.0;
        }
    }

    return {
        data: finalDataItems,
        legends: legends
    };
});

function clickItem(item: TrendsBarChartDataItem): void {
    let itemId = '';

    for (const item of props.items) {
        if (item.hidden) {
            continue;
        }

        const id = item.id ?? getItemName(item.name);

        if (unselectedLegends.value[id]) {
            continue;
        }

        if (itemId.length) {
            itemId += ',';
        }

        itemId += id;
    }

    const dateRange = item.dateRange;
    let minUnixTime = dateRange.minUnixTime;
    let maxUnixTime = dateRange.maxUnixTime;

    if (props.chartMode === 'daily') {
        if (props.startTime) {
            if (props.startTime > minUnixTime) {
                minUnixTime = props.startTime;
            }
        }

        if (props.endTime) {
            if (props.endTime < maxUnixTime) {
                maxUnixTime = props.endTime;
            }
        }
    } else if (props.chartMode === 'monthly') {
        if (props.startYearMonth) {
            const startMinUnixTime = getYearMonthFirstUnixTime(props.startYearMonth);

            if (startMinUnixTime > minUnixTime) {
                minUnixTime = startMinUnixTime;
            }
        }

        if (props.endYearMonth) {
            const endMaxUnixTime = getYearMonthLastUnixTime(props.endYearMonth);

            if (endMaxUnixTime < maxUnixTime) {
                maxUnixTime = endMaxUnixTime;
            }
        }
    }

    const dateRangeType = getDateTypeByDateRange(minUnixTime, maxUnixTime, userStore.currentUserFirstDayOfWeek, userStore.currentUserFiscalYearStart, DateRangeScene.Normal);

    emit('click', {
        itemId: itemId,
        dateRange: {
            minTime: minUnixTime,
            maxTime: maxUnixTime,
            dateType: dateRangeType
        }
    });
}

function toggleLegend(legend: TrendsBarChartLegend): void {
    if (unselectedLegends.value[legend.id]) {
        delete unselectedLegends.value[legend.id];
    } else {
        unselectedLegends.value[legend.id] = true;
    }
}

function renderExternal(vl: unknown, vlData: TrendsBarChartVirtualListData): void {
    virtualDataItems.value = vlData;
}

watch(allDisplayDataItems, () => {
    allDisplayDataItemsVersion.value++;
}, {
    deep: true
});
</script>

<style>
.trends-bar-chart {
    --f7-progressbar-bg-color: var(--ebk-chart-track-color);
    --f7-progressbar-border-radius: 999px;

    .statistics-percent-line .progressbar {
        --f7-progressbar-bg-color: var(--ebk-chart-track-color);
        height: 6px;
    }

    .statistics-list-item-text {
        color: var(--ebk-chart-text-primary);
    }
}

.trends-bar-chart-icon .f7-icons {
    color: var(--ebk-chart-text-secondary);
}

.trends-bar-chart-amount {
    color: var(--ebk-chart-text-primary);
    font-variant-numeric: tabular-nums;
}

.trends-bar-chart-legends {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
}

.trends-bar-chart-legend {
    gap: 6px;
    min-height: 32px;
    cursor: pointer;
    border-radius: var(--ebk-radius-control);
}

.trends-bar-chart-legend-text {
    color: var(--ebk-chart-text-primary);
    font-size: var(--ebk-trends-bar-chart-legend-text-font-size);
}

.trends-bar-chart-legend-unselected {
    .trends-bar-chart-symbol {
        color: var(--ebk-chart-inactive-color) !important;
    }

    .trends-bar-chart-legend-text {
        color: var(--ebk-chart-inactive-color);
        text-decoration: line-through;
    }
}

/* series shapes, the same order as the desktop chart symbols (ChartThemeBase CHART_SERIES_SYMBOLS) */
.trends-bar-chart-symbol {
    display: inline-block;
    flex: 0 0 auto;
    width: 10px;
    height: 10px;
    background-color: currentColor;
}

.trends-bar-chart-bar-symbol {
    width: 8px;
    height: 8px;
    margin-inline-end: 6px;
}

.trends-bar-chart-symbol-circle {
    border-radius: 50%;
}

.trends-bar-chart-symbol-roundRect {
    border-radius: 3px;
}

.trends-bar-chart-symbol-triangle {
    clip-path: polygon(50% 0, 100% 100%, 0 100%);
}

.trends-bar-chart-symbol-diamond {
    clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%);
}

.trends-bar-chart-symbol-pin {
    border-radius: 50% 50% 50% 0;
    transform: rotate(-45deg);
}

.trends-bar-chart-symbol-arrow {
    clip-path: polygon(50% 0, 100% 100%, 50% 70%, 0 100%);
}

@media (forced-colors: active) {
    .trends-bar-chart-symbol {
        forced-color-adjust: none;
        background-color: CanvasText;
    }
}
</style>

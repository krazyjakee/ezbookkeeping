import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { type TransactionStatisticsFilter, useStatisticsStore } from '@/stores/statistics.ts';

import type { TypeAndDisplayName } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import { type LocalizedDateRange, type WeekDayValue, DateRangeScene, DateRange } from '@/core/datetime.ts';
import type { ColorStyleValue } from '@/core/color.ts';
import type { AxisChartSourceDataItem } from '@/core/chart.ts';
import { CategoryType } from '@/core/category.ts';
import { TransactionRelatedAccountType } from '@/core/transaction.ts';
import {
    StatisticsAnalysisType,
    ChartDataType,
    ChartSortingType,
    ChartDateAggregationType,
    TrendChartType
} from '@/core/statistics.ts';

import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';

import type {
    TransactionCategoricalOverviewAnalysisData,
    TransactionCategoricalAnalysisData,
    TransactionCategoricalAnalysisDataItem,
    TransactionTrendsAnalysisData,
    TransactionAssetTrendsAnalysisData
} from '@/models/transaction.ts';

import { limitText, findNameByType, findDisplayNameByType } from '@/lib/common.ts';
import { BIG_DECIMAL_ZERO, isBigDecimal, parseBigDecimal } from '@/lib/numeral.ts';
import {
    parseDateTimeFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime
} from '@/lib/datetime.ts';
import { getDisplayColor, getCategoryDisplayColor, getAccountDisplayColor } from '@/lib/color.ts';

/**
 * Income, expense and net movement of the loaded statistics data, shown in the KPI tiles and summaries (#32).
 * They are calculated from the data the page has already loaded, no additional requests are made.
 */
export interface StatisticsIncomeExpenseTotals {
    readonly income: BigDecimal;
    readonly expense: BigDecimal;
    readonly net: BigDecimal;
}

export type StatisticsAmountKind = 'income' | 'expense' | 'none';

export function useStatisticsTransactionPageBase() {
    const {
        tt,
        getAllDateRanges,
        getAllStatisticsSortingTypes,
        getAllStatisticsDateAggregationTypes,
        formatDateTimeToLongDateTime,
        formatDateTimeToGregorianLikeLongYearMonth,
        formatDateRange,
        formatAmountToLocalizedNumerals,
        formatAmountToLocalizedNumeralsWithCurrency
    } = useI18n();

    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const exchangeRatesStore = useExchangeRatesStore();
    const statisticsStore = useStatisticsStore();

    const loading = ref<boolean>(true);
    const analysisType = ref<StatisticsAnalysisType>(StatisticsAnalysisType.CategoricalAnalysis);
    const trendDateAggregationType = ref<number>(ChartDateAggregationType.Default.type);
    const assetTrendsDateAggregationType = ref<number>(ChartDateAggregationType.Default.type);

    const showAccountBalance = computed<boolean>(() => settingsStore.appSettings.showAccountBalance);
    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const firstDayOfWeek = computed<WeekDayValue>(() => userStore.currentUserFirstDayOfWeek);
    const fiscalYearStart = computed<number>(() => userStore.currentUserFiscalYearStart);

    const allDateRanges = computed<LocalizedDateRange[]>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return getAllDateRanges(DateRangeScene.Normal, { includeCustom: true });
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return getAllDateRanges(DateRangeScene.TrendAnalysis, { includeCustom: true });
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return getAllDateRanges(DateRangeScene.AssetTrends, { includeCustom: true });
        } else {
            return [];
        }
    });
    const allSortingTypes = computed<TypeAndDisplayName[]>(() => getAllStatisticsSortingTypes());
    const allTrendAnalysisDateAggregationTypes = computed<TypeAndDisplayName[]>(() => getAllStatisticsDateAggregationTypes(StatisticsAnalysisType.TrendAnalysis, false));
    const allAssetTrendsDateAggregationTypes = computed<TypeAndDisplayName[]>(() => getAllStatisticsDateAggregationTypes(StatisticsAnalysisType.AssetTrends, false));

    const query = computed<TransactionStatisticsFilter>(() => statisticsStore.transactionStatisticsFilter);
    const queryChartDataCategory = computed<string>(() => statisticsStore.categoricalAnalysisChartDataCategory);
    const queryDateType = computed<number | null>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return query.value.categoricalChartDateType;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return query.value.trendChartDateType;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return query.value.assetTrendsChartDateType;
        } else {
            return null;
        }
    });

    const queryStartTime = computed<string>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(query.value.categoricalChartStartTime));
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(getYearMonthFirstUnixTime(query.value.trendChartStartYearMonth)));
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(query.value.assetTrendsChartStartTime));
        } else {
            return '';
        }
    });

    const queryEndTime = computed<string>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(query.value.categoricalChartEndTime));
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(getYearMonthLastUnixTime(query.value.trendChartEndYearMonth)));
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(query.value.assetTrendsChartEndTime));
        } else {
            return '';
        }
    });

    const queryDateRangeName = computed<string>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
                query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
                query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
                return tt(DateRange.All.name);
            }

            return formatDateRange(query.value.categoricalChartDateType, query.value.categoricalChartStartTime, query.value.categoricalChartEndTime);
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return formatDateRange(query.value.trendChartDateType, getYearMonthFirstUnixTime(query.value.trendChartStartYearMonth), getYearMonthLastUnixTime(query.value.trendChartEndYearMonth));
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return formatDateRange(query.value.assetTrendsChartDateType, query.value.assetTrendsChartStartTime, query.value.assetTrendsChartEndTime);
        } else {
            return '';
        }
    });

    const queryChartDataTypeName = computed<string>(() => {
        const queryChartDataTypeName = findNameByType(ChartDataType.values(), query.value.chartDataType) || 'Statistics';
        return tt(queryChartDataTypeName);
    });

    const querySortingTypeName = computed<string>(() => {
        const querySortingTypeName = findNameByType(ChartSortingType.values(), query.value.sortingType) || 'System Default';
        return tt(querySortingTypeName);
    });

    const queryTrendDateAggregationTypeName = computed<string>(() => findDisplayNameByType(allTrendAnalysisDateAggregationTypes.value, trendDateAggregationType.value) || '');
    const queryAssetTrendsDateAggregationTypeName = computed<string>(() => findDisplayNameByType(allAssetTrendsDateAggregationTypes.value, assetTrendsDateAggregationType.value) || '');

    const isQueryDateRangeChanged = computed<boolean>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
                query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
                query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
                return false;
            }

            if (query.value.categoricalChartDateType === settingsStore.appSettings.statistics.defaultCategoricalChartDataRangeType) {
                return false;
            }

            return !!query.value.categoricalChartStartTime || !!query.value.categoricalChartEndTime;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            if (query.value.trendChartDateType === settingsStore.appSettings.statistics.defaultTrendChartDataRangeType) {
                return false;
            }

            return !!query.value.trendChartStartYearMonth || !!query.value.trendChartEndYearMonth;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            if (query.value.assetTrendsChartDateType === settingsStore.appSettings.statistics.defaultAssetTrendsChartDataRangeType) {
                return false;
            }

            return !!query.value.assetTrendsChartStartTime || !!query.value.assetTrendsChartEndTime;
        } else {
            return false;
        }
    });

    const canChangeDateRange = computed<boolean>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type || query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type || query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
                return false;
            }

            return true;
        } else {
            return true;
        }
    });

    const canShiftDateRange = computed<boolean>(() => {
        if (!canChangeDateRange.value) {
            return false;
        }

        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return query.value.categoricalChartDateType !== DateRange.All.type;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return query.value.trendChartDateType !== DateRange.All.type;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return query.value.assetTrendsChartDateType !== DateRange.All.type;
        } else {
            return false;
        }
    });

    const canUseCategoryFilter = computed<boolean>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type || query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type || query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
                return false;
            }
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return false;
        }

        return true;
    });

    const canUseServerCustomFilter = computed<boolean>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type || query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type || query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
                return false;
            }
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return false;
        }

        return true;
    });

    const canUseTagFilter = computed<boolean>(() => {
        return canUseServerCustomFilter.value;
    });

    const canUseKeywordFilter = computed<boolean>(() => {
        return canUseServerCustomFilter.value;
    });

    const showAmountInChart = computed<boolean>(() => {
        if (!showAccountBalance.value) {
            if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis
                && (query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
                    query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                    query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
                    query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type)) {
                return false;
            }
        }

        return true;
    });

    const totalAmountName = computed<string>(() => {
        if (query.value.chartDataType === ChartDataType.InflowsByAccount.type) {
            return tt('Total Inflows');
        } else if (query.value.chartDataType === ChartDataType.OutflowsByAccount.type) {
            return tt('Total Outflows');
        } else if (query.value.chartDataType === ChartDataType.IncomeByAccount.type
            || query.value.chartDataType === ChartDataType.IncomeByPrimaryCategory.type
            || query.value.chartDataType === ChartDataType.IncomeBySecondaryCategory.type) {
            return tt('Total Income');
        } else if (query.value.chartDataType === ChartDataType.ExpenseByAccount.type
            || query.value.chartDataType === ChartDataType.ExpenseByPrimaryCategory.type
            || query.value.chartDataType === ChartDataType.ExpenseBySecondaryCategory.type) {
            return tt('Total Expense');
        } else if (query.value.chartDataType === ChartDataType.AccountTotalAssets.type || query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type) {
            return tt('Total Assets');
        } else if (query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type || query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
            return tt('Total Liabilities');
        }

        return tt('Total Amount');
    });

    const showPercentInCategoricalChart = computed<boolean>(() => {
        return query.value.chartDataType !== ChartDataType.OutflowsByAccount.type &&
            query.value.chartDataType !== ChartDataType.InflowsByAccount.type;
    });

    const showTotalAmountInTrendsChart = computed<boolean>(() => {
        return query.value.chartDataType !== ChartDataType.OutflowsByAccount.type &&
            query.value.chartDataType !== ChartDataType.InflowsByAccount.type &&
            query.value.chartDataType !== ChartDataType.TotalOutflows.type &&
            query.value.chartDataType !== ChartDataType.TotalExpense.type &&
            query.value.chartDataType !== ChartDataType.TotalInflows.type &&
            query.value.chartDataType !== ChartDataType.TotalIncome.type &&
            query.value.chartDataType !== ChartDataType.NetCashFlow.type &&
            query.value.chartDataType !== ChartDataType.NetIncome.type &&
            query.value.chartDataType !== ChartDataType.NetWorth.type;
    });

    const showStackedInTrendsChart = computed<boolean>(() => {
        return (query.value.trendChartType === TrendChartType.Area.type || query.value.trendChartType === TrendChartType.Column.type) &&
            query.value.chartDataType !== ChartDataType.OutflowsByAccount.type &&
            query.value.chartDataType !== ChartDataType.InflowsByAccount.type;
    });

    const translateNameInTrendsChart = computed<boolean>(() => {
        return query.value.chartDataType === ChartDataType.TotalOutflows.type ||
            query.value.chartDataType === ChartDataType.TotalExpense.type ||
            query.value.chartDataType === ChartDataType.TotalInflows.type ||
            query.value.chartDataType === ChartDataType.TotalIncome.type ||
            query.value.chartDataType === ChartDataType.NetCashFlow.type ||
            query.value.chartDataType === ChartDataType.NetIncome.type ||
            query.value.chartDataType === ChartDataType.NetWorth.type;
    });

    const categoricalOverviewAnalysisData = computed<TransactionCategoricalOverviewAnalysisData | null>(() => statisticsStore.categoricalOverviewAnalysisData);
    const categoricalAnalysisData = computed<TransactionCategoricalAnalysisData>(() => {
        const data = statisticsStore.categoricalAnalysisData;

        if (query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
            query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type) {
            return {
                value: data.value,
                items: data.items.map(item => {
                    if (!isBigDecimal(item.originalValue) || !item.originalCurrency) {
                        return item;
                    }

                    return {
                        ...item,
                        name: `${item.name} ${formatAmountToLocalizedNumerals(item.originalValue, item.originalCurrency)}`
                    };
                })
            };
        } else {
            return data;
        }
    });
    const trendsAnalysisData = computed<TransactionTrendsAnalysisData | null>(() => statisticsStore.trendsAnalysisData);
    const assetTrendsData = computed<TransactionAssetTrendsAnalysisData | null>(() => statisticsStore.assetTrendsData);

    const isAccountBalanceChartDataType = computed<boolean>(() => {
        return query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
            query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
            query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
            query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type;
    });

    // whether the amounts of the current chart data type are income, expense or neither (balances, net amounts)
    const chartDataAmountKind = computed<StatisticsAmountKind>(() => {
        const chartDataType = query.value.chartDataType;

        if (chartDataType === ChartDataType.OutflowsByAccount.type ||
            chartDataType === ChartDataType.ExpenseByAccount.type ||
            chartDataType === ChartDataType.ExpenseByPrimaryCategory.type ||
            chartDataType === ChartDataType.ExpenseBySecondaryCategory.type ||
            chartDataType === ChartDataType.TotalOutflows.type ||
            chartDataType === ChartDataType.TotalExpense.type) {
            return 'expense';
        } else if (chartDataType === ChartDataType.InflowsByAccount.type ||
            chartDataType === ChartDataType.IncomeByAccount.type ||
            chartDataType === ChartDataType.IncomeByPrimaryCategory.type ||
            chartDataType === ChartDataType.IncomeBySecondaryCategory.type ||
            chartDataType === ChartDataType.TotalInflows.type ||
            chartDataType === ChartDataType.TotalIncome.type) {
            return 'income';
        }

        return 'none';
    });

    const incomeExpenseTotals = computed<StatisticsIncomeExpenseTotals | null>(() => {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            if (isAccountBalanceChartDataType.value) {
                return null;
            }

            const overviewData = statisticsStore.categoricalOverviewAnalysisData;

            if (!overviewData) {
                return null;
            }

            return {
                income: overviewData.totalIncome,
                expense: overviewData.totalExpense,
                net: overviewData.totalIncome.subtract(overviewData.totalExpense)
            };
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            const trendsData = statisticsStore.transactionCategoryTrendsData;

            if (!trendsData || !trendsData.length) {
                return null;
            }

            const currency = defaultCurrency.value;
            let income = BIG_DECIMAL_ZERO;
            let expense = BIG_DECIMAL_ZERO;

            for (const trendItem of trendsData) {
                for (const item of trendItem.items || []) {
                    const account = accountsStore.allAccountsMap[item.accountId];
                    const category = transactionCategoriesStore.allTransactionCategoriesMap[item.categoryId];

                    if (!account || !category) {
                        continue;
                    }

                    if (item.relatedAccountId && item.relatedAccountType === TransactionRelatedAccountType.TransferFrom) {
                        continue;
                    }

                    if (query.value.filterAccountIds && query.value.filterAccountIds[account.id]) {
                        continue;
                    }

                    if (query.value.filterCategoryIds && query.value.filterCategoryIds[category.id]) {
                        continue;
                    }

                    let amount: BigDecimal | null = parseBigDecimal(item.amount);

                    if (account.currency !== currency) {
                        amount = exchangeRatesStore.getExchangedAmount(amount, account.currency, currency);
                        amount = amount ? amount.truncate() : null;
                    }

                    if (!amount) {
                        continue;
                    }

                    if (category.type === CategoryType.Income) {
                        income = income.add(amount);
                    } else if (category.type === CategoryType.Expense) {
                        expense = expense.add(amount);
                    }
                }
            }

            return {
                income: income,
                expense: expense,
                net: income.subtract(expense)
            };
        }

        return null;
    });

    const radarChartCategoryNames = computed<string[]>(() => {
        return categoricalAnalysisData.value?.items.filter(item => !item.hidden).map(item => item.name) ?? [];
    });

    const radarChartData = computed<AxisChartSourceDataItem[]>(() => {
        const items = categoricalAnalysisData.value?.items.filter(item => !item.hidden) ?? [];

        if (!items.length) {
            return [];
        }

        return [{
            id: 'amount',
            name: tt('Amount'),
            values: items.map(item => item.value),
            displayOrders: [0]
        }];
    });

    function canShowCustomDateRange(dateRangeType: number): boolean {
        if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis) {
            return query.value.categoricalChartDateType === dateRangeType && !!query.value.categoricalChartStartTime && !!query.value.categoricalChartEndTime;
        } else if (analysisType.value === StatisticsAnalysisType.TrendAnalysis) {
            return query.value.trendChartDateType === dateRangeType && !!query.value.trendChartStartYearMonth && !!query.value.trendChartEndYearMonth;
        } else if (analysisType.value === StatisticsAnalysisType.AssetTrends) {
            return query.value.assetTrendsChartDateType === dateRangeType && !!query.value.assetTrendsChartStartTime && !!query.value.assetTrendsChartEndTime;
        } else {
            return false;
        }
    }

    function getTransactionCategoricalAnalysisDataItemDisplayColor(item: TransactionCategoricalAnalysisDataItem): ColorStyleValue {
        if (item.type === 'category') {
            return getCategoryDisplayColor(item.color);
        } else if (item.type === 'account') {
            return getAccountDisplayColor(item.color);
        } else {
            return getDisplayColor(item.color);
        }
    }

    function getDisplayAmount(amount: BigDecimal, currency: string, textLimit?: number): string {
        const finalAmount = formatAmountToLocalizedNumeralsWithCurrency(amount, currency);

        if (!showAccountBalance.value) {
            if (analysisType.value === StatisticsAnalysisType.CategoricalAnalysis
                && (query.value.chartDataType === ChartDataType.AccountTotalAssets.type ||
                    query.value.chartDataType === ChartDataType.AccountTotalLiabilities.type ||
                    query.value.chartDataType === ChartDataType.TotalAssetsByCurrency.type ||
                    query.value.chartDataType === ChartDataType.TotalLiabilitiesByCurrency.type)) {
                return DISPLAY_HIDDEN_AMOUNT;
            }
        }

        if (textLimit) {
            return limitText(finalAmount, textLimit);
        }

        return finalAmount;
    }

    return {
        // states
        loading,
        analysisType,
        trendDateAggregationType,
        assetTrendsDateAggregationType,
        // computed states
        showAccountBalance,
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
        showTotalAmountInTrendsChart,
        showStackedInTrendsChart,
        translateNameInTrendsChart,
        categoricalOverviewAnalysisData,
        categoricalAnalysisData,
        trendsAnalysisData,
        assetTrendsData,
        radarChartCategoryNames,
        radarChartData,
        isAccountBalanceChartDataType,
        chartDataAmountKind,
        incomeExpenseTotals,
        // functions
        canShowCustomDateRange,
        getTransactionCategoricalAnalysisDataItemDisplayColor,
        getDisplayAmount
    };
}

import { useI18n } from '@/locales/helpers.ts';

import type { BigDecimal } from '@/core/numeral.ts';

/**
 * One-sentence text summaries generated from chart data (design guideline D7). Every chart panel pairs the chart
 * with a ranked list or table and one of these sentences, so the information is available without the chart.
 * The helpers only describe values that the caller already shows, they never compute new figures.
 */
export interface ChartSummaryRankedItem {
    readonly name: string;
    readonly value: BigDecimal;
    readonly percent?: number;
}

export interface ChartSummarySeriesPoint {
    readonly label: string;
    readonly value: BigDecimal;
}

export interface ChartRankedSummaryOptions {
    readonly totalName: string;
    readonly formatValue: (value: BigDecimal) => string;
    readonly showPercent?: boolean;
}

export function useChartTextSummary() {
    const { tt, formatPercentToLocalizedNumerals } = useI18n();

    // translates a sentence, and fills in the parameters itself when the sentence has not been translated yet
    function translateSentence(key: string, params: Record<string, string>): string {
        const translated = tt(key, params);

        if (translated !== key) {
            return translated;
        }

        return key.replace(/\{(\w+)\}/g, (match: string, name: string) => params[name] ?? match);
    }

    /**
     * e.g. "Payroll accounts for 73.88% of Total Expense." or "Payroll is the largest item at £4,200.00."
     */
    function getRankedItemsSummary(items: ChartSummaryRankedItem[], options: ChartRankedSummaryOptions): string {
        if (!items || !items.length) {
            return '';
        }

        let topItem: ChartSummaryRankedItem | null = null;

        for (const item of items) {
            if (!topItem || item.value.greaterThan(topItem.value)) {
                topItem = item;
            }
        }

        if (!topItem) {
            return '';
        }

        if (items.length === 1) {
            return translateSentence('{name} is {amount}.', {
                name: topItem.name,
                amount: options.formatValue(topItem.value)
            });
        }

        if (options.showPercent && typeof topItem.percent === 'number' && topItem.percent >= 0 && topItem.value.isPositive()) {
            return translateSentence('{name} accounts for {percent} of {totalName}.', {
                name: topItem.name,
                percent: formatPercentToLocalizedNumerals(topItem.percent, 2, '<0.01'),
                totalName: options.totalName
            });
        }

        return translateSentence('{name} is the largest item at {amount}.', {
            name: topItem.name,
            amount: options.formatValue(topItem.value)
        });
    }

    /**
     * e.g. "Payroll peaked at £4,200.00 in Nov, 2026; the lowest value was £0.00 in Jan, 2026."
     */
    function getSeriesSummary(seriesName: string, points: ChartSummarySeriesPoint[], formatValue: (value: BigDecimal) => string): string {
        if (!points || !points.length) {
            return '';
        }

        let highest: ChartSummarySeriesPoint | null = null;
        let lowest: ChartSummarySeriesPoint | null = null;

        for (const point of points) {
            if (!highest || point.value.greaterThan(highest.value)) {
                highest = point;
            }

            if (!lowest || point.value.lessThan(lowest.value)) {
                lowest = point;
            }
        }

        if (!highest || !lowest) {
            return '';
        }

        if (points.length === 1) {
            return translateSentence('{name} was {amount} in {period}.', {
                name: seriesName,
                amount: formatValue(highest.value),
                period: highest.label
            });
        } else if (highest.value.equals(lowest.value)) {
            return translateSentence('{name} was {amount} in every period.', {
                name: seriesName,
                amount: formatValue(highest.value)
            });
        }

        return translateSentence('{name} peaked at {amount} in {period}; the lowest value was {lowestAmount} in {lowestPeriod}.', {
            name: seriesName,
            amount: formatValue(highest.value),
            period: highest.label,
            lowestAmount: formatValue(lowest.value),
            lowestPeriod: lowest.label
        });
    }

    /**
     * e.g. "Income of +£8,017.20 and expense of −£5,684.70 give a net movement of +£2,332.50."
     * The caller passes amounts already formatted with their signs (D1).
     */
    function getIncomeExpenseSummary(displayIncome: string, displayExpense: string, displayNet: string): string {
        return translateSentence('Income of {income} and expense of {expense} give a net movement of {net}.', {
            income: displayIncome,
            expense: displayExpense,
            net: displayNet
        });
    }

    /**
     * e.g. "The balance moved from £1,000.00 to £1,250.00 with inflows of +£500.00 and outflows of −£250.00."
     * The caller passes amounts already formatted, inflows and outflows with their signs (D1).
     */
    function getBalanceMovementSummary(displayOpeningBalance: string, displayClosingBalance: string, displayInflows: string, displayOutflows: string): string {
        return translateSentence('The balance moved from {openingBalance} to {closingBalance} with inflows of {inflows} and outflows of {outflows}.', {
            openingBalance: displayOpeningBalance,
            closingBalance: displayClosingBalance,
            inflows: displayInflows,
            outflows: displayOutflows
        });
    }

    return {
        getBalanceMovementSummary,
        getRankedItemsSummary,
        getSeriesSummary,
        getIncomeExpenseSummary
    };
}

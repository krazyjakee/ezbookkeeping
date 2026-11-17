import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useOverviewStore } from '@/stores/overview.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { DateRange } from '@/core/datetime.ts';

import {
    DISPLAY_HIDDEN_AMOUNT,
    DISPLAY_HIDDEN_PERCENT,
    INCOMPLETE_AMOUNT_SUFFIX
} from '@/consts/numeral.ts';

import { isInteger } from '@/lib/common.ts';
import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

import type {
    TransactionOverviewData,
    TransactionOverviewDisplayTime,
    TransactionOverviewDataItem
} from '@/models/transaction.ts';

export interface CommonPeriodStatisticsWidgetProps {
    dateType?: number;
}

/**
 * An amount prepared for the SignedAmount component (design guidelines D1): the formatted absolute amount
 * without sign, the transaction type that decides the sign and colour, and whether the sign must be hidden
 * (zero, hidden or unavailable amounts)
 */
export interface SignedDisplayAmount {
    text: string;
    type: 'income' | 'expense' | 'none';
    hideSign: boolean;
}

export function usePeriodStatisticsWidgetBase(props: CommonPeriodStatisticsWidgetProps) {
    const {
        tt,
        formatRange,
        formatDateTimeToLongDate,
        formatDateTimeToLongMonthDay,
        formatDateTimeToGregorianLikeLongYear,
        formatDateTimeToGregorianLikeLongMonth,
        formatAmountToLocalizedNumeralsWithCurrency,
        formatPercentToLocalizedNumerals
    } = useI18n();

    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const overviewStore = useOverviewStore();

    const showAmountInHomePage = computed<boolean>({
        get: () => settingsStore.appSettings.showAmountInHomePage,
        set: (value) => settingsStore.setShowAmountInHomePage(value)
    });

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);

    const transactionOverview = computed<TransactionOverviewData>(() => overviewStore.transactionOverview);

    const displayDateRange = computed<TransactionOverviewDisplayTime>(() => {
        return {
            today: {
                displayTime: formatDateTimeToLongDate(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.today.startTime)),
            },
            thisWeek: {
                startTime: formatDateTimeToLongMonthDay(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisWeek.startTime)),
                endTime: formatDateTimeToLongMonthDay(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisWeek.endTime))
            },
            thisMonth: {
                displayTime: formatDateTimeToGregorianLikeLongMonth(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisMonth.startTime)),
                startTime: formatDateTimeToLongMonthDay(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisMonth.startTime)),
                endTime: formatDateTimeToLongMonthDay(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisMonth.endTime))
            },
            thisYear: {
                displayTime: formatDateTimeToGregorianLikeLongYear(parseDateTimeFromUnixTime(overviewStore.transactionDataRange.thisYear.startTime))
            }
        };
    });

    const currentOverviewItem = computed<TransactionOverviewDataItem | undefined>(() => getOverviewItem(props.dateType));
    const currentPeriodTitle = computed<string>(() => tt(isInteger(props.dateType) ? (DateRange.valueOf(props.dateType)?.name ?? 'Unknown') : 'Unknown'));
    const currentDisplayDateTime = computed<string>(() => getDisplayDateTime(props.dateType));

    const currentIncomeAmount = computed<BigDecimal>(() => currentOverviewItem.value?.incomeAmount ?? BIG_DECIMAL_ZERO);
    const currentExpenseAmount = computed<BigDecimal>(() => currentOverviewItem.value?.expenseAmount ?? BIG_DECIMAL_ZERO);
    const currentNetIncomeAmount = computed<BigDecimal>(() => currentIncomeAmount.value.subtract(currentExpenseAmount.value));

    const currentDisplayIncomeAmount = computed<string>(() => currentOverviewItem.value?.valid ? getDisplayIncomeAmount(currentOverviewItem.value) : '');
    const currentDisplayExpenseAmount = computed<string>(() => currentOverviewItem.value?.valid ? getDisplayExpenseAmount(currentOverviewItem.value) : '');
    const currentDisplayNetIncomeAmount = computed<string>(() => {
        if (!currentOverviewItem.value?.valid) {
            return '';
        }

        const netIncomeAmount: BigDecimal = currentNetIncomeAmount.value;
        const incomplete: boolean = currentOverviewItem.value.incompleteIncomeAmount || currentOverviewItem.value.incompleteExpenseAmount;

        return getDisplayAmount(netIncomeAmount, incomplete);
    });

    const currentDisplaySavingsRate = computed<string>(() => {
        if (!showAmountInHomePage.value) {
            return DISPLAY_HIDDEN_PERCENT;
        }

        if (!currentOverviewItem.value?.valid) {
            return '';
        }

        const incomeAmount: BigDecimal = currentIncomeAmount.value;
        const expenseAmount: BigDecimal = currentExpenseAmount.value;

        if (incomeAmount.isZero()) {
            return expenseAmount.isZero() ? formatPercentToLocalizedNumerals(0, 2, '<0.01') : '-';
        }

        const rate = incomeAmount.subtract(expenseAmount).divide(incomeAmount).multiply(100).toDoubleNumber();
        return formatPercentToLocalizedNumerals(rate, 2, '<0.01');
    });

    const currentSignedDisplayIncomeAmount = computed<SignedDisplayAmount>(() => getSignedDisplayAmount(currentOverviewItem.value, 'income'));
    const currentSignedDisplayExpenseAmount = computed<SignedDisplayAmount>(() => getSignedDisplayAmount(currentOverviewItem.value, 'expense'));
    const currentSignedDisplayNetIncomeAmount = computed<SignedDisplayAmount>(() => getSignedDisplayAmount(currentOverviewItem.value, 'net'));

    const currentDetailsUrl = computed<string>(() => {
        return `/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: props.dateType })}`
    });

    function getOverviewItem(dateType: number | undefined): TransactionOverviewDataItem | undefined {
        if (dateType === DateRange.Today.type) {
            return transactionOverview.value.today;
        } else if (dateType === DateRange.ThisWeek.type) {
            return transactionOverview.value.thisWeek;
        } else if (dateType === DateRange.ThisMonth.type) {
            return transactionOverview.value.thisMonth;
        } else if (dateType === DateRange.ThisYear.type) {
            return transactionOverview.value.thisYear;
        } else {
            return undefined;
        }
    }

    function getDisplayDateTime(dateType: number | undefined): string {
        if (dateType === DateRange.Today.type) {
            return displayDateRange.value.today?.displayTime || '';
        } else if (dateType === DateRange.ThisWeek.type) {
            return formatRange(displayDateRange.value.thisWeek?.startTime ?? '', displayDateRange.value.thisWeek?.endTime ?? '');
        } else if (dateType === DateRange.ThisMonth.type) {
            return formatRange(displayDateRange.value.thisMonth?.startTime ?? '', displayDateRange.value.thisMonth?.endTime ?? '');
        } else if (dateType === DateRange.ThisYear.type) {
            return displayDateRange.value.thisYear?.displayTime || '';
        } else {
            return '';
        }
    }

    function getDisplayAmount(amount: BigDecimal, incomplete: boolean): string {
        if (!showAmountInHomePage.value) {
            return formatAmountToLocalizedNumeralsWithCurrency(DISPLAY_HIDDEN_AMOUNT, defaultCurrency.value);
        }

        return formatAmountToLocalizedNumeralsWithCurrency(amount, defaultCurrency.value) + (incomplete ? INCOMPLETE_AMOUNT_SUFFIX : '');
    }

    function getSignedDisplayAmount(item: TransactionOverviewDataItem | undefined, kind: 'income' | 'expense' | 'net'): SignedDisplayAmount {
        if (!item || !item.valid) {
            return {
                text: '',
                type: kind === 'net' ? 'none' : kind,
                hideSign: true
            };
        }

        let amount: BigDecimal;
        let incomplete: boolean;
        let type: 'income' | 'expense' | 'none';

        if (kind === 'income') {
            amount = item.incomeAmount;
            incomplete = item.incompleteIncomeAmount;
            type = 'income';
        } else if (kind === 'expense') {
            amount = item.expenseAmount;
            incomplete = item.incompleteExpenseAmount;
            type = 'expense';
        } else {
            amount = item.incomeAmount.subtract(item.expenseAmount);
            incomplete = item.incompleteIncomeAmount || item.incompleteExpenseAmount;
            type = amount.isZero() ? 'none' : (amount.isNegative() ? 'expense' : 'income');
        }

        return {
            text: getDisplayAmount(amount.abs(), incomplete),
            type: type,
            hideSign: !showAmountInHomePage.value || amount.isZero()
        };
    }

    function getDisplayIncomeAmount(category: TransactionOverviewDataItem): string {
        return getDisplayAmount(category.incomeAmount, category.incompleteIncomeAmount);
    }

    function getDisplayExpenseAmount(category: TransactionOverviewDataItem): string {
        return getDisplayAmount(category.expenseAmount, category.incompleteExpenseAmount);
    }

    return {
        // computed states
        showAmountInHomePage,
        defaultCurrency,
        transactionOverview,
        displayDateRange,
        currentOverviewItem,
        currentPeriodTitle,
        currentDisplayDateTime,
        currentIncomeAmount,
        currentExpenseAmount,
        currentNetIncomeAmount,
        currentDisplayIncomeAmount,
        currentDisplayExpenseAmount,
        currentDisplayNetIncomeAmount,
        currentDisplaySavingsRate,
        currentSignedDisplayIncomeAmount,
        currentSignedDisplayExpenseAmount,
        currentSignedDisplayNetIncomeAmount,
        currentDetailsUrl,
        // functions
        getOverviewItem,
        getDisplayDateTime,
        getDisplayAmount,
        getSignedDisplayAmount,
        getDisplayIncomeAmount,
        getDisplayExpenseAmount
    };
}

import { ref, computed, type Ref, type ComputedRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { TemplateType } from '@/core/template.ts';
import type { TransactionTemplate } from '@/models/transaction_template.ts';

import { getCurrentUnixTime, parseDateTimeFromUnixTimeWithTimezoneOffset } from '@/lib/datetime.ts';
import {
    type ScheduledTransactionStatusFilterType,
    type ScheduledTransactionStatusInfo,
    ScheduledTransactionStatus,
    getScheduledTransactionStatusInfo,
    isScheduledTransactionMatchStatusFilter
} from '@/lib/scheduled_transaction.ts';

export interface ScheduledTransactionStatusFilterOption {
    readonly type: ScheduledTransactionStatusFilterType;
    readonly displayName: string;
}

export function useScheduledTransactionListPageBase(templateType: Ref<number>, templates: ComputedRef<TransactionTemplate[]>) {
    const { tt, formatNumberToLocalizedNumerals, formatGregorianTextualYearMonthDayToLongDate } = useI18n();

    const statusFilter = ref<ScheduledTransactionStatusFilterType>('all');

    const isScheduledTransaction = computed<boolean>(() => templateType.value === TemplateType.Schedule.type);
    const canReorderByStatusFilter = computed<boolean>(() => !isScheduledTransaction.value || statusFilter.value === 'all');

    const allStatusFilterTypes = computed<ScheduledTransactionStatusFilterOption[]>(() => [
        { type: 'all', displayName: tt('All') },
        { type: ScheduledTransactionStatus.Active, displayName: tt('Active') },
        { type: ScheduledTransactionStatus.NotStarted, displayName: tt('Not Started') },
        { type: ScheduledTransactionStatus.Expired, displayName: tt('Expired') },
        { type: ScheduledTransactionStatus.Disabled, displayName: tt('Disabled') }
    ]);

    const statusFilterDisplayName = computed<string>(() => {
        for (const option of allStatusFilterTypes.value) {
            if (option.type === statusFilter.value) {
                return option.displayName;
            }
        }

        return tt('All');
    });

    const allStatusInfos = computed<Record<string, ScheduledTransactionStatusInfo>>(() => {
        const result: Record<string, ScheduledTransactionStatusInfo> = {};

        if (!isScheduledTransaction.value) {
            return result;
        }

        const currentUnixTime = getCurrentUnixTime();

        for (const template of templates.value) {
            const today = parseDateTimeFromUnixTimeWithTimezoneOffset(currentUnixTime, template.utcOffset).getGregorianCalendarYearDashMonthDashDay();
            result[template.id] = getScheduledTransactionStatusInfo(template, today);
        }

        return result;
    });

    function getStatusInfo(template: TransactionTemplate): ScheduledTransactionStatusInfo | undefined {
        return allStatusInfos.value[template.id];
    }

    function getStatusDisplayName(template: TransactionTemplate): string {
        const status = getStatusInfo(template)?.status;

        for (const option of allStatusFilterTypes.value) {
            if (option.type === status) {
                return option.displayName;
            }
        }

        return '';
    }

    function getStatusColor(template: TransactionTemplate): string {
        switch (getStatusInfo(template)?.status) {
            case ScheduledTransactionStatus.Active:
                return 'green';
            case ScheduledTransactionStatus.NotStarted:
                return 'blue';
            case ScheduledTransactionStatus.Expired:
                return 'orange';
            default:
                return 'gray';
        }
    }

    function getNextOccurrenceDisplayText(template: TransactionTemplate): string {
        const statusInfo = getStatusInfo(template);

        if (!statusInfo || (statusInfo.status !== ScheduledTransactionStatus.Active && statusInfo.status !== ScheduledTransactionStatus.NotStarted)) {
            return '';
        }

        if (!statusInfo.nextOccurrenceDate) {
            return tt('No upcoming occurrences');
        }

        return tt('format.misc.nextOccurrence', {
            date: formatGregorianTextualYearMonthDayToLongDate(statusInfo.nextOccurrenceDate)
        });
    }

    function getRemainingOccurrencesDisplayText(template: TransactionTemplate): string {
        const statusInfo = getStatusInfo(template);

        if (!statusInfo || (statusInfo.status !== ScheduledTransactionStatus.Active && statusInfo.status !== ScheduledTransactionStatus.NotStarted) || !statusInfo.nextOccurrenceDate) {
            return '';
        }

        if (statusInfo.remainingOccurrences === undefined) {
            return tt('Unlimited');
        }

        return tt('format.misc.occurrencesLeft', {
            count: formatNumberToLocalizedNumerals(statusInfo.remainingOccurrences)
        });
    }

    function isShownByStatusFilter(template: TransactionTemplate): boolean {
        if (!isScheduledTransaction.value) {
            return true;
        }

        const statusInfo = getStatusInfo(template);

        if (!statusInfo) {
            return true;
        }

        return isScheduledTransactionMatchStatusFilter(statusInfo.status, statusFilter.value);
    }

    return {
        // states
        statusFilter,
        // computed states
        isScheduledTransaction,
        canReorderByStatusFilter,
        allStatusFilterTypes,
        statusFilterDisplayName,
        // functions
        getStatusDisplayName,
        getStatusColor,
        getNextOccurrenceDisplayText,
        getRemainingOccurrencesDisplayText,
        isShownByStatusFilter
    };
}

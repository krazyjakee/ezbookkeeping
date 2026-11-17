import type { TextualYearMonthDay } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';

const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000;
const MAX_SEARCH_DAYS_FOR_NEXT_OCCURRENCE = 366 * 8 + 1; // February 29 may only occur once every 8 years

export enum ScheduledTransactionStatus {
    Active = 'active',
    NotStarted = 'notStarted',
    Expired = 'expired',
    Disabled = 'disabled'
}

export type ScheduledTransactionStatusFilterType = 'all' | ScheduledTransactionStatus;

export interface ScheduledTransactionDefinition {
    readonly scheduledFrequencyType?: number;
    readonly scheduledFrequency?: string;
    readonly scheduledStartDate?: string;
    readonly scheduledEndDate?: string;
}

export interface ScheduledTransactionStatusInfo {
    readonly status: ScheduledTransactionStatus;
    readonly nextOccurrenceDate?: TextualYearMonthDay;
    readonly remainingOccurrences?: number; // undefined means unlimited (no end date)
}

function parseDayNumber(date: string | undefined): number | undefined {
    if (!date) {
        return undefined;
    }

    const match = /^(\d{4})-(\d{1,2})-(\d{1,2})$/.exec(date);

    if (!match) {
        return undefined;
    }

    const year = parseInt(match[1] as string, 10);
    const month = parseInt(match[2] as string, 10);
    const day = parseInt(match[3] as string, 10);

    return Math.floor(Date.UTC(year, month - 1, day) / MILLISECONDS_PER_DAY);
}

function dayNumberToDate(dayNumber: number): Date {
    return new Date(dayNumber * MILLISECONDS_PER_DAY);
}

function formatDayNumber(dayNumber: number): TextualYearMonthDay {
    const date = dayNumberToDate(dayNumber);
    const month = (date.getUTCMonth() + 1).toString().padStart(2, '0');
    const day = date.getUTCDate().toString().padStart(2, '0');

    return `${date.getUTCFullYear()}-${month}-${day}` as TextualYearMonthDay;
}

function parseFrequencyValues(frequency: string | undefined): number[] {
    const values: number[] = [];

    if (!frequency) {
        return values;
    }

    for (const item of frequency.split(',')) {
        const value = parseInt(item, 10);

        if (Number.isFinite(value)) {
            values.push(value);
        }
    }

    return values;
}

function isValidSchedule(frequencyType: number | undefined, frequencyValues: number[], startDayNumber: number | undefined): boolean {
    switch (frequencyType) {
        case ScheduledTemplateFrequencyType.Daily.type:
            return true;
        case ScheduledTemplateFrequencyType.EveryNDays.type:
            return startDayNumber !== undefined && frequencyValues.length === 1 && (frequencyValues[0] as number) > 0;
        case ScheduledTemplateFrequencyType.Weekly.type:
        case ScheduledTemplateFrequencyType.Monthly.type:
        case ScheduledTemplateFrequencyType.Yearly.type:
            return frequencyValues.length > 0;
        default:
            return false;
    }
}

function isOccurrenceDay(dayNumber: number, frequencyType: number | undefined, frequencyValues: number[], startDayNumber: number | undefined): boolean {
    const date = dayNumberToDate(dayNumber);

    switch (frequencyType) {
        case ScheduledTemplateFrequencyType.Daily.type:
            return true;
        case ScheduledTemplateFrequencyType.EveryNDays.type: {
            if (startDayNumber === undefined || !frequencyValues.length) {
                return false;
            }

            const n = frequencyValues[0] as number;
            const daysDiff = dayNumber - startDayNumber;
            return n > 0 && daysDiff >= 0 && daysDiff % n === 0;
        }
        case ScheduledTemplateFrequencyType.Weekly.type:
            return frequencyValues.includes(date.getUTCDay());
        case ScheduledTemplateFrequencyType.Monthly.type: {
            const dayOfMonth = date.getUTCDate();
            const maxDayOfMonth = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 0)).getUTCDate();

            for (const value of frequencyValues) {
                // negative value means counting from the end of month (e.g. -1 is the last day of month)
                if (value === dayOfMonth || (value < 0 && maxDayOfMonth + value + 1 === dayOfMonth)) {
                    return true;
                }
            }

            return false;
        }
        case ScheduledTemplateFrequencyType.Yearly.type:
            return frequencyValues.includes((date.getUTCMonth() + 1) * 100 + date.getUTCDate());
        default:
            return false;
    }
}

export function getScheduledTransactionStatus(schedule: ScheduledTransactionDefinition, today: TextualYearMonthDay): ScheduledTransactionStatus {
    const frequencyValues = parseFrequencyValues(schedule.scheduledFrequency);
    const startDayNumber = parseDayNumber(schedule.scheduledStartDate);
    const endDayNumber = parseDayNumber(schedule.scheduledEndDate);
    const todayDayNumber = parseDayNumber(today);

    if (!isValidSchedule(schedule.scheduledFrequencyType, frequencyValues, startDayNumber)) {
        return ScheduledTransactionStatus.Disabled;
    }

    if (todayDayNumber !== undefined && endDayNumber !== undefined && endDayNumber < todayDayNumber) {
        return ScheduledTransactionStatus.Expired;
    }

    if (todayDayNumber !== undefined && startDayNumber !== undefined && startDayNumber > todayDayNumber) {
        return ScheduledTransactionStatus.NotStarted;
    }

    return ScheduledTransactionStatus.Active;
}

/**
 * Computes the status, next occurrence date (inclusive of today) and remaining occurrences
 * (inclusive of today, until the end date) of a scheduled transaction.
 * The next occurrence and remaining occurrences are only computed for active or not started schedules.
 */
export function getScheduledTransactionStatusInfo(schedule: ScheduledTransactionDefinition, today: TextualYearMonthDay): ScheduledTransactionStatusInfo {
    const status = getScheduledTransactionStatus(schedule, today);

    if (status !== ScheduledTransactionStatus.Active && status !== ScheduledTransactionStatus.NotStarted) {
        return {
            status: status
        };
    }

    const frequencyType = schedule.scheduledFrequencyType;
    const frequencyValues = parseFrequencyValues(schedule.scheduledFrequency);
    const startDayNumber = parseDayNumber(schedule.scheduledStartDate);
    const endDayNumber = parseDayNumber(schedule.scheduledEndDate);
    const todayDayNumber = parseDayNumber(today);

    if (todayDayNumber === undefined) {
        return {
            status: status
        };
    }

    const fromDayNumber = startDayNumber !== undefined && startDayNumber > todayDayNumber ? startDayNumber : todayDayNumber;
    const lastSearchDayNumber = endDayNumber !== undefined ? endDayNumber : fromDayNumber + MAX_SEARCH_DAYS_FOR_NEXT_OCCURRENCE;

    let nextOccurrenceDayNumber: number | undefined = undefined;
    let remainingOccurrences = 0;

    for (let dayNumber = fromDayNumber; dayNumber <= lastSearchDayNumber; dayNumber++) {
        if (!isOccurrenceDay(dayNumber, frequencyType, frequencyValues, startDayNumber)) {
            continue;
        }

        if (nextOccurrenceDayNumber === undefined) {
            nextOccurrenceDayNumber = dayNumber;

            if (endDayNumber === undefined) {
                break;
            }
        }

        remainingOccurrences++;
    }

    return {
        status: status,
        nextOccurrenceDate: nextOccurrenceDayNumber !== undefined ? formatDayNumber(nextOccurrenceDayNumber) : undefined,
        remainingOccurrences: endDayNumber !== undefined ? remainingOccurrences : undefined
    };
}

export function isScheduledTransactionMatchStatusFilter(status: ScheduledTransactionStatus, filterType: ScheduledTransactionStatusFilterType): boolean {
    return filterType === 'all' || status === filterType;
}

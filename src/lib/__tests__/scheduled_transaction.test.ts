import { describe, expect, test } from 'vitest';

import { ScheduledTemplateFrequencyType } from '@/core/template.ts';

import {
    ScheduledTransactionStatus,
    getScheduledTransactionStatus,
    getScheduledTransactionStatusInfo,
    isScheduledTransactionMatchStatusFilter
} from '@/lib/scheduled_transaction.ts';

describe('getScheduledTransactionStatus', () => {
    test('returns active when today is within the date range', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-01-01',
            scheduledEndDate: '2026-12-31'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Active);
    });

    test('returns active when start date and end date are today', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-06-15',
            scheduledEndDate: '2026-06-15'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Active);
    });

    test('returns active when there is no start date or end date', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Monthly.type,
            scheduledFrequency: '1'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Active);
    });

    test('returns not started when start date is after today', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-06-16'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.NotStarted);
    });

    test('returns expired when end date is before today', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-01-01',
            scheduledEndDate: '2026-06-14'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Expired);
    });

    test('returns disabled when frequency type is disabled or invalid', () => {
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Disabled.type,
            scheduledFrequency: ''
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Disabled);
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Weekly.type,
            scheduledFrequency: ''
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Disabled);
        expect(getScheduledTransactionStatus({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.EveryNDays.type,
            scheduledFrequency: '3'
        }, '2026-06-15')).toBe(ScheduledTransactionStatus.Disabled);
    });
});

describe('getScheduledTransactionStatusInfo', () => {
    test('daily with end date counts today and end date', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-06-01',
            scheduledEndDate: '2026-06-20'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-06-15',
            remainingOccurrences: 6
        });
    });

    test('no end date means unlimited remaining occurrences', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Monthly.type,
            scheduledFrequency: '1',
            scheduledStartDate: '2026-01-01'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-07-01',
            remainingOccurrences: undefined
        });
    });

    test('weekly with multiple weekdays', () => {
        // 2026-06-15 is Monday
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Weekly.type,
            scheduledFrequency: '3,5',
            scheduledEndDate: '2026-06-30'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-06-17',
            remainingOccurrences: 4 // 17, 19, 24 and 26 June (Wednesday and Friday)
        });
    });

    test('monthly with last day of month', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Monthly.type,
            scheduledFrequency: '-1',
            scheduledEndDate: '2026-12-31'
        }, '2026-01-31')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-01-31',
            remainingOccurrences: 12
        });
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Monthly.type,
            scheduledFrequency: '-1'
        }, '2028-02-01')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2028-02-29',
            remainingOccurrences: undefined
        });
    });

    test('monthly on day 31 skips months without 31 days', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Monthly.type,
            scheduledFrequency: '31',
            scheduledEndDate: '2026-12-31'
        }, '2026-04-01')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-05-31',
            remainingOccurrences: 5 // May, Jul, Aug, Oct, Dec
        });
    });

    test('yearly on February 29 finds the next leap year', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Yearly.type,
            scheduledFrequency: '229'
        }, '2026-03-01')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2028-02-29',
            remainingOccurrences: undefined
        });
    });

    test('every N days is aligned to start date', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.EveryNDays.type,
            scheduledFrequency: '10',
            scheduledStartDate: '2026-06-01',
            scheduledEndDate: '2026-07-01'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: '2026-06-21',
            remainingOccurrences: 2 // 21 Jun, 1 Jul
        });
    });

    test('not started schedule counts from start date', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledStartDate: '2026-07-01',
            scheduledEndDate: '2026-07-10'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.NotStarted,
            nextOccurrenceDate: '2026-07-01',
            remainingOccurrences: 10
        });
    });

    test('active schedule without any remaining occurrence', () => {
        // 2026-06-16 is Tuesday, 2026-06-17 is Wednesday, only Monday is scheduled
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Weekly.type,
            scheduledFrequency: '1',
            scheduledEndDate: '2026-06-17'
        }, '2026-06-16')).toEqual({
            status: ScheduledTransactionStatus.Active,
            nextOccurrenceDate: undefined,
            remainingOccurrences: 0
        });
    });

    test('expired and disabled schedules have no occurrence info', () => {
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Daily.type,
            scheduledFrequency: '0',
            scheduledEndDate: '2026-01-01'
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Expired
        });
        expect(getScheduledTransactionStatusInfo({
            scheduledFrequencyType: ScheduledTemplateFrequencyType.Disabled.type
        }, '2026-06-15')).toEqual({
            status: ScheduledTransactionStatus.Disabled
        });
    });
});

describe('isScheduledTransactionMatchStatusFilter', () => {
    test('all filter matches every status', () => {
        for (const status of Object.values(ScheduledTransactionStatus)) {
            expect(isScheduledTransactionMatchStatusFilter(status, 'all')).toBe(true);
        }
    });

    test('status filter only matches the same status', () => {
        expect(isScheduledTransactionMatchStatusFilter(ScheduledTransactionStatus.Active, ScheduledTransactionStatus.Active)).toBe(true);
        expect(isScheduledTransactionMatchStatusFilter(ScheduledTransactionStatus.Expired, ScheduledTransactionStatus.Active)).toBe(false);
    });
});

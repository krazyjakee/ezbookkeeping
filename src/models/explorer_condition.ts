import type { PartialRecord } from '@/core/base.ts';
import { NormalizedText } from '@/core/text.ts';
import { type DateTime } from '@/core/datetime.ts';
import { AccountType } from '@/core/account.ts';
import { TransactionType } from '@/core/transaction.ts';
import {
    TransactionExplorerConditionFieldType,
    TransactionExplorerConditionOperatorType
} from '@/core/explorer.ts';

import { Account } from '@/models/account.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';
import { type TransactionInsightDataItem } from '@/models/transaction.ts';

export interface InsightsExplorerMatchContext {
    getTransactionDateTime(): DateTime;
    getNormalizedDescription(): NormalizedText;
}

export interface TransactionExplorerCondition<T = TransactionExplorerConditionFieldType, V = string | string[] | number[]> {
    readonly field: T;
    readonly operator: TransactionExplorerConditionOperatorType;
    value: V;

    getValueForStore(): V;
    match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean;
    toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>, allTagsMap: Record<string, TransactionTag>): string;
}

export class TransactionExplorerUndefinedCondition implements TransactionExplorerCondition {
    public readonly field = TransactionExplorerConditionFieldType.Undefined;
    public readonly operator = TransactionExplorerConditionOperatorType.Equals;
    public value = '';

    public getValueForStore(): string {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        return !!transaction;
    }

    public toExpression(): string {
        return '';
    }
}

export type TransactionTimeDayOfWeekConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionTimeDayOfWeekCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTimeDayOfWeek, number[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTimeDayOfWeek;
    public readonly operator: TransactionTimeDayOfWeekConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: number[];

    constructor(operator: TransactionTimeDayOfWeekConditionOperator, value: number[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): number[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        const transactionDateTime = context.getTransactionDateTime();

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transactionDateTime.getWeekDay().type);
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transactionDateTime.getWeekDay().type);
        }

        return false;
    }

    public toExpression(): string {
        const textualDayOfWeeks = this.value.join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `DAY_OF_WEEK(transaction_time) IN (${textualDayOfWeeks})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `DAY_OF_WEEK(transaction_time) NOT IN (${textualDayOfWeeks})`;
        } else {
            return '';
        }
    }
}

export type TransactionTimeDayOfMonthConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionTimeDayOfMonthCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTimeDayOfMonth, number[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTimeDayOfMonth;
    public readonly operator: TransactionTimeDayOfMonthConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: number[];

    constructor(operator: TransactionTimeDayOfMonthConditionOperator, value: number[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): number[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        const transactionDateTime = context.getTransactionDateTime();
        const normalizedValue: number[] = this.value.map(day => day >= 0 ? day : transactionDateTime.getMaxDayOfGregorianCalendarMonth() + day + 1);

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return normalizedValue.includes(transactionDateTime.getGregorianCalendarDay());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !normalizedValue.includes(transactionDateTime.getGregorianCalendarDay());
        }

        return false;
    }

    public toExpression(): string {
        const textualDayOfMonths = this.value.join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `DAY(transaction_time) IN (${textualDayOfMonths})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `DAY(transaction_time) NOT IN (${textualDayOfMonths})`;
        } else {
            return '';
        }
    }
}

export type TransactionTimeMonthOfYearConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionTimeMonthOfYearCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTimeMonthOfYear, number[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTimeMonthOfYear;
    public readonly operator: TransactionTimeMonthOfYearConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: number[];

    constructor(operator: TransactionTimeMonthOfYearConditionOperator, value: number[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): number[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        const transactionDateTime = context.getTransactionDateTime();

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transactionDateTime.getGregorianCalendarMonth());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transactionDateTime.getGregorianCalendarMonth());
        }

        return false;
    }

    public toExpression(): string {
        const textualMonthOfYears = this.value.join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `MONTH(transaction_time) IN (${textualMonthOfYears})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `MONTH(transaction_time) NOT IN (${textualMonthOfYears})`;
        } else {
            return '';
        }
    }
}

export type TransactionTimeHourOfDayConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionTimeHourOfDayCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTimeHourOfDay, number[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTimeHourOfDay;
    public readonly operator: TransactionTimeHourOfDayConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: number[];

    constructor(operator: TransactionTimeHourOfDayConditionOperator, value: number[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): number[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        const transactionDateTime = context.getTransactionDateTime();

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transactionDateTime.getHour());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transactionDateTime.getHour());
        }

        return false;
    }

    public toExpression(): string {
        const textualHourOfDays = this.value.join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `HOUR(transaction_time) IN (${textualHourOfDays})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `HOUR(transaction_time) NOT IN (${textualHourOfDays})`;
        } else {
            return '';
        }
    }
}

export type TransactionTimezoneConditionOperator = TransactionExplorerConditionOperatorType.MinuteOffsetBetween |
    TransactionExplorerConditionOperatorType.MinuteOffsetNotBetween;

export class TransactionExplorerTransactionTimezoneCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTimezone, [number, number]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.MinuteOffsetBetween]: true,
        [TransactionExplorerConditionOperatorType.MinuteOffsetNotBetween]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTimezone;
    public readonly operator: TransactionTimezoneConditionOperator = TransactionExplorerConditionOperatorType.MinuteOffsetBetween;
    public value: [number, number];

    constructor(operator: TransactionTimezoneConditionOperator, value: [number, number]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): [number, number] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.MinuteOffsetBetween) {
            return transaction.utcOffset >= this.value[0] && transaction.utcOffset <= this.value[1];
        } else if (this.operator === TransactionExplorerConditionOperatorType.MinuteOffsetNotBetween) {
            return transaction.utcOffset < this.value[0] || transaction.utcOffset > this.value[1];
        }

        return false;
    }

    public toExpression(): string {
        if (this.operator === TransactionExplorerConditionOperatorType.MinuteOffsetBetween) {
            return `(UTC_OFFSET(timezone) >= ${this.value[0]} AND UTC_OFFSET(timezone) <= ${this.value[1]})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.MinuteOffsetNotBetween) {
            return `(UTC_OFFSET(timezone) < ${this.value[0]} OR UTC_OFFSET(timezone) > ${this.value[1]})`;
        }

        return '';
    }
}

export type TransactionTypeConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionTypeCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionType, number[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionType;
    public readonly operator: TransactionTypeConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: number[];

    constructor(operator: TransactionTypeConditionOperator, value: number[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): number[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transaction.type);
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transaction.type);
        }

        return false;
    }

    public toExpression(): string {
        const textualTypes = this.value.map(type => {
            if (type === TransactionType.Income) {
                return 'Income';
            } else if (type === TransactionType.Expense) {
                return 'Expense';
            } else if (type === TransactionType.Transfer) {
                return 'Transfer';
            } else {
                return type.toString();
            }
        }).join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `type IN (${textualTypes})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `type NOT IN (${textualTypes})`;
        } else {
            return '';
        }
    }
}

export type TransactionCategoryConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerTransactionCategoryCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionCategory, string[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionCategory;
    public readonly operator: TransactionCategoryConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: string[];

    constructor(operator: TransactionCategoryConditionOperator, value: string[]) {
        this.operator = operator
        this.value = value;
    }

    public getValueForStore(): string[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transaction.primaryCategory?.id ?? '') || this.value.includes(transaction.secondaryCategory?.id ?? transaction.categoryId);
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transaction.primaryCategory?.id ?? '') && !this.value.includes(transaction.secondaryCategory?.id ?? transaction.categoryId);
        }

        return false;
    }

    public toExpression(allCategoriesMap: Record<string, TransactionCategory>): string {
        const textualCategories = this.value.map(id => {
            const category = allCategoriesMap[id];

            if (category) {
                if (!category.parentId || category.parentId === '0') {
                    return '';
                }

                return `'${category.name}'`;
            } else {
                return `'${id}'`;
            }
        }).filter(item => !!item).join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `category IN (${textualCategories})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `category NOT IN (${textualCategories})`;
        } else {
            return '';
        }
    }
}

export type AccountConditionOperator = TransactionExplorerConditionOperatorType.In |
    TransactionExplorerConditionOperatorType.NotIn;

export class TransactionExplorerSourceAccountCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.SourceAccount, string[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.SourceAccount;
    public readonly operator: AccountConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: string[];

    constructor(operator: AccountConditionOperator, value: string[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): string[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transaction.sourceAccountId);
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transaction.sourceAccountId);
        }

        return false;
    }

    public toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>): string {
        const textualAccounts = this.value.map(id => {
            const account = allAccountsMap[id];

            if (account) {
                if (account.type === AccountType.MultiSubAccounts.type) {
                    return '';
                }

                return `'${account.name}'`;
            } else {
                return `'${id}'`;
            }
        }).filter(item => !!item).join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `source_account IN (${textualAccounts})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `source_account NOT IN (${textualAccounts})`;
        } else {
            return '';
        }
    }
}

export class TransactionExplorerDestinationAccountCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.DestinationAccount, string[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.In]: true,
        [TransactionExplorerConditionOperatorType.NotIn]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.DestinationAccount;
    public readonly operator: AccountConditionOperator = TransactionExplorerConditionOperatorType.In;
    public value: string[];

    constructor(operator: AccountConditionOperator, value: string[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): string[] {
        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return this.value.includes(transaction.destinationAccountId);
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return !this.value.includes(transaction.destinationAccountId);
        }

        return false;
    }

    public toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>): string {
        const textualAccounts = this.value.map(id => {
            const account = allAccountsMap[id];

            if (account) {
                if (account.type === AccountType.MultiSubAccounts.type) {
                    return '';
                }

                return `'${account.name}'`;
            } else {
                return `'${id}'`;
            }
        }).filter(item => !!item).join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.In) {
            return `destination_account IN (${textualAccounts})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotIn) {
            return `destination_account NOT IN (${textualAccounts})`;
        } else {
            return '';
        }
    }
}

type AmountConditionField = TransactionExplorerConditionFieldType.SourceAmount | TransactionExplorerConditionFieldType.DestinationAmount;
export type AmountConditionOperator = TransactionExplorerConditionOperatorType.Equals |
    TransactionExplorerConditionOperatorType.NotEquals |
    TransactionExplorerConditionOperatorType.GreaterThan |
    TransactionExplorerConditionOperatorType.LessThan |
    TransactionExplorerConditionOperatorType.Between |
    TransactionExplorerConditionOperatorType.NotBetween;

export abstract class AbstractTransactionExplorerAmountCondition<T = AmountConditionField> implements TransactionExplorerCondition<T, [number, number]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.Equals]: true,
        [TransactionExplorerConditionOperatorType.NotEquals]: true,
        [TransactionExplorerConditionOperatorType.GreaterThan]: true,
        [TransactionExplorerConditionOperatorType.LessThan]: true,
        [TransactionExplorerConditionOperatorType.Between]: true,
        [TransactionExplorerConditionOperatorType.NotBetween]: true
    };
    public abstract readonly field: T;
    public readonly operator: AmountConditionOperator = TransactionExplorerConditionOperatorType.Between;
    public value: [number, number];

    protected constructor(operator: AmountConditionOperator, value: [number, number]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): [number, number] {
        if (this.operator === TransactionExplorerConditionOperatorType.Between || this.operator === TransactionExplorerConditionOperatorType.NotBetween) {
            return [this.value[0], this.value[1]];
        } else {
            return [this.value[0], this.value[0]];
        }
    }

    public abstract match(transaction: TransactionInsightDataItem): boolean;
    public abstract toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>, allTagsMap: Record<string, TransactionTag>): string;

    protected matchAmount(amount: number): boolean {
        switch (this.operator) {
            case TransactionExplorerConditionOperatorType.GreaterThan:
                return amount > this.value[0];
            case TransactionExplorerConditionOperatorType.LessThan:
                return amount < this.value[0];
            case TransactionExplorerConditionOperatorType.Equals:
                return amount === this.value[0];
            case TransactionExplorerConditionOperatorType.NotEquals:
                return amount !== this.value[0];
            case TransactionExplorerConditionOperatorType.Between:
                return amount >= this.value[0] && amount <= this.value[1];
            case TransactionExplorerConditionOperatorType.NotBetween:
                return amount < this.value[0] || amount > this.value[1];
            default:
                return false;
        }
    }

    protected getExpression(amountFieldName: string): string {
        let expressionAmount1 = this.value[0].toString(10);
        let expressionAmount2 = this.value[1].toString(10);

        if (expressionAmount1.length > 2) {
            expressionAmount1 = `${expressionAmount1.substring(0, expressionAmount1.length - 2)}.${expressionAmount1.substring(expressionAmount1.length - 2)}`;
        }

        if (expressionAmount2.length > 2) {
            expressionAmount2 = `${expressionAmount2.substring(0, expressionAmount2.length - 2)}.${expressionAmount2.substring(expressionAmount2.length - 2)}`;
        }

        switch (this.operator) {
            case TransactionExplorerConditionOperatorType.GreaterThan:
                return `${amountFieldName} > ${expressionAmount1}`;
            case TransactionExplorerConditionOperatorType.LessThan:
                return `${amountFieldName} < ${expressionAmount1}`;
            case TransactionExplorerConditionOperatorType.Equals:
                return `${amountFieldName} = ${expressionAmount1}`;
            case TransactionExplorerConditionOperatorType.NotEquals:
                return `${amountFieldName} <> ${expressionAmount1}`;
            case TransactionExplorerConditionOperatorType.Between:
                return `(${amountFieldName} >= ${expressionAmount1} AND ${amountFieldName} <= ${expressionAmount2})`;
            case TransactionExplorerConditionOperatorType.NotBetween:
                return `(${amountFieldName} < ${expressionAmount1} OR ${amountFieldName} > ${expressionAmount2})`;
            default:
                return '';
        }
    }
}

export class TransactionExplorerSourceAmountCondition extends AbstractTransactionExplorerAmountCondition<TransactionExplorerConditionFieldType.SourceAmount> {
    public readonly field = TransactionExplorerConditionFieldType.SourceAmount;

    constructor(operator: AmountConditionOperator, value: [number, number]) {
        super(operator, value);
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        return super.matchAmount(transaction.sourceAmount);
    }

    public toExpression(): string {
        return this.getExpression('source_amount');
    }
}

export class TransactionExplorerDestinationAmountCondition extends AbstractTransactionExplorerAmountCondition<TransactionExplorerConditionFieldType.DestinationAmount> {
    public readonly field = TransactionExplorerConditionFieldType.DestinationAmount;

    constructor(operator: AmountConditionOperator, value: [number, number]) {
        super(operator, value);
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        return super.matchAmount(transaction.destinationAmount);
    }

    public toExpression(): string {
        return this.getExpression('destination_amount');
    }
}

export type GeoLocationConditionOperator = TransactionExplorerConditionOperatorType.IsEmpty |
    TransactionExplorerConditionOperatorType.IsNotEmpty |
    TransactionExplorerConditionOperatorType.LatitudeBetween |
    TransactionExplorerConditionOperatorType.LatitudeNotBetween |
    TransactionExplorerConditionOperatorType.LongitudeBetween |
    TransactionExplorerConditionOperatorType.LongitudeNotBetween;

export class TransactionExplorerGeoLocationCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.GeoLocation, [number, number, number, number]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.IsEmpty]: true,
        [TransactionExplorerConditionOperatorType.IsNotEmpty]: true,
        [TransactionExplorerConditionOperatorType.LatitudeBetween]: true,
        [TransactionExplorerConditionOperatorType.LatitudeNotBetween]: true,
        [TransactionExplorerConditionOperatorType.LongitudeBetween]: true,
        [TransactionExplorerConditionOperatorType.LongitudeNotBetween]: true
    };
    public static readonly MIN_LATITUDE: number = -90.0;
    public static readonly MAX_LATITUDE: number = 90.0;
    public static readonly MIN_LONGITUDE: number = -180.0;
    public static readonly MAX_LONGITUDE: number = 180.0;

    public readonly field = TransactionExplorerConditionFieldType.GeoLocation;
    public readonly operator: GeoLocationConditionOperator = TransactionExplorerConditionOperatorType.IsNotEmpty;
    public value: [number, number, number, number];

    constructor(operator: GeoLocationConditionOperator, value: [number, number, number, number]) {
        this.operator = operator;
        this.value = [
            value[0] ?? TransactionExplorerGeoLocationCondition.MIN_LATITUDE,
            value[1] ?? TransactionExplorerGeoLocationCondition.MAX_LATITUDE,
            value[2] ?? TransactionExplorerGeoLocationCondition.MIN_LONGITUDE,
            value[3] ?? TransactionExplorerGeoLocationCondition.MAX_LONGITUDE
        ];
    }

    public getValueForStore(): [number, number, number, number] {
        return [
            this.value[0] ?? TransactionExplorerGeoLocationCondition.MIN_LATITUDE,
            this.value[1] ?? TransactionExplorerGeoLocationCondition.MAX_LATITUDE,
            this.value[2] ?? TransactionExplorerGeoLocationCondition.MIN_LONGITUDE,
            this.value[3] ?? TransactionExplorerGeoLocationCondition.MAX_LONGITUDE
        ];
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return !transaction.geoLocation;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return !!transaction.geoLocation;
        } else if (this.operator === TransactionExplorerConditionOperatorType.LatitudeBetween) {
            if (!transaction.geoLocation) {
                return false;
            }

            const latitude = transaction.geoLocation.latitude;

            if (typeof(this.value[0]) === 'number' && latitude < this.value[0]) {
                return false;
            }

            if (typeof(this.value[1]) === 'number' && latitude > this.value[1]) {
                return false;
            }

            return true;
        } else if (this.operator === TransactionExplorerConditionOperatorType.LatitudeNotBetween) {
            if (!transaction.geoLocation) {
                return false;
            }

            const latitude = transaction.geoLocation.latitude;

            if (typeof(this.value[0]) === 'number' && typeof(this.value[1]) === 'number' && latitude >= this.value[0] && latitude <= this.value[1]) {
                return false;
            }

            return true;
        } else if (this.operator === TransactionExplorerConditionOperatorType.LongitudeBetween) {
            if (!transaction.geoLocation) {
                return false;
            }

            const longitude = transaction.geoLocation.longitude;

            if (typeof(this.value[2]) === 'number' && longitude < this.value[2]) {
                return false;
            }

            if (typeof(this.value[3]) === 'number' && longitude > this.value[3]) {
                return false;
            }

            return true;
        } else if (this.operator === TransactionExplorerConditionOperatorType.LongitudeNotBetween) {
            if (!transaction.geoLocation) {
                return false;
            }

            const longitude = transaction.geoLocation.longitude;

            if (typeof(this.value[2]) === 'number' && typeof(this.value[3]) === 'number' && longitude >= this.value[2] && longitude <= this.value[3]) {
                return false;
            }

            return true;
        }

        return false;
    }

    public toExpression(): string {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return `geo_location IS EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return `geo_location IS NOT EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.LatitudeBetween || this.operator === TransactionExplorerConditionOperatorType.LatitudeNotBetween) {
            const minLatitude: number = this.value[0] ?? TransactionExplorerGeoLocationCondition.MIN_LATITUDE;
            const maxLatitude: number = this.value[1] ?? TransactionExplorerGeoLocationCondition.MAX_LATITUDE;

            if (this.operator === TransactionExplorerConditionOperatorType.LatitudeBetween) {
                return `(geo_location.latitude >= ${minLatitude} AND geo_location.latitude <= ${maxLatitude})`;
            } else if (this.operator === TransactionExplorerConditionOperatorType.LatitudeNotBetween) {
                return `(geo_location.latitude < ${minLatitude} OR geo_location.latitude > ${maxLatitude})`;
            }
        } else if (this.operator === TransactionExplorerConditionOperatorType.LongitudeBetween || this.operator === TransactionExplorerConditionOperatorType.LongitudeNotBetween) {
            const minLongitude: number = this.value[2] ?? TransactionExplorerGeoLocationCondition.MIN_LONGITUDE;
            const maxLongitude: number = this.value[3] ?? TransactionExplorerGeoLocationCondition.MAX_LONGITUDE;

            if (this.operator === TransactionExplorerConditionOperatorType.LongitudeBetween) {
                return `(geo_location.longitude >= ${minLongitude} AND geo_location.longitude <= ${maxLongitude})`;
            } else if (this.operator === TransactionExplorerConditionOperatorType.LongitudeNotBetween) {
                return `(geo_location.longitude < ${minLongitude} OR geo_location.longitude > ${maxLongitude})`;
            }
        }

        return '';
    }
}

export type TransactionTagConditionOperator = TransactionExplorerConditionOperatorType.IsEmpty |
    TransactionExplorerConditionOperatorType.IsNotEmpty |
    TransactionExplorerConditionOperatorType.Equals |
    TransactionExplorerConditionOperatorType.NotEquals |
    TransactionExplorerConditionOperatorType.HasAny |
    TransactionExplorerConditionOperatorType.HasAll |
    TransactionExplorerConditionOperatorType.NotHasAny |
    TransactionExplorerConditionOperatorType.NotHasAll;

export class TransactionExplorerTransactionTagCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.TransactionTag, string[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.IsEmpty]: true,
        [TransactionExplorerConditionOperatorType.IsNotEmpty]: true,
        [TransactionExplorerConditionOperatorType.Equals]: true,
        [TransactionExplorerConditionOperatorType.NotEquals]: true,
        [TransactionExplorerConditionOperatorType.HasAny]: true,
        [TransactionExplorerConditionOperatorType.HasAll]: true,
        [TransactionExplorerConditionOperatorType.NotHasAny]: true,
        [TransactionExplorerConditionOperatorType.NotHasAll]: true
    };
    public readonly field = TransactionExplorerConditionFieldType.TransactionTag;
    public readonly operator: TransactionTagConditionOperator = TransactionExplorerConditionOperatorType.HasAny;
    public value: string[];

    constructor(operator: TransactionTagConditionOperator, value: string[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): string[] {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty || this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return [];
        }

        return this.value;
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        const transactionTags: Record<string, boolean> = {};

        for (const tagId of transaction.tagIds) {
            transactionTags[tagId] = true;
        }

        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty || this.value.length < 1) {
            return transaction.tagIds.length < 1;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return transaction.tagIds.length > 0;
        } else if (this.operator === TransactionExplorerConditionOperatorType.Equals || this.operator === TransactionExplorerConditionOperatorType.NotEquals) {
            let hasAll = true;

            for (const tagId of this.value) {
                if (!transactionTags[tagId]) {
                    hasAll = false;
                    break;
                }
            }

            const hasSameCount = transaction.tagIds.length === this.value.length;

            if (this.operator === TransactionExplorerConditionOperatorType.Equals && hasAll && hasSameCount) {
                return true;
            } else if (this.operator === TransactionExplorerConditionOperatorType.NotEquals && (!hasAll || !hasSameCount)) {
                return true;
            }
        } else if (this.operator === TransactionExplorerConditionOperatorType.HasAny || this.operator === TransactionExplorerConditionOperatorType.NotHasAny) {
            let hasAny = false;

            for (const tagId of this.value) {
                if (transactionTags[tagId]) {
                    hasAny = true;
                    break;
                }
            }

            if (this.operator === TransactionExplorerConditionOperatorType.HasAny && hasAny) {
                return true;
            } else if (this.operator === TransactionExplorerConditionOperatorType.NotHasAny && !hasAny) {
                return true;
            }
        } else if (this.operator === TransactionExplorerConditionOperatorType.HasAll || this.operator === TransactionExplorerConditionOperatorType.NotHasAll) {
            let hasAll = true;

            for (const tagId of this.value) {
                if (!transactionTags[tagId]) {
                    hasAll = false;
                    break;
                }
            }

            if (this.operator === TransactionExplorerConditionOperatorType.HasAll && hasAll) {
                return true;
            } else if (this.operator === TransactionExplorerConditionOperatorType.NotHasAll && !hasAll) {
                return true;
            }
        }

        return false;
    }

    public toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>, allTagsMap: Record<string, TransactionTag>): string {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return `tags IS EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return `tags IS NOT EMPTY`;
        }

        const textualTags = this.value.map(id => {
            const tag = allTagsMap[id];

            if (tag) {
                return `'${tag.name}'`;
            } else {
                return `'${id}'`;
            }
        }).join(', ');

        if (this.operator === TransactionExplorerConditionOperatorType.Equals) {
            return `tags FULL MATCHES (${textualTags})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotEquals) {
            return `tags NOT FULL MATCHES (${textualTags})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.HasAny) {
            return `tags HAS ANY (${textualTags})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.HasAll) {
            return `tags HAS ALL (${textualTags})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotHasAny) {
            return `tags NOT HAS ANY (${textualTags})`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotHasAll) {
            return `tags NOT HAS ALL (${textualTags})`;
        } else {
            return '';
        }
    }
}

export type PicturesConditionOperator = TransactionExplorerConditionOperatorType.IsEmpty |
    TransactionExplorerConditionOperatorType.IsNotEmpty;

export class TransactionExplorerPicturesCondition implements TransactionExplorerCondition<TransactionExplorerConditionFieldType.Pictures, string[]> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.IsEmpty]: true,
        [TransactionExplorerConditionOperatorType.IsNotEmpty]: true
    };

    public readonly field = TransactionExplorerConditionFieldType.Pictures;
    public readonly operator: PicturesConditionOperator = TransactionExplorerConditionOperatorType.IsNotEmpty;
    public value: string[];

    constructor(operator: PicturesConditionOperator, value: string[]) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): string[] {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty || this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return [];
        }

        return [];
    }

    public match(transaction: TransactionInsightDataItem): boolean {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return !transaction.pictures || transaction.pictures.length < 1;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return !!transaction.pictures && transaction.pictures.length > 0;
        }

        return false;
    }

    public toExpression(): string {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return `pictures IS EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return `pictures IS NOT EMPTY`;
        }

        return '';
    }
}

type DescriptionConditionField = TransactionExplorerConditionFieldType.Description | TransactionExplorerConditionFieldType.DescriptionCaseInsensitive | TransactionExplorerConditionFieldType.DescriptionNormalized;
export type DescriptionConditionOperator = TransactionExplorerConditionOperatorType.IsEmpty |
    TransactionExplorerConditionOperatorType.IsNotEmpty |
    TransactionExplorerConditionOperatorType.Equals |
    TransactionExplorerConditionOperatorType.NotEquals |
    TransactionExplorerConditionOperatorType.Contains |
    TransactionExplorerConditionOperatorType.NotContains |
    TransactionExplorerConditionOperatorType.StartsWith |
    TransactionExplorerConditionOperatorType.NotStartsWith |
    TransactionExplorerConditionOperatorType.EndsWith |
    TransactionExplorerConditionOperatorType.NotEndsWith |
    TransactionExplorerConditionOperatorType.RegexMatch |
    TransactionExplorerConditionOperatorType.NotRegexMatch;

enum DescriptionMatchMode {
    Exact,
    CaseInsensitive,
    Normalized
}

export abstract class AbstractTransactionExplorerDescriptionCondition<T = DescriptionConditionField> implements TransactionExplorerCondition<T, string> {
    public static readonly supportedOperators: PartialRecord<TransactionExplorerConditionOperatorType, true> = {
        [TransactionExplorerConditionOperatorType.IsEmpty]: true,
        [TransactionExplorerConditionOperatorType.IsNotEmpty]: true,
        [TransactionExplorerConditionOperatorType.Equals]: true,
        [TransactionExplorerConditionOperatorType.NotEquals]: true,
        [TransactionExplorerConditionOperatorType.Contains]: true,
        [TransactionExplorerConditionOperatorType.NotContains]: true,
        [TransactionExplorerConditionOperatorType.StartsWith]: true,
        [TransactionExplorerConditionOperatorType.NotStartsWith]: true,
        [TransactionExplorerConditionOperatorType.EndsWith]: true,
        [TransactionExplorerConditionOperatorType.NotEndsWith]: true,
        [TransactionExplorerConditionOperatorType.RegexMatch]: true,
        [TransactionExplorerConditionOperatorType.NotRegexMatch]: true
    };

    public abstract readonly field: T;
    public readonly operator: DescriptionConditionOperator = TransactionExplorerConditionOperatorType.Contains;
    public value: string;
    protected abstract readonly matchMode: DescriptionMatchMode;
    private cachedNormalizedValue: NormalizedText | undefined = undefined;
    private cachedRegex: RegExp | undefined = undefined;

    constructor(operator: DescriptionConditionOperator, value: string) {
        this.operator = operator;
        this.value = value;
    }

    public getValueForStore(): string {
        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty || this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return '';
        }

        return this.value;
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        let description: string;

        if (!transaction.comment) {
            description = '';
        } else if (this.matchMode === DescriptionMatchMode.Normalized) {
            description = context.getNormalizedDescription().normalizedText;
        } else if (this.matchMode === DescriptionMatchMode.CaseInsensitive) {
            description = context.getNormalizedDescription().lowerCaseText;
        } else {
            description = transaction.comment;
        }

        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return description.length === 0;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return description.length > 0;
        } else if (this.operator === TransactionExplorerConditionOperatorType.Equals) {
            return description === this.getCachedNormalizedValue();
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotEquals) {
            return description !== this.getCachedNormalizedValue();
        } else if (this.operator === TransactionExplorerConditionOperatorType.Contains) {
            return description.includes(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotContains) {
            return !description.includes(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.StartsWith) {
            return description.startsWith(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotStartsWith) {
            return !description.startsWith(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.EndsWith) {
            return description.endsWith(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotEndsWith) {
            return !description.endsWith(this.getCachedNormalizedValue());
        } else if (this.operator === TransactionExplorerConditionOperatorType.RegexMatch) {
            return this.getCachedRegex()?.test(description) ?? false;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotRegexMatch) {
            return !(this.getCachedRegex()?.test(description) ?? false);
        }

        return false;
    }

    public toExpression(): string {
        const expression: string = this.getExpression();

        if (expression && this.matchMode === DescriptionMatchMode.Normalized) {
            return `${expression} NORMALIZED`;
        } else if (expression && this.matchMode === DescriptionMatchMode.CaseInsensitive) {
            return `${expression} CASE INSENSITIVE`;
        } else {
            return expression;
        }
    }

    private getExpression(): string {
        const escapedValue = this.value.replace(/'/g, "''");

        if (this.operator === TransactionExplorerConditionOperatorType.IsEmpty) {
            return `description IS EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.IsNotEmpty) {
            return `description IS NOT EMPTY`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.Equals) {
            return `description = '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotEquals) {
            return `description <> '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.Contains) {
            return `description CONTAINS '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotContains) {
            return `description NOT CONTAINS '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.StartsWith) {
            return `description STARTS WITH '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotStartsWith) {
            return `description NOT STARTS WITH '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.EndsWith) {
            return `description ENDS WITH '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotEndsWith) {
            return `description NOT ENDS WITH '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.RegexMatch) {
            return `description REGEXP LIKE '${escapedValue}'`;
        } else if (this.operator === TransactionExplorerConditionOperatorType.NotRegexMatch) {
            return `description NOT REGEXP LIKE '${escapedValue}'`;
        }

        return '';
    }

    private getCachedNormalizedValue(): string {
        let normalizedValue = this.cachedNormalizedValue;

        if (!normalizedValue || normalizedValue.originalText !== this.value) {
            normalizedValue = NormalizedText.of(this.value);
            this.cachedNormalizedValue = normalizedValue;
            this.cachedRegex = undefined;
        }

        if (this.matchMode === DescriptionMatchMode.Normalized) {
            return normalizedValue.normalizedText;
        } else if (this.matchMode === DescriptionMatchMode.CaseInsensitive) {
            return normalizedValue.lowerCaseText;
        } else {
            return normalizedValue.originalText;
        }
    }

    private getCachedRegex(): RegExp | undefined {
        if (this.operator !== TransactionExplorerConditionOperatorType.RegexMatch && this.operator !== TransactionExplorerConditionOperatorType.NotRegexMatch) {
            return undefined;
        }

        if (!this.cachedNormalizedValue || this.cachedNormalizedValue.originalText !== this.value || !this.cachedRegex) {
            try {
                let regex: RegExp;

                if (this.matchMode === DescriptionMatchMode.CaseInsensitive || this.matchMode === DescriptionMatchMode.Normalized) {
                    regex = new RegExp(this.getCachedNormalizedValue(), 'i');
                } else {
                    regex = new RegExp(this.getCachedNormalizedValue());
                }

                this.cachedRegex = regex;
            } catch {
                this.cachedRegex = undefined;
            }
        }

        return this.cachedRegex;
    }
}

export class TransactionExplorerDescriptionCondition extends AbstractTransactionExplorerDescriptionCondition<TransactionExplorerConditionFieldType.Description> {
    public readonly field = TransactionExplorerConditionFieldType.Description;
    protected readonly matchMode = DescriptionMatchMode.Exact;

    constructor(operator: DescriptionConditionOperator, value: string) {
        super(operator, value);
    }
}

export class TransactionExplorerDescriptionCaseInsensitiveCondition extends AbstractTransactionExplorerDescriptionCondition<TransactionExplorerConditionFieldType.DescriptionCaseInsensitive> {
    public readonly field = TransactionExplorerConditionFieldType.DescriptionCaseInsensitive;
    protected readonly matchMode = DescriptionMatchMode.CaseInsensitive;

    constructor(operator: DescriptionConditionOperator, value: string) {
        super(operator, value);
    }
}

export class TransactionExplorerDescriptionNormalizedCondition extends AbstractTransactionExplorerDescriptionCondition<TransactionExplorerConditionFieldType.DescriptionNormalized> {
    public readonly field = TransactionExplorerConditionFieldType.DescriptionNormalized;
    protected readonly matchMode = DescriptionMatchMode.Normalized;

    constructor(operator: DescriptionConditionOperator, value: string) {
        super(operator, value);
    }
}

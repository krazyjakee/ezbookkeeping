import { type PartialRecord, itemAndIndex, keysIfValueEquals } from '@/core/base.ts';
import { TimezoneTypeForStatistics } from '@/core/timezone.ts';
import { TransactionType } from '@/core/transaction.ts';
import { ChartSortingType } from '@/core/statistics.ts';
import {
    type TransactionExplorerSubConditionStartRelation,
    TransactionExplorerConditionRelation,
    TransactionExplorerSubConditionStartRelationPlaceholder,
    TransactionExplorerConditionRelationPriority,
    TransactionExplorerConditionField,
    TransactionExplorerConditionOperatorType,
    TransactionExplorerConditionOperator,
    TransactionExplorerChartTypeValue,
    TransactionExplorerChartType,
    TransactionExplorerDataDimensionType,
    TransactionExplorerDataDimension,
    TransactionExplorerValueMetricType,
    TransactionExplorerValueMetric,
    TransactionExplorerCustomChartDisplayLayout
} from '@/core/explorer.ts';

import { Account } from '@/models/account.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';
import { type TransactionInsightDataItem } from '@/models/transaction.ts';
import {
    type InsightsExplorerMatchContext,
    type TransactionExplorerCondition,
    type TransactionTimeDayOfWeekConditionOperator,
    type TransactionTimeDayOfMonthConditionOperator,
    type TransactionTimeMonthOfYearConditionOperator,
    type TransactionTimeHourOfDayConditionOperator,
    type TransactionTimezoneConditionOperator,
    type TransactionTypeConditionOperator,
    type TransactionCategoryConditionOperator,
    type AccountConditionOperator,
    type AmountConditionOperator,
    type GeoLocationConditionOperator,
    type TransactionTagConditionOperator,
    type PicturesConditionOperator,
    type DescriptionConditionOperator,
    TransactionExplorerUndefinedCondition,
    TransactionExplorerTransactionTimeDayOfWeekCondition,
    TransactionExplorerTransactionTimeDayOfMonthCondition,
    TransactionExplorerTransactionTimeMonthOfYearCondition,
    TransactionExplorerTransactionTimeHourOfDayCondition,
    TransactionExplorerTransactionTimezoneCondition,
    TransactionExplorerTransactionTypeCondition,
    TransactionExplorerTransactionCategoryCondition,
    TransactionExplorerSourceAccountCondition,
    TransactionExplorerDestinationAccountCondition,
    TransactionExplorerSourceAmountCondition,
    TransactionExplorerDestinationAmountCondition,
    TransactionExplorerGeoLocationCondition,
    TransactionExplorerTransactionTagCondition,
    TransactionExplorerPicturesCondition,
    TransactionExplorerDescriptionCondition,
    TransactionExplorerDescriptionCaseInsensitiveCondition,
    TransactionExplorerDescriptionNormalizedCondition
} from '@/models/explorer_condition.ts';

export class InsightsExplorerBasicInfo implements InsightsExplorerInfoResponse {
    public id: string;
    public name: string;
    public displayOrder: number;
    public hidden: boolean;
    public data: Record<string, string | number | object[]> = {};

    private constructor(id: string, name: string, displayOrder: number, hidden: boolean) {
        this.id = id;
        this.name = name;
        this.displayOrder = displayOrder;
        this.hidden = hidden;
    }

    public static of(explorerResponse: InsightsExplorerInfoResponse): InsightsExplorerBasicInfo {
        return new InsightsExplorerBasicInfo(
            explorerResponse.id,
            explorerResponse.name,
            explorerResponse.displayOrder,
            explorerResponse.hidden
        );
    }

    public static ofMulti(explorerResponses: InsightsExplorerInfoResponse[]): InsightsExplorerBasicInfo[] {
        const explorers: InsightsExplorerBasicInfo[] = [];

        for (const explorerResponse of explorerResponses) {
            explorers.push(InsightsExplorerBasicInfo.of(explorerResponse));
        }

        return explorers;
    }
}

export class InsightsExplorer implements InsightsExplorerInfoResponse {
    public id: string;
    public name: string;
    public displayOrder: number;
    public hidden: boolean;
    public queries: TransactionExplorerQuery[];
    public timezoneUsedForDateRange: number;
    public datatableQuerySource: string;
    public countPerPage: number;
    public chartType: TransactionExplorerChartTypeValue;
    public categoryDimension: TransactionExplorerDataDimensionType;
    public seriesDimension: TransactionExplorerDataDimensionType;
    public amountRangeCount: number;
    public valueMetric: TransactionExplorerValueMetricType;
    public chartSortingType: number;
    public customChartDisplayLayout: number;
    public customChartScript: string;

    public static readonly Default: InsightsExplorer = new InsightsExplorer(
        '',
        '',
        0,
        false,
        [],
        TimezoneTypeForStatistics.Default.type,
        '',
        15,
        TransactionExplorerChartType.Default.value,
        TransactionExplorerDataDimension.CategoryDimensionDefault.value,
        TransactionExplorerDataDimension.SeriesDimensionDefault.value,
        5,
        TransactionExplorerValueMetric.Default.value,
        ChartSortingType.Default.type,
        TransactionExplorerCustomChartDisplayLayout.Default.type,
        ''
    );

    private constructor(id: string, name: string, displayOrder: number, hidden: boolean, queries: TransactionExplorerQuery[], timezoneUsedForDateRange: number, datatableQuerySource: string, countPerPage: number, chartType: TransactionExplorerChartTypeValue, categoryDimension: TransactionExplorerDataDimensionType, seriesDimension: TransactionExplorerDataDimensionType, amountRangeCount: number, valueMetric: TransactionExplorerValueMetricType, chartSortingType: number, customChartDisplayLayout: number, customChartScript: string) {
        this.id = id;
        this.name = name;
        this.displayOrder = displayOrder;
        this.hidden = hidden;
        this.queries = queries;
        this.timezoneUsedForDateRange = timezoneUsedForDateRange;
        this.datatableQuerySource = datatableQuerySource;
        this.countPerPage = countPerPage;
        this.chartType = chartType;
        this.categoryDimension = categoryDimension;
        this.seriesDimension = seriesDimension;
        this.amountRangeCount = amountRangeCount;
        this.valueMetric = valueMetric;
        this.chartSortingType = chartSortingType;
        this.customChartDisplayLayout = customChartDisplayLayout;
        this.customChartScript = customChartScript;
    }

    public get data(): Record<string, string | number | object[]> {
        return {
            queries: this.queries.map(q => q.toJsonObject()),
            timezoneUsedForDateRange: this.timezoneUsedForDateRange,
            datatableQuerySource: this.datatableQuerySource,
            countPerPage: this.countPerPage,
            chartType: this.chartType,
            categoryDimension: this.categoryDimension,
            seriesDimension: this.seriesDimension,
            amountRangeCount: this.amountRangeCount,
            valueMetric: this.valueMetric,
            chartSortingType: this.chartSortingType,
            customChartDisplayLayout: this.customChartDisplayLayout,
            customChartScript: this.customChartScript
        };
    }

    public getQueryiesPrettyJson(): string {
        return JSON.stringify(this.queries.map(q => q.toJsonObject()), null, 4);
    }

    public toCreateRequest(clientSessionId: string): InsightsExplorerCreateRequest {
        return {
            name: this.name,
            data: this.data,
            clientSessionId: clientSessionId
        };
    }

    public toModifyRequest(): InsightsExplorerModifyRequest {
        return {
            id: this.id,
            name: this.name,
            data: this.data,
            hidden: this.hidden
        };
    }

    public static of(explorerResponse: InsightsExplorerInfoResponse): InsightsExplorer {
        const data = explorerResponse.data;
        const queries: TransactionExplorerQuery[] = [];
        let timezoneUsedForDateRange = InsightsExplorer.Default.timezoneUsedForDateRange;
        let datatableQuerySource = InsightsExplorer.Default.datatableQuerySource;
        let countPerPage = InsightsExplorer.Default.countPerPage;
        let chartType = InsightsExplorer.Default.chartType;
        let categoryDimension = InsightsExplorer.Default.categoryDimension;
        let seriesDimension = InsightsExplorer.Default.seriesDimension;
        let amountRangeCount = InsightsExplorer.Default.amountRangeCount;
        let valueMetric = InsightsExplorer.Default.valueMetric;
        let chartSortingType = InsightsExplorer.Default.chartSortingType;
        let customChartDisplayLayout = InsightsExplorer.Default.customChartDisplayLayout;
        let customChartScript = InsightsExplorer.Default.customChartScript;
        let hasDatatableQuerySource = false;

        if (data) {
            if (typeof data['timezoneUsedForDateRange'] === 'number') {
                timezoneUsedForDateRange = data['timezoneUsedForDateRange'] as number;
            }

            if (typeof data['datatableQuerySource'] === 'string') {
                datatableQuerySource = data['datatableQuerySource'] as string;
            }

            if (typeof data['countPerPage'] === 'number') {
                countPerPage = data['countPerPage'] as number;
            }

            if (typeof data['chartType'] === 'string') {
                chartType = data['chartType'] as TransactionExplorerChartTypeValue;
            }

            if (typeof data['categoryDimension'] === 'string') {
                categoryDimension = data['categoryDimension'] as TransactionExplorerDataDimensionType;
            }

            if (typeof data['seriesDimension'] === 'string') {
                seriesDimension = data['seriesDimension'] as TransactionExplorerDataDimensionType;
            }

            if (typeof data['amountRangeCount'] === 'number') {
                amountRangeCount = data['amountRangeCount'] as number;
            }

            if (typeof data['valueMetric'] === 'string') {
                valueMetric = data['valueMetric'] as TransactionExplorerValueMetricType;
            }

            if (typeof data['chartSortingType'] === 'number') {
                chartSortingType = data['chartSortingType'] as number;
            }

            if (typeof data['customChartDisplayLayout'] === 'number') {
                customChartDisplayLayout = data['customChartDisplayLayout'] as number;
            }

            if (typeof data['customChartScript'] === 'string') {
                customChartScript = data['customChartScript'] as string;
            }

            if (Array.isArray(data['queries'])) {
                const queryItems = data['queries'] as object[];

                for (const queryItem of queryItems) {
                    const query = TransactionExplorerQuery.parse(queryItem);

                    if (query) {
                        queries.push(query);
                    }

                    if (query && query.id === datatableQuerySource) {
                        hasDatatableQuerySource = true;
                    }
                }
            }

            if (!hasDatatableQuerySource) {
                datatableQuerySource = '';
            }
        }

        return new InsightsExplorer(
            explorerResponse.id,
            explorerResponse.name,
            explorerResponse.displayOrder,
            explorerResponse.hidden,
            queries,
            timezoneUsedForDateRange,
            datatableQuerySource,
            countPerPage,
            chartType,
            categoryDimension,
            seriesDimension,
            amountRangeCount,
            valueMetric,
            chartSortingType,
            customChartDisplayLayout,
            customChartScript
        );
    }

    public static createNewExplorer(newQueryId: string): InsightsExplorer {
        return new InsightsExplorer(
            '',
            '',
            0,
            false,
            [TransactionExplorerQuery.create(newQueryId)],
            InsightsExplorer.Default.timezoneUsedForDateRange,
            InsightsExplorer.Default.datatableQuerySource,
            InsightsExplorer.Default.countPerPage,
            InsightsExplorer.Default.chartType,
            InsightsExplorer.Default.categoryDimension,
            InsightsExplorer.Default.seriesDimension,
            InsightsExplorer.Default.amountRangeCount,
            InsightsExplorer.Default.valueMetric,
            InsightsExplorer.Default.chartSortingType,
            InsightsExplorer.Default.customChartDisplayLayout,
            InsightsExplorer.Default.customChartScript
        );
    }
}

export interface InsightsExplorerCreateRequest {
    readonly name: string;
    readonly data: Record<string, string | number | object[]>;
    readonly clientSessionId?: string;
}

export interface InsightsExplorerModifyRequest {
    readonly id: string;
    readonly name: string;
    readonly data: Record<string, string | number | object[]>;
    readonly hidden: boolean;
    readonly clientSessionId?: string;
}

export interface InsightsExplorerHideRequest {
    readonly id: string;
    readonly hidden: boolean;
}

export interface InsightsExplorerMoveRequest {
    readonly newDisplayOrders: InsightsExplorerNewDisplayOrderRequest[];
}

export interface InsightsExplorerNewDisplayOrderRequest {
    readonly id: string;
    readonly displayOrder: number;
}

export interface InsightsExplorerDeleteRequest {
    readonly id: string;
}

export interface InsightsExplorerInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly displayOrder: number;
    readonly hidden: boolean;
    readonly data: Record<string, string | number | object[]>;
}

interface ExpressionNode {
    textualExpression: string;
    operator?: TransactionExplorerConditionRelation;
}

export class TransactionExplorerQuery {
    public id: string;
    public name: string;
    public conditions: TransactionExplorerConditionWithRelation[];

    private constructor(id: string, name: string, conditions: TransactionExplorerConditionWithRelation[]) {
        this.id = id;
        this.name = name;
        this.conditions = conditions;
    }

    public addNewCondition(field: TransactionExplorerConditionField, isFirst: boolean): TransactionExplorerConditionWithRelation {
        let condition: TransactionExplorerCondition;

        switch (field) {
            case TransactionExplorerConditionField.TransactionTimeDayOfWeek:
                condition = new TransactionExplorerTransactionTimeDayOfWeekCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.TransactionTimeDayOfMonth:
                condition = new TransactionExplorerTransactionTimeDayOfMonthCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.TransactionTimeMonthOfYear:
                condition = new TransactionExplorerTransactionTimeMonthOfYearCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.TransactionTimeHourOfDay:
                condition = new TransactionExplorerTransactionTimeHourOfDayCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.TransactionTimezone:
                condition = new TransactionExplorerTransactionTimezoneCondition(TransactionExplorerConditionOperatorType.MinuteOffsetBetween, [ 0, 0 ]);
                break;
            case TransactionExplorerConditionField.TransactionType:
                condition = new TransactionExplorerTransactionTypeCondition(TransactionExplorerConditionOperatorType.In, [ TransactionType.Expense, TransactionType.Income, TransactionType.Transfer ]);
                break;
            case TransactionExplorerConditionField.TransactionCategory:
                condition = new TransactionExplorerTransactionCategoryCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.SourceAccount:
                condition = new TransactionExplorerSourceAccountCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.DestinationAccount:
                condition = new TransactionExplorerDestinationAccountCondition(TransactionExplorerConditionOperatorType.In, []);
                break;
            case TransactionExplorerConditionField.SourceAmount:
                condition = new TransactionExplorerSourceAmountCondition(TransactionExplorerConditionOperatorType.Between, [0, 0]);
                break;
            case TransactionExplorerConditionField.DestinationAmount:
                condition = new TransactionExplorerDestinationAmountCondition(TransactionExplorerConditionOperatorType.Between, [0, 0]);
                break;
            case TransactionExplorerConditionField.GeoLocation:
                condition = new TransactionExplorerGeoLocationCondition(TransactionExplorerConditionOperatorType.IsNotEmpty, [TransactionExplorerGeoLocationCondition.MIN_LATITUDE, TransactionExplorerGeoLocationCondition.MAX_LATITUDE, TransactionExplorerGeoLocationCondition.MIN_LONGITUDE, TransactionExplorerGeoLocationCondition.MAX_LONGITUDE]);
                break;
            case TransactionExplorerConditionField.TransactionTag:
                condition = new TransactionExplorerTransactionTagCondition(TransactionExplorerConditionOperatorType.HasAny, []);
                break;
            case TransactionExplorerConditionField.Pictures:
                condition = new TransactionExplorerPicturesCondition(TransactionExplorerConditionOperatorType.IsNotEmpty, []);
                break;
            case TransactionExplorerConditionField.Description:
                condition = new TransactionExplorerDescriptionCondition(TransactionExplorerConditionOperatorType.Contains, '');
                break;
            case TransactionExplorerConditionField.DescriptionCaseInsensitive:
                condition = new TransactionExplorerDescriptionCaseInsensitiveCondition(TransactionExplorerConditionOperatorType.Contains, '');
                break;
            case TransactionExplorerConditionField.DescriptionNormalized:
                condition = new TransactionExplorerDescriptionNormalizedCondition(TransactionExplorerConditionOperatorType.Contains, '');
                break;
            default:
                condition = new TransactionExplorerTransactionTypeCondition(TransactionExplorerConditionOperatorType.In, [ TransactionType.Expense, TransactionType.Income, TransactionType.Transfer ]);
                break;
        }

        return new TransactionExplorerConditionWithRelation(
            condition,
            isFirst ? TransactionExplorerConditionRelation.First : TransactionExplorerConditionRelation.And
        );
    }

    public addSubConditionEnd(): TransactionExplorerConditionWithRelation {
        return new TransactionExplorerConditionWithRelation(new TransactionExplorerUndefinedCondition(), TransactionExplorerConditionRelation.SubEnd);
    }

    public match(transaction: TransactionInsightDataItem, context: InsightsExplorerMatchContext): boolean {
        if (!this.conditions || this.conditions.length < 1) {
            return true;
        }

        const postfixExprTokens = this.getPostfixExprTokens();
        const stack: boolean[] = [];

        for (const token of postfixExprTokens) {
            if (token === TransactionExplorerConditionRelation.And || token === TransactionExplorerConditionRelation.Or) {
                const right = stack.pop();
                const left = stack.pop();

                if (left === undefined || right === undefined) {
                    throw new Error('invalid postfix expression');
                }

                if (token === TransactionExplorerConditionRelation.And) {
                    stack.push(left && right);
                } else if (token === TransactionExplorerConditionRelation.Or) {
                    stack.push(left || right);
                } else {
                    throw new Error('invalid postfix expression');
                }
            } else {
                stack.push(token.match(transaction, context));
            }
        }

        if (stack.length !== 1) {
            throw new Error('invalid postfix evaluation result');
        }

        return stack[0] as boolean;
    }

    public toExpression(allCategoriesMap: Record<string, TransactionCategory>, allAccountsMap: Record<string, Account>, allTagsMap: Record<string, TransactionTag>): string {
        if (!this.conditions || this.conditions.length < 1) {
            return '';
        }

        const postfixExprTokens = this.getPostfixExprTokens();
        const stack: ExpressionNode[] = [];

        for (const token of postfixExprTokens) {
            if (token === TransactionExplorerConditionRelation.And || token === TransactionExplorerConditionRelation.Or) {
                const right = stack.pop();
                const left = stack.pop();

                if (left === undefined || right === undefined) {
                    throw new Error('invalid postfix expression');
                }

                let leftExpression = left.textualExpression;
                let rightExpression = right.textualExpression;

                if (left.operator && left.operator !== token) {
                    leftExpression = `(${leftExpression})`;
                }

                if (right.operator && right.operator !== token) {
                    rightExpression = `(${rightExpression})`;
                }

                stack.push({
                    textualExpression: `${leftExpression} ${token.toUpperCase()} ${rightExpression}`,
                    operator: token
                });
            } else {
                stack.push({
                    textualExpression: token.toExpression(allCategoriesMap, allAccountsMap, allTagsMap)
                });
            }
        }

        if (stack.length !== 1) {
            throw new Error('invalid postfix evaluation result');
        }

        const finalNode = stack[0];

        if (!finalNode) {
            throw new Error('invalid postfix evaluation result');
        }

        return finalNode.textualExpression;
    }

    public getPostfixExprTokens(): (TransactionExplorerCondition | TransactionExplorerConditionRelation.And | TransactionExplorerConditionRelation.Or)[] {
        const finalTokens: (TransactionExplorerCondition | TransactionExplorerConditionRelation.And | TransactionExplorerConditionRelation.Or)[] = [];

        if (this.conditions.length < 1) {
            return finalTokens;
        }

        const operatorStack: (TransactionExplorerConditionRelation | TransactionExplorerSubConditionStartRelation)[] = [];
        const firstCondition = this.conditions[0] as TransactionExplorerConditionWithRelation;

        if (firstCondition.relation !== TransactionExplorerConditionRelation.First) {
            throw new Error('the first condition must have relation "first"');
        }

        finalTokens.push(firstCondition.condition);

        for (const [item, index] of itemAndIndex(this.conditions)) {
            if (index < 1) {
                continue;
            }

            if (item.relation === TransactionExplorerConditionRelation.First) {
                throw new Error('only the first condition can have relation "first"');
            }

            if (item.relation === TransactionExplorerConditionRelation.SubEnd) {
                while (operatorStack.length > 0) {
                    const topOperator = operatorStack.pop();

                    if (topOperator === TransactionExplorerSubConditionStartRelationPlaceholder) {
                        break;
                    }

                    const isAndOrOperator = topOperator === TransactionExplorerConditionRelation.And || topOperator === TransactionExplorerConditionRelation.Or;

                    if (isAndOrOperator) {
                        finalTokens.push(topOperator);
                    } else {
                        throw new Error('invalid operator in stack');
                    }
                }
            } else { // And, Or, AndSub, OrSub
                let currentOperator: TransactionExplorerConditionRelation.And | TransactionExplorerConditionRelation.Or;
                let startNewSubCondition = false;

                switch (item.relation) {
                    case TransactionExplorerConditionRelation.AndSub:
                        currentOperator = TransactionExplorerConditionRelation.And;
                        startNewSubCondition = true;
                        break;
                    case TransactionExplorerConditionRelation.OrSub:
                        currentOperator = TransactionExplorerConditionRelation.Or;
                        startNewSubCondition = true;
                        break;
                    case TransactionExplorerConditionRelation.And:
                        currentOperator = item.relation;
                        break;
                    case TransactionExplorerConditionRelation.Or:
                        currentOperator = item.relation;
                        break;
                    default:
                        throw new Error('invalid operator in stack');
                }

                while (operatorStack.length > 0) {
                    const topOperator = operatorStack[operatorStack.length - 1];

                    if (topOperator === TransactionExplorerSubConditionStartRelationPlaceholder) {
                        break;
                    }

                    const isAndOrOperator = topOperator === TransactionExplorerConditionRelation.And || topOperator === TransactionExplorerConditionRelation.Or;

                    if (isAndOrOperator && TransactionExplorerConditionRelationPriority[topOperator] >= TransactionExplorerConditionRelationPriority[currentOperator]) {
                        finalTokens.push(topOperator);
                        operatorStack.pop();
                    } else {
                        break;
                    }
                }

                operatorStack.push(currentOperator);

                if (startNewSubCondition) {
                    operatorStack.push(TransactionExplorerSubConditionStartRelationPlaceholder);
                }

                finalTokens.push(item.condition);
            }
        }

        while (operatorStack.length > 0) {
            const topOperator = operatorStack.pop();

            if (topOperator !== TransactionExplorerConditionRelation.And && topOperator !== TransactionExplorerConditionRelation.Or) {
                throw new Error('invalid operator in stack');
            }

            finalTokens.push(topOperator);
        }

        return finalTokens;
    }

    public getConditionNestingDepths(): number[] {
        const depths: number[] = [];
        let depth = 0;

        for (const item of this.conditions) {
            if (item.relation === TransactionExplorerConditionRelation.SubEnd) {
                depth--;
                depths.push(depth);
            } else if (item.relation === TransactionExplorerConditionRelation.AndSub || item.relation === TransactionExplorerConditionRelation.OrSub) {
                depth++;
                depths.push(depth);
            } else {
                depths.push(depth);
            }
        }

        return depths;
    }

    public clone(newId: string): TransactionExplorerQuery {
        const clonedConditions: TransactionExplorerConditionWithRelation[] = [];

        for (const condition of this.conditions) {
            const clonedCondition = TransactionExplorerConditionWithRelation.parse(condition.toJsonObject());

            if (!clonedCondition) {
                continue;
            }

            clonedConditions.push(clonedCondition);
        }

        return new TransactionExplorerQuery(newId, this.name, clonedConditions);
    }

    public toJsonObject(): object {
        return {
            id: this.id,
            name: this.name,
            conditions: this.conditions.map(condition => condition.toJsonObject())
        };
    }

    public static create(id: string): TransactionExplorerQuery {
        return new TransactionExplorerQuery(id, '', []);
    }

    public static parse(obj: object): TransactionExplorerQuery | null {
        if (!('id' in obj) || !('name' in obj) || !('conditions' in obj)) {
            return null;
        }

        const idFieldValue = obj['id'] as unknown;
        const nameFieldValue = obj['name'] as unknown;
        const conditionsFieldValue = obj['conditions'] as unknown;

        if (typeof idFieldValue !== 'string' || typeof nameFieldValue !== 'string' || !Array.isArray(conditionsFieldValue)) {
            return null;
        }

        const id: string = idFieldValue;
        const name: string = nameFieldValue;
        const conditions: TransactionExplorerConditionWithRelation[] = [];
        let conditionDepth = 0;

        for (const [item, index] of itemAndIndex(conditionsFieldValue)) {
            const condition = TransactionExplorerConditionWithRelation.parse(item);

            if (condition === null) {
                return null;
            }

            if (index === 0 && condition.relation !== TransactionExplorerConditionRelation.First) {
                return null;
            } else if (index > 0 && condition.relation === TransactionExplorerConditionRelation.First) {
                return null;
            }

            if (condition.relation === TransactionExplorerConditionRelation.AndSub ||
                condition.relation === TransactionExplorerConditionRelation.OrSub) {
                conditionDepth++;
            } else if (condition.relation === TransactionExplorerConditionRelation.SubEnd) {
                conditionDepth--;
            }

            conditions.push(condition);
        }

        if (conditionDepth !== 0) {
            return null; // unbalanced parentheses
        }

        return new TransactionExplorerQuery(id, name, conditions);
    }
}

export class TransactionExplorerConditionWithRelation {
    public condition: TransactionExplorerCondition;
    public relation: TransactionExplorerConditionRelation;

    constructor(condition: TransactionExplorerCondition, relation: TransactionExplorerConditionRelation) {
        this.condition = condition;
        this.relation = relation;
    }

    public getSupportedOperators(): TransactionExplorerConditionOperator[] {
        let operatorTypes: PartialRecord<TransactionExplorerConditionOperatorType, true> = {};

        switch (this.condition.field) {
            case TransactionExplorerConditionField.TransactionTimeDayOfWeek.value:
                operatorTypes = TransactionExplorerTransactionTimeDayOfWeekCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionTimeDayOfMonth.value:
                operatorTypes = TransactionExplorerTransactionTimeDayOfMonthCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionTimeMonthOfYear.value:
                operatorTypes = TransactionExplorerTransactionTimeMonthOfYearCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionTimeHourOfDay.value:
                operatorTypes = TransactionExplorerTransactionTimeHourOfDayCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionTimezone.value:
                operatorTypes = TransactionExplorerTransactionTimezoneCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionType.value:
                operatorTypes = TransactionExplorerTransactionTypeCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionCategory.value:
                operatorTypes = TransactionExplorerTransactionCategoryCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.SourceAccount.value:
                operatorTypes = TransactionExplorerSourceAccountCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.DestinationAccount.value:
                operatorTypes = TransactionExplorerDestinationAccountCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.SourceAmount.value:
                operatorTypes = TransactionExplorerSourceAmountCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.DestinationAmount.value:
                operatorTypes = TransactionExplorerDestinationAmountCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.GeoLocation.value:
                operatorTypes = TransactionExplorerGeoLocationCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.TransactionTag.value:
                operatorTypes = TransactionExplorerTransactionTagCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.Pictures.value:
                operatorTypes = TransactionExplorerPicturesCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.Description.value:
                operatorTypes = TransactionExplorerDescriptionCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.DescriptionCaseInsensitive.value:
                operatorTypes = TransactionExplorerDescriptionCaseInsensitiveCondition.supportedOperators;
                break;
            case TransactionExplorerConditionField.DescriptionNormalized.value:
                operatorTypes = TransactionExplorerDescriptionNormalizedCondition.supportedOperators;
                break;
            default:
                return [];
        }

        const result: TransactionExplorerConditionOperator[] = [];

        for (const key of keysIfValueEquals(operatorTypes, true)) {
            const operator = TransactionExplorerConditionOperator.valueOf(key);

            if (operator) {
                result.push(operator);
            }
        }

        return result;
    }

    public toJsonObject(): unknown {
        if (this.relation === TransactionExplorerConditionRelation.SubEnd) {
            return {
                relation: this.relation
            };
        }

        return {
            condition: {
                field: this.condition.field,
                operator: this.condition.operator,
                value: this.condition.getValueForStore()
            },
            relation: this.relation
        };
    }

    public static parse(data: unknown): TransactionExplorerConditionWithRelation | null {
        if (typeof data !== 'object' || !data || !('relation' in data)) {
            return null;
        }

        const relation = data['relation'];
        let condition: TransactionExplorerCondition | null = null;

        if (relation === TransactionExplorerConditionRelation.First ||
            relation === TransactionExplorerConditionRelation.And || relation === TransactionExplorerConditionRelation.Or ||
            relation === TransactionExplorerConditionRelation.AndSub || relation === TransactionExplorerConditionRelation.OrSub) {
            if (!('condition' in data)) {
                return null;
            }

            const conditionObject = data['condition'];

            if (typeof conditionObject !== 'object' || !conditionObject || !('field' in conditionObject) || !('operator' in conditionObject) || !('value' in conditionObject) || typeof relation !== 'string') {
                return null;
            }

            const conditionField = conditionObject['field'];
            const conditionOperator = conditionObject['operator'] as TransactionExplorerConditionOperatorType;
            const conditionValue = conditionObject['value'];

            switch (conditionField) {
                case TransactionExplorerConditionField.TransactionTimeDayOfWeek.value:
                    if (TransactionExplorerTransactionTimeDayOfWeekCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTimeDayOfWeekCondition(conditionOperator as TransactionTimeDayOfWeekConditionOperator, conditionValue as number[]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionTimeDayOfMonth.value:
                    if (TransactionExplorerTransactionTimeDayOfMonthCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTimeDayOfMonthCondition(conditionOperator as TransactionTimeDayOfMonthConditionOperator, conditionValue as number[]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionTimeMonthOfYear.value:
                    if (TransactionExplorerTransactionTimeMonthOfYearCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTimeMonthOfYearCondition(conditionOperator as TransactionTimeMonthOfYearConditionOperator, conditionValue as number[]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionTimeHourOfDay.value:
                    if (TransactionExplorerTransactionTimeHourOfDayCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTimeHourOfDayCondition(conditionOperator as TransactionTimeHourOfDayConditionOperator, conditionValue as number[]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionTimezone.value:
                    if (TransactionExplorerTransactionTimezoneCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue) && conditionValue.length === 2) {
                        condition = new TransactionExplorerTransactionTimezoneCondition(conditionOperator as TransactionTimezoneConditionOperator, conditionValue as [number, number]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionType.value:
                    if (TransactionExplorerTransactionTypeCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTypeCondition(conditionOperator as TransactionTypeConditionOperator, conditionValue as number[]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionCategory.value:
                    if (TransactionExplorerTransactionCategoryCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionCategoryCondition(conditionOperator as TransactionCategoryConditionOperator, conditionValue as string[]);
                    }
                    break;
                case TransactionExplorerConditionField.SourceAccount.value:
                    if (TransactionExplorerSourceAccountCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerSourceAccountCondition(conditionOperator as AccountConditionOperator, conditionValue as string[]);
                    }
                    break;
                case TransactionExplorerConditionField.DestinationAccount.value:
                    if (TransactionExplorerDestinationAccountCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerDestinationAccountCondition(conditionOperator as AccountConditionOperator, conditionValue as string[]);
                    }
                    break;
                case TransactionExplorerConditionField.SourceAmount.value:
                    if (TransactionExplorerSourceAmountCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue) && conditionValue.length === 2) {
                        condition = new TransactionExplorerSourceAmountCondition(conditionOperator as AmountConditionOperator, conditionValue as [number, number]);
                    }
                    break;
                case TransactionExplorerConditionField.DestinationAmount.value:
                    if (TransactionExplorerDestinationAmountCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue) && conditionValue.length === 2) {
                        condition = new TransactionExplorerDestinationAmountCondition(conditionOperator as AmountConditionOperator, conditionValue as [number, number]);
                    }
                    break;
                case TransactionExplorerConditionField.GeoLocation.value:
                    if (TransactionExplorerGeoLocationCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerGeoLocationCondition(conditionOperator as GeoLocationConditionOperator, conditionValue as [number, number, number, number]);
                    }
                    break;
                case TransactionExplorerConditionField.TransactionTag.value:
                    if (TransactionExplorerTransactionTagCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerTransactionTagCondition(conditionOperator as TransactionTagConditionOperator, conditionValue as string[]);
                    }
                    break;
                case TransactionExplorerConditionField.Pictures.value:
                    if (TransactionExplorerPicturesCondition.supportedOperators[conditionOperator] && Array.isArray(conditionValue)) {
                        condition = new TransactionExplorerPicturesCondition(conditionOperator as PicturesConditionOperator, conditionValue as string[]);
                    }
                    break;
                case TransactionExplorerConditionField.Description.value:
                    if (TransactionExplorerDescriptionCondition.supportedOperators[conditionOperator] && typeof conditionValue === 'string') {
                        condition = new TransactionExplorerDescriptionCondition(conditionOperator as DescriptionConditionOperator, conditionValue);
                    }
                    break;
                case TransactionExplorerConditionField.DescriptionCaseInsensitive.value:
                    if (TransactionExplorerDescriptionCaseInsensitiveCondition.supportedOperators[conditionOperator] && typeof conditionValue === 'string') {
                        condition = new TransactionExplorerDescriptionCaseInsensitiveCondition(conditionOperator as DescriptionConditionOperator, conditionValue);
                    }
                    break;
                case TransactionExplorerConditionField.DescriptionNormalized.value:
                    if (TransactionExplorerDescriptionNormalizedCondition.supportedOperators[conditionOperator] && typeof conditionValue === 'string') {
                        condition = new TransactionExplorerDescriptionNormalizedCondition(conditionOperator as DescriptionConditionOperator, conditionValue);
                    }
                    break;
                default:
                    break;
            }
        } else if (relation === TransactionExplorerConditionRelation.SubEnd) {
            condition = new TransactionExplorerUndefinedCondition();
        } else {
            return null;
        }

        if (condition === null) {
            return null;
        }

        return new TransactionExplorerConditionWithRelation(condition, relation);
    }
}

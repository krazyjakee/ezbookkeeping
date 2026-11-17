import { describe, expect, it } from 'vitest';

import { TransactionExplorerDataDimension } from '@/core/explorer.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { calculateAmountRanges } from '@/lib/explorer.ts';

function ranges(amounts: number[], dimension: TransactionExplorerDataDimension, rangeCount: number): string[] {
    return calculateAmountRanges(amounts.map(amount => parseBigDecimal(amount)), dimension, rangeCount).map(amount => amount.toString());
}

describe('calculateAmountRanges', () => {
    const amounts = [100, 250, 300, 1200, 1500, 9000, 9100, 20000];
    const amountsWithNegative = [-5000, -100, 0, 50, 75, 80, 4000, 4100, 4200, 100000];

    it('should return empty ranges for empty input or non-positive range count', () => {
        expect(ranges([], TransactionExplorerDataDimension.SourceAmountRangeNaturalBreaks, 3)).toEqual([]);
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmountRangeNaturalBreaks, 0)).toEqual([]);
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmount, 3)).toEqual([]);
    });

    it('should calculate equal frequency and equal width ranges', () => {
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmountRangeEqualFrequency, 4)).toEqual(['100', '250', '1200', '9000', '20000']);
        expect(ranges(amounts, TransactionExplorerDataDimension.DestinationAmountRangeEqualWidth, 4)).toEqual(['100', '5075', '10050', '15025', '20000']);
        expect(ranges([700, 700], TransactionExplorerDataDimension.SourceAmountRangeEqualWidth, 3)).toEqual(['700', '700']);
    });

    it('should calculate log scale and standard deviation ranges', () => {
        expect(ranges(amountsWithNegative, TransactionExplorerDataDimension.SourceAmountRangeLogScale, 3)).toEqual(['-5000', '-6.2996052484751847721', '125.99210498847975692', '100000']);
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmountRangeStandardDeviation, 3)).toEqual(['100', '5181.25', '11822.97123304644577', '20000']);
    });

    it('should calculate natural breaks ranges', () => {
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmountRangeNaturalBreaks, 3)).toEqual(['100', '9000', '20000', '20000']);
        expect(ranges(amounts, TransactionExplorerDataDimension.SourceAmountRangeNaturalBreaks, 4)).toEqual(['100', '1200', '9000', '20000', '20000']);
        expect(ranges(amountsWithNegative, TransactionExplorerDataDimension.DestinationAmountRangeNaturalBreaks, 4)).toEqual(['-5000', '-100', '4000', '100000', '100000']);
        expect(ranges([700, 700], TransactionExplorerDataDimension.SourceAmountRangeNaturalBreaks, 4)).toEqual(['700', '700']);
    });
});

import { describe, expect, it } from 'vitest';

import { isArrayEquals, isTextualUUID } from '@/lib/common.ts';

describe('isTextualUUID', () => {
	it('should return false for null', () => {
		expect(isTextualUUID(null)).toBe(false);
	});

	it('should return false for undefined', () => {
		expect(isTextualUUID(undefined)).toBe(false);
	});

	it('should return false for number', () => {
		expect(isTextualUUID(12345)).toBe(false);
	});

	it('should return false for boolean', () => {
		expect(isTextualUUID(true)).toBe(false);
	});

	it('should return false for object', () => {
		expect(isTextualUUID({})).toBe(false);
	});

	it('should return false for empty string', () => {
		expect(isTextualUUID('')).toBe(false);
	});

	it('should return false for string without hyphens', () => {
		expect(isTextualUUID('notauuid')).toBe(false);
	});

	it('should return false for UUID with wrong number of segments', () => {
		expect(isTextualUUID('1234-5678')).toBe(false);
	});

	it('should return false for UUID with wrong segment lengths', () => {
		expect(isTextualUUID('1234567-1234-1234-1234-123456789012')).toBe(false);
	});

	it('should return false for UUID containing non-hex characters', () => {
		expect(isTextualUUID('1234567g-1234-1234-1234-123456789012')).toBe(false);
	});

	it('should return false for UUID with uppercase non-hex characters', () => {
		expect(isTextualUUID('1234567G-1234-1234-1234-123456789012')).toBe(false);
	});

	it('should return true for valid lowercase UUID', () => {
		expect(isTextualUUID('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
	});

	it('should return true for valid uppercase UUID', () => {
		expect(isTextualUUID('550E8400-E29B-41D4-A716-446655440000')).toBe(true);
	});

	it('should return true for valid mixed-case UUID', () => {
		expect(isTextualUUID('550e8400-E29B-41d4-A716-446655440000')).toBe(true);
	});

	it('should return true for all-zero UUID', () => {
		expect(isTextualUUID('00000000-0000-0000-0000-000000000000')).toBe(true);
	});

	it('should return true for all-f UUID', () => {
		expect(isTextualUUID('ffffffff-ffff-ffff-ffff-ffffffffffff')).toBe(true);
	});
});

describe('isArrayEquals', () => {
    interface Item {
        key: string;
        value: number;
    }

    const getKey = (item: Item): string => item.key;
    const itemsEqual = (left: Item, right: Item): boolean => left.key === right.key && left.value === right.value;

    it.each(['__proto__', 'constructor', 'toString'])('compares items with the key %s', key => {
        expect(isArrayEquals([{ key, value: 1 }], [{ key, value: 1 }], getKey, itemsEqual)).toBe(true);
        expect(isArrayEquals([{ key, value: 1 }], [{ key, value: 2 }], getKey, itemsEqual)).toBe(false);
    });

    it('matches reordered duplicates without modifying either array', () => {
        const left = [
            Object.freeze({ key: 'account', value: 1 }),
            Object.freeze({ key: 'account', value: 2 }),
            Object.freeze({ key: 'account', value: 1 })
        ];
        const right = [left[1]!, left[0]!, left[2]!];
        Object.freeze(left);
        Object.freeze(right);

        expect(isArrayEquals(left, right, getKey, itemsEqual)).toBe(true);
        expect(left.map(item => item.value)).toEqual([1, 2, 1]);
        expect(right.map(item => item.value)).toEqual([2, 1, 1]);
    });

    it('rejects different duplicate counts even when lengths and keys match', () => {
        const left = [{ key: 'account', value: 1 }, { key: 'account', value: 1 }, { key: 'account', value: 2 }];
        const right = [{ key: 'account', value: 1 }, { key: 'account', value: 2 }, { key: 'account', value: 2 }];

        expect(isArrayEquals(left, right, getKey, itemsEqual)).toBe(false);
    });

    it('rejects missing keys and different lengths', () => {
        expect(isArrayEquals([{ key: 'first', value: 1 }], [{ key: 'second', value: 1 }], getKey, itemsEqual)).toBe(false);
        expect(isArrayEquals([{ key: 'first', value: 1 }], [], getKey, itemsEqual)).toBe(false);
    });

    it('compares empty arrays', () => {
        expect(isArrayEquals([], [], getKey, itemsEqual)).toBe(true);
    });
});

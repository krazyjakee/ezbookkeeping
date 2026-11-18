import { ref, computed, useId } from 'vue';

/**
 * Data table alternative for chart panels (design guideline D7, accessibility audit T1 in #36).
 * Every chart panel pairs the chart with a table of the exact values it draws, built from the same data.
 * Amount cells carry a type so they can be rendered with `SignedAmount` (D1); cells without a type
 * (for example balances, which keep their natural sign) are rendered as plain numeric text.
 */
export interface ChartDataTableColumn {
    readonly key: string;
    readonly name: string;
    readonly numeric: boolean;
}

export interface ChartDataTableCell {
    readonly text: string;
    readonly type?: 'income' | 'expense' | 'transfer' | 'none';
    readonly hideSign?: boolean;
}

export interface ChartDataTableRow {
    readonly key: string;
    readonly name: string;
    readonly cells: ChartDataTableCell[];
}

export interface ChartDataTable {
    readonly columns: ChartDataTableColumn[];
    readonly rows: ChartDataTableRow[];
}

/**
 * State of a "data table" disclosure button (`aria-expanded` / `aria-controls`).
 * The table stays collapsed by default so dashboards remain compact, and is only rendered after it was opened once.
 */
export function useChartDataTableDisclosure() {
    const regionId = `chart-data-table-${useId()}`;
    const expanded = ref<boolean>(false);
    const rendered = ref<boolean>(false);

    const buttonAttrs = computed<Record<string, string>>(() => ({
        'aria-expanded': expanded.value ? 'true' : 'false',
        'aria-controls': regionId
    }));

    function toggle(): void {
        expanded.value = !expanded.value;

        if (expanded.value) {
            rendered.value = true;
        }
    }

    function collapse(): void {
        expanded.value = false;
    }

    return {
        regionId,
        expanded,
        rendered,
        buttonAttrs,
        toggle,
        collapse
    };
}

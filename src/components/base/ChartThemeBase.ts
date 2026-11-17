import { ref } from 'vue';

/**
 * Shared ECharts presentation for the "Copper Instrument Panel" design (guidelines §4.5 / D7).
 *
 * ECharts draws on a canvas, so it cannot read the CSS design tokens. The values below mirror
 * `src/styles/common/_tokens.scss` (text primary / secondary, hairline, raised surface) and must be kept in sync with it.
 * Series colours are never set here: they keep coming from the user chart colour scheme, the user amount colours
 * or the item (category / account) colours.
 */
export interface ChartThemePalette {
    readonly textPrimary: string;
    readonly textSecondary: string;
    readonly hairline: string;
    readonly gridLine: string;
    readonly axisLine: string;
    readonly surface: string;
    readonly surfaceRaised: string;
    readonly surfaceSunken: string;
    readonly copper: string;
    readonly copperLine: string;
    readonly tooltipShadow: string;
    readonly emphasisShadow: string;
}

const LIGHT_PALETTE: ChartThemePalette = {
    textPrimary: '#2B2421',
    textSecondary: '#6B605A',
    hairline: 'rgba(54, 47, 42, 0.14)',
    gridLine: 'rgba(54, 47, 42, 0.10)',
    axisLine: 'rgba(54, 47, 42, 0.28)',
    surface: '#FFFFFF',
    surfaceRaised: '#FFFFFF',
    surfaceSunken: '#F1ECE5',
    copper: '#C67E48',
    copperLine: 'rgba(198, 126, 72, 0.45)',
    tooltipShadow: '0 6px 18px rgba(43, 36, 33, 0.14)',
    emphasisShadow: 'rgba(43, 36, 33, 0.28)'
};

const DARK_PALETTE: ChartThemePalette = {
    textPrimary: '#F4EBE1',
    textSecondary: '#B9ADA2',
    hairline: 'rgba(236, 226, 217, 0.12)',
    gridLine: 'rgba(236, 226, 217, 0.09)',
    axisLine: 'rgba(236, 226, 217, 0.24)',
    surface: '#161412',
    surfaceRaised: '#1E1B18',
    surfaceSunken: '#100E0C',
    copper: '#C67E48',
    copperLine: 'rgba(198, 126, 72, 0.55)',
    tooltipShadow: '0 8px 22px rgba(0, 0, 0, 0.5)',
    emphasisShadow: 'rgba(0, 0, 0, 0.6)'
};

/**
 * Canvas text cannot use `font-variant-numeric`, so axis and value labels use a stack whose default figures are tabular
 * (Segoe UI, Roboto, Helvetica Neue and Arial all ship tabular digits by default) before falling back to the system font.
 */
export const CHART_NUMERIC_FONT_FAMILY = '"Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", system-ui, sans-serif';

/** Symbol shapes that keep line / scatter series distinguishable without colour (D7) */
export const CHART_SERIES_SYMBOLS: string[] = ['circle', 'rect', 'triangle', 'diamond', 'roundRect', 'pin', 'arrow'];

/** Dash pattern used by expense series (D7: income solid, expense dashed) */
export const CHART_EXPENSE_LINE_TYPE: number[] = [6, 4];

const highContrastActive = ref<boolean>(false);
let highContrastListenerInstalled = false;

function installHighContrastListener(): void {
    if (highContrastListenerInstalled || typeof window === 'undefined' || !window.matchMedia) {
        return;
    }

    highContrastListenerInstalled = true;

    const queries = [
        window.matchMedia('(forced-colors: active)'),
        window.matchMedia('(prefers-contrast: more)')
    ];

    const update = () => {
        highContrastActive.value = queries.some(query => query.matches);
    };

    for (const query of queries) {
        if (query.addEventListener) {
            query.addEventListener('change', update);
        }
    }

    update();
}

/**
 * Whether the forced-colors / high-contrast mode is active. Reactive when read inside computed properties.
 * Pattern fills (decals) are only used in this mode (D7).
 */
export function isChartHighContrastMode(): boolean {
    installHighContrastListener();
    return highContrastActive.value;
}

export function getChartThemePalette(isDarkMode: boolean | undefined): ChartThemePalette {
    return isDarkMode ? DARK_PALETTE : LIGHT_PALETTE;
}

/** Tooltip rendered as a raised panel: raised surface, hairline border, panel radius and tabular figures */
export function getChartTooltipStyleOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        backgroundColor: palette.surfaceRaised,
        borderColor: palette.hairline,
        borderWidth: 1,
        padding: [8, 12],
        textStyle: {
            color: palette.textPrimary,
            fontSize: 13
        },
        extraCssText: `border-radius: 10px; box-shadow: ${palette.tooltipShadow}; font-variant-numeric: tabular-nums;`
    };
}

/** Axis pointer label styled like a small raised chip */
export function getChartAxisPointerLabelOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        backgroundColor: palette.surfaceRaised,
        borderColor: palette.hairline,
        borderWidth: 1,
        color: palette.textPrimary,
        fontFamily: CHART_NUMERIC_FONT_FAMILY,
        shadowBlur: 0
    };
}

export function getChartAxisLabelStyleOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        color: palette.textSecondary,
        fontFamily: CHART_NUMERIC_FONT_FAMILY,
        fontSize: 11
    };
}

export function getChartAxisLineOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        lineStyle: {
            color: palette.axisLine,
            width: 1
        }
    };
}

/** Grid / split lines at hairline strength */
export function getChartSplitLineOptions(isDarkMode: boolean | undefined, show?: boolean): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        show: show ?? true,
        lineStyle: {
            color: palette.gridLine,
            width: 1
        }
    };
}

export function getChartLegendTextStyleOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        color: palette.textPrimary,
        fontSize: 12
    };
}

/** Legend paging arrows and inactive items in the panel palette */
export function getChartLegendStyleOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        textStyle: getChartLegendTextStyleOptions(isDarkMode),
        inactiveColor: palette.axisLine,
        pageIconColor: palette.copper,
        pageIconInactiveColor: palette.axisLine,
        pageTextStyle: {
            color: palette.textSecondary,
            fontFamily: CHART_NUMERIC_FONT_FAMILY
        },
        itemGap: 14
    };
}

/** Value / data labels drawn inside the chart */
export function getChartLabelTextStyleOptions(isDarkMode: boolean | undefined): Record<string, unknown> {
    const palette = getChartThemePalette(isDarkMode);

    return {
        color: palette.textPrimary,
        fontFamily: CHART_NUMERIC_FONT_FAMILY,
        fontSize: 12
    };
}

/**
 * Pattern fill for a series or data item. Returns undefined unless the forced-colors / high-contrast mode is active,
 * pattern fills are not used otherwise (D7).
 */
export function getChartDecal(index: number, isDarkMode: boolean | undefined): Record<string, unknown> | undefined {
    if (!isChartHighContrastMode()) {
        return undefined;
    }

    const patterns: Record<string, unknown>[] = [
        { symbol: 'none', dashArrayX: [1, 0], dashArrayY: [2, 5], rotation: Math.PI / 4 },
        { symbol: 'circle', symbolSize: 0.6, dashArrayX: [[8, 8], [0, 8, 8, 0]], dashArrayY: [6, 0] },
        { symbol: 'none', dashArrayX: [1, 0], dashArrayY: [2, 5], rotation: -Math.PI / 4 },
        { symbol: 'rect', symbolSize: 0.7, dashArrayX: [[6, 6], [0, 6, 6, 0]], dashArrayY: [5, 0] },
        { symbol: 'none', dashArrayX: [1, 0], dashArrayY: [3, 3], rotation: 0 },
        { symbol: 'triangle', symbolSize: 0.8, dashArrayX: [[8, 8], [0, 8, 8, 0]], dashArrayY: [7, 0] },
        { symbol: 'none', dashArrayX: [[1, 0], [1, 6]], dashArrayY: [1, 0, 6, 0], rotation: Math.PI / 2 }
    ];

    const pattern = patterns[Math.abs(index) % patterns.length] as Record<string, unknown>;

    return {
        ...pattern,
        color: isDarkMode ? 'rgba(255, 255, 255, 0.75)' : 'rgba(0, 0, 0, 0.6)'
    };
}

export function getChartSeriesSymbol(index: number): string {
    return CHART_SERIES_SYMBOLS[Math.abs(index) % CHART_SERIES_SYMBOLS.length] as string;
}

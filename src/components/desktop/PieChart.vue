<template>
    <div class="pie-chart-wrapper">
        <v-chart autoresize class="pie-chart-container" :class="{ 'transition-in': skeleton }"
                 :option="chartOptions" :update-options="{ notMerge: true }"
                 @click="clickItem" @legendselectchanged="onLegendSelectChanged" />
        <div class="pie-chart-center" aria-hidden="true"
             v-if="styleType === 'donut' && !skeleton && (centerTitle || centerValue)">
            <div class="pie-chart-center-title ebk-technical-heading" v-if="centerTitle">{{ centerTitle }}</div>
            <div class="pie-chart-center-value" v-if="centerValue">{{ centerValue }}</div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useTheme } from 'vuetify';

import type { ECElementEvent } from 'echarts/core';
import type { CallbackDataParams } from 'echarts/types/dist/shared';

import { type CommonPieChartDataItem, type CommonPieChartProps, usePieChartBase } from '@/components/base/PieChartBase.ts'

import { itemAndIndex } from '@/core/base.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import type { ColorStyleValue } from '@/core/color.ts';
import { ThemeType } from '@/core/theme.ts';

import { getObjectOwnFieldCount } from '@/lib/common.ts';
import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import { getChartAnimationOptions } from '@/lib/motion.ts';
import {
    getChartThemePalette,
    getChartTooltipStyleOptions,
    getChartLegendStyleOptions,
    getChartLabelTextStyleOptions,
    getChartDecal
} from '@/components/base/ChartThemeBase.ts';

type PieChartStyleType = 'pie' | 'donut' | 'nightingaleRose';

interface DesktopPieChartDataItem extends CommonPieChartDataItem {
    itemStyle: {
        color: ColorStyleValue;
        decal?: Record<string, unknown>;
    };
    selected: boolean;
}

interface DesktopPieChartProps extends CommonPieChartProps {
    styleType?: PieChartStyleType;
    centerTitle?: string; // only shown in the donut style, e.g. "Total Expense"
    centerValue?: string; // only shown in the donut style, e.g. the formatted total amount
}

const props = defineProps<DesktopPieChartProps>();

const emit = defineEmits<{
    (e: 'click', value: Record<string, unknown>): void;
}>();

const theme = useTheme();

const { selectedIndex, validItems, allItemsMap } = usePieChartBase(props);

const selectedLegends = ref<Record<string, boolean>>({});

const validItemsMap = computed<Record<string, CommonPieChartDataItem>>(() => {
    const map: Record<string, CommonPieChartDataItem> = {};

    for (const item of validItems.value) {
        map[item.id] = item;
    }

    return map;
});

const isDarkMode = computed<boolean>(() => theme.global.name.value === ThemeType.Dark);

const seriesRadius = computed<[number | string, string]>(() => {
    if (props.styleType === 'donut') {
        return ['48%', '72%'];
    }

    return [0, '75%'];
});

const seriesData = computed<DesktopPieChartDataItem[]>(() => {
    const ret: DesktopPieChartDataItem[] = [];

    for (const [item, index] of itemAndIndex(validItems.value)) {
        const decal = getChartDecal(index, isDarkMode.value);

        ret.push({
            ...item,
            itemStyle: {
                color: item.color,
                ...(decal ? { decal: decal } : {})
            },
            selected: true
        });
    }

    return ret;
});

const hasUnselectedItem = computed<boolean>(() => {
    for (const item of validItems.value) {
        if (getObjectOwnFieldCount(selectedLegends.value) && !selectedLegends.value[item.id]) {
            return true;
        }
    }

    return false;
});

const firstItemAndHalfCurrentItemTotalPercent = computed<number>(() => {
    let totalValue: BigDecimal = BIG_DECIMAL_ZERO;
    let firstValue: string | null = null;
    let firstToCurrentTotalValue: BigDecimal = BIG_DECIMAL_ZERO;

    for (const [item, index] of itemAndIndex(validItems.value)) {
        if (getObjectOwnFieldCount(selectedLegends.value) && !selectedLegends.value[item.id]) {
            continue;
        }

        if (firstValue === null) {
            firstValue = item.originalValue;
        }

        let positiveValue: BigDecimal = parseBigDecimal(item.value);

        if (positiveValue.isNegative()) {
            positiveValue = BIG_DECIMAL_ZERO;
        }

        if (firstValue !== null) {
            if (index < selectedIndex.value) {
                firstToCurrentTotalValue = firstToCurrentTotalValue.add(positiveValue);
            } else if (index === selectedIndex.value) {
                firstToCurrentTotalValue = firstToCurrentTotalValue.add(positiveValue.divide(2));
            }
        }

        totalValue = totalValue.add(positiveValue);
    }

    if (firstToCurrentTotalValue && totalValue.isPositive()) {
        return firstToCurrentTotalValue.divide(totalValue).toDoubleNumber();
    } else {
        return 0;
    }
});

const chartOptions = computed<object>(() => {
    const palette = getChartThemePalette(isDarkMode.value);

    return {
        backgroundColor: 'transparent',
        tooltip: {
            trigger: 'item',
            ...getChartTooltipStyleOptions(isDarkMode.value),
            formatter: (params: CallbackDataParams) => {
                const dataItem = params.data as DesktopPieChartDataItem;
                let percent = dataItem.displayPercent;

                if (hasUnselectedItem.value) {
                    percent = params.percent + '%';
                }

                let tooltip = `<div><span class="chart-pointer" style="background-color: ${params.color}"></span>`;

                if (dataItem.displayName) {
                    tooltip += `<div class="d-inline-flex">${dataItem.displayName}</div><br/>`;
                }

                const showValue = props.showValue;
                const showPercent = props.showPercent && dataItem.value > 0;

                if (showValue && showPercent) {
                    tooltip += `<div class="d-inline-flex"><span>${dataItem.displayValue}</span><span class="ms-1">(${percent})</span></div>`;
                } else if (showValue && !showPercent) {
                    tooltip += `<div class="d-inline-flex">${dataItem.displayValue}</div>`;
                } else if (!showValue && showPercent) {
                    tooltip += `<div class="d-inline-flex">${percent}</div>`;
                }

                tooltip += '</div>';

                return tooltip;
            }
        },
        legend: {
            orient: 'horizontal',
            type: 'scroll',
            top: 0,
            data: validItems.value.map(item => item.name),
            selected: selectedLegends.value,
            ...getChartLegendStyleOptions(isDarkMode.value),
            icon: 'roundRect',
            itemWidth: 12,
            itemHeight: 12,
            formatter: (id: string) => {
                const name = allItemsMap.value[id]?.name ?? id;
                const validItem = validItemsMap.value[id];

                if (!validItem || props.skeleton) {
                    return name;
                }

                const parts: string[] = [name];

                if (props.showValue) {
                    parts.push(validItem.displayValue);
                }

                if (props.showPercent && validItem.value > 0) {
                    parts.push(validItem.displayPercent);
                }

                return parts.join('  ');
            }
        },
        series: [
            {
                type: 'pie',
                data: seriesData.value,
                top: 50,
                startAngle: -90 + firstItemAndHalfCurrentItemTotalPercent.value * 360,
                radius: seriesRadius.value,
                roseType: props.styleType === 'nightingaleRose' ? 'radius' : undefined,
                itemStyle: {
                    borderColor: palette.surface,
                    borderWidth: validItems.value.length > 1 ? 1 : 0
                },
                emphasis: {
                    itemStyle: {
                        shadowBlur: 8,
                        shadowOffsetX: 0,
                        shadowColor: palette.emphasisShadow
                    }
                },
                label: {
                    ...getChartLabelTextStyleOptions(isDarkMode.value),
                    formatter: (params: CallbackDataParams) => {
                        const dataItem = params.data as DesktopPieChartDataItem;

                        if (!dataItem) {
                            return '';
                        }

                        // donut segments carry percentage labels (design guideline D7)
                        if (props.showPercent && !props.skeleton && dataItem.value > 0) {
                            const percent = hasUnselectedItem.value ? `${params.percent}%` : dataItem.displayPercent;
                            return `${dataItem.displayName}\n{percent|${percent}}`;
                        }

                        return dataItem.displayName;
                    },
                    rich: {
                        percent: {
                            ...getChartLabelTextStyleOptions(isDarkMode.value),
                            color: palette.textSecondary,
                            fontSize: 11,
                            lineHeight: 16
                        }
                    }
                },
                labelLine: {
                    lineStyle: {
                        color: palette.axisLine
                    }
                },
                ...getChartAnimationOptions(!props.skeleton)
            }
        ],
        media: [
            {
                query: {
                    minWidth: 600,
                },
                option: {
                    legend: {
                        orient: 'vertical',
                        left: 'left'
                    },
                    series: [
                        {
                            type: 'pie',
                            top: 0
                        }
                    ]
                },
            }
        ]
    };
});

function clickItem(e: ECElementEvent): void {
    if (!props.enableClickItem || e.componentType !== 'series' || e.seriesType !=='pie') {
        return;
    }

    if (e.event && e.event.target && e.event.target.currentStates && e.event.target.currentStates[0] && e.event.target.currentStates[0] === 'emphasis') {
        selectedIndex.value = e.dataIndex;
        return;
    }

    if (!e.data) {
        return;
    }

    const data = e.data as object;

    if ('sourceItem' in data) {
        emit('click', data.sourceItem as Record<string, unknown>);
    }
}

function onLegendSelectChanged(e: { selected: Record<string, boolean> }): void {
    selectedLegends.value = e.selected;
    const selectedItem = validItems.value[selectedIndex.value];

    if (!selectedItem || !selectedLegends.value[selectedItem.id]) {
        let newSelectedIndex = 0;

        for (const [item, index] of itemAndIndex(validItems.value)) {
            if (selectedLegends.value[item.id]) {
                newSelectedIndex = index;
                break;
            }
        }

        selectedIndex.value = newSelectedIndex;
    }
}
</script>

<style scoped>
.pie-chart-wrapper {
    position: relative;
    width: 100%;
}

.pie-chart-center {
    position: absolute;
    inset-block-start: calc(50% + 25px);
    inset-inline-start: 50%;
    transform: translate(-50%, -50%);
    max-width: 30%;
    text-align: center;
    pointer-events: none;
}

[dir="rtl"] .pie-chart-center {
    transform: translate(50%, -50%);
}

.pie-chart-center-title {
    color: var(--ebk-color-text-secondary);
}

.pie-chart-center-value {
    margin-block-start: var(--ebk-space-1);
    font-size: 1.25rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    overflow-wrap: anywhere;
}

@media (min-width: 600px) {
    .pie-chart-center {
        inset-block-start: 50%;
    }
}

.pie-chart-container {
    width: 100%;
    height: 460px;
}

@media (min-width: 600px) {
    .pie-chart-container {
        height: 660px;
    }
}

.pie-chart-container.transition-in {
    animation: pie-chart-skeleton-fade-in 2s 1;
}

@keyframes pie-chart-skeleton-fade-in {
    0% {
        opacity: 0;
    }
    20% {
        opacity: 0;
    }
    100% {
        opacity: 1;
    }
}
</style>

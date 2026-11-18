<template>
    <section class="overview-widget ebk-panel"
             :class="{ 'ebk-panel--bracketed': bracketed, 'overview-widget--busy': loading }"
             :aria-busy="loading ? 'true' : 'false'"
             :aria-labelledby="titleId">
        <overview-widget-header :id="titleId" :title="title" :meta="meta">
            <template #default v-if="hasDataTable || $slots['actions']">
                <!-- data table disclosure: the exact values of the chart (D7, #36) -->
                <v-btn density="comfortable" size="small" color="default" variant="text"
                       :aria-label="tt('Data Table')" :icon="true"
                       v-bind="dataTableButtonAttrs"
                       @click="toggleDataTable"
                       v-if="hasDataTable">
                    <v-icon :icon="dataTableExpanded ? mdiChartBoxOutline : mdiTableLarge" size="20" />
                    <v-tooltip activator="parent">{{ tt('Data Table') }}</v-tooltip>
                </v-btn>
                <slot name="actions" />
            </template>
        </overview-widget-header>
        <v-progress-linear class="overview-widget__busy-bar" color="primary" height="2" indeterminate
                           :aria-label="tt('Loading...')" v-if="loading && !showSkeletonOnly" />
        <!-- chart widgets: the canvas is exposed as one image named by the panel title and described by the footer summary (D7, #36) -->
        <div class="overview-widget__body" :class="bodyClass"
             :role="isChartImage ? 'img' : undefined"
             :aria-labelledby="isChartImage ? titleId : undefined"
             :aria-describedby="isChartImage && $slots['footer'] ? footerId : undefined"
             v-show="!showDataTable">
            <div class="overview-widget__state" v-if="showError">
                <state-panel type="error" compact :title="tt('An error occurred')">
                    <template #actions>
                        <v-btn size="small" variant="tonal" :disabled="loading" @click="retry">{{ tt('Refresh') }}</v-btn>
                    </template>
                </state-panel>
            </div>
            <div class="overview-widget__state" v-else-if="showEmpty">
                <state-panel type="empty" compact :title="emptyText || tt('No data')" />
            </div>
            <slot v-else />
        </div>
        <div class="overview-widget__body overview-widget__data" role="region" tabindex="0"
             :id="dataTableRegionId" :aria-labelledby="`${titleId} ${dataTableLabelId}`"
             v-show="showDataTable"
             v-if="hasDataTable">
            <span class="ebk-visually-hidden" :id="dataTableLabelId">{{ tt('Data Table') }}</span>
            <table class="overview-widget__data-table" :aria-labelledby="`${titleId} ${dataTableLabelId}`"
                   v-if="dataTableRendered && dataTable">
                <thead>
                    <tr>
                        <th scope="col" :key="column.key"
                            :class="{ 'overview-widget__data-numeric': column.numeric }"
                            v-for="column in dataTable.columns">{{ column.name }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr :key="row.key" v-for="row in dataTable.rows">
                        <th scope="row">{{ row.name }}</th>
                        <td class="overview-widget__data-numeric" :key="cellIdx" v-for="(cell, cellIdx) in row.cells">
                            <signed-amount :text="cell.text" :type="cell.type" :hide-sign="cell.hideSign"
                                           v-if="cell.type && cell.type !== 'none'" />
                            <span class="ebk-numeric" v-else>{{ cell.text }}</span>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
        <div class="overview-widget__footer" :id="footerId" v-if="$slots['footer'] && !showError && !showEmpty">
            <slot name="footer" />
        </div>
    </section>
</template>

<script setup lang="ts">
import OverviewWidgetHeader from './OverviewWidgetHeader.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed, useId } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { type ChartDataTable, useChartDataTableDisclosure } from '@/components/base/ChartDataTableBase.ts';

import { useOverviewWidgetState } from '../overviewWidgetState.ts';

import {
    mdiChartBoxOutline,
    mdiTableLarge
} from '@mdi/js';

/**
 * Shared frame of every desktop overview widget (design guidelines §4.1 / §4.6):
 * header strip with copper start tick, technical label, meta and actions, then the body.
 * Empty and error states use the shared StatePanel, loading keeps the content (or a skeleton
 * rendered by the widget) in place and shows a thin progress line under the header strip.
 * Chart widgets can pass a `dataTable` with the exact values of the chart (D7): a disclosure button in the
 * header strip swaps the chart for the table inside the same panel, so the dashboard layout does not change.
 */
const props = defineProps<{
    title: string;
    meta?: string;
    loading?: boolean;
    empty?: boolean;
    emptyText?: string;
    bracketed?: boolean;
    bodyClass?: string | Record<string, boolean> | string[];
    chart?: boolean;
    dataTable?: ChartDataTable | null;
}>();

const { tt } = useI18n();

const titleId = `overview-widget-${useId()}`;
const footerId = `${titleId}-footer`;
const dataTableLabelId = `${titleId}-data-label`;
const { loadFailed, retry } = useOverviewWidgetState();
const {
    regionId: dataTableRegionId,
    expanded: dataTableExpanded,
    rendered: dataTableRendered,
    buttonAttrs: dataTableButtonAttrs,
    toggle: toggleDataTable
} = useChartDataTableDisclosure();

const showError = computed<boolean>(() => !props.loading && !!props.empty && loadFailed.value);
const showEmpty = computed<boolean>(() => !props.loading && !!props.empty && !loadFailed.value);
const showSkeletonOnly = computed<boolean>(() => !!props.empty);
const isChartImage = computed<boolean>(() => !!props.chart && !props.empty);
const hasDataTable = computed<boolean>(() => !props.empty && !!props.dataTable && props.dataTable.rows.length > 0);
const showDataTable = computed<boolean>(() => hasDataTable.value && dataTableExpanded.value);
</script>

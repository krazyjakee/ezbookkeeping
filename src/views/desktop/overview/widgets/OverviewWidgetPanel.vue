<template>
    <section class="overview-widget ebk-panel"
             :class="{ 'ebk-panel--bracketed': bracketed, 'overview-widget--busy': loading }"
             :aria-busy="loading ? 'true' : 'false'"
             :aria-labelledby="titleId">
        <overview-widget-header :id="titleId" :title="title" :meta="meta">
            <slot name="actions" />
        </overview-widget-header>
        <v-progress-linear class="overview-widget__busy-bar" color="primary" height="2" indeterminate
                           :aria-label="tt('Loading...')" v-if="loading && !showSkeletonOnly" />
        <!-- chart widgets: the canvas is exposed as one image named by the panel title and described by the footer summary (D7, #36) -->
        <div class="overview-widget__body" :class="bodyClass"
             :role="isChartImage ? 'img' : undefined"
             :aria-labelledby="isChartImage ? titleId : undefined"
             :aria-describedby="isChartImage && $slots['footer'] ? footerId : undefined">
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
        <div class="overview-widget__footer" :id="footerId" v-if="$slots['footer'] && !showError && !showEmpty">
            <slot name="footer" />
        </div>
    </section>
</template>

<script setup lang="ts">
import OverviewWidgetHeader from './OverviewWidgetHeader.vue';
import StatePanel from '@/components/common/StatePanel.vue';

import { computed, useId } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useOverviewWidgetState } from '../overviewWidgetState.ts';

/**
 * Shared frame of every desktop overview widget (design guidelines §4.1 / §4.6):
 * header strip with copper start tick, technical label, meta and actions, then the body.
 * Empty and error states use the shared StatePanel, loading keeps the content (or a skeleton
 * rendered by the widget) in place and shows a thin progress line under the header strip.
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
}>();

const { tt } = useI18n();

const titleId = `overview-widget-${useId()}`;
const footerId = `${titleId}-footer`;
const { loadFailed, retry } = useOverviewWidgetState();

const showError = computed<boolean>(() => !props.loading && !!props.empty && loadFailed.value);
const showEmpty = computed<boolean>(() => !props.loading && !!props.empty && !loadFailed.value);
const showSkeletonOnly = computed<boolean>(() => !!props.empty);
const isChartImage = computed<boolean>(() => !!props.chart && !props.empty);
</script>

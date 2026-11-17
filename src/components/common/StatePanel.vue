<template>
    <div class="ebk-state-panel" :class="[`ebk-state-panel--${type ?? 'empty'}`, { 'ebk-state-panel--compact': compact }]"
         :role="type === 'error' ? 'alert' : 'status'">
        <div class="ebk-state-panel-icon" aria-hidden="true">
            <slot name="icon">
                <svg viewBox="0 0 24 24" focusable="false"><path :d="defaultIconPath" /></svg>
            </slot>
        </div>
        <div class="ebk-state-panel-body">
            <div class="ebk-state-panel-title" v-if="title">{{ title }}</div>
            <div class="ebk-state-panel-text" v-if="text">{{ text }}</div>
            <slot />
        </div>
        <div class="ebk-state-panel-actions" v-if="$slots['actions']">
            <slot name="actions" />
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

// The paths are copied from @mdi/js 7.4.47 (mdiAlertCircleOutline, mdiCloudOffOutline, mdiTrayRemove) instead of importing it:
// this component is in the shared "common" chunk, and the build groups @mdi/js into "vendor-desktop" (Vuetify, vue-echarts),
// so importing it here made the mobile entry download and evaluate Vuetify, its CSS and ECharts on every page load (#37).
const mdiAlertCircleOutline = 'M11,15H13V17H11V15M11,7H13V13H11V7M12,2C6.47,2 2,6.5 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20Z';
const mdiCloudOffOutline = 'M19.8 22.6L17.15 20H6.5Q4.2 20 2.6 18.4T1 14.5Q1 12.58 2.19 11.08 3.38 9.57 5.25 9.15 5.33 8.95 5.4 8.76 5.5 8.57 5.55 8.35L1.4 4.2L2.8 2.8L21.2 21.2M6.5 18H15.15L7.1 9.95Q7.05 10.23 7.03 10.5 7 10.73 7 11H6.5Q5.05 11 4.03 12.03 3 13.05 3 14.5 3 15.95 4.03 17 5.05 18 6.5 18M11.13 14M21.6 18.75L20.15 17.35Q20.58 17 20.79 16.54 21 16.08 21 15.5 21 14.45 20.27 13.73 19.55 13 18.5 13H17V11Q17 8.93 15.54 7.46 14.08 6 12 6 11.33 6 10.7 6.16 10.07 6.33 9.5 6.68L8.05 5.23Q8.93 4.63 9.91 4.31 10.9 4 12 4 14.93 4 16.96 6.04 19 8.07 19 11 20.73 11.2 21.86 12.5 23 13.78 23 15.5 23 16.5 22.63 17.31 22.25 18.15 21.6 18.75M14.83 12.03Z';
const mdiTrayRemove = 'M2 17A2 2 0 0 0 4 19H20A2 2 0 0 0 22 17V12H20V17H4V12H2M14.12 5.46L15.54 6.88L13.41 9L15.54 11.12L14.12 12.54L12 10.41L9.88 12.54L8.46 11.12L10.59 9L8.46 6.88L9.88 5.46L12 7.59Z';

/**
 * Presentational empty / error / offline state shared by desktop and mobile (#27).
 * Callers pass already translated texts, and put a retry or create button in the `actions` slot.
 */
const props = defineProps<{
    type?: 'empty' | 'error' | 'offline';
    title?: string;
    text?: string;
    compact?: boolean;
}>();

const defaultIconPath = computed<string>(() => {
    if (props.type === 'error') {
        return mdiAlertCircleOutline;
    } else if (props.type === 'offline') {
        return mdiCloudOffOutline;
    } else {
        return mdiTrayRemove;
    }
});
</script>

<style>
.ebk-state-panel {
    --ebk-state-panel-accent: var(--ebk-color-copper-line);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--ebk-space-3);
    padding: var(--ebk-space-6) var(--ebk-space-4);
    border: 1px dashed var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-panel);
    background-color: rgba(var(--ebk-color-surface-sunken-rgb), 0.5);
    color: var(--ebk-color-text-secondary);
    text-align: center;
    overflow-wrap: anywhere;
}

.ebk-state-panel--error {
    --ebk-state-panel-accent: var(--ebk-color-error);
    border-style: solid;
    border-color: color-mix(in srgb, var(--ebk-color-error) 35%, transparent);
    border-inline-start: 3px solid var(--ebk-color-error);
}

.ebk-state-panel--offline {
    --ebk-state-panel-accent: var(--ebk-color-warning);
}

.ebk-state-panel--compact {
    flex-direction: row;
    gap: var(--ebk-space-2);
    padding: var(--ebk-space-2) var(--ebk-space-3);
    text-align: start;
}

.ebk-state-panel-icon {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border: 1px solid var(--ebk-state-panel-accent);
    border-radius: var(--ebk-radius-pill);
    color: var(--ebk-state-panel-accent);
}

.ebk-state-panel--empty .ebk-state-panel-icon {
    color: var(--ebk-color-primary-text);
}

.ebk-state-panel--compact .ebk-state-panel-icon {
    width: 1.75rem;
    height: 1.75rem;
}

.ebk-state-panel-icon > svg {
    width: 60%;
    height: 60%;
    fill: currentColor;
}

.ebk-state-panel-body {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-1);
    min-width: 0;
}

.ebk-state-panel-title {
    color: var(--ebk-color-text-primary);
    font-size: 1em;
    font-weight: 600;
    line-height: 1.35;
}

.ebk-state-panel-text {
    font-size: 0.875em;
    line-height: 1.45;
}

.ebk-state-panel-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--ebk-space-2);
}

.ebk-state-panel--compact .ebk-state-panel-actions {
    margin-inline-start: auto;
}

@media (forced-colors: active) {
    .ebk-state-panel,
    .ebk-state-panel-icon {
        border-color: CanvasText;
    }
}
</style>

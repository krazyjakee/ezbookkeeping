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

import {
    mdiAlertCircleOutline,
    mdiCloudOffOutline,
    mdiTrayRemove
} from '@mdi/js';

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

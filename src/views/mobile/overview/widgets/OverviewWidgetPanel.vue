<template>
    <section class="ebk-panel mobile-overview-panel"
             :class="{ 'ebk-panel--bracketed': bracketed, 'mobile-overview-panel--custom-color': !!customBackgroundColor }"
             :style="panelStyle"
             :aria-busy="loading ? 'true' : 'false'"
             :aria-label="!hasHeader ? title : undefined">
        <header class="ebk-panel-header-strip mobile-overview-panel-header" v-if="hasHeader">
            <h2 class="mobile-overview-panel-title">{{ title }}</h2>
            <span class="mobile-overview-panel-meta ebk-numeric" v-if="meta">{{ meta }}</span>
            <div class="ebk-panel-header-actions mobile-overview-panel-actions" v-if="$slots['actions']">
                <slot name="actions" />
            </div>
        </header>
        <div class="mobile-overview-panel-body" :class="{ 'mobile-overview-panel-body--flush': flush }">
            <div class="mobile-overview-panel-state" v-if="!loading && empty">
                <state-panel type="empty" compact :title="emptyText || tt('No data')" />
            </div>
            <slot v-else />
        </div>
        <div class="mobile-overview-panel-footer" v-if="$slots['footer'] && !(!loading && empty)">
            <slot name="footer" />
        </div>
    </section>
</template>

<script setup lang="ts">
import StatePanel from '@/components/common/StatePanel.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

const props = withDefaults(defineProps<{
    title?: string;
    meta?: string;
    showTitle?: boolean;
    loading?: boolean;
    empty?: boolean;
    emptyText?: string;
    bracketed?: boolean;
    flush?: boolean;
    customBackgroundColor?: string;
    customTextColor?: string;
}>(), {
    showTitle: true
});

const { tt } = useI18n();

const hasHeader = computed<boolean>(() => props.showTitle && !!props.title);

const panelStyle = computed<Record<string, string>>(() => {
    if (!props.customBackgroundColor) {
        return {};
    }

    const style: Record<string, string> = {
        'background-color': props.customBackgroundColor
    };

    if (props.customTextColor) {
        style['color'] = props.customTextColor;
        style['--mobile-overview-panel-text-secondary'] = `color-mix(in srgb, ${props.customTextColor} 72%, transparent)`;
        style['--mobile-overview-panel-hairline'] = `color-mix(in srgb, ${props.customTextColor} 18%, transparent)`;
    }

    return style;
});
</script>

<style>
.mobile-overview-panel {
    --mobile-overview-panel-padding: var(--ebk-space-3);
    --mobile-overview-panel-text-secondary: var(--ebk-color-text-secondary);
    --mobile-overview-panel-hairline: var(--ebk-color-hairline);
    min-width: 0;
    font-size: 1rem;
    line-height: 1.4;
}

.mobile-overview-panel--custom-color {
    border-color: var(--mobile-overview-panel-hairline);
}

.mobile-overview-panel--custom-color .mobile-overview-panel-header {
    color: var(--mobile-overview-panel-text-secondary);
    border-block-end-color: var(--mobile-overview-panel-hairline);
}

.mobile-overview-panel-header {
    padding-inline: var(--ebk-space-3);
}

/* keep the label clear of the copper start tick */
.mobile-overview-panel:not(.ebk-panel--bracketed) > .mobile-overview-panel-header {
    padding-inline-start: var(--ebk-space-4);
}

/* hero panels carry the corner brackets instead of the start tick (guideline D3) */
.mobile-overview-panel.ebk-panel--bracketed > .mobile-overview-panel-header::before {
    display: none;
}

.mobile-overview-panel-title {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    font: inherit;
    letter-spacing: inherit;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.mobile-overview-panel-meta {
    flex: 0 1 auto;
    min-width: 0;
    margin-inline-start: auto;
    overflow: hidden;
    font-weight: 400;
    letter-spacing: normal;
    text-overflow: ellipsis;
    text-transform: none;
    white-space: nowrap;
}

.mobile-overview-panel-meta + .mobile-overview-panel-actions {
    margin-inline-start: 0 !important;
}

.mobile-overview-panel-actions {
    display: flex;
    align-items: center;
    margin-inline-end: calc(-1 * var(--ebk-space-2));
}

.mobile-overview-panel-actions .link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: var(--ebk-touch-target-min);
    min-height: var(--ebk-header-strip-height);
    color: inherit;
}

.mobile-overview-panel-actions .link .icon {
    font-size: 20px;
}

.mobile-overview-panel-body {
    padding: var(--mobile-overview-panel-padding);
}

.mobile-overview-panel-body--flush {
    padding: 0;
}

.mobile-overview-panel-body--flush > .mobile-overview-panel-state {
    padding: var(--mobile-overview-panel-padding);
}

.mobile-overview-panel-footer {
    padding: var(--ebk-space-2) var(--mobile-overview-panel-padding);
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.8125rem;
    line-height: 1.4;
}

/* shared content pieces used by the mobile overview widgets */
.mobile-overview-label {
    display: inline-flex;
    align-items: center;
    gap: var(--ebk-space-1);
    min-width: 0;
    color: var(--mobile-overview-panel-text-secondary);
}

.mobile-overview-label-arrow {
    font-weight: 600;
}

.mobile-overview-hero {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-1);
    min-width: 0;
}

.mobile-overview-hero-amount {
    font-size: 1.75rem;
    font-weight: 600;
    line-height: 1.2;
    font-variant-numeric: var(--ebk-font-numeric);
    overflow-wrap: anywhere;
}

.mobile-overview-pair {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin-block-start: var(--ebk-space-3);
    padding-block-start: var(--ebk-space-3);
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
}

.mobile-overview-pair > div {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-1);
    min-width: 0;
}

.mobile-overview-pair > div + div {
    padding-inline-start: var(--ebk-space-3);
    border-inline-start: 1px solid var(--mobile-overview-panel-hairline);
}

.mobile-overview-pair-amount {
    font-size: 1.125rem;
    font-weight: 600;
    line-height: 1.3;
    overflow-wrap: anywhere;
}

.mobile-overview-pair-amount .ebk-signed-amount {
    white-space: normal;
}

.mobile-overview-detail-row {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--ebk-space-2) var(--ebk-space-4);
    min-width: 0;
}

.mobile-overview-detail-row > :first-child {
    min-width: 0;
    color: var(--mobile-overview-panel-text-secondary);
}

.mobile-overview-detail-row > :last-child {
    flex: 0 0 auto;
    max-width: 60%;
    font-weight: 600;
    text-align: end;
    font-variant-numeric: var(--ebk-font-numeric);
    overflow-wrap: anywhere;
}

.mobile-overview-net-line {
    margin-block-start: var(--ebk-space-3);
    padding-block-start: var(--ebk-space-3);
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
}

/* rows (accounts, rankings, transactions) */
.mobile-overview-rows {
    display: flex;
    flex-direction: column;
}

.mobile-overview-row {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-3);
    min-height: 52px;
    padding: var(--ebk-space-2) var(--ebk-space-3);
    color: inherit;
    text-decoration: none;
}

.mobile-overview-row + .mobile-overview-row {
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
}

a.mobile-overview-row:active {
    background-color: var(--ebk-color-selected-background);
}

.mobile-overview-row-icon {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
}

.mobile-overview-row-icon .icon {
    font-size: 20px;
}

.mobile-overview-row-main {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
}

.mobile-overview-row-title {
    overflow: hidden;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.mobile-overview-row-meta {
    overflow: hidden;
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.8125rem;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.mobile-overview-row-end {
    display: flex;
    flex: 0 0 auto;
    flex-direction: column;
    align-items: flex-end;
    max-width: 50%;
    font-weight: 600;
    text-align: end;
    font-variant-numeric: var(--ebk-font-numeric);
}

.mobile-overview-row-end .ebk-signed-amount-value {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
}

.mobile-overview-row-chevron {
    flex: 0 0 auto;
    margin-inline-start: calc(-1 * var(--ebk-space-1));
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 14px !important;
    opacity: 0.7;
}

.mobile-overview-summary-text {
    padding: var(--ebk-space-2) var(--ebk-space-3);
    border-block-start: 1px solid var(--mobile-overview-panel-hairline);
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.8125rem;
}

.mobile-overview-skeleton-block {
    display: inline-block;
    border-radius: 4px;
}
</style>

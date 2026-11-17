<template>
    <div class="mobile-overview-dashboard">
        <state-panel class="mobile-overview-dashboard-error" :type="isOffline ? 'offline' : 'error'" compact
                     :title="isOffline ? tt('You are offline. Changes cannot be saved until the connection is restored.') : tt('Unable to retrieve transaction overview')"
                     v-if="loadFailed && !loading">
            <template #actions>
                <f7-button tonal small :text="tt('Retry')" @click="emit('retry')"></f7-button>
            </template>
        </state-panel>

        <div class="mobile-overview-dashboard-item ebk-animate-reveal" :style="getStaggerStyle(index)"
             :key="widget.id" v-for="(widget, index) in layout.widgets">
            <overview-widget :widget="widget" :loading="loading" @navigate="onNavigate" />
        </div>

        <state-panel type="empty" :title="tt('No widgets')" v-if="!layout.widgets.length" />
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';

import OverviewWidget from './OverviewWidget.vue';
import StatePanel from '@/components/common/StatePanel.vue';

import { useI18n } from '@/locales/helpers.ts';

import { type MobileOverviewLayout, MobileOverviewWidgetNavigationType } from '@/core/overview_layout.ts';

import { getStaggerStyle } from '@/lib/motion.ts';

defineProps<{
    layout: MobileOverviewLayout;
    loading: boolean;
    loadFailed?: boolean;
}>();

const emit = defineEmits<{
    (e: 'navigate', type: MobileOverviewWidgetNavigationType, path?: string): void;
    (e: 'retry'): void;
}>();

const { tt } = useI18n();

const isOffline = ref<boolean>(typeof navigator !== 'undefined' && navigator.onLine === false);

function updateOnlineState(): void {
    isOffline.value = navigator.onLine === false;
}

onMounted(() => {
    window.addEventListener('online', updateOnlineState);
    window.addEventListener('offline', updateOnlineState);
});

onBeforeUnmount(() => {
    window.removeEventListener('online', updateOnlineState);
    window.removeEventListener('offline', updateOnlineState);
});

function onNavigate(type: MobileOverviewWidgetNavigationType, path?: string): void {
    emit('navigate', type, path);
}
</script>

<style>
.mobile-overview-dashboard {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-4);
    padding-block: var(--ebk-space-3) var(--ebk-space-4);
    padding-inline: calc(var(--ebk-space-3) + var(--f7-safe-area-left)) calc(var(--ebk-space-3) + var(--f7-safe-area-right));
}

.mobile-overview-dashboard-item {
    min-width: 0;
}

.mobile-overview-dashboard-item:empty {
    display: none;
}

.mobile-overview-dashboard-item > .card,
.mobile-overview-dashboard-item > .list {
    margin-block-start: 0 !important;
    margin-block-end: 0 !important;
}
</style>

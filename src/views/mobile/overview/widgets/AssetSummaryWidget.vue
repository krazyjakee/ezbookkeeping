<template>
    <overview-widget-panel class="mobile-asset-summary"
                           :class="[`mobile-asset-summary--height-${height || 3}`, `mobile-asset-summary--${scene}`]"
                           :title="tt('Asset Summary')"
                           :loading="loading"
                           :custom-background-color="customBackgroundColor"
                           :custom-text-color="customTextColor">
        <template #actions v-if="scene === 'accountList'">
            <f7-link :aria-label="showAccountBalance ? tt('Hide Account Balance') : tt('Show Account Balance')"
                     @click="showAccountBalance = !showAccountBalance">
                <f7-icon :f7="showAccountBalance ? 'eye_slash' : 'eye'"></f7-icon>
            </f7-link>
        </template>

        <div class="mobile-overview-hero">
            <span class="mobile-overview-label ebk-technical-heading">{{ tt('Net assets') }}</span>
            <span class="mobile-overview-hero-amount ebk-numeric" :class="{ 'skeleton-text': loading }">{{ loading ? '0.00 GBP' : netAssets }}</span>
        </div>

        <div class="mobile-overview-pair">
            <div>
                <span class="mobile-overview-label ebk-technical-heading">{{ tt('Total assets') }}</span>
                <span class="mobile-overview-pair-amount ebk-numeric" :class="{ 'skeleton-text': loading }">{{ loading ? '0.00 GBP' : totalAssets }}</span>
            </div>
            <div>
                <span class="mobile-overview-label ebk-technical-heading">{{ tt('Total liabilities') }}</span>
                <span class="mobile-overview-pair-amount ebk-numeric" :class="{ 'skeleton-text': loading }">{{ loading ? '0.00 GBP' : totalLiabilities }}</span>
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAssetSummaryWidgetBase } from '@/views/base/overview/AssetSummaryWidgetBase.ts';

import { useEnvironmentsStore } from '@/stores/environment.ts';

import type { ColorValue } from '@/core/color.ts';
import {
    DEFAULT_MOBILE_OVERVIEW_WIDGET_LIGHT_BACKGROUND_COLOR,
    DEFAULT_MOBILE_OVERVIEW_WIDGET_DARK_BACKGROUND_COLOR
} from '@/consts/color.ts';

import { getDisplayColor, getContrastTextColor } from '@/lib/color.ts';

const props = defineProps<{
    loading: boolean;
    scene: 'overview' | 'accountList';
    height?: number;
    lightBackgroundColor?: ColorValue;
    darkBackgroundColor?: ColorValue;
}>();

const { tt } = useI18n();

const {
    showAccountBalance,
    netAssets,
    totalAssets,
    totalLiabilities
} = useAssetSummaryWidgetBase(props.scene);

const environmentsStore = useEnvironmentsStore();

const isDarkMode = computed<boolean>(() => environmentsStore.framework7DarkMode || false);
const customBackground = computed<ColorValue | undefined>(() => {
    if (isDarkMode.value) {
        return props.darkBackgroundColor && props.darkBackgroundColor !== DEFAULT_MOBILE_OVERVIEW_WIDGET_DARK_BACKGROUND_COLOR ? props.darkBackgroundColor : undefined;
    } else {
        return props.lightBackgroundColor && props.lightBackgroundColor !== DEFAULT_MOBILE_OVERVIEW_WIDGET_LIGHT_BACKGROUND_COLOR ? props.lightBackgroundColor : undefined;
    }
});
const customBackgroundColor = computed<string | undefined>(() => customBackground.value ? getDisplayColor(customBackground.value) : undefined);
const customTextColor = computed<string | undefined>(() => customBackground.value ? getDisplayColor(getContrastTextColor(customBackground.value)) : undefined);
</script>

<style>
.mobile-asset-summary--accountList {
    margin-inline: calc(var(--ebk-space-3) + var(--f7-safe-area-left)) calc(var(--ebk-space-3) + var(--f7-safe-area-right));
}

.mobile-asset-summary--height-1 {
    --mobile-overview-panel-padding: var(--ebk-space-2) var(--ebk-space-3);
}

.mobile-asset-summary--height-3 .mobile-overview-panel-body {
    padding-block: var(--ebk-space-4);
}
</style>

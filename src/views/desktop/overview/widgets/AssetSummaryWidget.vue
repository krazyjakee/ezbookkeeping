<template>
    <overview-widget-panel class="asset-summary-widget"
                           :title="title || tt('Asset Summary')"
                           :meta="hasData ? tt('format.misc.youHaveAccounts', { count: displayAccountCount }) : ''"
                           :loading="loading" :empty="!hasData">
        <div class="asset-summary-widget__metrics">
            <div class="asset-summary-widget__metric" :class="{ 'asset-summary-widget__metric--primary': item.primary }"
                 :key="item.title" v-for="item in summaryItems">
                <span class="overview-widget__label ebk-technical-heading">{{ tt(item.title) }}</span>
                <span class="overview-widget__kpi-amount ebk-numeric" :class="{ 'overview-widget__hero-amount': item.primary }"
                      v-if="hasData">{{ item.value }}</span>
                <v-skeleton-loader class="skeleton-no-margin overview-widget__skeleton-amount" type="text" width="140px" v-else />
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAssetSummaryWidgetBase } from '@/views/base/overview/AssetSummaryWidgetBase.ts';

interface SummaryItem {
    title: string;
    value: string;
    primary: boolean;
}

defineProps<{
    loading: boolean;
    title?: string;
}>();

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const {
    allAccounts,
    netAssets,
    totalAssets,
    totalLiabilities
} = useAssetSummaryWidgetBase('overview');

const hasData = computed<boolean>(() => !!allAccounts.value && allAccounts.value.length > 0);
const displayAccountCount = computed<string>(() => formatNumberToLocalizedNumerals(allAccounts.value?.length ?? 0));

const summaryItems = computed<SummaryItem[]>(() => [
    {
        title: 'Total assets',
        value: totalAssets.value,
        primary: false
    },
    {
        title: 'Total liabilities',
        value: totalLiabilities.value,
        primary: false
    },
    {
        title: 'Net assets',
        value: netAssets.value,
        primary: true
    }
]);
</script>

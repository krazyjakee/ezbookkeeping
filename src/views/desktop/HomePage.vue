<template>
    <main-page-layout no-navbar>
        <template #content>
            <div class="overview-currency-quick-switch-bar" v-if="showQuickSwitch">
                <currency-quick-switch :currencies="quickSwitchCurrencies"
                                       :selected-currency="selectedCurrency"
                                       :switching-currency="switchingCurrency"
                                       :disabled="loadingOverview"
                                       @switch="onSwitchCurrency" />
            </div>
            <overview-dashboard :layout="layout" :loading="loadingOverview" :load-failed="loadingFailed" @refresh="reload" />
        </template>
    </main-page-layout>
</template>

<script setup lang="ts">
import CurrencyQuickSwitch from '@/components/common/CurrencyQuickSwitch.vue';
import OverviewDashboard from './overview/OverviewDashboard.vue';

import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useCurrencyQuickSwitchBase } from '@/views/base/overview/CurrencyQuickSwitchBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useOverviewStore } from '@/stores/overview.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { PartialRecord } from '@/core/base.ts';
import { type DesktopOverviewLayout, OverviewWidgetDataRequirement } from '@/core/overview_layout.ts';
import { DESKTOP_OVERVIEW_WIDGET_DEFINITIONS, DEFAULT_DESKTOP_OVERVIEW_LAYOUT } from '@/consts/overview_layout.ts';

import { isUserLogined, isUserUnlocked } from '@/lib/userstate.ts';
import { getShareCacheImageBlob } from '@/lib/cache.ts';
import {
    getOverviewDataRequirements,
    getOverviewTransactionOverviewMonths,
    getOverviewUnreconciledTransactionAccountIds,
    getOverviewRecentTransactionsQueries,
    getOverviewAssetTrendMonths,
    getOverviewCalendarHeatmapMonths,
    getOverviewTransactionCategoryStatisticDateTypes,
    parseDesktopOverviewLayout
} from '@/lib/overview_layout.ts';
import logger from '@/lib/logger.ts';

const { getCurrencyName } = useI18n();
const { switchingCurrency, quickSwitchCurrencies, selectedCurrency, showQuickSwitch, switchCurrency } = useCurrencyQuickSwitchBase();

const settingsStore = useSettingsStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const overviewStore = useOverviewStore();
const desktopPageStore = useDesktopPageStore();

const loadingOverview = ref<boolean>(true);
const loadingFailed = ref<boolean>(false);

const layout = computed<DesktopOverviewLayout>(() => {
    try {
        return parseDesktopOverviewLayout(settingsStore.appSettings.desktopOverviewPageLayout);
    } catch (error) {
        logger.warn('failed to parse desktop overview page layout, fallback to default layout', error);
        return DEFAULT_DESKTOP_OVERVIEW_LAYOUT;
    }
});

function clearShareImageCache(): void {
    getShareCacheImageBlob().then(blob => {
        if (blob) {
            logger.warn('desktop version does not support receving shared image, the share image cache has been cleared');
        }
    });
}

function reload(force: boolean): void {
    loadingOverview.value = true;

    const requirements: PartialRecord<OverviewWidgetDataRequirement, boolean> = getOverviewDataRequirements(layout.value, DESKTOP_OVERVIEW_WIDGET_DEFINITIONS);
    const promises: Promise<unknown>[] = [
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false })
    ];

    if (requirements[OverviewWidgetDataRequirement.TransactionOverview]) {
        promises.push(overviewStore.loadTransactionOverview({
            force: force,
            months: getOverviewTransactionOverviewMonths(layout.value)
        }));
    }

    if (requirements[OverviewWidgetDataRequirement.AccountUnreconciledTransactionCounts]) {
        promises.push(overviewStore.loadTransactionUnreconciledCounts({
            force: force,
            accountIds: getOverviewUnreconciledTransactionAccountIds(layout.value)
        }));
    }

    if (requirements[OverviewWidgetDataRequirement.TransactionCategoryStatistics]) {
        for (const dateType of getOverviewTransactionCategoryStatisticDateTypes(layout.value)) {
            promises.push(overviewStore.loadTransactionCategoryStatistics({
                force: force,
                dateType: dateType
            }));
        }
    }

    if (requirements[OverviewWidgetDataRequirement.AssetTrends]) {
        promises.push(overviewStore.loadTransactionAssetTrends({
            force: force,
            months: getOverviewAssetTrendMonths(layout.value)
        }));
    }

    if (requirements[OverviewWidgetDataRequirement.RecentTransactions]) {
        promises.push(overviewStore.loadRecentTransactions({
            force: force,
            queries: getOverviewRecentTransactionsQueries(layout.value)
        }));
    }

    if (requirements[OverviewWidgetDataRequirement.CurrentMonthTransactions]) {
        promises.push(overviewStore.loadCurrentMonthTransactions({
            force: force
        }));
    }

    if (requirements[OverviewWidgetDataRequirement.DailyTransactionAmounts]) {
        promises.push(overviewStore.loadTransactionDailyAmounts({
            force: force,
            months: getOverviewCalendarHeatmapMonths(layout.value)
        }));
    }

    Promise.all(promises).then(() => {
        loadingOverview.value = false;
        loadingFailed.value = false;

        if (force) {
            desktopPageStore.addToast('Data has been updated');
        }
    }).catch(error => {
        loadingOverview.value = false;
        loadingFailed.value = !error.isUpToDate;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function onSwitchCurrency(currency: string): void {
    switchCurrency(currency).then(switched => {
        if (!switched) {
            return;
        }

        desktopPageStore.addToast('format.misc.defaultCurrencyChangedTo', { currency: `${getCurrencyName(currency)} (${currency})` });
        reload(false);
    }).catch(error => {
        // the default currency has not been changed, the selection has already gone back to the saved currency
        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

clearShareImageCache();

if (isUserLogined() && isUserUnlocked()) {
    reload(false);
}
</script>

<style>
.overview-currency-quick-switch-bar {
    display: flex;
    margin-block-end: var(--ebk-space-4);
}
</style>

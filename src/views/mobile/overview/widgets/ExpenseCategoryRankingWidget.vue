<template>
    <overview-widget-panel flush :title="title || tt('Expense Category Ranking')" :show-title="showTitle"
                           :meta="displayPeriodName" :loading="loading"
                           :empty="!loading && !rankingItems.length">
        <div class="mobile-overview-rows" v-if="loading && !rankingItems.length">
            <div class="mobile-overview-row mobile-ranking-row skeleton-text" :key="itemIdx" v-for="itemIdx in itemCount">
                <span class="mobile-overview-row-icon"><f7-icon f7="app_fill"></f7-icon></span>
                <span class="mobile-overview-row-main">
                    <span class="mobile-ranking-line">
                        <span class="mobile-overview-row-title">Category Name</span>
                        <span class="mobile-ranking-amount">0.00 GBP</span>
                    </span>
                    <span class="mobile-ranking-line">
                        <span class="mobile-ranking-bar"><span class="mobile-ranking-bar-fill" style="inline-size: 50%"></span></span>
                        <span class="mobile-ranking-percent">33.33%</span>
                    </span>
                </span>
            </div>
        </div>

        <template v-else-if="rankingItems.length">
            <div class="mobile-overview-rows">
                <a class="mobile-overview-row mobile-ranking-row" :href="getTransactionItemLinkUrl(item.id)"
                   :key="item.id" v-for="item in rankingItems">
                    <span class="mobile-overview-row-icon">
                        <ItemIcon :icon-type="getCategoryIconType(item.iconType)" :icon-id="item.icon" :color="item.color" v-if="item.icon" />
                        <f7-icon f7="pencil_ellipsis_rectangle" v-else></f7-icon>
                    </span>
                    <span class="mobile-overview-row-main">
                        <span class="mobile-ranking-line">
                            <span class="mobile-overview-row-title">{{ item.name }}</span>
                            <signed-amount class="mobile-ranking-amount" type="expense"
                                           :text="getDisplayAmount(item.value, rankingData.incomplete)"
                                           :hide-sign="!showAmountInHomePage || item.value.isZero()" />
                        </span>
                        <span class="mobile-ranking-line">
                            <span class="mobile-ranking-bar" aria-hidden="true">
                                <span class="mobile-ranking-bar-fill"
                                      :style="{ 'inline-size': `${Math.max(0, Math.min(100, item.percent))}%`, 'background-color': item.color ? getCategoryDisplayColor(item.color) : undefined }"></span>
                            </span>
                            <span class="mobile-ranking-percent ebk-numeric" v-if="item.percent >= 0 && item.value.isPositiveOrZero()">{{ formatPercentToLocalizedNumerals(item.percent, 2, '<0.01') }}</span>
                        </span>
                    </span>
                    <f7-icon class="mobile-overview-row-chevron icon-with-direction" f7="chevron_right" aria-hidden="true"></f7-icon>
                </a>
            </div>
        </template>

        <template #footer v-if="summaryText">{{ summaryText }}</template>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type CommonExpenseCategoryRankingWidgetProps,
    useExpenseCategoryRankingWidgetBase
} from '@/views/base/overview/ExpenseCategoryRankingWidgetBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';

import { DateRange } from '@/core/datetime.ts';

import { getCategoryIconType } from '@/lib/icon.ts';
import { getCategoryDisplayColor } from '@/lib/color.ts';

interface MobileExpenseCategoryRankingWidgetProps extends CommonExpenseCategoryRankingWidgetProps {
    showTitle: boolean;
}

const props = defineProps<MobileExpenseCategoryRankingWidgetProps>();

const { tt, formatPercentToLocalizedNumerals } = useI18n();
const settingsStore = useSettingsStore();

const {
    rankingData,
    rankingItems,
    getDisplayAmount,
    getTransactionItemLinkUrl
} = useExpenseCategoryRankingWidgetBase(props);

const showAmountInHomePage = computed<boolean>(() => settingsStore.appSettings.showAmountInHomePage);
const displayPeriodName = computed<string>(() => {
    const dateRange = DateRange.valueOf(props.dateType);
    return dateRange ? tt(dateRange.name) : '';
});

const summaryText = computed<string>(() => {
    const topItem = rankingItems.value[0];

    if (props.loading || !topItem || topItem.percent < 0 || !topItem.value.isPositive()) {
        return '';
    }

    return tt('format.misc.categoryAccountsForPercentOfExpense', {
        name: topItem.name,
        percent: formatPercentToLocalizedNumerals(topItem.percent, 2, '<0.01')
    });
});
</script>

<style>
.mobile-ranking-row {
    align-items: center;
}

.mobile-ranking-line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--ebk-space-2);
    min-width: 0;
}

.mobile-ranking-amount {
    flex: 0 0 auto;
    font-weight: 600;
}

.mobile-ranking-bar {
    position: relative;
    flex: 1 1 auto;
    height: 6px;
    overflow: hidden;
    border-radius: var(--ebk-radius-pill);
    background-color: var(--ebk-color-surface-sunken);
}

.mobile-ranking-bar-fill {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    border-radius: inherit;
    background-color: var(--ebk-color-primary);
}

.mobile-ranking-percent {
    flex: 0 0 auto;
    min-width: 3.75rem;
    color: var(--mobile-overview-panel-text-secondary);
    font-size: 0.75rem;
    text-align: end;
}

@media (forced-colors: active) {
    .mobile-ranking-bar {
        border: 1px solid CanvasText;
    }

    .mobile-ranking-bar-fill {
        background-color: CanvasText !important;
    }
}
</style>

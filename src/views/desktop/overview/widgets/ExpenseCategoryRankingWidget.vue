<template>
    <overview-widget-panel class="expense-ranking-widget"
                           :title="title || tt('Expense Category Ranking')"
                           :meta="getDisplayPeriodName()"
                           :loading="loading" :empty="!rankingItems.length">
        <ol class="expense-ranking-widget__list" v-if="rankingItems.length">
            <li class="expense-ranking-widget__item" :key="item.id" v-for="item in rankingItems">
                <router-link class="expense-ranking-widget__link" :to="getTransactionItemLinkUrl(item.id)">
                    <span class="overview-widget__item-icon">
                        <ItemIcon size="22px" :icon-type="getCategoryIconType(item.iconType)"
                                  :icon-id="item.icon" :color="item.color" />
                    </span>
                    <span class="expense-ranking-widget__main">
                        <span class="expense-ranking-widget__label-row">
                            <span class="expense-ranking-widget__name text-truncate" :title="item.name">{{ item.name }}</span>
                            <signed-amount class="overview-widget__row-amount" type="expense"
                                           :text="getDisplayAmount(item.value, rankingData.incomplete)"
                                           :hide-sign="!showAmountInHomePage || item.value.isZero()" />
                        </span>
                        <span class="expense-ranking-widget__bar-row">
                            <span class="expense-ranking-widget__bar" aria-hidden="true">
                                <span class="expense-ranking-widget__bar-value"
                                      :style="{ inlineSize: `${Math.max(0, Math.min(100, item.percent))}%`, backgroundColor: item.color ? getCategoryDisplayColor(item.color) : undefined }"></span>
                            </span>
                            <span class="expense-ranking-widget__percent ebk-numeric">{{ getDisplayPercent(item) }}</span>
                        </span>
                    </span>
                </router-link>
            </li>
        </ol>
        <div class="expense-ranking-widget__list" v-else-if="loading">
            <div class="expense-ranking-widget__item expense-ranking-widget__skeleton" :key="idx" v-for="idx in props.itemCount">
                <v-skeleton-loader class="skeleton-no-margin" type="avatar" width="32px" height="32px" />
                <div class="flex-grow-1">
                    <v-skeleton-loader class="skeleton-no-margin" type="text" />
                    <v-skeleton-loader class="skeleton-no-margin" type="text" width="70%" />
                </div>
            </div>
        </div>

        <template #footer v-if="summaryText">
            <span class="overview-widget__summary">{{ summaryText }}</span>
        </template>
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

import { getCategoryIconType } from '@/lib/icon.ts';
import { getCategoryDisplayColor } from '@/lib/color.ts';

const props = defineProps<CommonExpenseCategoryRankingWidgetProps>();

const { tt } = useI18n();
const {
    rankingData,
    rankingItems,
    showAmountInHomePage,
    getDisplayAmount,
    getDisplayPercent,
    getDisplayPeriodName,
    getSummaryText,
    getTransactionItemLinkUrl
} = useExpenseCategoryRankingWidgetBase(props);

const summaryText = computed<string>(() => getSummaryText());
</script>

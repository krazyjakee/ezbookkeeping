<template>
    <overview-widget-panel flush :title="title || tt('Account Balance List')" :show-title="showTitle"
                           :loading="loading" :empty="!loading && !displayAccounts.length">
        <div class="mobile-overview-rows" v-if="displayAccounts.length">
            <a class="mobile-overview-row" :key="account.id"
               :href="`/transaction/list?accountIds=${account.id}&dateType=${DateRange.All.type}`"
               v-for="account in displayAccounts">
                <span class="mobile-overview-row-icon">
                    <ItemIcon :icon-type="getAccountIconType(account.iconType)" :icon-id="account.icon" :color="account.color" />
                </span>
                <span class="mobile-overview-row-main">
                    <span class="mobile-overview-row-title">{{ account.name }}</span>
                </span>
                <span class="mobile-overview-row-end ebk-numeric">{{ accountBalanceOrAvailableCredit(account, undefined, showAvailableCreditForCreditCard, showAmount, accountIds) || '' }}</span>
                <f7-icon class="mobile-overview-row-chevron icon-with-direction" f7="chevron_right" aria-hidden="true"></f7-icon>
            </a>
        </div>
        <div class="mobile-overview-rows skeleton-text" v-else-if="loading">
            <div class="mobile-overview-row" :key="idx" v-for="idx in itemCount">
                <span class="mobile-overview-row-icon"><f7-icon f7="app_fill"></f7-icon></span>
                <span class="mobile-overview-row-main"><span class="mobile-overview-row-title">Account Name</span></span>
                <span class="mobile-overview-row-end">0.00 GBP</span>
            </div>
        </div>
    </overview-widget-panel>
</template>

<script setup lang="ts">
import OverviewWidgetPanel from './OverviewWidgetPanel.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAccountListPageBase } from '@/views/base/accounts/AccountListPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useAccountsStore } from '@/stores/account.ts';

import { DateRange } from '@/core/datetime.ts';
import type { Account } from '@/models/account.ts';

import { getAccountIconType } from '@/lib/icon.ts';

const props = defineProps<{
    loading: boolean;
    title?: string;
    showTitle: boolean;
    accountIds: string[];
    itemCount: number;
    sortBy: string;
    alwaysShowAmount: boolean;
    showAvailableCreditForCreditCard: boolean;
}>();

const { tt } = useI18n();

const { accountBalanceOrAvailableCredit } = useAccountListPageBase();

const settingsStore = useSettingsStore();
const accountsStore = useAccountsStore();

const showAmount = computed<boolean>(() => !!props.alwaysShowAmount || settingsStore.appSettings.showAmountInHomePage);
const displayAccounts = computed<Account[]>(() => accountsStore.getSortedAccounts(props.accountIds, props.sortBy, props.itemCount));
</script>

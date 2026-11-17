<template>
    <overview-widget-panel class="overview-widget--list"
                           :title="title || tt('Account Balance List')"
                           :loading="loading" :empty="!displayAccounts.length">
        <v-list class="overview-widget__list py-0" density="compact" v-if="displayAccounts.length">
            <v-list-item class="overview-widget__list-item" :key="account.id" :title="account.name"
                         :to="editing ? undefined : `/transaction/list?accountIds=${account.id}&dateType=${DateRange.All.type}`"
                         v-for="account in displayAccounts">
                <template #prepend>
                    <span class="overview-widget__item-icon">
                        <ItemIcon size="22px" :icon-type="getAccountIconType(account.iconType)" :icon-id="account.icon" :color="account.color" />
                    </span>
                </template>
                <template #append>
                    <span class="overview-widget__list-amount ebk-numeric">{{ accountBalanceOrAvailableCredit(account, undefined, showAvailableCreditForCreditCard, showAmount, accountIds) }}</span>
                </template>
            </v-list-item>
        </v-list>
        <div class="overview-widget__list" v-else-if="loading">
            <div class="overview-widget__list-skeleton" :key="idx" v-for="idx in props.itemCount">
                <v-skeleton-loader class="skeleton-no-margin" type="avatar" width="32px" height="32px" />
                <v-skeleton-loader class="skeleton-no-margin flex-grow-1" type="text" />
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
    accountIds: string[];
    itemCount: number;
    sortBy: string;
    alwaysShowAmount: boolean;
    showAvailableCreditForCreditCard: boolean;
    editing?: boolean
}>();

const { tt } = useI18n();

const { accountBalanceOrAvailableCredit } = useAccountListPageBase();

const settingsStore = useSettingsStore();
const accountsStore = useAccountsStore();

const showAmount = computed<boolean>(() => !!props.alwaysShowAmount || settingsStore.appSettings.showAmountInHomePage);
const displayAccounts = computed<Account[]>(() => accountsStore.getSortedAccounts(props.accountIds, props.sortBy, props.itemCount));
</script>

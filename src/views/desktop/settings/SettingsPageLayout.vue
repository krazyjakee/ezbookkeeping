<template>
    <main-page-layout nav-items-class="ebk-settings-nav" active-nav-key="settings"
                      :context-label="tt('Settings')"
                      :page-title="currentItem?.title"
                      :breadcrumbs="currentBreadcrumbs">
        <template #nav-items>
            <template :key="group.key" v-for="group in visibleGroups">
                <li class="nav-section-title">
                    <div class="title-wrapper">
                        <span class="title-text">{{ group.title }}</span>
                    </div>
                </li>
                <li class="nav-link" :key="item.path" v-for="item in group.items">
                    <router-link :to="item.path">
                        <v-icon class="nav-item-icon" :icon="item.icon"/>
                        <span class="nav-item-title">{{ item.title }}</span>
                    </router-link>
                </li>
            </template>
        </template>

        <template #content>
            <router-view v-slot="{ Component }">
                <transition name="ebk-route">
                    <component :is="Component" :key="currentRoutePath" />
                </transition>
            </router-view>
        </template>
    </main-page-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import {
    isUserCustomIconEnabled,
    isUserScheduledTransactionEnabled,
    isLanguagePreviewEnabled
} from '@/lib/server_settings.ts';

import {
    mdiAccountOutline,
    mdiShieldAccountOutline,
    mdiOnepassword,
    mdiRhombusSplitOutline,
    mdiPoundBoxOutline,
    mdiClipboardTextOutline,
    mdiClipboardTextClockOutline,
    mdiShapePlusOutline,
    mdiFileTableBoxOutline,
    mdiDatabaseCogOutline,
    mdiCogOutline,
    mdiLockOpenOutline,
    mdiChartPieOutline,
    mdiCloudOutline,
    mdiSwapHorizontal,
    mdiDatabaseClockOutline,
    mdiCodeBraces
} from '@mdi/js';

interface SettingsNavItem {
    path: string;
    icon: string;
    title: string;
    visible?: boolean;
}

interface SettingsNavGroup {
    key: string;
    title: string;
    items: SettingsNavItem[];
}

const route = useRoute();

const { tt } = useI18n();

const currentRoutePath = computed<string>(() => route.path);

const visibleGroups = computed<SettingsNavGroup[]>(() => {
    const groups: SettingsNavGroup[] = [
        {
            key: 'user',
            title: tt('User Settings'),
            items: [
                { path: '/settings/user/basic', icon: mdiAccountOutline, title: tt('User Profile') },
                { path: '/settings/user/security', icon: mdiShieldAccountOutline, title: tt('Security') },
                { path: '/settings/user/two_factor', icon: mdiOnepassword, title: tt('Two-Factor Authentication') }
            ]
        },
        {
            key: 'basis',
            title: tt('Basis Data'),
            items: [
                { path: '/category/list', icon: mdiRhombusSplitOutline, title: tt('Transaction Categories') },
                { path: '/tag/list', icon: mdiPoundBoxOutline, title: tt('Transaction Tags') },
                { path: '/template/list', icon: mdiClipboardTextOutline, title: tt('Transaction Templates') },
                { path: '/schedule/list', icon: mdiClipboardTextClockOutline, title: tt('Scheduled Transactions'), visible: isUserScheduledTransactionEnabled() },
                { path: '/custom_icon/list', icon: mdiShapePlusOutline, title: tt('Custom Icons'), visible: isUserCustomIconEnabled() },
                { path: '/presets/import/list', icon: mdiFileTableBoxOutline, title: tt('Import Presets') },
                { path: '/settings/user/data_management', icon: mdiDatabaseCogOutline, title: tt('Data Management') }
            ]
        },
        {
            key: 'application',
            title: tt('Application Settings'),
            items: [
                { path: '/settings/preferences', icon: mdiCogOutline, title: tt('Preferences') },
                { path: '/settings/application_lock', icon: mdiLockOpenOutline, title: tt('Application Lock') },
                { path: '/settings/statistics', icon: mdiChartPieOutline, title: tt('Statistics Settings') },
                { path: '/settings/cloud_sync', icon: mdiCloudOutline, title: tt('Settings Sync') },
                { path: '/exchange_rate', icon: mdiSwapHorizontal, title: tt('Exchange Rates Data') },
                { path: '/settings/browser_cache', icon: mdiDatabaseClockOutline, title: tt('Browser Cache Management') },
                { path: '/settings/developer_tools', icon: mdiCodeBraces, title: tt('Developer Tools'), visible: isLanguagePreviewEnabled() }
            ]
        }
    ];

    return groups.map(group => ({
        ...group,
        items: group.items.filter(item => item.visible !== false)
    }));
});

const currentGroup = computed<SettingsNavGroup | undefined>(() => visibleGroups.value.find(group => group.items.some(item => item.path === route.path)));
const currentItem = computed<SettingsNavItem | undefined>(() => currentGroup.value?.items.find(item => item.path === route.path));
const currentBreadcrumbs = computed<string[]>(() => currentGroup.value ? [tt('Settings'), currentGroup.value.title] : [tt('Settings')]);
</script>

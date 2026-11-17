<template>
    <v-row>
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loadingDataStatistics ? 'true' : 'false'">
                <header class="ebk-panel-header-strip ebk-mgmt-header">
                    <h2 class="ebk-mgmt-header-label">{{ tt('Data Management') }}</h2>
                    <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                        <v-btn density="comfortable" color="default" variant="text"
                               :aria-label="tt('Refresh')" :icon="true" :loading="loadingDataStatistics" @click="reloadUserDataStatistics(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="22" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </header>

                <div class="ebk-mgmt-section-body" v-if="loadDataStatisticsError && !loadingDataStatistics && !dataStatistics">
                    <state-panel type="error" :title="tt('Unable to retrieve user statistics data')">
                        <template #actions>
                            <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reloadUserDataStatistics(false)">{{ tt('Retry') }}</v-btn>
                        </template>
                    </state-panel>
                </div>

                <ul class="data-statistics-grid ebk-mgmt-section-body" v-else>
                    <li class="data-statistics-tile" :key="idx" v-for="(item, idx) in [
                        {
                            title: 'Transactions',
                            count: displayDataStatistics ? displayDataStatistics.totalTransactionCount : '-',
                            icon: mdiListBoxOutline
                        },
                        {
                            title: 'Transaction Pictures',
                            count: displayDataStatistics ? displayDataStatistics.totalTransactionPictureCount : '-',
                            icon: mdiImage
                        },
                        {
                            title: 'Accounts',
                            count: displayDataStatistics ? displayDataStatistics.totalAccountCount : '-',
                            icon: mdiCreditCardOutline
                        },
                        {
                            title: 'Explorations',
                            count: displayDataStatistics ? displayDataStatistics.totalExplorationCount : '-',
                            icon: mdiCompassOutline
                        },
                        {
                            title: 'Transaction Categories',
                            count: displayDataStatistics ? displayDataStatistics.totalTransactionCategoryCount : '-',
                            icon: mdiRhombusSplitOutline
                        },
                        {
                            title: 'Transaction Tags',
                            count: displayDataStatistics ? displayDataStatistics.totalTransactionTagCount : '-',
                            icon: mdiPoundBoxOutline
                        },
                        {
                            title: 'Transaction Templates',
                            count: displayDataStatistics ? displayDataStatistics.totalTransactionTemplateCount : '-',
                            icon: mdiClipboardTextOutline
                        },
                        {
                            title: 'Scheduled Transactions',
                            count: displayDataStatistics ? displayDataStatistics.totalScheduledTransactionCount : '-',
                            icon: mdiClipboardTextClockOutline
                        },
                        {
                            title: 'Custom Icons',
                            count: displayDataStatistics ? displayDataStatistics.totalCustomIconCount : '-',
                            icon: mdiShapePlusOutline
                        },
                        {
                            title: 'Import Presets',
                            count: displayDataStatistics ? displayDataStatistics.totalImportPresetCount : '-',
                            icon: mdiFileTableBoxOutline
                        }
                    ]">
                        <span class="ebk-mgmt-icon-tile" aria-hidden="true">
                            <v-icon size="18" :icon="item.icon"/>
                        </span>
                        <span class="data-statistics-text">
                            <span class="data-statistics-label">{{ tt(item.title) }}</span>
                            <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="48" :loading="true" v-if="loadingDataStatistics"></v-skeleton-loader>
                            <span class="data-statistics-count ebk-numeric" v-else>{{ item.count }}</span>
                        </span>
                    </li>
                </ul>
            </v-card>
        </v-col>

        <v-col cols="12" v-if="isDataExportingEnabled()">
            <v-card class="ebk-mgmt-panel" :aria-busy="exportingData ? 'true' : 'false'">
                <header class="ebk-panel-header-strip ebk-mgmt-header">
                    <h2 class="ebk-mgmt-header-label">{{ tt('Export Data') }}</h2>
                </header>

                <div class="ebk-mgmt-section-body">
                    <p class="ebk-mgmt-section-text">{{ tt('Export all transaction data to file.') }}&nbsp;{{ tt('It may take a long time, please wait for a few minutes.') }}</p>

                    <div class="d-flex flex-wrap align-center gap-3">
                        <v-btn color="default" variant="outlined" :prepend-icon="mdiTrayArrowUp" :append-icon="mdiMenuDown"
                               :loading="exportingData"
                               :disabled="loadingDataStatistics || exportingData || !dataStatistics || !dataStatistics.totalTransactionCount || dataStatistics.totalTransactionCount === '0'">
                            {{ tt('Export Data') }}
                            <v-menu activator="parent">
                                <v-list :disabled="loadingDataStatistics || exportingData || !dataStatistics || !dataStatistics.totalTransactionCount || dataStatistics.totalTransactionCount === '0'">
                                    <v-list-item @click="exportData('csv')">
                                        <v-list-item-title>{{ tt('CSV (Comma-separated values) File') }}</v-list-item-title>
                                    </v-list-item>
                                    <v-list-item @click="exportData('tsv')">
                                        <v-list-item-title>{{ tt('TSV (Tab-separated values) File') }}</v-list-item-title>
                                    </v-list-item>
                                </v-list>
                            </v-menu>
                        </v-btn>
                        <span class="data-export-meta" v-if="displayDataStatistics && !loadingDataStatistics">
                            <span class="ebk-technical-heading">{{ tt('Transactions') }}</span>
                            <span class="ebk-numeric ms-2">{{ displayDataStatistics.totalTransactionCount }}</span>
                        </span>
                    </div>
                </div>
            </v-card>
        </v-col>

        <v-col cols="12">
            <v-card class="ebk-mgmt-panel ebk-mgmt-danger" :aria-busy="clearingData ? 'true' : 'false'">
                <header class="ebk-panel-header-strip ebk-mgmt-header">
                    <h2 class="ebk-mgmt-header-label text-error">{{ tt('Danger Zone') }}</h2>
                </header>

                <div class="ebk-mgmt-section-body">
                    <v-alert type="error" variant="tonal" :icon="mdiAlert">
                        {{ tt('You CANNOT undo this action. "Clear All Transactions" will clear all your transactions data, and "Clear All Data" will clear your accounts, categories, tags and transactions data. Please enter your current password to confirm.') }}
                    </v-alert>

                    <v-row class="mt-4 mb-1">
                        <v-col cols="12" md="6">
                            <v-text-field
                                autocomplete="current-password"
                                ref="currentPasswordInput"
                                type="password"
                                persistent-placeholder
                                :label="tt('Current Password')"
                                :disabled="loadingDataStatistics || clearingData"
                                :placeholder="tt('Current Password')"
                                v-model="currentPasswordForClearData"
                            />
                        </v-col>
                    </v-row>

                    <v-btn color="error" variant="tonal" :prepend-icon="mdiDeleteSweepOutline" :append-icon="mdiMenuDown"
                           :loading="clearingData"
                           :disabled="loadingDataStatistics || !currentPasswordForClearData || clearingData">
                        {{ tt('Clear User Data') }}
                        <v-menu activator="parent">
                            <v-list :disabled="loadingDataStatistics || !currentPasswordForClearData || clearingData">
                                <v-list-item class="text-error" @click="clearAllTransactions">
                                    <v-list-item-title>{{ tt('Clear All Transactions') }}</v-list-item-title>
                                </v-list-item>
                                <v-list-item class="text-error" @click="clearAllData">
                                    <v-list-item-title>{{ tt('Clear All Data') }}</v-list-item-title>
                                </v-list-item>
                            </v-list>
                        </v-menu>
                    </v-btn>
                </div>
            </v-card>
        </v-col>
    </v-row>

    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useDataManagementPageBase } from '@/views/base/users/DataManagementPageBase.ts';

import { useRootStore } from '@/stores/index.ts';
import { useUserStore } from '@/stores/user.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { isEquals } from '@/lib/common.ts';
import { isDataExportingEnabled } from '@/lib/server_settings.ts';
import { startDownloadFile } from '@/lib/ui/common.ts';

import {
    mdiRefresh,
    mdiListBoxOutline,
    mdiCreditCardOutline,
    mdiImage,
    mdiCompassOutline,
    mdiRhombusSplitOutline,
    mdiPoundBoxOutline,
    mdiClipboardTextOutline,
    mdiClipboardTextClockOutline,
    mdiShapePlusOutline,
    mdiFileTableBoxOutline,
    mdiAlert,
    mdiTrayArrowUp,
    mdiMenuDown,
    mdiDeleteSweepOutline
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;

const { tt } = useI18n();
const { dataStatistics, displayDataStatistics, getExportFileName } = useDataManagementPageBase();

const rootStore = useRootStore();
const userStore = useUserStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');

const loadingDataStatistics = ref<boolean>(true);
const loadDataStatisticsError = ref<boolean>(false);
const exportingData = ref<boolean>(false);
const currentPasswordForClearData = ref<string>('');
const clearingData = ref<boolean>(false);

function reloadUserDataStatistics(force: boolean): void {
    loadingDataStatistics.value = true;

    userStore.getUserDataStatistics().then(dataStatisticsResponse => {
        if (force) {
            if (isEquals(dataStatistics.value, dataStatisticsResponse)) {
                desktopPageStore.addToast('Data is up to date');
            } else {
                desktopPageStore.addToast('Data has been updated');
            }
        }

        dataStatistics.value = dataStatisticsResponse;
        loadingDataStatistics.value = false;
        loadDataStatisticsError.value = false;
    }).catch(error => {
        loadingDataStatistics.value = false;
        loadDataStatisticsError.value = true;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function exportData(fileType: string): void {
    if (exportingData.value) {
        return;
    }

    exportingData.value = true;

    userStore.getExportedUserData(fileType).then(data => {
        startDownloadFile(getExportFileName(fileType), data);
        exportingData.value = false;
    }).catch(error => {
        exportingData.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function clearAllTransactions(): void {
    if (!currentPasswordForClearData.value) {
        desktopPageStore.addToast('Current password cannot be blank');
        return;
    }

    if (clearingData.value) {
        return;
    }

    confirmDialog.value?.open('Are you sure you want to clear all transactions?', { color: 'error' }).then(() => {
        clearingData.value = true;

        rootStore.clearAllUserTransactions({
            password: currentPasswordForClearData.value
        }).then(() => {
            clearingData.value = false;
            currentPasswordForClearData.value = '';

            desktopPageStore.addToast('All transactions has been cleared');
            reloadUserDataStatistics(false);
        }).catch(error => {
            clearingData.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function clearAllData(): void {
    if (!currentPasswordForClearData.value) {
        desktopPageStore.addToast('Current password cannot be blank');
        return;
    }

    if (clearingData.value) {
        return;
    }

    confirmDialog.value?.open('Are you sure you want to clear all data?', { color: 'error' }).then(() => {
        clearingData.value = true;

        rootStore.clearAllUserData({
            password: currentPasswordForClearData.value
        }).then(() => {
            clearingData.value = false;
            currentPasswordForClearData.value = '';

            desktopPageStore.addToast('All user data has been cleared');
            reloadUserDataStatistics(false);
        }).catch(error => {
            clearingData.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

reloadUserDataStatistics(false);
</script>

<style>
.data-statistics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: var(--ebk-space-3);
    margin: 0;
    list-style: none;
}

.data-statistics-tile {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-3);
    min-width: 0;
    padding: var(--ebk-space-3);
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
}

.data-statistics-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
}

.data-statistics-label {
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
    line-height: 1.2;
    overflow-wrap: anywhere;
}

.data-statistics-count {
    color: var(--ebk-color-text-primary);
    font-size: 1.25rem;
    font-weight: 600;
    line-height: 1.5;
}

.data-export-meta {
    color: var(--ebk-color-text-secondary);
}
</style>

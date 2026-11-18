<template>
    <v-row>
        <v-col cols="12">
            <v-card class="ebk-settings-panel">
                <header class="ebk-panel-header-strip ebk-settings-panel-header">
                    <h2 class="ebk-settings-panel-title">{{ tt('Settings Sync') }}</h2>
                    <div class="ebk-panel-header-actions">
                        <v-progress-circular indeterminate size="18" v-if="loading"></v-progress-circular>
                        <span class="ebk-settings-status" :class="{ 'ebk-settings-status--on': isEnableCloudSync }" v-if="!loading">
                            <v-icon size="16" :icon="isEnableCloudSync ? mdiCloudCheckOutline : mdiCloudOffOutline" />
                            <span>{{ isEnableCloudSync ? tt('Enabled') : tt('Disabled') }}</span>
                        </span>
                    </div>
                </header>

                <v-card-text class="mt-2">
                    <v-skeleton-loader class="skeleton-no-margin pt-1 pb-2" type="text" style="width: 150px" :loading="true" v-if="loading"></v-skeleton-loader>
                    <span class="text-body-large" v-if="!loading && !isEnableCloudSync">
                        {{ tt('Settings sync is not enabled') }}
                    </span>
                    <span class="text-body-large" v-if="!loading && isEnableCloudSync">
                        {{ tt('Settings sync has been enabled') }}
                    </span>
                </v-card-text>

                <v-card-text class="pt-0">
                    <v-expansion-panels class="synchronized-settings" multiple
                                        :readonly="true" :hide-actions="true"
                                        :disabled="loading || enabling || disabling"
                                        v-model="openedPanel">
                        <v-expansion-panel class="border" value="synchronizedSettings">
                            <!-- the panel is always open and read-only: a static header (not a button), so the More menu button is not nested in another button -->
                            <div class="v-expansion-panel-title v-expansion-panel-title--active expand-panel-title-with-bg py-0">
                                <div class="d-flex align-center justify-center w-100">
                                    <div class="w-100">
                                        <span role="heading" aria-level="3">{{ tt('Synchronized Settings') }}</span>
                                    </div>
                                    <v-btn density="comfortable" color="default" variant="text" class="ms-2"
                                           :aria-label="tt('More')" :disabled="loading || enabling || disabling" :icon="true">
                                        <v-icon :icon="mdiDotsVertical" />
                                        <v-menu activator="parent">
                                            <v-list>
                                                <v-list-item :disable="loading || enabling || disabling"
                                                             :prepend-icon="mdiSelectAll"
                                                             :title="tt('Select All')"
                                                             @click="selectAllSettings"></v-list-item>
                                                <v-list-item :disable="loading || enabling || disabling"
                                                             :prepend-icon="mdiSelect"
                                                             :title="tt('Select None')"
                                                             @click="selectNoneSettings"></v-list-item>
                                                <v-list-item :disable="loading || enabling || disabling"
                                                             :prepend-icon="mdiSelectInverse"
                                                             :title="tt('Invert Selection')"
                                                             @click="selectInvertSettings"></v-list-item>
                                            </v-list>
                                        </v-menu>
                                    </v-btn>
                                </div>
                            </div>
                            <v-expansion-panel-text>
                                <v-list rounded density="comfortable" class="pa-0">
                                    <template :key="categorizedItems.categoryName"
                                              v-for="(categorizedItems, categoryIdx) in ALL_APPLICATION_CLOUD_SETTINGS">
                                        <v-divider role="presentation" v-if="categoryIdx > 0"/>

                                        <v-list-item>
                                            <template #prepend>
                                                <v-checkbox :disabled="loading || enabling || disabling"
                                                            :model-value="isAllSettingsSelected(categorizedItems)"
                                                            :indeterminate="hasSettingSelectedButNotAllChecked(categorizedItems)"
                                                            @update:model-value="updateSettingsSelected(categorizedItems, !!$event)">
                                                    <template #label>
                                                        <span class="ms-1">{{ tt(categorizedItems.categoryName) }}</span>
                                                        <span class="mx-2" v-if="categorizedItems.categorySubName">/</span>
                                                        <span v-if="categorizedItems.categorySubName">{{ tt(categorizedItems.categorySubName) }}</span>
                                                    </template>
                                                </v-checkbox>
                                            </template>
                                        </v-list-item>

                                        <v-divider role="presentation"/>

                                        <div role="listitem">
                                        <v-list rounded density="comfortable" class="pa-0 ms-4">
                                            <template :key="settingItem.settingKey"
                                                      v-for="(settingItem, itemIdx) in categorizedItems.items">
                                                <v-divider role="presentation" v-if="itemIdx > 0"/>

                                                <v-list-item>
                                                    <template #prepend>
                                                        <v-checkbox :disabled="loading || enabling || disabling"
                                                                    :model-value="enabledApplicationCloudSettings[settingItem.settingKey]"
                                                                    @update:model-value="updateSettingSelected(settingItem, !!$event)">
                                                            <template #label>
                                                                <span class="ms-1">{{ tt(settingItem.settingName) }}</span>
                                                                <v-icon class="ms-2 me-0" start size="16" :icon="mdiCellphone" v-if="settingItem.mobile"/>
                                                                <v-icon class="ms-2 me-0" start size="16" :icon="mdiMonitor" v-if="settingItem.desktop"/>
                                                            </template>
                                                        </v-checkbox>
                                                    </template>
                                                </v-list-item>
                                            </template>
                                        </v-list>
                                        </div>
                                    </template>
                                </v-list>
                            </v-expansion-panel-text>
                        </v-expansion-panel>
                    </v-expansion-panels>
                </v-card-text>

                <v-card-text class="ebk-settings-panel-footer">
                    <v-btn color="primary" :loading="enabling" :disabled="loading || enabling || disabling || !hasEnabledApplicationCloudSettings" v-if="!isEnableCloudSync" @click="enable(false)">
                        {{ tt('Enable Settings Sync') }}
                    </v-btn>
                    <v-btn color="primary" :loading="enabling" :disabled="loading || enabling || disabling || !hasEnabledApplicationCloudSettings" v-if="isEnableCloudSync" @click="enable(true)">
                        {{ tt('Update Synchronized Settings') }}
                    </v-btn>
                    <v-btn color="error" variant="tonal" :loading="disabling" :disabled="loading || enabling || disabling" v-if="isEnableCloudSync" @click="disable">
                        {{ tt('Disable Settings Sync') }}
                    </v-btn>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>
</template>

<script setup lang="ts">
import { ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAppCloudSyncBase } from '@/views/base/settings/AppCloudSyncPageBase.ts';

import { useUserStore } from '@/stores/user.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import {
    mdiDotsVertical,
    mdiSelectAll,
    mdiSelect,
    mdiSelectInverse,
    mdiCellphone,
    mdiMonitor,
    mdiCloudCheckOutline,
    mdiCloudOffOutline
} from '@mdi/js';

const { tt } = useI18n();
const {
    ALL_APPLICATION_CLOUD_SETTINGS,
    loading,
    enabling,
    disabling,
    enabledApplicationCloudSettings,
    isEnableCloudSync,
    hasEnabledApplicationCloudSettings,
    enabledApplicationCloudSettingKeys,
    isAllSettingsSelected,
    hasSettingSelectedButNotAllChecked,
    updateSettingsSelected,
    updateSettingSelected,
    selectAllSettings,
    selectNoneSettings,
    selectInvertSettings,
    setUserApplicationCloudSettings
} = useAppCloudSyncBase();

const userStore = useUserStore();
const desktopPageStore = useDesktopPageStore();

const openedPanel = ref<string[]>(['synchronizedSettings']);

function init(): void {
    loading.value = true;

    userStore.getUserApplicationCloudSettings().then(response => {
        setUserApplicationCloudSettings(response);
        loading.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function enable(update: boolean): void {
    enabling.value = true;

    userStore.fullUpdateUserApplicationCloudSettings(enabledApplicationCloudSettingKeys.value).then(() => {
        enabling.value = false;

        if (!update) {
            desktopPageStore.addToast('Settings sync has been enabled');
        } else {
            desktopPageStore.addToast('Synchronized settings have been updated');
        }
    }).catch(error => {
        enabling.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function disable(): void {
    disabling.value = true;

    userStore.disableUserApplicationCloudSettings().then(() => {
        enabledApplicationCloudSettings.value = {};
        disabling.value = false;
        desktopPageStore.addToast('Settings sync has been disabled');
    }).catch(error => {
        disabling.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

init();
</script>

<style>
.synchronized-settings .v-expansion-panel-title {
    cursor: inherit;
}

.synchronized-settings .v-expansion-panel-text__wrapper {
    padding: 0 0 0 0;
}
</style>

<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loading ? 'true' : 'false'">
                <header class="ebk-panel-header-strip ebk-mgmt-header">
                    <h2 class="ebk-mgmt-header-label">{{ tt('Import Presets') }}</h2>
                    <span class="ebk-mgmt-header-meta ebk-numeric" v-if="!loading && presets.length > 0">{{ formatNumberToLocalizedNumerals(presets.length) }}</span>
                    <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                        <v-btn color="primary" variant="tonal"
                               :disabled="loading || updating" @click="saveSortResult"
                               v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                        <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                               :disabled="loading || updating" @click="add">{{ tt('Add') }}</v-btn>
                        <v-btn density="comfortable" color="default" variant="text"
                               :aria-label="tt('Refresh')" :icon="true" :disabled="loading || updating"
                               :loading="loading" @click="reload">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="22" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </header>

                <v-table class="ebk-mgmt-table import-presets-table" :hover="!loading">
                    <thead>
                    <tr>
                        <th class="ebk-mgmt-column-name">{{ tt('Preset Name') }}</th>
                        <th class="ebk-mgmt-column-operation">
                            <span class="ebk-visually-hidden">{{ tt('Operation') }}</span>
                        </th>
                    </tr>
                    </thead>

                    <tbody aria-hidden="true" v-if="loading && !presets.length">
                    <tr class="ebk-mgmt-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                        <td class="ebk-mgmt-column-name">
                            <div class="ebk-mgmt-row-content">
                                <span class="ebk-mgmt-drag disabled"></span>
                                <span class="ebk-mgmt-icon-tile ebk-mgmt-icon-tile--skeleton"></span>
                                <v-skeleton-loader class="ebk-mgmt-skeleton ms-3" type="text" width="180" :loading="true"></v-skeleton-loader>
                            </div>
                        </td>
                        <td></td>
                    </tr>
                    </tbody>

                    <tbody v-if="!loading && !presets.length">
                    <tr class="no-hover">
                        <td class="ebk-mgmt-column-state" colspan="2">
                            <state-panel type="error" :title="tt('Unable to retrieve import preset list')" v-if="loadError">
                                <template #actions>
                                    <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload">{{ tt('Retry') }}</v-btn>
                                </template>
                            </state-panel>
                            <state-panel type="empty" :title="tt('No available import presets')" v-else>
                                <template #actions>
                                    <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus" :disabled="updating" @click="add">{{ tt('Add') }}</v-btn>
                                </template>
                            </state-panel>
                        </td>
                    </tr>
                    </tbody>

                    <draggable-list tag="tbody"
                                    item-key="id"
                                    handle=".drag-handle"
                                    ghost-class="dragging-item"
                                    :disabled="!presets.length || loading || updating"
                                    v-model="presets"
                                    @change="onMove">
                        <template #item="{ element, index }">
                            <tr class="ebk-mgmt-row">
                                <td class="ebk-mgmt-column-name">
                                    <div class="ebk-mgmt-row-content">
                                        <span class="ebk-mgmt-drag" :class="{ 'drag-handle': !loading && !updating && presets.length > 1, 'disabled': loading || updating || presets.length <= 1 }">
                                            <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                            <v-tooltip activator="parent" v-if="!loading && !updating && presets.length > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                        </span>
                                        <span class="ebk-mgmt-icon-tile" aria-hidden="true">
                                            <v-icon size="18" :icon="mdiFileTableBoxOutline"/>
                                        </span>
                                        <div class="ebk-mgmt-name">
                                            <span class="ebk-mgmt-name-text">{{ element.name }}</span>
                                        </div>
                                    </div>
                                </td>
                                <td class="ebk-mgmt-column-operation">
                                    <div class="ebk-mgmt-actions">
                                        <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Edit')" :disabled="loading || updating"
                                               @click="edit(element)">
                                            <v-icon :icon="mdiPencilOutline" size="20" />
                                            <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                        </v-btn>
                                        <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Delete')"
                                               :loading="presetRemoving[element.id]"
                                               :disabled="loading || updating"
                                               @click="remove(element)">
                                            <template #loader>
                                                <v-progress-circular indeterminate size="20" width="2"/>
                                            </template>
                                            <v-icon :icon="mdiDeleteOutline" size="20" />
                                            <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                        </v-btn>
                                        <span class="ebk-mgmt-actions-divider" aria-hidden="true"></span>
                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Move Up')" :disabled="loading || updating || index <= 0"
                                               @click="movePreset(element, index, -1)">
                                            <v-icon :icon="mdiArrowUp" size="18" />
                                            <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                        </v-btn>
                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Move Down')" :disabled="loading || updating || index >= presets.length - 1"
                                               @click="movePreset(element, index, 1)">
                                            <v-icon :icon="mdiArrowDown" size="18" />
                                            <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                        </v-btn>
                                    </div>
                                </td>
                            </tr>
                        </template>
                    </draggable-list>
                </v-table>
            </v-card>
        </v-col>
    </v-row>

    <import-preset-edit-dialog ref="editDialog" />
    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import ImportPresetEditDialog from './dialogs/ImportPresetEditDialog.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useImportPresetsStore } from '@/stores/importPreset.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { type ImportPresetBasicInfo } from '@/models/import_preset.ts';

import {
    mdiRefresh,
    mdiPencilOutline,
    mdiDeleteOutline,
    mdiDrag,
    mdiFileTableBoxOutline,
    mdiPlus,
    mdiArrowUp,
    mdiArrowDown
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type EditDialogType = InstanceType<typeof ImportPresetEditDialog>;

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const importPresetsStore = useImportPresetsStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const editDialog = useTemplateRef<EditDialogType>('editDialog');

const loading = ref<boolean>(true);
const updating = ref<boolean>(false);
const loadError = ref<boolean>(false);
const presetRemoving = ref<Record<string, boolean>>({});
const displayOrderModified = ref<boolean>(false);

const presets = computed<ImportPresetBasicInfo[]>(() => importPresetsStore.allImportPresetBasicInfos);

function init(): void {
    loading.value = true;

    importPresetsStore.loadAllPresetBasicInfos({
        force: false
    }).then(() => {
        loading.value = false;
        loadError.value = false;
    }).catch(error => {
        loading.value = false;
        loadError.value = presets.value.length < 1;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function reload(): void {
    loading.value = true;

    importPresetsStore.loadAllPresetBasicInfos({
        force: true
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        desktopPageStore.addToast('Import preset list has been updated');
    }).catch(error => {
        loading.value = false;

        if (error.isUpToDate) {
            loadError.value = false;
            displayOrderModified.value = false;
        } else if (presets.value.length < 1) {
            loadError.value = true;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function add(): void {
    editDialog.value?.open({}).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function edit(preset: ImportPresetBasicInfo): void {
    editDialog.value?.open({
        id: preset.id
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(preset: ImportPresetBasicInfo): void {
    confirmDialog.value?.open('Are you sure you want to delete this import preset?', { color: 'error' }).then(() => {
        updating.value = true;
        presetRemoving.value[preset.id] = true;

        importPresetsStore.deletePreset({
            preset: preset
        }).then(() => {
            updating.value = false;
            presetRemoving.value[preset.id] = false;
        }).catch(error => {
            updating.value = false;
            presetRemoving.value[preset.id] = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function saveSortResult(): void {
    if (!displayOrderModified.value) {
        return;
    }

    loading.value = true;

    importPresetsStore.updatePresetDisplayOrders().then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

// non-drag alternative to the drag handle (guidelines §6.4), uses the same display order update as dragging
function movePreset(preset: ImportPresetBasicInfo, index: number, direction: number): void {
    const to = index + direction;

    if (to < 0 || to >= presets.value.length) {
        return;
    }

    importPresetsStore.changePresetDisplayOrder({
        presetId: preset.id,
        from: index,
        to: to
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function onMove(event: { moved: { element: { id: string }, oldIndex: number, newIndex: number } }): void {
    if (!event || !event.moved) {
        return;
    }

    const moveEvent = event.moved;

    if (!moveEvent.element || !moveEvent.element.id) {
        desktopPageStore.addToast('Unable to move import preset');
        return;
    }

    importPresetsStore.changePresetDisplayOrder({
        presetId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

init();
</script>

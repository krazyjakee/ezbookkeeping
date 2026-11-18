<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loading ? 'true' : 'false'">
                <header class="ebk-panel-header-strip ebk-mgmt-header">
                    <h2 class="ebk-mgmt-header-label">{{ tt('Custom Icons') }}</h2>
                    <span class="ebk-mgmt-header-meta ebk-numeric" v-if="!loading && allCustomIcons.length > 0">{{ formatNumberToLocalizedNumerals(allCustomIcons.length) }}</span>
                    <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                        <v-btn color="primary" variant="tonal"
                               :disabled="loading || updating" @click="saveSortResult"
                               v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                        <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                               :disabled="loading || updating" @click="add">{{ tt('Add') }}</v-btn>
                        <v-btn density="comfortable" color="default" variant="text"
                               :aria-label="tt('Refresh')" :icon="true" :disabled="loading || updating"
                               :loading="loading" @click="reload(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="22" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </header>

                <div class="custom-icon-body" v-if="allCustomIcons.length < 1">
                    <div class="custom-icon-grid" aria-hidden="true" v-if="loading">
                        <div class="custom-icon-card custom-icon-card--skeleton" :key="idx" v-for="idx in [ 1, 2, 3, 4 ]">
                            <div class="custom-icon-preview">
                                <v-skeleton-loader class="skeleton-no-margin" type="avatar" :loading="true"></v-skeleton-loader>
                            </div>
                            <div class="custom-icon-toolbar"></div>
                        </div>
                    </div>

                    <state-panel type="error" :title="tt('Unable to retrieve custom icon list')" v-else-if="loadError">
                        <template #actions>
                            <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload(true)">{{ tt('Retry') }}</v-btn>
                        </template>
                    </state-panel>
                    <state-panel type="empty" :title="tt('No available custom icons')" v-else>
                        <template #actions>
                            <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus" :disabled="updating" @click="add">{{ tt('Add') }}</v-btn>
                        </template>
                    </state-panel>
                </div>

                <div class="custom-icon-body" v-else-if="allCustomIcons.length > 0">
                    <draggable-list class="custom-icon-grid"
                                    item-key="id"
                                    handle=".drag-handle"
                                    ghost-class="dragging-item"
                                    :disabled="loading || updating"
                                    v-model="allCustomIcons"
                                    @change="onMove">
                        <template #item="{ element, index }">
                            <div class="custom-icon-card ebk-mgmt-card">
                                <div class="custom-icon-preview">
                                    <v-img class="custom-icon-thumbnail" aspect-ratio="1" cover :src="getCustomIconUrl(element)" />
                                </div>
                                <div class="custom-icon-toolbar">
                                    <span class="ebk-mgmt-drag me-0" :class="{ 'drag-handle': !loading && !updating && allCustomIcons.length > 1, 'disabled': loading || updating || allCustomIcons.length <= 1 }">
                                        <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="18"/>
                                        <v-tooltip activator="parent" v-if="!loading && !updating && allCustomIcons.length > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                    </span>
                                    <v-btn class="ebk-mgmt-reveal button-icon-with-direction" color="default" density="compact" size="small" variant="text" :icon="true"
                                           :aria-label="tt('Move Up')" :disabled="loading || updating || index <= 0"
                                           @click="moveCustomIcon(element, index, -1)">
                                        <v-icon :icon="mdiChevronLeft" size="18" />
                                        <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn class="ebk-mgmt-reveal button-icon-with-direction" color="default" density="compact" size="small" variant="text" :icon="true"
                                           :aria-label="tt('Move Down')" :disabled="loading || updating || index >= allCustomIcons.length - 1"
                                           @click="moveCustomIcon(element, index, 1)">
                                        <v-icon :icon="mdiChevronRight" size="18" />
                                        <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                    </v-btn>
                                    <v-spacer/>
                                    <v-btn color="error" density="compact" size="small" variant="text" :icon="true"
                                           :aria-label="tt('Delete')" :disabled="updating"
                                           @click="deleteCustomIcon(element)">
                                        <v-icon :icon="mdiDeleteOutline" size="18" v-if="deletingCustomIconId !== element.id" />
                                        <v-progress-circular indeterminate size="16" width="2" v-else-if="deletingCustomIconId === element.id" />
                                        <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                    </v-btn>
                                </div>
                            </div>
                        </template>
                    </draggable-list>
                </div>
            </v-card>
        </v-col>
    </v-row>

    <upload-dialog ref="uploadDialog" />

    <confirm-dialog ref="confirmDialog" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import UploadDialog from '@/views/desktop/customicons/dialogs/UploadDialog.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserCustomIconsStore } from '@/stores/userCustomIcon.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { UserCustomIconInfoResponse } from '@/models/user_custom_icon.ts';

import {
    mdiRefresh,
    mdiDrag,
    mdiDeleteOutline,
    mdiPlus,
    mdiChevronLeft,
    mdiChevronRight
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type UploadDialogType = InstanceType<typeof UploadDialog>;

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const customIconsStore = useUserCustomIconsStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const uploadDialog = useTemplateRef<UploadDialogType>('uploadDialog');

const loading = ref<boolean>(true);
const updating = ref<boolean>(false);
const loadError = ref<boolean>(false);
const displayOrderModified = ref<boolean>(false);
const deletingCustomIconId = ref<string>('');

const allCustomIcons = computed<UserCustomIconInfoResponse[]>(() => customIconsStore.allCustomIcons);

function getCustomIconUrl(customIcon: UserCustomIconInfoResponse): string | undefined {
    return customIconsStore.getUserCustomIconUrl(customIcon);
}

function reload(force: boolean): void {
    loading.value = true;

    customIconsStore.loadAllCustomIcons({
        force: force
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        if (force) {
            desktopPageStore.addToast('Custom icon list has been updated');
        }
    }).catch(error => {
        loading.value = false;
        loadError.value = !(error && error.isUpToDate) && allCustomIcons.value.length < 1;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function add(): void {
    uploadDialog.value?.open().then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function deleteCustomIcon(customIcon: UserCustomIconInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to delete this custom icon?', { color: 'error' }).then(() => {
        updating.value = true;
        deletingCustomIconId.value = customIcon.id;

        customIconsStore.deleteCustomIcon({
            customIcon: customIcon
        }).then(() => {
            updating.value = false;
            deletingCustomIconId.value = '';
        }).catch(error => {
            updating.value = false;
            deletingCustomIconId.value = '';

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

    customIconsStore.updateCustomIconDisplayOrders().then(() => {
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
function moveCustomIcon(customIcon: UserCustomIconInfoResponse, index: number, direction: number): void {
    const to = index + direction;

    if (to < 0 || to >= allCustomIcons.value.length) {
        return;
    }

    customIconsStore.changeCustomIconDisplayOrder({
        iconId: customIcon.id,
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
        desktopPageStore.addToast('Unable to move custom icon');
        return;
    }

    customIconsStore.changeCustomIconDisplayOrder({
        iconId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

reload(false);
</script>

<style>
.custom-icon-body {
    padding: var(--ebk-space-4);
}

.custom-icon-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
    gap: var(--ebk-space-3);
}

.custom-icon-card {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface);

    &.dragging-item {
        opacity: 0.6;
    }
}

.custom-icon-card--skeleton {
    border-style: dashed;
}

.custom-icon-preview {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--ebk-space-3);
    background-color: var(--ebk-color-surface-sunken);
}

.custom-icon-thumbnail {
    width: 100%;
    max-width: 64px;
    border-radius: var(--ebk-radius-sm);
}

.custom-icon-toolbar {
    display: flex;
    align-items: center;
    min-height: 32px;
    padding: 2px var(--ebk-space-1);
    border-block-start: 1px solid var(--ebk-color-hairline);
}
</style>

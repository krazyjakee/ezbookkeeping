<template>
    <v-dialog width="800" :persistent="loading || updating || displayOrderModified" v-model="showState">
        <one-column-dialog-layout content-class="pa-0" :disabled="loading || updating"
                                  :title="tt('Change Group Display Order')" :cancel-button-title="tt('Close')"
                                  @cancel="close">
            <template #after-title>
                <v-btn density="compact" color="default" variant="text" class="ms-2"
                       :aria-label="tt('Refresh')" :icon="true" :disabled="loading || updating"
                       :loading="loading" @click="reload">
                    <template #loader>
                        <v-progress-circular indeterminate size="20"/>
                    </template>
                    <v-icon :icon="mdiRefresh" size="22" />
                    <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                </v-btn>
                <v-btn density="compact" color="primary" variant="text" class="ms-1" :icon="true"
                       :aria-label="tt('Save Display Order')" :disabled="loading || updating || !displayOrderModified" @click="saveDisplayOrder">
                    <v-icon :icon="mdiCheck" size="22" />
                    <v-tooltip activator="parent">{{ tt('Save Display Order') }}</v-tooltip>
                </v-btn>
            </template>

            <template #content>
                <v-table hover density="comfortable" class="w-100 ebk-mgmt-table">
                    <tbody v-if="loading && (!allTagGroups || allTagGroups.length < 1)">
                    <tr :key="itemIdx" v-for="itemIdx in [ 1, 2, 3, 4, 5, 6 ]">
                        <td class="px-0">
                            <v-skeleton-loader type="text" :loading="true"></v-skeleton-loader>
                        </td>
                    </tr>
                    </tbody>

                    <tbody v-if="!loading && (!allTagGroups || allTagGroups.length < 1)">
                    <tr>
                        <td>{{ tt('No available tag group') }}</td>
                    </tr>
                    </tbody>

                    <draggable-list tag="tbody"
                                    item-key="id"
                                    handle=".drag-handle"
                                    ghost-class="dragging-item"
                                    v-model="allTagGroups"
                                    @change="onMove">
                        <template #item="{ element, index }">
                            <tr class="ebk-mgmt-row">
                                <td class="ebk-mgmt-column-name">
                                    <div class="ebk-mgmt-row-content">
                                        <span class="ebk-mgmt-drag" :class="{ 'drag-handle': !loading && !updating && allTagGroups && allTagGroups.length > 1, 'disabled': loading || updating || !allTagGroups || allTagGroups.length <= 1 }">
                                            <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                            <v-tooltip activator="parent" v-if="!loading && !updating && allTagGroups && allTagGroups.length > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                        </span>
                                        <span class="ebk-mgmt-name-text">{{ element.name }}</span>
                                    </div>
                                </td>
                                <td class="ebk-mgmt-column-operation">
                                    <div class="ebk-mgmt-actions">
                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Move Up')" :disabled="loading || updating || index <= 0"
                                               @click="moveTagGroup(element, index, -1)">
                                            <v-icon :icon="mdiArrowUp" size="18" />
                                            <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                        </v-btn>
                                        <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="tt('Move Down')" :disabled="loading || updating || index >= allTagGroups.length - 1"
                                               @click="moveTagGroup(element, index, 1)">
                                            <v-icon :icon="mdiArrowDown" size="18" />
                                            <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                        </v-btn>
                                    </div>
                                </td>
                            </tr>
                        </template>
                    </draggable-list>
                </v-table>
            </template>
        </one-column-dialog-layout>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { type TransactionTagGroup } from '@/models/transaction_tag_group.ts';

import {
    mdiRefresh,
    mdiCheck,
    mdiDrag,
    mdiArrowUp,
    mdiArrowDown
} from '@mdi/js';

const { tt } = useI18n();

const transactionTagsStore = useTransactionTagsStore();
const desktopPageStore = useDesktopPageStore();

let resolveFunc: (() => void) | null = null;

const showState = ref<boolean>(false);
const loading = ref<boolean>(true);
const updating = ref<boolean>(false);
const displayOrderModified = ref<boolean>(false);

const allTagGroups = computed<TransactionTagGroup[]>(() => transactionTagsStore.allTransactionTagGroups);

function open(): Promise<void> {
    showState.value = true;
    loading.value = true;

    transactionTagsStore.loadAllTagGroups({
        force: false
    }).then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });

    return new Promise<void>((resolve) => {
        resolveFunc = resolve;
    });
}

function reload(): void {
    loading.value = true;

    transactionTagsStore.loadAllTagGroups({
        force: true
    }).then(() => {
        loading.value = false;
        displayOrderModified.value = false;

        desktopPageStore.addToast('Tag group list has been updated');
    }).catch(error => {
        loading.value = false;

        if (error && error.isUpToDate) {
            displayOrderModified.value = false;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function saveDisplayOrder(): void {
    if (!displayOrderModified.value) {
        return;
    }

    loading.value = true;

    transactionTagsStore.updateTagGroupDisplayOrders().then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function close(): void {
    if (loading.value || updating.value) {
        return;
    }

    resolveFunc?.();
    showState.value = false;
}

// non-drag alternative to the drag handle (guidelines §6.4), uses the same display order update as dragging
function moveTagGroup(tagGroup: { id: string }, index: number, direction: number): void {
    const to = index + direction;

    if (!allTagGroups.value || to < 0 || to >= allTagGroups.value.length) {
        return;
    }

    transactionTagsStore.changeTagGroupDisplayOrder({
        tagGroupId: tagGroup.id,
        from: index,
        to: to
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function onMove(event: { moved: { element: { id: string }; oldIndex: number; newIndex: number } }): void {
    if (!event || !event.moved) {
        return;
    }

    const moveEvent = event.moved;

    if (!moveEvent.element || !moveEvent.element.id) {
        desktopPageStore.addToast('Unable to move tag group');
        return;
    }

    transactionTagsStore.changeTagGroupDisplayOrder({
        tagGroupId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

defineExpose({
    open
});
</script>

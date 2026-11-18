<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loading ? 'true' : 'false'">
                <v-layout class="page-with-navigation-drawer">
                    <v-navigation-drawer class="ebk-mgmt-subnav scrollable-tabs-navigation-drawer" :permanent="alwaysShowNav" v-model="showNav">
                        <div class="ebk-mgmt-subnav-section">
                            <div class="ebk-mgmt-kv">
                                <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Total tags') }}</span>
                                <span class="ebk-mgmt-kv-value" v-if="!loading || totalAvailableTagsCount > 0">{{ displayTotalAvailableTagsCount }}</span>
                                <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="60" :loading="true" v-else></v-skeleton-loader>
                            </div>
                        </div>
                        <v-tabs show-arrows
                                class="ebk-mgmt-subnav-tabs scrollable-vertical-tabs my-2"
                                style="max-height: calc(100% - 90px)"
                                direction="vertical"
                                :prev-icon="mdiMenuUp" :next-icon="mdiMenuDown"
                                :disabled="loading || updating" v-model="activeTagGroupId">
                            <v-tab class="tab-text-truncate" :disabled="loading || updating || displayOrderModified || hasEditingTag"
                                   :key="tagGroup.id" :value="tagGroup.id"
                                   v-for="tagGroup in allTagGroupsWithDefault"
                                   @click="switchTagGroup(tagGroup.id)">
                                <span class="ebk-mgmt-subnav-tab-content">
                                    <span class="text-truncate">{{ tagGroup.name }}</span>
                                    <span class="ebk-mgmt-subnav-tab-count" v-if="getTagGroupTagCount(tagGroup.id) > 0">{{ formatNumberToLocalizedNumerals(getTagGroupTagCount(tagGroup.id)) }}</span>
                                </span>
                            </v-tab>
                            <template v-if="loading && (!allTagGroupsWithDefault || allTagGroupsWithDefault.length < 2)">
                                <v-skeleton-loader class="skeleton-no-margin mx-4 mt-h1 py-3" type="text"
                                                   :key="itemIdx" :loading="true" v-for="itemIdx in [ 1, 2, 3, 4, 5 ]"></v-skeleton-loader>
                            </template>
                        </v-tabs>
                    </v-navigation-drawer>
                    <v-main>
                        <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                            <v-window-item value="tagListPage">
                                <v-card variant="flat" class="tag-list-content" min-height="780">
                                    <header class="ebk-panel-header-strip ebk-mgmt-header">
                                        <v-btn class="ebk-mgmt-header-menu-toggle d-lg-none" density="comfortable" color="default" variant="text"
                                               :aria-label="tt('Open Menu')" :icon="true" @click="showNav = !showNav">
                                            <v-icon :icon="mdiMenu" size="22" />
                                        </v-btn>
                                        <h2 class="ebk-mgmt-header-label">{{ activeTagGroupName || tt('Transaction Tags') }}</h2>
                                        <span class="ebk-mgmt-header-meta ebk-numeric" v-if="!loading && availableTagCount > 0">{{ formatNumberToLocalizedNumerals(availableTagCount) }}</span>
                                        <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                                            <v-btn color="primary" variant="tonal"
                                                   :disabled="loading || updating || hasEditingTag" @click="saveSortResult"
                                                   v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                                            <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                                                   :disabled="loading || updating || hasEditingTag" @click="add">{{ tt('Add') }}</v-btn>
                                            <v-btn density="comfortable" color="default" variant="text"
                                                   :aria-label="tt('Refresh')" :icon="true" :disabled="loading || updating || hasEditingTag"
                                                   :loading="loading" @click="reload">
                                                <template #loader>
                                                    <v-progress-circular indeterminate size="20"/>
                                                </template>
                                                <v-icon :icon="mdiRefresh" size="22" />
                                                <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                                            </v-btn>
                                            <v-btn density="comfortable" color="default" variant="text"
                                                   :aria-label="tt('More')" :disabled="loading || updating || hasEditingTag" :icon="true">
                                                <v-icon :icon="mdiDotsVertical" />
                                                <v-tooltip activator="parent">{{ tt('More') }}</v-tooltip>
                                                <v-menu activator="parent">
                                                    <v-list>
                                                        <v-list-item :prepend-icon="mdiPlus" @click="addTagGroup">
                                                            <v-list-item-title>{{ tt('Add Tag Group') }}</v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item :prepend-icon="mdiPencilOutline"
                                                                     @click="renameTagGroup"
                                                                     v-if="activeTagGroupId && activeTagGroupId !== DEFAULT_TAG_GROUP_ID">
                                                            <v-list-item-title>{{ tt('Rename Tag Group') }}</v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item class="text-error" :prepend-icon="mdiDeleteOutline"
                                                                     :disabled="tags && tags.length > 0"
                                                                     @click="removeTagGroup"
                                                                     v-if="activeTagGroupId && activeTagGroupId !== DEFAULT_TAG_GROUP_ID">
                                                            <v-list-item-title>{{ tt('Delete Tag Group') }}</v-list-item-title>
                                                        </v-list-item>
                                                        <v-divider class="my-2" v-if="allTagGroupsWithDefault.length >= 2"/>
                                                        <v-list-item :prepend-icon="mdiSort"
                                                                     :disabled="!allTagGroupsWithDefault || allTagGroupsWithDefault.length < 2"
                                                                     :title="tt('Change Group Display Order')"
                                                                     v-if="allTagGroupsWithDefault.length >= 2"
                                                                     @click="showChangeGroupDisplayOrderDialog"></v-list-item>
                                                        <v-divider class="my-2"/>
                                                        <v-list-item :prepend-icon="mdiSortAlphabeticalAscending"
                                                                     :disabled="!tags || tags.length < 2"
                                                                     :title="tt('Sort by Name (A to Z)')"
                                                                     @click="sortByName(false)"></v-list-item>
                                                        <v-list-item :prepend-icon="mdiSortAlphabeticalDescending"
                                                                     :disabled="!tags || tags.length < 2"
                                                                     :title="tt('Sort by Name (Z to A)')"
                                                                     @click="sortByName(true)"></v-list-item>
                                                        <v-divider class="my-2"/>
                                                        <v-list-item :prepend-icon="mdiEyeOutline"
                                                                     :title="tt('Show Hidden Transaction Tags')"
                                                                     v-if="!showHidden" @click="showHidden = true"></v-list-item>
                                                        <v-list-item :prepend-icon="mdiEyeOffOutline"
                                                                     :title="tt('Hide Hidden Transaction Tags')"
                                                                     v-if="showHidden" @click="showHidden = false"></v-list-item>
                                                    </v-list>
                                                </v-menu>
                                            </v-btn>
                                        </div>
                                    </header>

                                    <v-table class="ebk-mgmt-table transaction-tags-table" density="default" :hover="!loading">
                                        <thead>
                                        <tr>
                                            <th class="ebk-mgmt-column-name">{{ tt('Tag Title') }}</th>
                                            <th class="ebk-mgmt-column-operation">
                                                <span class="ebk-visually-hidden">{{ tt('Operation') }}</span>
                                            </th>
                                        </tr>
                                        </thead>

                                        <tbody aria-hidden="true" v-if="loading && noAvailableTag && !newTag">
                                        <tr class="ebk-mgmt-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3, 4, 5 ]">
                                            <td class="ebk-mgmt-column-name">
                                                <div class="ebk-mgmt-row-content">
                                                    <span class="ebk-mgmt-drag disabled"></span>
                                                    <span class="ebk-mgmt-icon-tile ebk-mgmt-icon-tile--skeleton"></span>
                                                    <v-skeleton-loader class="ebk-mgmt-skeleton ms-3" type="text" width="160" :loading="true"></v-skeleton-loader>
                                                </div>
                                            </td>
                                            <td></td>
                                        </tr>
                                        </tbody>

                                        <tbody v-if="!loading && noAvailableTag && !newTag">
                                        <tr class="no-hover">
                                            <td class="ebk-mgmt-column-state" colspan="2">
                                                <state-panel type="error" :title="tt('Unable to retrieve tag list')" v-if="loadError">
                                                    <template #actions>
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload">{{ tt('Retry') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                                <state-panel type="empty" :title="tt('No available tag')" v-else>
                                                    <template #actions>
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                                                               :disabled="updating || hasEditingTag" @click="add">{{ tt('Add') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                            </td>
                                        </tr>
                                        </tbody>

                                        <draggable-list tag="tbody"
                                                        item-key="id"
                                                        handle=".drag-handle"
                                                        ghost-class="dragging-item"
                                                        :disabled="noAvailableTag || loading || updating || hasEditingTag"
                                                        v-model="tags"
                                                        @change="onMove">
                                            <template #item="{ element }">
                                                <tr class="ebk-mgmt-row transaction-tags-table-row-tag"
                                                    :class="{ 'ebk-mgmt-row--hidden': element.hidden && editingTag.id !== element.id }"
                                                    v-if="showHidden || !element.hidden">
                                                    <td class="ebk-mgmt-column-name">
                                                        <div class="ebk-mgmt-row-content">
                                                            <span class="ebk-mgmt-drag" :class="{ 'drag-handle': !loading && !updating && !hasEditingTag && availableTagCount > 1, 'disabled': loading || updating || hasEditingTag || availableTagCount <= 1 }">
                                                                <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                                                <v-tooltip activator="parent" v-if="!loading && !updating && !hasEditingTag && availableTagCount > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                                            </span>
                                                            <span class="ebk-mgmt-icon-tile ebk-mgmt-dimmable" aria-hidden="true">
                                                                <v-icon size="18" :icon="mdiPound"/>
                                                            </span>
                                                            <div class="ebk-mgmt-name" v-if="editingTag.id !== element.id">
                                                                <div class="ebk-mgmt-name-line">
                                                                    <span class="ebk-mgmt-name-text ebk-mgmt-dimmable">{{ element.name }}</span>
                                                                    <span class="ebk-mgmt-hidden-chip" v-if="element.hidden">
                                                                        <v-icon :icon="mdiEyeOffOutline" size="12" />
                                                                        {{ tt('Hidden') }}
                                                                    </span>
                                                                </div>
                                                            </div>
                                                            <v-text-field class="tag-list-name-field ms-3" type="text" autocomplete="off"
                                                                          density="compact" hide-details
                                                                          :disabled="loading || updating"
                                                                          :aria-label="tt('Tag Title')"
                                                                          :placeholder="tt('Tag Title')"
                                                                          v-model="editingTag.name"
                                                                          v-else-if="editingTag.id === element.id"
                                                                          @keyup.enter="save(editingTag)"
                                                                          @keyup.esc="cancelSave(editingTag)" />
                                                        </div>
                                                    </td>
                                                    <td class="ebk-mgmt-column-operation">
                                                        <div class="ebk-mgmt-actions" v-if="editingTag.id !== element.id">
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="element.hidden ? tt('Show') : tt('Hide')"
                                                                   :loading="tagHiding[element.id]"
                                                                   :disabled="loading || updating || hasEditingTag"
                                                                   @click="hide(element, !element.hidden)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                <v-icon :icon="element.hidden ? mdiEyeOutline : mdiEyeOffOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ element.hidden ? tt('Show') : tt('Hide') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Move to...')"
                                                                   :loading="tagMoving[element.id]"
                                                                   :disabled="loading || updating || hasEditingTag || allTagGroupsWithDefault.length < 2"
                                                                   @click="showMoveTagDialog(element)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                <v-icon :icon="mdiFolderMoveOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Move to...') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Edit')"
                                                                   :loading="tagUpdating[element.id]"
                                                                   :disabled="loading || updating || hasEditingTag"
                                                                   @click="edit(element)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                <v-icon :icon="mdiPencilOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Delete')"
                                                                   :loading="tagRemoving[element.id]"
                                                                   :disabled="loading || updating || hasEditingTag"
                                                                   @click="remove(element)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                <v-icon :icon="mdiDeleteOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                                            </v-btn>
                                                            <span class="ebk-mgmt-actions-divider" aria-hidden="true"></span>
                                                            <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Move Up')"
                                                                   :disabled="loading || updating || hasEditingTag || !canMoveTag(element, -1)"
                                                                   @click="moveTag(element, -1)">
                                                                <v-icon :icon="mdiArrowUp" size="18" />
                                                                <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Move Down')"
                                                                   :disabled="loading || updating || hasEditingTag || !canMoveTag(element, 1)"
                                                                   @click="moveTag(element, 1)">
                                                                <v-icon :icon="mdiArrowDown" size="18" />
                                                                <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                                            </v-btn>
                                                        </div>
                                                        <div class="ebk-mgmt-actions" v-else>
                                                            <v-btn color="primary" density="comfortable" variant="tonal"
                                                                   :prepend-icon="mdiCheck"
                                                                   :loading="tagUpdating[element.id]"
                                                                   :disabled="loading || updating || !isTagModified(element)"
                                                                   @click="save(editingTag)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                {{ tt('Save') }}
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text"
                                                                   :prepend-icon="mdiClose"
                                                                   :disabled="loading || updating"
                                                                   @click="cancelSave(editingTag)">
                                                                {{ tt('Cancel') }}
                                                            </v-btn>
                                                        </div>
                                                    </td>
                                                </tr>
                                            </template>
                                        </draggable-list>

                                        <tbody ref="newTagRow" v-if="newTag">
                                        <tr class="ebk-mgmt-row no-hover tag-list-new-row">
                                            <td class="ebk-mgmt-column-name">
                                                <div class="ebk-mgmt-row-content">
                                                    <span class="ebk-mgmt-drag disabled" aria-hidden="true"></span>
                                                    <span class="ebk-mgmt-icon-tile" aria-hidden="true">
                                                        <v-icon size="18" :icon="mdiPound"/>
                                                    </span>
                                                    <v-text-field class="tag-list-name-field ms-3" type="text" autocomplete="off"
                                                                  density="compact" hide-details
                                                                  :disabled="loading || updating"
                                                                  :aria-label="tt('Tag Title')"
                                                                  :placeholder="tt('Tag Title')"
                                                                  v-model="newTag.name"
                                                                  @keyup.enter="save(newTag)"
                                                                  @keyup.esc="cancelSave(newTag)" />
                                                </div>
                                            </td>
                                            <td class="ebk-mgmt-column-operation">
                                                <div class="ebk-mgmt-actions">
                                                    <v-btn color="primary" density="comfortable" variant="tonal"
                                                           :prepend-icon="mdiCheck"
                                                           :loading="tagUpdating['']"
                                                           :disabled="loading || updating || !isTagModified(newTag)"
                                                           @click="save(newTag)">
                                                        <template #loader>
                                                            <v-progress-circular indeterminate size="20" width="2"/>
                                                        </template>
                                                        {{ tt('Save') }}
                                                    </v-btn>
                                                    <v-btn color="default" density="comfortable" variant="text"
                                                           :prepend-icon="mdiClose"
                                                           :disabled="loading || updating"
                                                           @click="cancelSave(newTag)">
                                                        {{ tt('Cancel') }}
                                                    </v-btn>
                                                </div>
                                            </td>
                                        </tr>
                                        </tbody>
                                    </v-table>
                                </v-card>
                            </v-window-item>
                        </v-window>
                    </v-main>
                </v-layout>
            </v-card>
        </v-col>
    </v-row>

    <v-dialog width="640" v-model="showTagMoveToDialog">
        <one-column-dialog-layout content-class="pa-0" :disabled="loading || updating"
                                  :title="tt('Move to...')" :cancel-button-title="tt('Close')"
                                  @cancel="showTagMoveToDialog = false">
            <template #content>
                <v-list class="w-100 py-0" density="comfortable">
                    <template :key="tagGroup.id" v-for="tagGroup in allTagGroupsWithDefault">
                        <v-list-item :title="tagGroup.name"
                                     :append-icon="mdiChevronRight"
                                     v-if="activeTagGroupId !== tagGroup.id"
                                     @click="moveTagToGroup(currentMovingTag, tagGroup.id)">
                            <template #subtitle>
                                <span class="ebk-numeric">{{ formatNumberToLocalizedNumerals(getTagGroupTagCount(tagGroup.id)) }}</span>
                            </template>
                        </v-list-item>
                    </template>
                </v-list>
            </template>
        </one-column-dialog-layout>
    </v-dialog>

    <tag-group-change-display-order-dialog ref="tagGroupChangeDisplayOrderDialog" />

    <rename-dialog ref="renameDialog"
                   :default-title="tt('Rename Tag Group')"
                   :label="tt('Tag Group Name')" :placeholder="tt('Tag Group Name')" />
    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import TagGroupChangeDisplayOrderDialog from './dialog/TagGroupChangeDisplayOrderDialog.vue';
import RenameDialog from '@/components/desktop/RenameDialog.vue';
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, computed, useTemplateRef, watch, nextTick } from 'vue';
import { useDisplay } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { useTagListPageBase } from '@/views/base/tags/TagListPageBase.ts';

import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { DEFAULT_TAG_GROUP_ID } from '@/consts/tag.ts';

import { TransactionTagGroup } from '@/models/transaction_tag_group.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';

import { getAvailableTagCount } from '@/lib/tag.ts';

import {
    mdiRefresh,
    mdiMenuUp,
    mdiMenuDown,
    mdiPencilOutline,
    mdiCheck,
    mdiClose,
    mdiSortAlphabeticalAscending,
    mdiSortAlphabeticalDescending,
    mdiEyeOffOutline,
    mdiEyeOutline,
    mdiSort,
    mdiMenu,
    mdiPlus,
    mdiFolderMoveOutline,
    mdiDeleteOutline,
    mdiDrag,
    mdiDotsVertical,
    mdiPound,
    mdiArrowUp,
    mdiArrowDown,
    mdiChevronRight
} from '@mdi/js';

type TagGroupChangeDisplayOrderDialogType = InstanceType<typeof TagGroupChangeDisplayOrderDialog>;
type RenameDialogType = InstanceType<typeof RenameDialog>;
type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;

const { lgAndUp } = useDisplay();

const { tt, formatNumberToLocalizedNumerals } = useI18n();

const {
    activeTagGroupId,
    newTag,
    editingTag,
    loading,
    showHidden,
    displayOrderModified,
    allTagGroupsWithDefault,
    tags,
    noAvailableTag,
    hasEditingTag,
    isTagModified,
    switchTagGroup,
    createNewTag,
    edit
} = useTagListPageBase();

const transactionTagsStore = useTransactionTagsStore();
const desktopPageStore = useDesktopPageStore();

const newTagRow = useTemplateRef<HTMLElement>('newTagRow');
const tagGroupChangeDisplayOrderDialog = useTemplateRef<TagGroupChangeDisplayOrderDialogType>('tagGroupChangeDisplayOrderDialog');
const renameDialog = useTemplateRef<RenameDialogType>('renameDialog');
const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');

const updating = ref<boolean>(false);
const activeTab = ref<string>('tagListPage');
const alwaysShowNav = ref<boolean>(lgAndUp.value);
const showNav = ref<boolean>(lgAndUp.value);
const loadError = ref<boolean>(false);
const tagUpdating = ref<Record<string, boolean>>({});
const tagHiding = ref<Record<string, boolean>>({});
const tagMoving = ref<Record<string, boolean>>({});
const tagRemoving = ref<Record<string, boolean>>({});
const currentMovingTag = ref<TransactionTag | null>(null);
const currentMoveTargetGroupId = ref<string>(DEFAULT_TAG_GROUP_ID);
const showTagMoveToDialog = ref<boolean>(false);

const totalAvailableTagsCount = computed<number>(() => transactionTagsStore.allAvailableTagsCount);
const displayTotalAvailableTagsCount = computed<string>(() => formatNumberToLocalizedNumerals(transactionTagsStore.allAvailableTagsCount));
const availableTagCount = computed<number>(() => getAvailableTagCount(tags.value, showHidden.value));
const activeTagGroupName = computed<string>(() => allTagGroupsWithDefault.value.find(tagGroup => tagGroup.id === activeTagGroupId.value)?.name ?? '');

function getTagGroupTagCount(tagGroupId: string): number {
    return getAvailableTagCount(transactionTagsStore.allTransactionTagsByGroupMap[tagGroupId] || [], showHidden.value);
}

function getAdjacentVisibleTagIndex(tag: TransactionTag, direction: number): { from: number, to: number } | null {
    const list = tags.value;
    const from = list.findIndex(item => item.id === tag.id);

    if (from < 0) {
        return null;
    }

    for (let to = from + direction; to >= 0 && to < list.length; to += direction) {
        if (showHidden.value || !list[to]!.hidden) {
            return { from, to };
        }
    }

    return null;
}

function canMoveTag(tag: TransactionTag, direction: number): boolean {
    return availableTagCount.value > 1 && !!getAdjacentVisibleTagIndex(tag, direction);
}

// non-drag alternative to the drag handle (guidelines §6.4), uses the same display order update as dragging
function moveTag(tag: TransactionTag, direction: number): void {
    const position = getAdjacentVisibleTagIndex(tag, direction);

    if (!position) {
        return;
    }

    transactionTagsStore.changeTagDisplayOrder({
        tagId: tag.id,
        from: position.from,
        to: position.to
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function reload(): void {
    if (hasEditingTag.value) {
        return;
    }

    loading.value = true;

    transactionTagsStore.loadAllTags({
        force: true
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        desktopPageStore.addToast('Tag list has been updated');
    }).catch(error => {
        loading.value = false;

        if (error && error.isUpToDate) {
            loadError.value = false;
            displayOrderModified.value = false;
        } else if (totalAvailableTagsCount.value < 1) {
            loadError.value = true;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function add(): void {
    createNewTag();

    nextTick(() => {
        newTagRow.value?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    });
}

function addTagGroup(): void {
    renameDialog.value?.open('', tt('New Tag Group Name')).then((newName: string) => {
        updating.value = true;

        transactionTagsStore.saveTagGroup({
            tagGroup: TransactionTagGroup.createNewTagGroup(newName)
        }).then(tagGroup => {
            updating.value = false;
            activeTagGroupId.value = tagGroup.id;
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function renameTagGroup(): void {
    const tagGroup = transactionTagsStore.allTransactionTagGroupsMap[activeTagGroupId.value];

    if (!tagGroup) {
        desktopPageStore.addToast('Unable to rename this tag group');
        return;
    }

    renameDialog.value?.open(tagGroup.name || '').then((newName: string) => {
        updating.value = true;

        const newTagGroup = tagGroup.clone();
        newTagGroup.name = newName;

        transactionTagsStore.saveTagGroup({
            tagGroup: newTagGroup
        }).then(() => {
            updating.value = false;
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function showChangeGroupDisplayOrderDialog(): void {
    tagGroupChangeDisplayOrderDialog.value?.open().then(() => {
        if (transactionTagsStore.transactionTagGroupListStateInvalid) {
            loading.value = true;

            transactionTagsStore.loadAllTagGroups({
                force: false
            }).then(() => {
                loading.value = false;
            }).catch(() => {
                loading.value = false;
            });
        }
    });
}

function removeTagGroup(): void {
    const tagGroup = transactionTagsStore.allTransactionTagGroupsMap[activeTagGroupId.value];

    if (!tagGroup) {
        desktopPageStore.addToast('Unable to delete this tag group');
        return;
    }

    const currentTagGroupIndex = allTagGroupsWithDefault.value.findIndex(group => group.id === tagGroup.id);

    confirmDialog.value?.open('Are you sure you want to delete this tag group?', { color: 'error' }).then(() => {
        updating.value = true;

        transactionTagsStore.deleteTagGroup({
            tagGroup: tagGroup
        }).then(() => {
            updating.value = false;

            if (allTagGroupsWithDefault.value[currentTagGroupIndex]) {
                const newActiveTagGroup = allTagGroupsWithDefault.value[currentTagGroupIndex];
                activeTagGroupId.value = newActiveTagGroup ? newActiveTagGroup.id : DEFAULT_TAG_GROUP_ID;
            } else if (allTagGroupsWithDefault.value[currentTagGroupIndex - 1]) {
                const newActiveTagGroup = allTagGroupsWithDefault.value[currentTagGroupIndex - 1];
                activeTagGroupId.value = newActiveTagGroup ? newActiveTagGroup.id : DEFAULT_TAG_GROUP_ID;
            } else {
                activeTagGroupId.value = DEFAULT_TAG_GROUP_ID;
            }
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function showMoveTagDialog(tag: TransactionTag): void {
    currentMovingTag.value = tag;
    currentMoveTargetGroupId.value = tag.groupId || DEFAULT_TAG_GROUP_ID;
    showTagMoveToDialog.value = true;
}

function moveTagToGroup(tag: TransactionTag | null, targetTagGroupId: string): void {
    if (!tag) {
        desktopPageStore.addToast('Unable to move tag');
        return;
    }

    updating.value = true;
    tagMoving.value[tag.id] = true;

    const newTag = tag.clone();
    newTag.groupId = targetTagGroupId;

    transactionTagsStore.saveTag({
        tag: newTag
    }).then(() => {
        updating.value = false;
        tagMoving.value[tag.id] = false;
    }).catch(error => {
        updating.value = false;
        tagMoving.value[tag.id] = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });

    showTagMoveToDialog.value = false;
}

function save(tag: TransactionTag): void {
    updating.value = true;
    tagUpdating.value[tag.id || ''] = true;

    transactionTagsStore.saveTag({
        tag: tag
    }).then(() => {
        updating.value = false;
        tagUpdating.value[tag.id || ''] = false;

        if (tag.id) {
            editingTag.value.id = '';
            editingTag.value.name = '';
        } else {
            newTag.value = null;
        }
    }).catch(error => {
        updating.value = false;
        tagUpdating.value[tag.id || ''] = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function cancelSave(tag: TransactionTag): void {
    if (tag.id) {
        editingTag.value.id = '';
        editingTag.value.name = '';
    } else {
        newTag.value = null;
    }
}

function sortByName(desc: boolean): void {
    const changed = transactionTagsStore.sortTagDisplayOrderByTagName(activeTagGroupId.value, desc);

    if (changed) {
        displayOrderModified.value = true;
    }
}

function saveSortResult(): void {
    if (!displayOrderModified.value) {
        return;
    }

    loading.value = true;

    transactionTagsStore.updateTagDisplayOrders(activeTagGroupId.value).then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function hide(tag: TransactionTag, hidden: boolean): void {
    updating.value = true;
    tagHiding.value[tag.id] = true;

    transactionTagsStore.hideTag({
        tag: tag,
        hidden: hidden
    }).then(() => {
        updating.value = false;
        tagHiding.value[tag.id] = false;
    }).catch(error => {
        updating.value = false;
        tagHiding.value[tag.id] = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(tag: TransactionTag): void {
    confirmDialog.value?.open('Are you sure you want to delete this tag?', { color: 'error' }).then(() => {
        updating.value = true;
        tagRemoving.value[tag.id] = true;

        transactionTagsStore.deleteTag({
            tag: tag
        }).then(() => {
            updating.value = false;
            tagRemoving.value[tag.id] = false;
        }).catch(error => {
            updating.value = false;
            tagRemoving.value[tag.id] = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function onMove(event: { moved: { element: { id: string }; oldIndex: number; newIndex: number } }): void {
    if (!event || !event.moved) {
        return;
    }

    const moveEvent = event.moved;

    if (!moveEvent.element || !moveEvent.element.id) {
        desktopPageStore.addToast('Unable to move tag');
        return;
    }

    transactionTagsStore.changeTagDisplayOrder({
        tagId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

transactionTagsStore.loadAllTags({
    force: false
}).then(() => {
    loading.value = false;
}).catch(error => {
    loading.value = false;
    loadError.value = true;

    if (!error.processed) {
        desktopPageStore.addErrorToast(error);
    }
});

watch(lgAndUp, (newValue) => {
    alwaysShowNav.value = newValue;

    if (!showNav.value) {
        showNav.value = newValue;
    }
});
</script>

<style>
.tag-list-content {
    border-radius: 0;
}

.tag-list-name-field {
    flex: 1 1 auto;
    max-width: 28rem;
}
</style>

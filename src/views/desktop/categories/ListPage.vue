<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loading ? 'true' : 'false'">
                <v-layout class="page-with-navigation-drawer">
                    <v-navigation-drawer ref="navbar" class="ebk-mgmt-subnav" :permanent="alwaysShowNav" v-model="showNav">
                        <div class="ebk-mgmt-subnav-section">
                            <btn-vertical-group :disabled="loading" :buttons="[
                                { name: tt('Expense'), value: CategoryType.Expense },
                                { name: tt('Income'), value: CategoryType.Income },
                                { name: tt('Transfer'), value: CategoryType.Transfer }
                            ]" v-model="activeCategoryType" @update:model-value="switchAllPrimaryCategories" />
                        </div>
                        <span class="ebk-mgmt-subnav-title ebk-technical-heading">{{ tt('Primary Category') }}</span>
                        <v-tabs show-arrows class="ebk-mgmt-subnav-tabs mb-3" direction="vertical"
                                :disabled="loading" v-model="primaryCategoryId">
                            <v-tab class="tab-text-truncate" value="0" @click="switchAllPrimaryCategories">
                                <span class="ebk-mgmt-subnav-tab-content">
                                    <span class="text-truncate">{{ tt('Primary Categories') }}</span>
                                    <span class="ebk-mgmt-subnav-tab-count" v-if="primaryCategories.length">{{ formatNumberToLocalizedNumerals(getAvailableCategoryCount(primaryCategories, showHidden)) }}</span>
                                </span>
                            </v-tab>
                            <template :key="category.id" v-for="category in primaryCategories">
                                <v-tab class="tab-text-truncate" :value="category.id" v-if="!category.hidden"
                                       @click="switchPrimaryCategory(category)">
                                    <span class="ebk-mgmt-subnav-tab-content">
                                        <ItemIcon size="1.25rem" :icon-type="getCategoryIconType(category.iconType)" :icon-id="category.icon" :color="category.color" />
                                        <span class="text-truncate">{{ category.name }}</span>
                                        <span class="ebk-mgmt-subnav-tab-count" v-if="category.subCategories && category.subCategories.length">{{ formatNumberToLocalizedNumerals(getAvailableCategoryCount(category.subCategories, showHidden)) }}</span>
                                    </span>
                                </v-tab>
                            </template>
                            <template v-if="loading && (!primaryCategories || primaryCategories.length < 1)">
                                <v-skeleton-loader class="skeleton-no-margin mx-4 mt-h1 py-3" type="text"
                                                   :key="itemIdx" :loading="true" v-for="itemIdx in [ 1, 2, 3, 4, 5, 6 ]"></v-skeleton-loader>
                            </template>
                        </v-tabs>
                    </v-navigation-drawer>
                    <v-main>
                        <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                            <v-window-item value="categoryPage">
                                <v-card variant="flat" class="category-list-content" :min-height="cardMinHeight">
                                    <header class="ebk-panel-header-strip ebk-mgmt-header">
                                        <v-btn class="ebk-mgmt-header-menu-toggle d-lg-none" density="comfortable" color="default" variant="text"
                                               :aria-label="tt('Open Menu')" :icon="true" @click="showNav = !showNav">
                                            <v-icon :icon="mdiMenu" size="22" />
                                        </v-btn>
                                        <v-btn class="ebk-mgmt-header-menu-toggle button-icon-with-direction" density="comfortable" color="default" variant="text"
                                               :aria-label="tt('Back')" :icon="true" :disabled="loading"
                                               v-if="!hasSubCategories" @click="switchAllPrimaryCategories">
                                            <v-icon :icon="mdiArrowLeft" size="20" />
                                            <v-tooltip activator="parent">{{ tt('Primary Categories') }}</v-tooltip>
                                        </v-btn>
                                        <h2 class="ebk-mgmt-header-label">{{ hasSubCategories ? tt('Primary Categories') : (currentPrimaryCategory?.name ?? tt('Secondary Category')) }}</h2>
                                        <span class="ebk-mgmt-header-meta">
                                            <span>{{ activeCategoryTypeName }}</span>
                                            <template v-if="!loading && availableCategoryCount > 0">
                                                <span aria-hidden="true"> · </span>
                                                <span class="ebk-numeric">{{ formatNumberToLocalizedNumerals(availableCategoryCount) }}</span>
                                            </template>
                                        </span>
                                        <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                                            <v-btn color="primary" variant="tonal"
                                                   :disabled="loading || updating" @click="saveSortResult"
                                                   v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                                            <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                                                   :disabled="loading || updating" @click="add">{{ tt('Add') }}</v-btn>
                                            <v-btn density="comfortable" color="default" variant="text"
                                                   :aria-label="tt('Refresh')" :icon="true" :loading="loading || updating" @click="reload(true)">
                                                <template #loader>
                                                    <v-progress-circular indeterminate size="20"/>
                                                </template>
                                                <v-icon :icon="mdiRefresh" size="22" />
                                                <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                                            </v-btn>
                                            <v-btn density="comfortable" color="default" variant="text"
                                                   :aria-label="tt('More')" :disabled="loading || updating" :icon="true">
                                                <v-icon :icon="mdiDotsVertical" />
                                                <v-tooltip activator="parent">{{ tt('More') }}</v-tooltip>
                                                <v-menu activator="parent">
                                                    <v-list>
                                                        <v-list-item :prepend-icon="mdiEyeOutline"
                                                                     :title="tt('Show Hidden Transaction Categories')"
                                                                     v-if="!showHidden" @click="showHidden = true"></v-list-item>
                                                        <v-list-item :prepend-icon="mdiEyeOffOutline"
                                                                     :title="tt('Hide Hidden Transaction Categories')"
                                                                     v-if="showHidden" @click="showHidden = false"></v-list-item>
                                                    </v-list>
                                                </v-menu>
                                            </v-btn>
                                        </div>
                                    </header>

                                    <v-table class="ebk-mgmt-table transaction-category-table" density="default" :hover="!loading">
                                        <thead>
                                        <tr>
                                            <th class="ebk-mgmt-column-name">{{ tt('Category Name') }}</th>
                                            <th class="ebk-mgmt-column-operation">
                                                <span class="ebk-visually-hidden">{{ tt('Operation') }}</span>
                                            </th>
                                        </tr>
                                        </thead>

                                        <tbody aria-hidden="true" v-if="loading && noAvailableCategory">
                                        <tr class="ebk-mgmt-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3, 4, 5 ]">
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

                                        <tbody v-if="!hasSubCategories && currentPrimaryCategory && !(loading && noAvailableCategory)">
                                        <tr class="ebk-mgmt-row ebk-mgmt-row--parent no-hover" :class="{ 'ebk-mgmt-row--hidden': currentPrimaryCategory.hidden }">
                                            <td class="ebk-mgmt-column-name">
                                                <div class="ebk-mgmt-row-content">
                                                    <span class="ebk-mgmt-drag disabled" aria-hidden="true"></span>
                                                    <span class="ebk-mgmt-icon-tile ebk-mgmt-dimmable" aria-hidden="true">
                                                        <ItemIcon size="1.25rem" :icon-type="getCategoryIconType(currentPrimaryCategory.iconType)"
                                                                  :icon-id="currentPrimaryCategory.icon" :color="currentPrimaryCategory.color" />
                                                    </span>
                                                    <div class="ebk-mgmt-name">
                                                        <div class="ebk-mgmt-name-line">
                                                            <span class="ebk-mgmt-name-text ebk-mgmt-dimmable">{{ currentPrimaryCategory.name }}</span>
                                                        </div>
                                                        <span class="ebk-mgmt-meta">{{ tt('Primary Category') }}</span>
                                                    </div>
                                                </div>
                                            </td>
                                            <td class="ebk-mgmt-column-operation">
                                                <div class="ebk-mgmt-actions">
                                                    <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                           :aria-label="tt('Edit')" :disabled="loading || updating"
                                                           @click="edit(currentPrimaryCategory)">
                                                        <v-icon :icon="mdiPencilOutline" size="20" />
                                                        <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                    </v-btn>
                                                </div>
                                            </td>
                                        </tr>
                                        </tbody>

                                        <tbody v-if="!loading && noAvailableCategory">
                                        <tr class="no-hover">
                                            <td class="ebk-mgmt-column-state" colspan="2">
                                                <state-panel type="error" :title="tt('Unable to retrieve category list')" v-if="loadError">
                                                    <template #actions>
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload(true)">{{ tt('Retry') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                                <state-panel type="empty" :title="tt('No available category')" v-else>
                                                    <template #actions>
                                                        <v-btn color="default" variant="outlined"
                                                               @click="showPresetDialog = true"
                                                               v-if="hasSubCategories && noCategory">
                                                            {{ tt('Add Default Categories') }}
                                                        </v-btn>
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiPlus"
                                                               :disabled="updating" @click="add">{{ tt('Add') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                            </td>
                                        </tr>
                                        </tbody>

                                        <draggable-list tag="tbody"
                                                        item-key="id"
                                                        handle=".drag-handle"
                                                        ghost-class="dragging-item"
                                                        :disabled="noAvailableCategory || loading || updating"
                                                        v-model="categories"
                                                        @change="onMove">
                                            <template #item="{ element }">
                                                <tr class="ebk-mgmt-row transaction-category-table-row"
                                                    :class="{ 'ebk-mgmt-row--hidden': element.hidden, 'ebk-mgmt-row--child': !hasSubCategories, 'ebk-mgmt-row--child-last': !hasSubCategories && isLastVisibleCategory(element) }"
                                                    v-if="showHidden || !element.hidden">
                                                    <td class="ebk-mgmt-column-name">
                                                        <div class="ebk-mgmt-row-content">
                                                            <span class="ebk-mgmt-drag" :class="{ 'drag-handle': !loading && !updating && availableCategoryCount > 1, 'disabled': loading || updating || availableCategoryCount <= 1 }">
                                                                <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                                                <v-tooltip activator="parent" v-if="!loading && !updating && availableCategoryCount > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                                            </span>
                                                            <span class="ebk-mgmt-icon-tile ebk-mgmt-dimmable" aria-hidden="true">
                                                                <ItemIcon size="1.25rem" :icon-type="getCategoryIconType(element.iconType)"
                                                                          :icon-id="element.icon" :color="element.color" />
                                                            </span>
                                                            <div class="ebk-mgmt-name">
                                                                <div class="ebk-mgmt-name-line">
                                                                    <span class="ebk-mgmt-name-text ebk-mgmt-dimmable"
                                                                          :class="{ 'ebk-mgmt-name-text--link': isCategorySupportSwitch(element) }"
                                                                          @click="switchPrimaryCategory(element)">{{ element.name }}</span>
                                                                    <span class="ebk-mgmt-hidden-chip" v-if="element.hidden">
                                                                        <v-icon :icon="mdiEyeOffOutline" size="12" />
                                                                        {{ tt('Hidden') }}
                                                                    </span>
                                                                </div>
                                                                <span class="ebk-mgmt-meta ebk-mgmt-dimmable" v-if="element.comment || (hasSubCategories && element.subCategories && element.subCategories.length)">
                                                                    <template v-if="hasSubCategories && element.subCategories && element.subCategories.length">
                                                                        <span>{{ tt('Secondary Category') }}</span>
                                                                        <span class="ebk-numeric">&nbsp;{{ formatNumberToLocalizedNumerals(getAvailableCategoryCount(element.subCategories, showHidden)) }}</span>
                                                                    </template>
                                                                    <span aria-hidden="true" v-if="element.comment && hasSubCategories && element.subCategories && element.subCategories.length"> · </span>
                                                                    <span v-if="element.comment">{{ element.comment }}</span>
                                                                </span>
                                                                <v-tooltip activator="parent" location="bottom start" open-delay="600" v-if="element.comment">{{ element.comment }}</v-tooltip>
                                                            </div>
                                                        </div>
                                                    </td>
                                                    <td class="ebk-mgmt-column-operation">
                                                        <div class="ebk-mgmt-actions">
                                                            <v-btn class="button-icon-with-direction" color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Secondary Category')"
                                                                   :disabled="loading || updating"
                                                                   v-if="isCategorySupportSwitch(element)"
                                                                   @click="switchPrimaryCategory(element)">
                                                                <v-icon :icon="mdiChevronRight" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Secondary Category') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="element.hidden ? tt('Show') : tt('Hide')"
                                                                   :loading="categoryHiding[element.id]"
                                                                   :disabled="loading || updating"
                                                                   @click="hide(element, !element.hidden)">
                                                                <template #loader>
                                                                    <v-progress-circular indeterminate size="20" width="2"/>
                                                                </template>
                                                                <v-icon :icon="element.hidden ? mdiEyeOutline : mdiEyeOffOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ element.hidden ? tt('Show') : tt('Hide') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Edit')"
                                                                   :disabled="loading || updating"
                                                                   @click="edit(element)">
                                                                <v-icon :icon="mdiPencilOutline" size="20" />
                                                                <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Delete')"
                                                                   :loading="categoryRemoving[element.id]"
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
                                                                   :aria-label="tt('Move Up')"
                                                                   :disabled="loading || updating || !canMoveCategory(element, -1)"
                                                                   @click="moveCategory(element, -1)">
                                                                <v-icon :icon="mdiArrowUp" size="18" />
                                                                <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                                            </v-btn>
                                                            <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text" :icon="true"
                                                                   :aria-label="tt('Move Down')"
                                                                   :disabled="loading || updating || !canMoveCategory(element, 1)"
                                                                   @click="moveCategory(element, 1)">
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
                            </v-window-item>
                        </v-window>
                    </v-main>
                </v-layout>
            </v-card>
        </v-col>
    </v-row>

    <edit-dialog ref="editDialog" />
    <preset-dialog :category-type="activeCategoryType" v-model:show="showPresetDialog"
                   @category:saved="onPresetCategorySaved" />

    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import { VNavigationDrawer } from 'vuetify/components/VNavigationDrawer';

import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import EditDialog from './list/dialogs/EditDialog.vue';
import PresetDialog from './list/dialogs/PresetDialog.vue';

import { ref, computed, useTemplateRef, watch, nextTick } from 'vue';
import { useDisplay } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { useCategoryListPageBase } from '@/views/base/categories/CategoryListPageBase.ts';

import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { CategoryType } from '@/core/category.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';

import { getCategoryIconType } from '@/lib/icon.ts';
import {
    isNoAvailableCategory,
    getAvailableCategoryCount
} from '@/lib/category.ts';
import { getNavSideBarOuterHeight } from '@/lib/ui/desktop.ts';

import {
    mdiRefresh,
    mdiMenu,
    mdiPencilOutline,
    mdiEyeOffOutline,
    mdiEyeOutline,
    mdiDeleteOutline,
    mdiDrag,
    mdiDotsVertical,
    mdiPlus,
    mdiArrowUp,
    mdiArrowDown,
    mdiArrowLeft,
    mdiChevronRight
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type EditDialogType = InstanceType<typeof EditDialog>;

const { lgAndUp } = useDisplay();
const { tt, formatNumberToLocalizedNumerals } = useI18n();
const { loading, primaryCategoryId, currentPrimaryCategory } = useCategoryListPageBase();

const transactionCategoriesStore = useTransactionCategoriesStore();
const desktopPageStore = useDesktopPageStore();

const navbar = useTemplateRef<VNavigationDrawer>('navbar');
const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const editDialog = useTemplateRef<EditDialogType>('editDialog');

const activeCategoryType = ref<CategoryType>(CategoryType.Expense);
const activeTab = ref<string>('categoryPage');
const updating = ref<boolean>(false);
const loadError = ref<boolean>(false);
const categoryHiding = ref<Record<string, boolean>>({});
const categoryRemoving = ref<Record<string, boolean>>({});
const displayOrderModified = ref<boolean>(false);
const cardMinHeight = ref<number>(680);
const alwaysShowNav = ref<boolean>(lgAndUp.value);
const showNav = ref<boolean>(lgAndUp.value);
const showHidden = ref<boolean>(false);
const showPresetDialog = ref<boolean>(false);

const primaryCategories = computed<TransactionCategory[]>(() => {
    if (!transactionCategoriesStore.allTransactionCategories || !transactionCategoriesStore.allTransactionCategories[activeCategoryType.value]) {
        return [];
    }

    return transactionCategoriesStore.allTransactionCategories[activeCategoryType.value] ?? [];
});

const secondaryCategories = computed<TransactionCategory[]>(() => {
    if (!transactionCategoriesStore.allTransactionCategoriesMap || !transactionCategoriesStore.allTransactionCategoriesMap[primaryCategoryId.value]) {
        return [];
    }

    return transactionCategoriesStore.allTransactionCategoriesMap[primaryCategoryId.value]?.subCategories ?? [];
});

const hasSubCategories = computed<boolean>(() => {
    return !primaryCategoryId.value || primaryCategoryId.value === '' || primaryCategoryId.value === '0';
});

const categories = computed<TransactionCategory[]>(() => {
    if (hasSubCategories.value) {
        return primaryCategories.value;
    } else {
        return secondaryCategories.value;
    }
});

const noAvailableCategory = computed<boolean>(() => isNoAvailableCategory(categories.value, showHidden.value));
const noCategory = computed<boolean>(() => categories.value.length < 1);
const availableCategoryCount = computed<number>(() => getAvailableCategoryCount(categories.value, showHidden.value));

const activeCategoryTypeName = computed<string>(() => {
    if (activeCategoryType.value === CategoryType.Income) {
        return tt('Income');
    } else if (activeCategoryType.value === CategoryType.Transfer) {
        return tt('Transfer');
    } else {
        return tt('Expense');
    }
});

function getAdjacentVisibleCategoryIndex(category: TransactionCategory, direction: number): { from: number, to: number } | null {
    const list = categories.value;
    const from = list.findIndex(item => item.id === category.id);

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

function isLastVisibleCategory(category: TransactionCategory): boolean {
    return !getAdjacentVisibleCategoryIndex(category, 1);
}

function canMoveCategory(category: TransactionCategory, direction: number): boolean {
    return availableCategoryCount.value > 1 && !!getAdjacentVisibleCategoryIndex(category, direction);
}

// non-drag alternative to the drag handle (guidelines §6.4), uses the same display order update as dragging
function moveCategory(category: TransactionCategory, direction: number): void {
    const position = getAdjacentVisibleCategoryIndex(category, direction);

    if (!position) {
        return;
    }

    transactionCategoriesStore.changeCategoryDisplayOrder({
        categoryId: category.id,
        from: position.from,
        to: position.to
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function updateCardMinHeight(): void {
    nextTick(() => {
        if (navbar.value && navbar.value.$el && navbar.value.$el.nextElementSibling) {
            const navbarHeight = getNavSideBarOuterHeight(navbar.value.$el.nextElementSibling);
            cardMinHeight.value = Math.max(navbarHeight, 680);
        }
    });
}

function isCategorySupportSwitch(category: TransactionCategory): boolean {
    if (!category || category.hidden) {
        return false;
    }

    return !category.parentId || category.parentId === '' || category.parentId === '0';
}

function switchAllPrimaryCategories(): void {
    primaryCategoryId.value = '0';
    updateCardMinHeight();
}

function switchPrimaryCategory(category: TransactionCategory): void {
    if (!category || category.hidden) {
        return;
    }

    if (!category.parentId || category.parentId === '' || category.parentId === '0') {
        primaryCategoryId.value = category.id;
    }

    updateCardMinHeight();
}

function reload(force: boolean): void {
    loading.value = true;

    transactionCategoriesStore.loadAllCategories({
        force: force
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        if (force) {
            desktopPageStore.addToast('Category list has been updated');
        }

        updateCardMinHeight();
    }).catch(error => {
        loading.value = false;

        if (error && error.isUpToDate) {
            loadError.value = false;
            displayOrderModified.value = false;
        } else if (noCategory.value) {
            loadError.value = true;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function add(): void {
    editDialog.value?.open({
        type: activeCategoryType.value,
        parentId: primaryCategoryId.value,
        color: currentPrimaryCategory.value?.color,
        icon: currentPrimaryCategory.value?.icon
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }

        updateCardMinHeight();
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function edit(category: TransactionCategory): void {
    editDialog.value?.open({
        id: category.id,
        currentCategory: category
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }

        if (transactionCategoriesStore.transactionCategoryListStateInvalid) {
            reload(true);
        }

        updateCardMinHeight();
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function hide(category: TransactionCategory, hidden: boolean): void {
    updating.value = true;
    categoryHiding.value[category.id] = true;

    transactionCategoriesStore.hideCategory({
        category: category,
        hidden: hidden
    }).then(() => {
        updating.value = false;
        categoryHiding.value[category.id] = false;

        updateCardMinHeight();
    }).catch(error => {
        updating.value = false;
        categoryHiding.value[category.id] = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(category: TransactionCategory): void {
    confirmDialog.value?.open('Are you sure you want to delete this category?', { color: 'error' }).then(() => {
        updating.value = true;
        categoryRemoving.value[category.id] = true;

        transactionCategoriesStore.deleteCategory({
            category: category
        }).then(() => {
            updating.value = false;
            categoryRemoving.value[category.id] = false;

            updateCardMinHeight();
        }).catch(error => {
            updating.value = false;
            categoryRemoving.value[category.id] = false;

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

    transactionCategoriesStore.updateCategoryDisplayOrders({
        type: activeCategoryType.value,
        parentId: primaryCategoryId.value
    }).then(() => {
        loading.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function onMove(event: { moved: { element: { id: string }, oldIndex: number, newIndex: number } }): void {
    if (!event || !event.moved) {
        return;
    }

    const moveEvent = event.moved;

    if (!moveEvent.element || !moveEvent.element.id) {
        desktopPageStore.addToast('Unable to move category');
        return;
    }

    transactionCategoriesStore.changeCategoryDisplayOrder({
        categoryId: moveEvent.element.id,
        from: moveEvent.oldIndex,
        to: moveEvent.newIndex
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        desktopPageStore.addErrorToast(error);
    });
}

function onPresetCategorySaved(e: { message: string }): void {
    if (e && e.message) {
        desktopPageStore.addToast(e.message);
        reload(false);
    }
}

watch(lgAndUp, (newValue) => {
    alwaysShowNav.value = newValue;

    if (!showNav.value) {
        showNav.value = newValue;
    }
});

reload(false);
</script>

<style>
.category-list-content {
    border-radius: 0;
}
</style>

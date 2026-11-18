<template>
    <v-dialog width="800" :persistent="loading || submitting || isCategoryModified" v-model="showState">
        <one-column-dialog-layout :disabled="loading || submitting" :loading="loading"
                                  :title="tt(title)" :cancel-button-title="tt('Cancel')"
                                  @cancel="cancel">
            <template #content>
                <v-form class="mt-2 ebk-dialog-form">
                    <v-row>
                        <v-col cols="12" md="12">
                            <v-text-field
                                type="text"
                                autocomplete="off"
                                persistent-placeholder
                                :disabled="loading || submitting"
                                :label="tt('Category Name')"
                                :placeholder="tt('Category Name')"
                                v-model="category.name"
                            />
                        </v-col>
                        <v-col cols="12" md="12" v-if="editCategoryId && category.parentId && category.parentId !== '0'">
                            <v-select
                                item-title="name"
                                item-value="id"
                                persistent-placeholder
                                :disabled="loading || submitting"
                                :label="tt('Primary Category')"
                                :placeholder="tt('Primary Category')"
                                :items="allAvailableCategories"
                                :no-data-text="tt('No available primary category')"
                                v-model="category.parentId"
                            >
                                <template #item="{ props, internalItem }">
                                    <v-list-item v-bind="props">
                                        <template #prepend>
                                            <ItemIcon class="me-2" :icon-type="getCategoryIconType(internalItem.raw.iconType)"
                                                      :icon-id="internalItem.raw.icon" :color="internalItem.raw.color"></ItemIcon>
                                        </template>
                                        <template #title>
                                            <div class="text-truncate">{{ internalItem.raw.name }}</div>
                                        </template>
                                    </v-list-item>
                                </template>
                            </v-select>
                        </v-col>
                        <v-col cols="12" md="6">
                            <icon-select type="category"
                                         :all-system-icon-infos="ALL_CATEGORY_ICONS"
                                         :label="tt('Category Icon')"
                                         :color="category.color"
                                         :disabled="loading || submitting"
                                         v-model:icon-type="category.iconType"
                                         v-model="category.icon" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <color-select :all-system-color-infos="ALL_CATEGORY_COLORS"
                                          :label="tt('Category Color')"
                                          :disabled="loading || submitting"
                                          v-model="category.color" />
                        </v-col>
                        <v-col cols="12" md="12">
                            <v-textarea
                                type="text"
                                autocomplete="off"
                                persistent-placeholder
                                rows="3"
                                :disabled="loading || submitting"
                                :label="tt('Description')"
                                :placeholder="tt('Your category description (optional)')"
                                v-model="category.comment"
                            />
                        </v-col>
                        <v-col class="py-0 my-n2" cols="12" md="12" v-if="editCategoryId">
                            <v-switch :disabled="loading || submitting"
                                      :label="tt('Visible')" v-model="category.visible"/>
                        </v-col>
                    </v-row>
                </v-form>
            </template>

            <template #footer>
                <div class="ebk-dialog-footer">
                    <div class="ebk-dialog-footer-status" role="status" aria-live="polite">
                        <template v-if="footerStatus">
                            <span class="ebk-dialog-footer-status-dot" :class="`ebk-dialog-footer-status-dot--${footerStatus.kind}`" aria-hidden="true"></span>
                            <span>{{ footerStatus.text }}</span>
                        </template>
                    </div>
                    <v-btn color="default" variant="outlined" :disabled="loading || submitting" @click="cancel">{{ tt('Cancel') }}</v-btn>
                    <v-tooltip :disabled="!inputIsEmpty" :text="inputEmptyProblemMessage ? tt(inputEmptyProblemMessage) : ''">
                        <template v-slot:activator="{ props }">
                            <div v-bind="props" class="d-inline-block">
                                <v-btn color="primary" variant="flat" :disabled="inputIsEmpty || loading || submitting" @click="save">
                                    {{ tt(saveButtonTitle) }}
                                    <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                                </v-btn>
                            </div>
                        </template>
                    </v-tooltip>
                </div>
            </template>
        </one-column-dialog-layout>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useCategoryEditPageBase } from '@/views/base/categories/CategoryEditPageBase.ts';

import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { ColorValue } from '@/core/color.ts';
import { CategoryType } from '@/core/category.ts';
import { ALL_CATEGORY_ICONS } from '@/consts/icon.ts';
import { ALL_CATEGORY_COLORS } from '@/consts/color.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';

import { isEquals } from '@/lib/common.ts';
import { getCategoryIconType } from '@/lib/icon.ts';
import { generateRandomUUID } from '@/lib/misc.ts';

interface TransactionCategoryEditResponse {
    message: string;
    category?: TransactionCategory;
}

const { tt } = useI18n();
const {
    editCategoryId,
    clientSessionId,
    loading,
    submitting,
    category,
    allAvailableCategories,
    title,
    saveButtonTitle,
    inputEmptyProblemMessage,
    inputIsEmpty
} = useCategoryEditPageBase();

const transactionCategoriesStore = useTransactionCategoriesStore();
const desktopPageStore = useDesktopPageStore();

let resolveFunc: ((value: TransactionCategoryEditResponse) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);
const initCategory = ref<TransactionCategory | null>(null);

const isCategoryModified = computed<boolean>(() => {
    if (!editCategoryId.value) { // Add
        return !!initCategory.value && !isEquals(category.value.toCreateRequest(clientSessionId.value), initCategory.value.toCreateRequest(clientSessionId.value));
    } else { // Edit
        return !!initCategory.value && !isEquals(category.value.toModifyRequest(), initCategory.value.toModifyRequest());
    }
});

// footer status text at the start of the dialog footer (design guideline 4.4)
const footerStatus = computed<{ kind: 'progress' | 'warning' | 'modified', text: string } | null>(() => {
    if (loading.value) {
        return null;
    }

    if (submitting.value) {
        return { kind: 'progress', text: tt('Saving...') };
    }

    if (isCategoryModified.value && inputIsEmpty.value && inputEmptyProblemMessage.value) {
        return { kind: 'warning', text: tt(inputEmptyProblemMessage.value) };
    }

    if (isCategoryModified.value) {
        return { kind: 'modified', text: tt('Unsaved changes') };
    }

    return null;
});

function open(options: { id?: string; parentId?: string; type?: CategoryType; currentCategory?: TransactionCategory, color?: ColorValue, icon?: string, name?: string }): Promise<TransactionCategoryEditResponse> {
    showState.value = true;
    loading.value = true;
    submitting.value = false;

    initCategory.value = TransactionCategory.createNewCategory();
    category.value.fillFrom(initCategory.value);

    if (options.id) {
        if (options.currentCategory) {
            category.value.fillFrom(options.currentCategory);
        }

        editCategoryId.value = options.id;
        transactionCategoriesStore.getCategory({
            categoryId: editCategoryId.value
        }).then(response => {
            category.value.fillFrom(response);
            initCategory.value = TransactionCategory.of(response);
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            showState.value = false;

            if (!error.processed) {
                if (rejectFunc) {
                    rejectFunc(error);
                }
            }
        });
    } else if (options.parentId) {
        editCategoryId.value = null;

        const categoryType = options.type;

        if (categoryType !== CategoryType.Income &&
            categoryType !== CategoryType.Expense &&
            categoryType !== CategoryType.Transfer) {
            loading.value = false;
            showState.value = false;

            return Promise.reject('Parameter Invalid');
        }

        initCategory.value.type = categoryType;
        initCategory.value.parentId = options.parentId;

        category.value.type = initCategory.value.type;
        category.value.parentId = initCategory.value.parentId;

        if (options.color) {
            initCategory.value.color = options.color;
            category.value.color = initCategory.value.color;
        }

        if (options.icon) {
            initCategory.value.icon = options.icon;
            category.value.icon = initCategory.value.icon;
        }

        if (options.name) {
            category.value.name = options.name;
        }

        clientSessionId.value = generateRandomUUID();
        loading.value = false;
    }

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function save(): void {
    const problemMessage = inputEmptyProblemMessage.value;

    if (problemMessage) {
        desktopPageStore.addToast(problemMessage);
        return;
    }

    submitting.value = true;

    transactionCategoriesStore.saveCategory({
        category: category.value,
        isEdit: !!editCategoryId.value,
        clientSessionId: clientSessionId.value
    }).then(savedCategory => {
        submitting.value = false;

        let message = 'You have saved this category';

        if (!editCategoryId.value) {
            message = 'You have added a new category';
        }

        resolveFunc?.({ message, category: savedCategory });
        showState.value = false;
    }).catch(error => {
        submitting.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function cancel(): void {
    rejectFunc?.();
    showState.value = false;
}

defineExpose({
    open
});
</script>

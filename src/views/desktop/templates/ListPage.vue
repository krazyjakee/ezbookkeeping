<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="transaction-templates-panel" :aria-busy="loading ? 'true' : 'false'">
                <header class="ebk-panel-header-strip transaction-templates-header">
                    <h2 class="transaction-templates-header-label">{{ templateType === TemplateType.Schedule.type ? tt('Scheduled Transactions') : tt('Transaction Templates') }}</h2>
                    <span class="transaction-templates-header-meta ebk-numeric" v-if="!loading && availableTemplateCount > 0">{{ formatNumberToLocalizedNumerals(availableTemplateCount) }}</span>
                    <div class="ebk-panel-header-actions transaction-templates-header-actions">
                        <v-btn color="primary" variant="tonal"
                               :disabled="loading || updating" @click="saveSortResult"
                               v-if="displayOrderModified">{{ tt('Save Display Order') }}</v-btn>
                        <v-btn color="default" variant="outlined"
                               :disabled="loading || updating" v-if="isScheduledTransaction">
                            {{ tt('Status') }}: {{ statusFilterDisplayName }}
                            <v-icon class="ms-1" :icon="mdiMenuDown" size="20" />
                            <v-menu activator="parent">
                                <v-list>
                                    <v-list-item :key="option.type"
                                                 :active="statusFilter === option.type"
                                                 :title="option.displayName"
                                                 v-for="option in allStatusFilterTypes"
                                                 @click="statusFilter = option.type"></v-list-item>
                                </v-list>
                            </v-menu>
                        </v-btn>
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
                        <v-btn density="comfortable" color="default" variant="text"
                               :aria-label="tt('More')" :disabled="loading || updating" :icon="true">
                            <v-icon :icon="mdiDotsVertical" />
                            <v-menu activator="parent">
                                <v-list>
                                    <v-list-item :prepend-icon="mdiEyeOutline"
                                                 :title="tt('Show Hidden Transaction Templates')"
                                                 v-if="!showHidden" @click="showHidden = true"></v-list-item>
                                    <v-list-item :prepend-icon="mdiEyeOffOutline"
                                                 :title="tt('Hide Hidden Transaction Templates')"
                                                 v-if="showHidden" @click="showHidden = false"></v-list-item>
                                </v-list>
                            </v-menu>
                        </v-btn>
                    </div>
                </header>

                <v-table class="transaction-templates-table" :hover="!loading">
                    <thead>
                    <tr>
                        <th class="transaction-templates-column-name">{{ tt('Template Name') }}</th>
                        <th class="transaction-templates-column-status" v-if="isScheduledTransaction">{{ tt('Status') }}</th>
                        <th class="transaction-templates-column-account">{{ tt('Account') }}</th>
                        <th class="transaction-templates-column-amount">{{ tt('Amount') }}</th>
                        <th class="transaction-templates-column-operation">
                            <span class="ebk-visually-hidden">{{ tt('Operation') }}</span>
                        </th>
                    </tr>
                    </thead>

                    <tbody aria-hidden="true" v-if="loading && noAvailableTemplate">
                    <tr class="transaction-templates-table-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                        <td>
                            <div class="d-flex align-center">
                                <span class="transaction-templates-icon-tile transaction-templates-icon-tile--skeleton"></span>
                                <v-skeleton-loader class="transaction-templates-skeleton ms-3" type="text" width="160" :loading="true"></v-skeleton-loader>
                            </div>
                        </td>
                        <td v-if="isScheduledTransaction"><v-skeleton-loader class="transaction-templates-skeleton" type="text" width="72" :loading="true"></v-skeleton-loader></td>
                        <td><v-skeleton-loader class="transaction-templates-skeleton" type="text" width="120" :loading="true"></v-skeleton-loader></td>
                        <td><v-skeleton-loader class="transaction-templates-skeleton ms-auto" type="text" width="80" :loading="true"></v-skeleton-loader></td>
                        <td></td>
                    </tr>
                    </tbody>

                    <tbody v-if="!loading && noAvailableTemplate">
                    <tr class="no-hover">
                        <td class="transaction-templates-state-cell" :colspan="tableColumnCount">
                            <state-panel type="error" :title="loadErrorTitle" v-if="loadError">
                                <template #actions>
                                    <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload">{{ tt('Retry') }}</v-btn>
                                </template>
                            </state-panel>
                            <state-panel type="empty" :title="emptyStateTitle" v-else>
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
                                    :disabled="noAvailableTemplate || !canReorderByStatusFilter"
                                    v-model="templates"
                                    @change="onMove">
                        <template #item="{ element }">
                            <tr class="transaction-templates-table-row" :class="{ 'transaction-templates-table-row--hidden': element.hidden }"
                                v-if="(showHidden || !element.hidden) && isShownByStatusFilter(element)">
                                <td class="transaction-templates-column-name">
                                    <div class="d-flex align-center">
                                        <span class="transaction-templates-drag" :class="{ 'drag-handle': !loading && !updating && canReorderByStatusFilter && availableTemplateCount > 1, 'disabled': loading || updating || !canReorderByStatusFilter || availableTemplateCount <= 1 }">
                                            <v-icon :aria-label="tt('Drag to Reorder')" :icon="mdiDrag" size="20"/>
                                            <v-tooltip activator="parent" v-if="!loading && !updating && canReorderByStatusFilter && availableTemplateCount > 1">{{ tt('Drag to Reorder') }}</v-tooltip>
                                        </span>
                                        <span class="transaction-templates-icon-tile" aria-hidden="true">
                                            <v-icon size="18" :icon="templateType === TemplateType.Schedule.type ? mdiClockTimeNineOutline : mdiTextBoxOutline"/>
                                        </span>
                                        <div class="transaction-templates-name">
                                            <div class="d-flex align-center">
                                                <span class="transaction-templates-name-text">{{ element.name }}</span>
                                                <span class="transaction-templates-hidden-chip" v-if="element.hidden">
                                                    <v-icon :icon="mdiEyeOffOutline" size="12" />
                                                    {{ tt('Hidden') }}
                                                </span>
                                            </div>
                                            <span class="transaction-templates-meta">
                                                <span>{{ getTemplateTypeName(element) }}</span>
                                                <template v-if="getTemplateCategoryName(element)">
                                                    <span aria-hidden="true"> · </span>
                                                    <span>{{ getTemplateCategoryName(element) }}</span>
                                                </template>
                                            </span>
                                        </div>
                                    </div>
                                </td>
                                <td class="transaction-templates-column-status" v-if="isScheduledTransaction">
                                    <span class="transaction-templates-status" :class="`transaction-templates-status--${getStatusColor(element)}`">
                                        <span class="transaction-templates-status-dot" aria-hidden="true"></span>
                                        {{ getStatusDisplayName(element) }}
                                    </span>
                                    <span class="transaction-templates-meta" v-if="getNextOccurrenceDisplayText(element)">
                                        {{ getNextOccurrenceDisplayText(element) }}<template v-if="getRemainingOccurrencesDisplayText(element)"> ({{ getRemainingOccurrencesDisplayText(element) }})</template>
                                    </span>
                                </td>
                                <td class="transaction-templates-column-account">
                                    <div class="d-flex align-center">
                                        <span class="text-truncate">{{ getTemplateAccountName(element.sourceAccountId) }}</span>
                                        <template v-if="element.type === TransactionType.Transfer && element.destinationAccountId && element.destinationAccountId !== '0'">
                                            <v-icon class="icon-with-direction mx-1 transaction-templates-arrow" size="14" :icon="mdiArrowRight" :aria-label="tt('Destination Account')" role="img" />
                                            <span class="text-truncate">{{ getTemplateAccountName(element.destinationAccountId) }}</span>
                                        </template>
                                    </div>
                                </td>
                                <td class="transaction-templates-column-amount">
                                    <signed-amount :text="getTemplateAmount(element)" :type="getSignedAmountType(element)"
                                                   :hide-sign="element.hideAmount" v-if="getTemplateAmount(element)" />
                                </td>
                                <td class="transaction-templates-column-operation">
                                    <div class="transaction-templates-actions">
                                        <v-btn color="default" density="comfortable" variant="text" :icon="true"
                                               :aria-label="element.hidden ? tt('Show') : tt('Hide')"
                                               :loading="templateHiding[element.id]"
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
                                               :loading="templateRemoving[element.id]"
                                               :disabled="loading || updating"
                                               @click="remove(element)">
                                            <template #loader>
                                                <v-progress-circular indeterminate size="20" width="2"/>
                                            </template>
                                            <v-icon :icon="mdiDeleteOutline" size="20" />
                                            <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
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

    <edit-dialog ref="editDialog" :type="TransactionEditPageType.Template" />

    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SignedAmount from '@/components/common/SignedAmount.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import EditDialog from '@/views/desktop/transactions/list/dialogs/EditDialog.vue';
import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useScheduledTransactionListPageBase } from '@/views/base/templates/ScheduledTransactionListPageBase.ts';

import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { TemplateType } from '@/core/template.ts';
import { TransactionType } from '@/core/transaction.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';

import {
    isNoAvailableTemplate,
    getAvailableTemplateCount
} from '@/lib/template.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';

import {
    mdiRefresh,
    mdiPencilOutline,
    mdiEyeOffOutline,
    mdiEyeOutline,
    mdiDeleteOutline,
    mdiDrag,
    mdiDotsVertical,
    mdiTextBoxOutline,
    mdiClockTimeNineOutline,
    mdiMenuDown,
    mdiPlus,
    mdiArrowRight
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type EditDialogType = InstanceType<typeof EditDialog>;

const props = defineProps<{
    initType: number;
}>();

const { tt, formatNumberToLocalizedNumerals, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const transactionTemplatesStore = useTransactionTemplatesStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const editDialog = useTemplateRef<EditDialogType>('editDialog');

const templateType = ref<number>(TemplateType.Normal.type);
const loading = ref<boolean>(true);
const updating = ref<boolean>(false);
const loadError = ref<boolean>(false);
const templateHiding = ref<Record<string, boolean>>({});
const templateRemoving = ref<Record<string, boolean>>({});
const displayOrderModified = ref<boolean>(false);
const showHidden = ref<boolean>(false);

const templates = computed<TransactionTemplate[]>(() => transactionTemplatesStore.allTransactionTemplates[templateType.value] || []);
const noAvailableTemplate = computed<boolean>(() => isNoAvailableTemplate(templates.value, showHidden.value));
const tableColumnCount = computed<number>(() => isScheduledTransaction.value ? 5 : 4);
const loadErrorTitle = computed<string>(() => tt('Unable to retrieve template list'));

const emptyStateTitle = computed<string>(() => {
    if (templateType.value === TemplateType.Normal.type) {
        return tt('No available template. Once you add templates, you can quickly add a new transaction using the dropdown menu of the Add button on the transaction list page');
    } else if (templateType.value === TemplateType.Schedule.type) {
        return tt('No available scheduled transactions');
    } else {
        return tt('No available template');
    }
});

const availableTemplateCount = computed<number>(() => getAvailableTemplateCount(templates.value, showHidden.value));

const {
    statusFilter,
    isScheduledTransaction,
    canReorderByStatusFilter,
    allStatusFilterTypes,
    statusFilterDisplayName,
    getStatusDisplayName,
    getStatusColor,
    getNextOccurrenceDisplayText,
    getRemainingOccurrencesDisplayText,
    isShownByStatusFilter
} = useScheduledTransactionListPageBase(templateType, templates);

function getTemplateTypeName(template: TransactionTemplate): string {
    switch (template.type) {
        case TransactionType.Income:
            return tt('Income');
        case TransactionType.Expense:
            return tt('Expense');
        case TransactionType.Transfer:
            return tt('Transfer');
        default:
            return tt('Transaction');
    }
}

function getTemplateCategoryName(template: TransactionTemplate): string {
    const categoryId = template.categoryId;

    if (!categoryId || categoryId === '0') {
        return '';
    }

    return transactionCategoriesStore.allTransactionCategoriesMap[categoryId]?.name ?? '';
}

function getTemplateAccountName(accountId?: string): string {
    if (!accountId || accountId === '0') {
        return '—';
    }

    return accountsStore.allAccountsMap[accountId]?.name ?? '—';
}

function getTemplateAmount(template: TransactionTemplate): string {
    const account = template.sourceAccountId ? accountsStore.allAccountsMap[template.sourceAccountId] : undefined;

    if (!account) {
        return '';
    }

    return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(template.sourceAmount), account.currency);
}

function getSignedAmountType(template: TransactionTemplate): 'income' | 'expense' | 'transfer' | 'none' {
    if (template.type === TransactionType.Income) {
        return 'income';
    } else if (template.type === TransactionType.Expense) {
        return 'expense';
    } else if (template.type === TransactionType.Transfer) {
        return 'transfer';
    }

    return 'none';
}

function loadReferenceData(): void {
    // account and category names are only displayed in the list, so failures here are not fatal
    accountsStore.loadAllAccounts({ force: false }).catch(() => undefined);
    transactionCategoriesStore.loadAllCategories({ force: false }).catch(() => undefined);
}

function init(): void {
    templateType.value = props.initType;
    loading.value = true;
    loadError.value = false;

    loadReferenceData();

    transactionTemplatesStore.loadAllTemplates({
        templateType: templateType.value,
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
}

function reload(): void {
    loading.value = true;
    loadReferenceData();

    transactionTemplatesStore.loadAllTemplates({
        templateType: templateType.value,
        force: true
    }).then(() => {
        loading.value = false;
        loadError.value = false;
        displayOrderModified.value = false;

        desktopPageStore.addToast('Template list has been updated');
    }).catch(error => {
        loading.value = false;

        if (error && error.isUpToDate) {
            loadError.value = false;
            displayOrderModified.value = false;
        } else {
            loadError.value = true;
        }

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function add(): void {
    editDialog.value?.open({
        templateType: templateType.value
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

function edit(template: TransactionTemplate): void {
    editDialog.value?.open({
        id: template.id,
        currentTemplate: template
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

function hide(template: TransactionTemplate, hidden: boolean): void {
    updating.value = true;
    templateHiding.value[template.id] = true;

    transactionTemplatesStore.hideTemplate({
        template: template,
        hidden: hidden
    }).then(() => {
        updating.value = false;
        templateHiding.value[template.id] = false;
    }).catch(error => {
        updating.value = false;
        templateHiding.value[template.id] = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(template: TransactionTemplate): void {
    confirmDialog.value?.open('Are you sure you want to delete this template?', { color: 'error' }).then(() => {
        updating.value = true;
        templateRemoving.value[template.id] = true;

        transactionTemplatesStore.deleteTemplate({
            template: template
        }).then(() => {
            updating.value = false;
            templateRemoving.value[template.id] = false;
        }).catch(error => {
            updating.value = false;
            templateRemoving.value[template.id] = false;

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

    transactionTemplatesStore.updateTemplateDisplayOrders({
        templateType: templateType.value
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
        desktopPageStore.addToast('Unable to move template');
        return;
    }

    transactionTemplatesStore.changeTemplateDisplayOrder({
        templateType: templateType.value,
        templateId: moveEvent.element.id,
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

<style>
.transaction-templates-header {
    flex-wrap: wrap;
    row-gap: var(--ebk-space-2);
    padding-block: var(--ebk-space-2);
}

.transaction-templates-header-label {
    margin: 0;
    color: var(--ebk-color-text-primary);
    font: inherit;
}

.transaction-templates-header-meta {
    color: var(--ebk-color-text-secondary);
    font-weight: 400;
}

.transaction-templates-header-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: var(--ebk-space-2);
}

.v-table.transaction-templates-table > .v-table__wrapper > table {
    th,
    td {
        white-space: nowrap;
    }

    .transaction-templates-column-name {
        width: 100%;
        max-width: 0;
    }

    .transaction-templates-column-amount {
        min-width: 120px;
        text-align: end;
    }

    .transaction-templates-column-operation {
        width: 1%;
        padding-inline: var(--ebk-space-2);
    }
}

.transaction-templates-table-row > td {
    height: var(--ebk-density-table-row-height) !important;
    padding-block: 4px !important;
}

.transaction-templates-table-row--hidden .transaction-templates-name,
.transaction-templates-table-row--hidden .transaction-templates-icon-tile {
    opacity: 0.6;
}

.transaction-templates-drag {
    display: inline-flex;
    margin-inline-end: var(--ebk-space-2);
    color: var(--ebk-color-text-secondary);

    &.drag-handle {
        cursor: grab;
    }

    &.disabled {
        opacity: 0.35;
    }
}

.transaction-templates-icon-tile {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--ebk-color-primary-text);
}

.transaction-templates-icon-tile--skeleton {
    border-style: dashed;
    background-color: transparent;
}

.transaction-templates-skeleton .v-skeleton-loader__text {
    margin: 0;
}

.transaction-templates-name {
    display: flex;
    flex-direction: column;
    min-width: 0;
    margin-inline-start: var(--ebk-space-3);
    line-height: 1.25rem;
}

.transaction-templates-name-text {
    overflow: hidden;
    color: var(--ebk-color-text-primary);
    font-weight: 500;
    text-overflow: ellipsis;
}

.transaction-templates-meta {
    display: block;
    color: var(--ebk-color-text-secondary);
    font-size: 0.75rem;
    line-height: 1rem;
}

.transaction-templates-hidden-chip {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    margin-inline-start: var(--ebk-space-2);
    padding-inline: var(--ebk-space-2);
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-pill);
    color: var(--ebk-color-text-secondary);
    font-size: 0.6875rem;
    line-height: 1.125rem;
}

.transaction-templates-arrow {
    color: var(--ebk-color-text-secondary);
}

.transaction-templates-column-amount .ebk-signed-amount {
    font-weight: 600;
}

/* Scheduled transaction status badge (#14): label plus colour, never colour alone */
.transaction-templates-status {
    --ebk-template-status-color: var(--ebk-color-text-secondary);
    display: inline-flex;
    align-items: center;
    gap: var(--ebk-space-1);
    padding: 1px var(--ebk-space-2);
    border: 1px solid color-mix(in srgb, var(--ebk-template-status-color) 45%, transparent);
    border-radius: var(--ebk-radius-pill);
    background-color: color-mix(in srgb, var(--ebk-template-status-color) 10%, transparent);
    color: var(--ebk-color-text-primary);
    font-size: 0.75rem;
    font-weight: 500;
    line-height: 1.125rem;
}

.transaction-templates-status--green {
    --ebk-template-status-color: var(--ebk-color-success);
}

.transaction-templates-status--blue {
    --ebk-template-status-color: var(--ebk-color-info);
}

.transaction-templates-status--orange {
    --ebk-template-status-color: var(--ebk-color-warning);
}

.transaction-templates-status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: var(--ebk-template-status-color);
}

.transaction-templates-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
    opacity: 0;
    transition: opacity var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);
}

.transaction-templates-table-row:hover .transaction-templates-actions,
.transaction-templates-table-row:focus-within .transaction-templates-actions {
    opacity: 1;
}

@media (hover: none) {
    .transaction-templates-actions {
        opacity: 1;
    }
}

.transaction-templates-state-cell {
    height: auto !important;
    padding: var(--ebk-space-6) var(--ebk-space-4) !important;
    white-space: normal !important;
}

@media (forced-colors: active) {
    .transaction-templates-status,
    .transaction-templates-icon-tile {
        border-color: CanvasText;
    }

    .transaction-templates-status-dot {
        forced-color-adjust: none;
    }
}
</style>

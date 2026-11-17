<template>
    <f7-page :ptr="!sortable" @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')" v-if="!sortable"></f7-nav-left>
            <f7-nav-left v-else-if="sortable">
                <f7-link icon-f7="xmark" :class="{ 'disabled': displayOrderSaving }" :aria-label="tt('Cancel')" @click="cancelSort"></f7-link>
            </f7-nav-left>
            <f7-nav-title :title="templateType === TemplateType.Schedule.type ? tt('Scheduled Transactions') : tt('Transaction Templates')"></f7-nav-title>
            <f7-nav-right :class="{ 'navbar-compact-icons': true, 'disabled': loading }">
                <f7-link icon-f7="ellipsis" :class="{ 'disabled': !templates.length || sortable }" :aria-label="tt('More')" @click="showMoreActionSheet = true"></f7-link>
                <f7-link icon-f7="plus" :aria-label="tt('Add')" :href="'/template/add?templateType=' + templateType" v-if="!sortable"></f7-link>
                <f7-link icon-f7="checkmark_alt" :class="{ 'disabled': displayOrderSaving || !displayOrderModified }" :aria-label="tt('Save')" @click="saveSortResult" v-else-if="sortable"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-list strong inset dividers class="margin-top-half skeleton-text" v-if="loading">
            <f7-list-item title="Template Name"
                          :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                <template #media>
                    <span class="template-icon-tile">
                        <f7-icon f7="app_fill"></f7-icon>
                    </span>
                </template>
            </f7-list-item>
        </f7-list>

        <div class="template-list-state margin-top-half" v-if="!loading && noAvailableTemplate">
            <state-panel type="empty" :title="tt('No available template')"
                         :text="tt('Once you add templates, you can long-press the Add button on the home page to quickly add a new transaction')"
                         v-if="templateType === TemplateType.Normal.type">
                <template #actions>
                    <f7-button tonal :text="tt('Add')" :href="'/template/add?templateType=' + templateType"></f7-button>
                </template>
            </state-panel>
            <state-panel type="empty" :title="tt('No available scheduled transactions')" v-else-if="templateType === TemplateType.Schedule.type">
                <template #actions>
                    <f7-button tonal :text="tt('Add')" :href="'/template/add?templateType=' + templateType"></f7-button>
                </template>
            </state-panel>
            <state-panel type="empty" :title="tt('No available template')" v-else />
        </div>

        <f7-list strong inset dividers class="margin-top-half margin-bottom-half"
                 v-if="!loading && !noAvailableTemplate && isScheduledTransaction && !sortable">
            <f7-list-item link="#" popover-open=".schedule-status-filter-popover-menu"
                          :title="tt('Status')" :after="statusFilterDisplayName">
            </f7-list-item>
        </f7-list>

        <f7-popover class="schedule-status-filter-popover-menu">
            <f7-list dividers>
                <f7-list-item link="#" no-chevron popover-close
                              :title="option.displayName"
                              :class="{ 'list-item-selected': statusFilter === option.type }"
                              :key="option.type"
                              v-for="option in allStatusFilterTypes"
                              @click="statusFilter = option.type">
                    <template #after>
                        <f7-icon class="list-item-checked-icon" f7="checkmark_alt" v-if="statusFilter === option.type"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>

        <f7-list strong inset dividers sortable class="margin-top-half template-list"
                 :sortable-enabled="sortable"
                 v-if="!loading"
                 @sortable:sort="onSort">
            <f7-list-item swipeout
                          :class="{ 'actual-first-child': template.id === firstShowingId, 'actual-last-child': template.id === lastShowingId }"
                          :id="getTemplateDomId(template)"
                          :title="template.name"
                          :footer="isScheduledTransaction ? getScheduleOccurrenceDisplayText(template) : undefined"
                          :key="template.id"
                          v-for="template in templates"
                          v-show="(showHidden || !template.hidden) && isShownByStatusFilter(template)"
                          @taphold="setSortable()">
                <template #after v-if="isScheduledTransaction">
                    <span class="template-status-badge" :class="`template-status-badge--${getStatusColor(template)}`">
                        <span class="template-status-badge-dot" aria-hidden="true"></span>
                        <span>{{ getStatusDisplayName(template) }}</span>
                    </span>
                </template>
                <template #media>
                    <span class="template-icon-tile">
                        <f7-icon :f7="templateType === TemplateType.Schedule.type ? 'clock' : 'doc_plaintext'">
                            <f7-badge color="gray" class="right-bottom-icon" v-if="template.hidden">
                                <f7-icon f7="eye_slash_fill"></f7-icon>
                            </f7-badge>
                        </f7-icon>
                    </span>
                </template>
                <f7-swipeout-actions :left="textDirection === TextDirection.LTR"
                                     :right="textDirection === TextDirection.RTL"
                                     v-if="sortable">
                    <f7-swipeout-button class="padding-horizontal" overswipe close
                                        :aria-label="template.hidden ? tt('Show') : tt('Hide')"
                                        :color="template.hidden ? 'blue' : 'gray'"
                                        @click="hide(template, !template.hidden)">
                        <f7-icon :f7="template.hidden ? 'eye' : 'eye_slash'"></f7-icon>
                    </f7-swipeout-button>
                </f7-swipeout-actions>
                <f7-swipeout-actions :left="textDirection === TextDirection.RTL"
                                     :right="textDirection === TextDirection.LTR"
                                     v-if="!sortable">
                    <f7-swipeout-button color="orange" close :text="tt('Edit')" @click="edit(template)"></f7-swipeout-button>
                    <f7-swipeout-button color="red" class="padding-horizontal" :aria-label="tt('Delete')" @click="remove(template, false)">
                        <f7-icon f7="trash"></f7-icon>
                    </f7-swipeout-button>
                </f7-swipeout-actions>
            </f7-list-item>
        </f7-list>

        <f7-actions close-by-outside-click close-on-escape :opened="showMoreActionSheet" @actions:closed="showMoreActionSheet = false">
            <f7-actions-group>
                <f7-actions-button :class="{ 'disabled': !templates || templates.length < 2 }" @click="setSortable()">{{ tt('Sort') }}</f7-actions-button>
                <f7-actions-button v-if="!showHidden" @click="showHidden = true">{{ tt('Show Hidden Transaction Templates') }}</f7-actions-button>
                <f7-actions-button v-if="showHidden" @click="showHidden = false">{{ tt('Hide Hidden Transaction Templates') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-actions close-by-outside-click close-on-escape :opened="showDeleteActionSheet" @actions:closed="showDeleteActionSheet = false">
            <f7-actions-group>
                <f7-actions-label>{{ tt('Are you sure you want to delete this template?') }}</f7-actions-label>
                <f7-actions-button color="red" @click="remove(templateToDelete, true)">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>
    </f7-page>
</template>

<script setup lang="ts">
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading, onSwipeoutDeleted } from '@/lib/ui/mobile.ts';
import { useScheduledTransactionListPageBase } from '@/views/base/templates/ScheduledTransactionListPageBase.ts';

import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';

import { TextDirection } from '@/core/text.ts';
import { TemplateType } from '@/core/template.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';

import { isDefined } from '@/lib/common.ts';
import {
    isNoAvailableTemplate,
    getFirstShowingId,
    getLastShowingId
} from '@/lib/template.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt, getCurrentLanguageTextDirection } = useI18n();
const { showAlert, showToast, routeBackOnError } = useI18nUIComponents();

const transactionTemplatesStore = useTransactionTemplatesStore();

const templateType = ref<number>(TemplateType.Normal.type);
const loading = ref<boolean>(true);
const loadingError = ref<unknown | null>(null);
const showHidden = ref<boolean>(false);
const sortable = ref<boolean>(false);
const templateToDelete = ref<TransactionTemplate | null>(null);
const showMoreActionSheet = ref<boolean>(false);
const showDeleteActionSheet = ref<boolean>(false);
const displayOrderModified = ref<boolean>(false);
const displayOrderSaving = ref<boolean>(false);

const textDirection = computed<TextDirection>(() => getCurrentLanguageTextDirection());
const templates = computed<TransactionTemplate[]>(() => transactionTemplatesStore.allTransactionTemplates[templateType.value] || []);
const noAvailableTemplate = computed<boolean>(() => isNoAvailableTemplate(templates.value, showHidden.value));

const {
    statusFilter,
    isScheduledTransaction,
    allStatusFilterTypes,
    statusFilterDisplayName,
    getStatusDisplayName,
    getStatusColor,
    getNextOccurrenceDisplayText,
    getRemainingOccurrencesDisplayText,
    isShownByStatusFilter
} = useScheduledTransactionListPageBase(templateType, templates);

const statusFilteredTemplates = computed<TransactionTemplate[]>(() => templates.value.filter(template => isShownByStatusFilter(template)));
const firstShowingId = computed<string | null>(() => getFirstShowingId(statusFilteredTemplates.value, showHidden.value));
const lastShowingId = computed<string | null>(() => getLastShowingId(statusFilteredTemplates.value, showHidden.value));

function getScheduleOccurrenceDisplayText(template: TransactionTemplate): string {
    const nextOccurrence = getNextOccurrenceDisplayText(template);
    const remainingOccurrences = getRemainingOccurrencesDisplayText(template);

    if (nextOccurrence && remainingOccurrences) {
        return `${nextOccurrence} (${remainingOccurrences})`;
    }

    return nextOccurrence || remainingOccurrences;
}

function getTemplateDomId(template: TransactionTemplate): string {
    return 'template_' + template.id;
}

function parseTemplateIdFromDomId(domId: string): string | null {
    if (!domId || domId.indexOf('template_') !== 0) {
        return null;
    }

    return domId.substring(9); // template_
}

function init(): void {
    if (props.f7route.path === '/template/list') {
        templateType.value = TemplateType.Normal.type;
    } else if (props.f7route.path === '/schedule/list') {
        templateType.value = TemplateType.Schedule.type;
    }

    loading.value = true;

    transactionTemplatesStore.loadAllTemplates({
        templateType: templateType.value,
        force: false
    }).then(() => {
        loading.value = false;
    }).catch(error => {
        if (error.processed) {
            loading.value = false;
        } else {
            loadingError.value = error;
            showToast(error.message || error);
        }
    });
}

function reload(done?: () => void): void {
    if (sortable.value) {
        done?.();
        return;
    }

    const force = !!done;

    transactionTemplatesStore.loadAllTemplates({
        templateType: templateType.value,
        force: force
    }).then(() => {
        done?.();

        if (force) {
            showToast('Template list has been updated');
        }
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function edit(template: TransactionTemplate): void {
    props.f7router.navigate(`/template/edit?id=${template.id}&templateType=${template.templateType}`);
}

function hide(template: TransactionTemplate, hidden: boolean): void {
    showLoading();

    transactionTemplatesStore.hideTemplate({
        template: template,
        hidden: hidden
    }).then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function remove(template: TransactionTemplate | null, confirm: boolean): void {
    if (!template) {
        showAlert('An error occurred');
        return;
    }

    if (!confirm) {
        templateToDelete.value = template;
        showDeleteActionSheet.value = true;
        return;
    }

    showDeleteActionSheet.value = false;
    templateToDelete.value = null;
    showLoading();

    transactionTemplatesStore.deleteTemplate({
        template: template,
        beforeResolve: (done) => {
            onSwipeoutDeleted(getTemplateDomId(template), done);
        }
    }).then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function setSortable(): void {
    if (sortable.value) {
        return;
    }

    showHidden.value = true;
    statusFilter.value = 'all';
    sortable.value = true;
    displayOrderModified.value = false;
}

function saveSortResult(): void {
    if (!displayOrderModified.value) {
        showHidden.value = false;
        sortable.value = false;
        return;
    }

    displayOrderSaving.value = true;
    showLoading();

    transactionTemplatesStore.updateTemplateDisplayOrders({
        templateType: templateType.value
    }).then(() => {
        displayOrderSaving.value = false;
        hideLoading();

        showHidden.value = false;
        sortable.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        displayOrderSaving.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function cancelSort(): void {
    if (!displayOrderModified.value) {
        showHidden.value = false;
        sortable.value = false;
        return;
    }

    displayOrderSaving.value = true;
    showLoading();

    transactionTemplatesStore.loadAllTemplates({
        templateType: templateType.value,
        force: false
    }).then(() => {
        displayOrderSaving.value = false;
        hideLoading();

        showHidden.value = false;
        sortable.value = false;
        displayOrderModified.value = false;
    }).catch(error => {
        displayOrderSaving.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function onSort(event: { el: { id: string }; from: number; to: number }): void {
    if (!event || !event.el || !event.el.id) {
        showToast('Unable to move template');
        return;
    }

    const id = parseTemplateIdFromDomId(event.el.id);

    if (!id) {
        showToast('Unable to move template');
        return;
    }

    transactionTemplatesStore.changeTemplateDisplayOrder({
        templateType: templateType.value,
        templateId: id,
        from: event.from,
        to: event.to
    }).then(() => {
        displayOrderModified.value = true;
    }).catch(error => {
        showToast(error.message || error);
    });
}

function onPageAfterIn(): void {
    if ((!isDefined(transactionTemplatesStore.transactionTemplateListStatesInvalid[templateType.value]) || transactionTemplatesStore.transactionTemplateListStatesInvalid[templateType.value]) && !loading.value) {
        reload();
    }

    routeBackOnError(props.f7router, loadingError);
}

init();
</script>

<style>
.template-list-state {
    margin-inline: var(--f7-list-inset-side-margin, 16px);
}

.template-list li > .item-content,
.template-list li > .swipeout-content > .item-content {
    min-height: 52px;
}

.template-icon-tile {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--ebk-color-text-secondary);
}

.template-icon-tile > .icon {
    font-size: 20px;
}

/* Scheduled transaction status (#14): label plus colour, never colour alone */
.template-status-badge {
    --template-status-color: var(--ebk-color-text-secondary);
    display: inline-flex;
    align-items: center;
    gap: 6px;
    max-width: 100%;
    padding: 2px 10px;
    border: 1px solid color-mix(in srgb, var(--template-status-color) 45%, transparent);
    border-radius: var(--ebk-radius-pill);
    background-color: color-mix(in srgb, var(--template-status-color) 12%, transparent);
    color: var(--ebk-color-text-primary);
    font-size: 0.75rem;
    font-weight: 600;
    line-height: 1.4;
    white-space: nowrap;
}

.template-status-badge--green {
    --template-status-color: var(--ebk-color-success);
}

.template-status-badge--blue {
    --template-status-color: var(--ebk-color-info);
}

.template-status-badge--orange {
    --template-status-color: var(--ebk-color-warning);
}

.template-status-badge-dot {
    flex: 0 0 auto;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--template-status-color);
}

@media (forced-colors: active) {
    .template-status-badge {
        border-color: CanvasText;
    }

    .template-status-badge-dot {
        background-color: CanvasText;
    }
}

.template-list {
    --f7-list-item-footer-font-size: var(--ebk-large-footer-font-size);
}

.template-list .item-footer {
    padding-top: 4px;
}
</style>

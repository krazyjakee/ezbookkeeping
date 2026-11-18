<template>
    <v-dialog width="1000" :persistent="loading || submitting || recognizing || isTransactionModified" v-model="showState">
        <two-column-dialog-layout :disabled="loading || submitting || recognizing" :loading="loading"
                                  :title="tt(title)" :cancel-button-title="tt(cancelButtonTitle)"
                                  @cancel="cancel">
            <template #after-title>
                <v-btn density="compact" color="default" variant="text" class="ms-2" :icon="true"
                       :aria-label="tt('Edit')" :disabled="loading || submitting || recognizing"
                       v-if="mode === TransactionEditPageMode.View && originalTransactionEditable"
                       @click="edit">
                    <v-icon :icon="mdiPencilOutline" size="22"/>
                    <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                </v-btn>
                <v-btn density="compact" color="default" variant="text" class="ms-2" :icon="true"
                       :aria-label="tt('AI Clipboard Text Recognition')" :disabled="loading || submitting || recognizing"
                       v-if="mode !== TransactionEditPageMode.View && type === TransactionEditPageType.Transaction && activeTab === 'basicInfo' && isTransactionFromAITextRecognitionEnabled()"
                       @click="recognizeFromClipboard">
                    <v-icon :icon="mdiMagicStaff" size="20" v-if="!recognizing"/>
                    <v-tooltip activator="parent">{{ tt('AI Clipboard Text Recognition') }}</v-tooltip>
                    <v-progress-circular indeterminate size="22" v-if="recognizing"></v-progress-circular>
                </v-btn>
                <small class="ms-2 text-truncate" v-if="recognizing">{{ tt('AI can make mistakes. Check important info.') }}</small>
            </template>

            <template #toolbar>
                <v-btn density="compact" color="default" variant="text" class="ms-2" :icon="true"
                       :aria-label="tt('More')" :disabled="loading || submitting || recognizing"
                       v-if="mode !== TransactionEditPageMode.View && (activeTab === 'basicInfo' || (activeTab === 'map' && isSupportGetGeoLocationByClick()))">
                    <v-icon :icon="mdiDotsVertical" size="22" />
                    <v-menu activator="parent">
                        <v-list v-if="activeTab === 'basicInfo'">
                            <v-list-item :prepend-icon="mdiSwapHorizontal"
                                         :title="tt('Swap Account')"
                                         v-if="transaction.type === TransactionType.Transfer"
                                         @click="swapTransactionData(true, false)"></v-list-item>
                            <v-list-item :prepend-icon="mdiSwapHorizontal"
                                         :title="tt('Swap Amount')"
                                         v-if="transaction.type === TransactionType.Transfer"
                                         @click="swapTransactionData(false, true)"></v-list-item>
                            <v-list-item :prepend-icon="mdiSwapHorizontal"
                                         :title="tt('Swap Account and Amount')"
                                         v-if="transaction.type === TransactionType.Transfer"
                                         @click="swapTransactionData(true, true)"></v-list-item>
                            <v-divider v-if="transaction.type === TransactionType.Transfer" />
                            <v-list-item :prepend-icon="mdiEyeOutline"
                                         :title="tt('Show Amount')"
                                         v-if="transaction.hideAmount" @click="transaction.hideAmount = false"></v-list-item>
                            <v-list-item :prepend-icon="mdiEyeOffOutline"
                                         :title="tt('Hide Amount')"
                                         v-if="!transaction.hideAmount" @click="transaction.hideAmount = true"></v-list-item>
                        </v-list>
                        <v-list v-if="activeTab === 'map'">
                            <v-list-item @click="setGeoLocationByClickMap = !setGeoLocationByClickMap; geoMenuState = false" key="setGeoLocationByClickMap" value="setGeoLocationByClickMap"
                                         :prepend-icon="mdiMapMarkerOutline"
                                         :disabled="!transaction.geoLocation" v-if="isSupportGetGeoLocationByClick()">
                                <v-list-item-title class="cursor-pointer">
                                    <div class="d-flex align-center">
                                        <span>{{ tt('Click on Map to Set Geographic Location') }}</span>
                                        <v-spacer/>
                                        <v-icon :icon="mdiCheck" v-if="setGeoLocationByClickMap" />
                                    </div>
                                </v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </v-btn>
            </template>

            <template #content-left-column>
                <div class="px-4">
                    <v-tabs direction="vertical" :disabled="loading || submitting || recognizing" v-model="activeTab">
                        <v-tab value="basicInfo">
                            <span>{{ tt('Basic Information') }}</span>
                        </v-tab>
                        <v-tab value="map" :disabled="!transaction.geoLocation" v-if="type === TransactionEditPageType.Transaction && !!getMapProvider()">
                            <span>{{ tt('Location on Map') }}</span>
                        </v-tab>
                        <v-tab value="pictures" v-if="type === TransactionEditPageType.Transaction && isTransactionPicturesEnabled()">
                            <span>{{ tt('Documents') }}</span>
                        </v-tab>
                    </v-tabs>
                </div>
            </template>

            <template #content-right-column>
                <div class="transaction-edit-main d-flex flex-column flex-grow-1">
                <div class="transaction-edit-type-bar">
                    <v-tabs class="v-tabs-pill transaction-edit-type-tabs" density="compact" :aria-label="tt('Transaction Type')" :class="{ 'readonly': type === TransactionEditPageType.Transaction && mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit }"
                            :disabled="loading || submitting || recognizing" v-model="transaction.type">
                        <v-tab :value="TransactionType.Expense" :disabled="type === TransactionEditPageType.Transaction && mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && transaction.type !== TransactionType.Expense" v-if="transaction.type !== TransactionType.ModifyBalance">
                            <span>{{ tt('Expense') }}</span>
                        </v-tab>
                        <v-tab :value="TransactionType.Income" :disabled="type === TransactionEditPageType.Transaction && mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && transaction.type !== TransactionType.Income" v-if="transaction.type !== TransactionType.ModifyBalance">
                            <span>{{ tt('Income') }}</span>
                        </v-tab>
                        <v-tab :value="TransactionType.Transfer" :disabled="type === TransactionEditPageType.Transaction && mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && transaction.type !== TransactionType.Transfer" v-if="transaction.type !== TransactionType.ModifyBalance">
                            <span>{{ tt('Transfer') }}</span>
                        </v-tab>
                        <v-tab :value="TransactionType.ModifyBalance" v-if="type === TransactionEditPageType.Transaction && transaction.type === TransactionType.ModifyBalance">
                            <span>{{ tt('Modify Balance') }}</span>
                        </v-tab>
                    </v-tabs>
                </div>
                <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container"
                          v-model="activeTab">
                    <v-window-item value="basicInfo">
                        <v-form class="my-4">
                            <v-row>
                                <v-col cols="12" v-if="type === TransactionEditPageType.Template && transaction instanceof TransactionTemplate">
                                    <v-text-field
                                        type="text"
                                        autocomplete="off"
                                        persistent-placeholder
                                        :disabled="loading || submitting || recognizing"
                                        :label="tt('Template Name')"
                                        :placeholder="tt('Template Name')"
                                        v-model="transaction.name"
                                    />
                                </v-col>
                                <v-col class="transaction-edit-amount-col" cols="12" :md="transaction.type === TransactionType.Transfer ? 6 : 12">
                                    <amount-input class="transaction-edit-amount font-weight-bold"
                                                  :color="sourceAmountColor"
                                                  :currency="sourceAccountCurrency"
                                                  :show-currency="true"
                                                  :readonly="mode === TransactionEditPageMode.View"
                                                  :disabled="loading || submitting || recognizing"
                                                  :persistent-placeholder="true"
                                                  :hide="transaction.hideAmount"
                                                  :label="sourceAmountTitle"
                                                  :placeholder="tt(sourceAmountName)"
                                                  :enable-formula="mode !== TransactionEditPageMode.View"
                                                  v-model="transaction.sourceAmount"/>
                                </v-col>
                                <v-col class="transaction-edit-amount-col" cols="12" :md="6" v-if="transaction.type === TransactionType.Transfer">
                                    <amount-input class="transaction-edit-amount font-weight-bold" color="primary"
                                                  :currency="destinationAccountCurrency"
                                                  :show-currency="true"
                                                  :readonly="mode === TransactionEditPageMode.View"
                                                  :disabled="loading || submitting || recognizing"
                                                  :persistent-placeholder="true"
                                                  :hide="transaction.hideAmount"
                                                  :label="transferInAmountTitle"
                                                  :placeholder="tt('Transfer In Amount')"
                                                  :enable-formula="mode !== TransactionEditPageMode.View"
                                                  v-model="transaction.destinationAmount"/>
                                </v-col>
                                <v-col cols="12" md="6" v-if="type === TransactionEditPageType.Transaction">
                                    <date-time-select
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing || (mode === TransactionEditPageMode.Edit && transaction.type === TransactionType.ModifyBalance)"
                                        :label="tt('Transaction Time')"
                                        :timezone-utc-offset="transaction.utcOffset"
                                        :model-value="transaction.time"
                                        @update:model-value="updateTransactionTime"
                                        @error="onShowDateTimeError" />
                                </v-col>
                                <v-col cols="12" md="6" v-if="type === TransactionEditPageType.Template && transaction instanceof TransactionTemplate && transaction.templateType === TemplateType.Schedule.type">
                                    <schedule-frequency-select
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :label="tt('Scheduled Transaction Frequency')"
                                        v-model:type="transaction.scheduledFrequencyType"
                                        v-model="transaction.scheduledFrequency" />
                                </v-col>
                                <v-col cols="12" md="6" v-if="type === TransactionEditPageType.Transaction || (type === TransactionEditPageType.Template && transaction instanceof TransactionTemplate && transaction.templateType === TemplateType.Schedule.type)">
                                    <v-autocomplete
                                        class="transaction-edit-timezone"
                                        item-title="displayNameWithUtcOffset"
                                        item-value="name"
                                        auto-select-first
                                        persistent-placeholder
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing || (mode === TransactionEditPageMode.Edit && transaction.type === TransactionType.ModifyBalance)"
                                        :label="tt('Transaction Timezone')"
                                        :placeholder="!transaction.timeZone && transaction.timeZone !== '' ? `(${transactionDisplayTimezone}) ${type === TransactionEditPageType.Transaction ? transactionTimezoneTimeDifference : ''}` : tt('Timezone')"
                                        :items="allTimezones"
                                        :no-data-text="tt('No results')"
                                        :model-value="transaction.timeZone"
                                        @update:model-value="updateTransactionTimezone"
                                    />
                                </v-col>
                                <v-col cols="12" md="6" v-if="type === TransactionEditPageType.Template && transaction instanceof TransactionTemplate && transaction.templateType === TemplateType.Schedule.type">
                                    <date-select
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :clearable="true"
                                        :label="tt('Start Date')"
                                        :no-data-text="tt('No limit')"
                                        v-model="transaction.scheduledStartDate" />
                                </v-col>
                                <v-col cols="12" md="6" v-if="type === TransactionEditPageType.Template && transaction instanceof TransactionTemplate && transaction.templateType === TemplateType.Schedule.type">
                                    <date-select
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :clearable="true"
                                        :label="tt('End Date')"
                                        :no-data-text="tt('No limit')"
                                        v-model="transaction.scheduledEndDate" />
                                </v-col>
                                <v-col cols="12" md="6">
                                    <v-tooltip :disabled="!!allVisibleAccounts.length" :text="allVisibleAccounts.length ? '' : tt('No available account')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="category"
                                                                   primary-title-field="name" primary-footer-field="displayBalance"
                                                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="account"
                                                                   primary-sub-items-field="accounts"
                                                                   :primary-title-i18n="true"
                                                                   secondary-key-field="id" secondary-value-field="id"
                                                                   secondary-title-field="name" secondary-footer-field="displayBalance"
                                                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="account" secondary-color-field="color"
                                                                   :readonly="mode === TransactionEditPageMode.View"
                                                                   :disabled="loading || submitting || recognizing || !allVisibleAccounts.length || (mode === TransactionEditPageMode.Edit && transaction.type === TransactionType.ModifyBalance)"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find account')" :filter-no-items-text="tt('No available account')"
                                                                   :custom-selection-primary-text="sourceAccountName"
                                                                   :label="tt(sourceAccountTitle)"
                                                                   :placeholder="tt(sourceAccountTitle)"
                                                                   :items="allVisibleCategorizedAccounts"
                                                                   v-model="transaction.sourceAccountId">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="6" v-if="transaction.type === TransactionType.Transfer">
                                    <v-tooltip :disabled="!!allVisibleAccounts.length" :text="allVisibleAccounts.length ? '' : tt('No available account')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="category"
                                                                   primary-title-field="name" primary-footer-field="displayBalance"
                                                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="account"
                                                                   primary-sub-items-field="accounts"
                                                                   :primary-title-i18n="true"
                                                                   secondary-key-field="id" secondary-value-field="id"
                                                                   secondary-title-field="name" secondary-footer-field="displayBalance"
                                                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="account" secondary-color-field="color"
                                                                   :readonly="mode === TransactionEditPageMode.View"
                                                                   :disabled="loading || submitting || recognizing || !allVisibleAccounts.length"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find account')" :filter-no-items-text="tt('No available account')"
                                                                   :custom-selection-primary-text="destinationAccountName"
                                                                   :label="tt('Destination Account')"
                                                                   :placeholder="tt('Destination Account')"
                                                                   :items="allVisibleCategorizedAccounts"
                                                                   v-model="transaction.destinationAccountId">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="6" v-if="transaction.type === TransactionType.Expense">
                                    <v-tooltip :disabled="hasVisibleExpenseCategories" :text="hasVisibleExpenseCategories ? '' : tt('No secondary expense categories are available')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="id" primary-title-field="name"
                                                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="category" primary-color-field="color"
                                                                   primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                                                   secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="category" secondary-color-field="color"
                                                                   secondary-hidden-field="hidden"
                                                                   :readonly="mode === TransactionEditPageMode.View"
                                                                   :disabled="loading || submitting || recognizing || (!hasVisibleExpenseCategories && !canAddNewCategory)"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                                                   :show-selection-primary-text="true"
                                                                   :custom-selection-primary-text="getTransactionPrimaryCategoryName(transaction.expenseCategoryId, allCategories[CategoryType.Expense])"
                                                                   :custom-selection-secondary-text="getTransactionSecondaryCategoryName(transaction.expenseCategoryId, allCategories[CategoryType.Expense])"
                                                                   :label="tt('Category')" :placeholder="tt('Category')"
                                                                   :items="allCategories[CategoryType.Expense] || []"
                                                                   :add-primary-item-text="canAddNewCategory ? tt('Add Primary Category') : undefined"
                                                                   :add-secondary-item-text="canAddNewCategory ? tt('Add Secondary Category') : undefined"
                                                                   v-model="transaction.expenseCategoryId"
                                                                   @add:primary="addNewCategory(CategoryType.Expense, undefined, $event)"
                                                                   @add:secondary="(primaryItem: unknown, name: string) => addNewCategory(CategoryType.Expense, primaryItem, name)">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="6" v-if="transaction.type === TransactionType.Income">
                                    <v-tooltip :disabled="hasVisibleIncomeCategories" :text="hasVisibleIncomeCategories ? '' : tt('No secondary income categories are available')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="id" primary-title-field="name"
                                                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="category" primary-color-field="color"
                                                                   primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                                                   secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="category" secondary-color-field="color"
                                                                   secondary-hidden-field="hidden"
                                                                   :readonly="mode === TransactionEditPageMode.View"
                                                                   :disabled="loading || submitting || recognizing || (!hasVisibleIncomeCategories && !canAddNewCategory)"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                                                   :show-selection-primary-text="true"
                                                                   :custom-selection-primary-text="getTransactionPrimaryCategoryName(transaction.incomeCategoryId, allCategories[CategoryType.Income])"
                                                                   :custom-selection-secondary-text="getTransactionSecondaryCategoryName(transaction.incomeCategoryId, allCategories[CategoryType.Income])"
                                                                   :label="tt('Category')" :placeholder="tt('Category')"
                                                                   :items="allCategories[CategoryType.Income] || []"
                                                                   :add-primary-item-text="canAddNewCategory ? tt('Add Primary Category') : undefined"
                                                                   :add-secondary-item-text="canAddNewCategory ? tt('Add Secondary Category') : undefined"
                                                                   v-model="transaction.incomeCategoryId"
                                                                   @add:primary="addNewCategory(CategoryType.Income, undefined, $event)"
                                                                   @add:secondary="(primaryItem: unknown, name: string) => addNewCategory(CategoryType.Income, primaryItem, name)">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="12" v-if="transaction.type === TransactionType.Transfer">
                                    <v-tooltip :disabled="hasVisibleTransferCategories" :text="hasVisibleTransferCategories ? '' : tt('No secondary transfer categories are available')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="id" primary-title-field="name"
                                                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="category" primary-color-field="color"
                                                                   primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                                                   secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="category" secondary-color-field="color"
                                                                   secondary-hidden-field="hidden"
                                                                   :readonly="mode === TransactionEditPageMode.View"
                                                                   :disabled="loading || submitting || recognizing || (!hasVisibleTransferCategories && !canAddNewCategory)"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                                                   :show-selection-primary-text="true"
                                                                   :custom-selection-primary-text="getTransactionPrimaryCategoryName(transaction.transferCategoryId, allCategories[CategoryType.Transfer])"
                                                                   :custom-selection-secondary-text="getTransactionSecondaryCategoryName(transaction.transferCategoryId, allCategories[CategoryType.Transfer])"
                                                                   :label="tt('Category')" :placeholder="tt('Category')"
                                                                   :items="allCategories[CategoryType.Transfer] || []"
                                                                   :add-primary-item-text="canAddNewCategory ? tt('Add Primary Category') : undefined"
                                                                   :add-secondary-item-text="canAddNewCategory ? tt('Add Secondary Category') : undefined"
                                                                   v-model="transaction.transferCategoryId"
                                                                   @add:primary="addNewCategory(CategoryType.Transfer, undefined, $event)"
                                                                   @add:secondary="(primaryItem: unknown, name: string) => addNewCategory(CategoryType.Transfer, primaryItem, name)">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="12">
                                    <v-textarea
                                        type="text"
                                        autocomplete="off"
                                        persistent-placeholder
                                        rows="3"
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :label="transactionDescriptionTitle"
                                        :placeholder="tt('Your transaction description (optional)')"
                                        v-model="transaction.comment"
                                    />
                                </v-col>
                                <v-col cols="12" md="12">
                                    <transaction-tag-auto-complete
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :show-label="true"
                                        :allow-add-new-tag="true"
                                        v-model="transaction.tagIds"
                                        @tag:saving="onSavingTag"
                                    />
                                </v-col>
                                <v-col cols="12" md="12" v-if="type === TransactionEditPageType.Transaction">
                                    <v-select
                                        persistent-placeholder
                                        :readonly="mode === TransactionEditPageMode.View"
                                        :disabled="loading || submitting || recognizing"
                                        :label="tt('Geographic Location')"
                                        v-model="transaction"
                                        v-model:menu="geoMenuState"
                                    >
                                        <template #selection>
                                            <span class="cursor-pointer" v-if="transaction.geoLocation">{{ `(${formatCoordinate(transaction.geoLocation)})` }}</span>
                                            <span class="cursor-pointer" v-else-if="!transaction.geoLocation">{{ geoLocationStatusInfo }}</span>
                                        </template>

                                        <template #no-data>
                                            <v-list class="py-0" @keydown.up.stop @keydown.down.stop>
                                                <v-list-item class="text-body-medium" v-if="mode !== TransactionEditPageMode.View" @click="updateGeoLocation(true)">{{ tt('Update Geographic Location') }}</v-list-item>
                                                <v-list-item class="text-body-medium" v-if="mode !== TransactionEditPageMode.View" @click="clearGeoLocation">{{ tt('Clear Geographic Location') }}</v-list-item>
                                            </v-list>
                                        </template>
                                    </v-select>
                                </v-col>
                                <v-col cols="12" v-if="type === TransactionEditPageType.Transaction && isTransactionPicturesEnabled() && (mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit || (transaction.pictures && transaction.pictures.length > 0))">
                                    <div class="transaction-edit-attach">
                                        <button type="button" class="transaction-edit-attach-dropzone"
                                                :class="{ 'transaction-edit-attach-dropzone--over': attachDragOver }"
                                                :disabled="!canAddTransactionPicture || loading || submitting || recognizing || uploadingPicture"
                                                :aria-busy="uploadingPicture ? 'true' : 'false'"
                                                v-if="mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit"
                                                @click="showOpenPictureDialog"
                                                @dragenter.prevent="onAttachDragOver"
                                                @dragover.prevent="onAttachDragOver"
                                                @dragleave="attachDragOver = false"
                                                @drop.prevent="onAttachDrop">
                                            <v-progress-circular indeterminate size="20" width="2" v-if="uploadingPicture" />
                                            <v-icon :icon="mdiPaperclip" size="20" v-else />
                                            <span class="transaction-edit-attach-text">{{ tt('Attach document — PDF, image, Office or text file') }}</span>
                                        </button>
                                        <v-btn class="transaction-edit-attach-count" color="default" variant="text" :prepend-icon="mdiFileDocumentMultipleOutline"
                                               :disabled="loading || submitting || recognizing"
                                               v-if="transaction.pictures && transaction.pictures.length > 0"
                                               @click="activeTab = 'pictures'">
                                            <span>{{ tt('Documents') }}</span>
                                            <span class="ms-1 ebk-numeric">({{ formatNumberToLocalizedNumerals(transaction.pictures.length) }})</span>
                                        </v-btn>
                                    </div>
                                </v-col>
                            </v-row>
                        </v-form>
                    </v-window-item>
                    <v-window-item value="map">
                        <map-view ref="map" map-class="transaction-edit-map-view mb-3 mb-sm-0"
                                  :enable-zoom-control="true" :geo-location="transaction.geoLocation"
                                  @click="updateSpecifiedGeoLocation">
                            <template #error-title="{ mapSupported, mapDependencyLoaded }">
                                <span class="text-body-large" v-if="!mapSupported"><b>{{ tt('Unsupported Map Provider') }}</b></span>
                                <span class="text-body-large" v-else-if="!mapDependencyLoaded"><b>{{ tt('Cannot Initialize Map') }}</b></span>
                            </template>
                            <template #error-content>
                                <span class="text-body-large">
                                    {{ tt('Please refresh the page and try again. If the error persists, ensure that the server\'s map settings are correctly configured.') }}
                                </span>
                            </template>
                        </map-view>
                    </v-window-item>
                    <v-window-item value="pictures">
                        <div class="transaction-pictures transaction-pictures--empty"
                             v-if="mode === TransactionEditPageMode.View && (!transaction.pictures || !transaction.pictures.length)">
                            <v-icon class="transaction-pictures-empty-icon" size="48" :icon="mdiFileDocumentMultipleOutline"/>
                            <div class="transaction-pictures-empty-title">{{ tt('No documents attached') }}</div>
                            <div class="transaction-pictures-empty-hint">{{ tt('Attach receipts, invoices or statements as proof of this transaction.') }}</div>
                            <v-btn class="mt-2" color="primary" variant="tonal" :prepend-icon="mdiFileDocumentPlusOutline"
                                   :disabled="loading || submitting || recognizing"
                                   v-if="originalTransactionEditable"
                                   @click="editAndAddDocument">{{ tt('Add Document') }}</v-btn>
                        </div>
                        <div class="transaction-pictures" :class="{ 'readonly': submitting || uploadingPicture || removingPictureId }" v-else>
                            <div class="transaction-pictures-grid">
                                <div class="transaction-picture-tile" :key="pictureInfo.pictureId || picIdx" v-for="(pictureInfo, picIdx) in transaction.pictures">
                                    <button type="button" class="transaction-picture-tile-preview"
                                            :aria-label="getPictureTileActionLabel(pictureInfo)"
                                            :title="getTransactionPictureKind(pictureInfo) !== 'image' ? getPictureDisplayName(pictureInfo) : undefined"
                                            :aria-busy="pictureInfo.pictureId === removingPictureId ? 'true' : 'false'"
                                            @click="viewOrRemovePicture(pictureInfo)">
                                        <span class="transaction-picture-tile-document"
                                              v-if="getTransactionPictureKind(pictureInfo) !== 'image'">
                                            <v-icon size="48" :icon="getTransactionAttachmentMdiIcon(getTransactionPictureExtension(pictureInfo))"/>
                                            <span class="transaction-picture-tile-document-name">{{ getPictureDisplayName(pictureInfo) }}</span>
                                            <span class="transaction-picture-tile-document-type" v-if="pictureInfo.originalFileName && getTransactionPictureExtension(pictureInfo)">{{ getTransactionPictureExtension(pictureInfo).toUpperCase() }}</span>
                                        </span>
                                        <v-img class="transaction-picture-tile-image" cover :src="getTransactionPictureUrl(pictureInfo)" v-else>
                                            <template #placeholder>
                                                <div class="transaction-picture-tile-placeholder">
                                                    <v-progress-circular indeterminate size="32"></v-progress-circular>
                                                </div>
                                            </template>
                                            <template #error>
                                                <div class="transaction-picture-tile-placeholder transaction-picture-tile-placeholder--error">
                                                    <v-icon size="28" :icon="mdiImageBrokenVariant" />
                                                    <span>{{ tt('Failed to load image, please check whether the config "domain" and "root_url" are set correctly.') }}</span>
                                                </div>
                                            </template>
                                        </v-img>
                                        <span class="transaction-picture-tile-overlay" :class="{ 'transaction-picture-tile-overlay--visible': pictureInfo.pictureId === removingPictureId }" aria-hidden="true">
                                            <span class="transaction-picture-tile-action" v-if="(mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit) && pictureInfo.pictureId !== removingPictureId">
                                                <v-icon size="22" :icon="mdiTrashCanOutline"/>
                                                <span>{{ tt('Remove Document') }}</span>
                                            </span>
                                            <v-progress-circular indeterminate size="32" v-if="(mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit) && pictureInfo.pictureId === removingPictureId"></v-progress-circular>
                                            <span class="transaction-picture-tile-action" v-if="mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && getTransactionPictureKind(pictureInfo) !== 'document'">
                                                <v-icon size="22" :icon="mdiFullscreen"/>
                                                <span>{{ tt('View') }}</span>
                                            </span>
                                            <span class="transaction-picture-tile-action" v-if="mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && getTransactionPictureKind(pictureInfo) === 'document'">
                                                <v-icon size="22" :icon="mdiDownload"/>
                                                <span>{{ tt('Download') }}</span>
                                            </span>
                                        </span>
                                    </button>
                                    <v-btn class="transaction-picture-download-button" density="comfortable" variant="flat" color="surface"
                                           :icon="mdiDownload" :aria-label="tt('Download')" :title="tt('Download')"
                                           @click.stop="downloadPicture(pictureInfo)"
                                           v-if="mode !== TransactionEditPageMode.Add && mode !== TransactionEditPageMode.Edit && getTransactionPictureKind(pictureInfo) !== 'image'"/>
                                </div>
                                <button type="button" class="transaction-picture-tile transaction-picture-add"
                                        :class="{ 'transaction-picture-add--over': attachDragOver }"
                                        :disabled="submitting || uploadingPicture"
                                        :aria-busy="uploadingPicture ? 'true' : 'false'"
                                        v-if="canAddTransactionPicture"
                                        @click="showOpenPictureDialog"
                                        @dragenter.prevent="onAttachDragOver"
                                        @dragover.prevent="onAttachDragOver"
                                        @dragleave="attachDragOver = false"
                                        @drop.prevent="onAttachDrop">
                                    <v-progress-circular indeterminate size="32" v-if="uploadingPicture"></v-progress-circular>
                                    <v-icon class="transaction-picture-add-icon" size="32" :icon="mdiFileDocumentPlusOutline" v-else/>
                                    <span>{{ tt('Add Document') }}</span>
                                </button>
                            </div>
                        </div>
                    </v-window-item>
                </v-window>
                </div>
            </template>

            <template #footer>
                <div class="transaction-edit-footer">
                    <v-btn color="error" variant="tonal" :disabled="loading || submitting || recognizing"
                           v-if="mode === TransactionEditPageMode.View && originalTransactionEditable" @click="remove">
                        {{ tt('Delete') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                    </v-btn>
                    <div class="transaction-edit-footer-status" role="status" aria-live="polite">
                        <template v-if="footerStatus">
                            <span class="transaction-edit-footer-status-dot" :class="`transaction-edit-footer-status-dot--${footerStatus.kind}`" aria-hidden="true"></span>
                            <span>{{ footerStatus.text }}</span>
                        </template>
                    </div>
                    <v-btn color="default" variant="outlined"
                           :disabled="loading || submitting || recognizing"
                           v-if="mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit"
                           @click="cancel">{{ tt('Cancel') }}</v-btn>
                    <v-tooltip :disabled="!inputIsEmpty" :text="inputEmptyProblemMessage ? tt(inputEmptyProblemMessage) : ''">
                        <template v-slot:activator="{ props }">
                            <div v-bind="props" class="d-inline-block">
                                <v-btn-group class="transaction-edit-save-group" density="comfortable" v-if="mode === TransactionEditPageMode.Add || mode === TransactionEditPageMode.Edit">
                                    <v-btn color="primary" :disabled="inputIsEmpty || loading || submitting || recognizing" @click="save(AfterSaveAction.GoBack)">
                                        {{ tt(saveButtonTitle) }}
                                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                                    </v-btn>
                                    <v-btn color="primary" density="compact"
                                           :aria-label="tt('More')" :disabled="inputIsEmpty || loading || submitting || recognizing" :icon="true"
                                           v-if="type === TransactionEditPageType.Transaction && mode === TransactionEditPageMode.Add">
                                        <v-icon :icon="mdiMenuDown" size="24" />
                                        <v-menu activator="parent">
                                            <v-list>
                                                <v-list-item :title="tt(TransactionQuickAddButtonActionType.SaveAndAddNewTransaction.name)"
                                                             @click="save(AfterSaveAction.StayWithNewTransaction)"></v-list-item>
                                                <v-list-item :title="tt(TransactionQuickAddButtonActionType.SaveAndKeepCurrentData.name)"
                                                             @click="save(AfterSaveAction.StayWithCurrentTransaction)"></v-list-item>
                                            </v-list>
                                        </v-menu>
                                    </v-btn>
                                </v-btn-group>
                            </div>
                        </template>
                    </v-tooltip>
                    <v-btn-group variant="tonal" density="comfortable"
                                 v-if="mode === TransactionEditPageMode.View && transaction.type !== TransactionType.ModifyBalance">
                        <v-btn :disabled="loading || submitting || recognizing"
                               @click="duplicate(false, false)">{{ tt('Duplicate') }}</v-btn>
                        <v-btn density="compact" :aria-label="tt('More')" :disabled="loading || submitting || recognizing" :icon="true">
                            <v-icon :icon="mdiMenuDown" size="24" />
                            <v-menu activator="parent">
                                <v-list>
                                    <v-list-item :title="tt('Duplicate (With Time)')"
                                                 @click="duplicate(true, false)"></v-list-item>
                                    <v-list-item :title="tt('Duplicate (With Geographic Location)')"
                                                 @click="duplicate(false, true)"
                                                 v-if="transaction.geoLocation"></v-list-item>
                                    <v-list-item :title="tt('Duplicate (With Time and Geographic Location)')"
                                                 @click="duplicate(true, true)"
                                                 v-if="transaction.geoLocation"></v-list-item>
                                </v-list>
                            </v-menu>
                        </v-btn>
                    </v-btn-group>
                </div>
            </template>
        </two-column-dialog-layout>
    </v-dialog>

    <v-dialog width="600" v-model="showPasteTextDialog">
        <one-column-dialog-layout content-class="pa-0" :disabled="recognizing"
                                  :title="tt('AI Clipboard Text Recognition')" :cancel-button-title="tt('Cancel')"
                                  @cancel="showPasteTextDialog = false; pastedText = '';">
            <template #toolbar>
                <v-btn class="me-2" density="comfortable" variant="outlined"
                       :disabled="!pastedText || !pastedText.trim() || recognizing"
                       @click="showPasteTextDialog = false; recognizeText(pastedText);">{{ tt('Recognize') }}</v-btn>
            </template>

            <template #content>
                <v-textarea no-resize persistent-placeholder
                            class="w-100 h-100 ps-4 always-cursor-text"
                            rows="10" autocomplete="off" density="compact" variant="plain" :rounded="false"
                            :disabled="recognizing"
                            :placeholder="tt('Click here to paste a transaction description')"
                            v-model="pastedText"></v-textarea>
            </template>
        </one-column-dialog-layout>
    </v-dialog>

    <category-edit-dialog ref="categoryEditDialog" />
    <confirm-dialog ref="confirmDialog"/>
    <input ref="pictureInput" type="file" style="display: none" :accept="SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS" @change="onUploadPicture($event)" />
</template>

<script setup lang="ts">
import MapView from '@/components/common/MapView.vue';
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import CategoryEditDialog from '@/views/desktop/categories/list/dialogs/EditDialog.vue';

import { ref, computed, useTemplateRef, watch, nextTick } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    TransactionEditPageMode,
    TransactionEditPageType,
    GeoLocationStatus,
    AfterSaveAction,
    useTransactionEditPageBase
} from '@/views/base/transactions/TransactionEditPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { Coordinate } from '@/core/coordinate.ts';
import { CategoryType } from '@/core/category.ts';
import { TransactionType, TransactionEditScopeType, TransactionQuickAddButtonActionType } from '@/core/transaction.ts';
import { TemplateType, ScheduledTemplateFrequencyType } from '@/core/template.ts';

import { KnownErrorCode } from '@/consts/api.ts';
import { SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS } from '@/consts/file.ts';

import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';
import type { TransactionPictureInfoBasicResponse } from '@/models/transaction_picture_info.ts';
import { Transaction } from '@/models/transaction.ts';

import { isDefined, isEquals } from '@/lib/common.ts';
import {
    getTimezoneOffsetMinutes,
    getCurrentUnixTime
} from '@/lib/datetime.ts';
import { generateRandomUUID } from '@/lib/misc.ts';
import {
    getTransactionPrimaryCategoryName,
    getTransactionSecondaryCategoryName
} from '@/lib/category.ts';
import { type SetTransactionOptions } from '@/lib/transaction.ts';
import {
    isTransactionFromAITextRecognitionEnabled,
    isTransactionPicturesEnabled,
    getMapProvider
} from '@/lib/server_settings.ts';
import {
    isSupportGetGeoLocationByClick
} from '@/lib/map/index.ts';
import { getFileDownloadUrl, getTransactionAttachmentDisplayName, isSupportedTransactionAttachmentFile } from '@/lib/file.ts';
import { getTransactionAttachmentMdiIcon } from '@/lib/ui/desktop.ts';
import logger from '@/lib/logger.ts';

import {
    mdiMagicStaff,
    mdiDotsVertical,
    mdiPencilOutline,
    mdiEyeOffOutline,
    mdiEyeOutline,
    mdiSwapHorizontal,
    mdiMapMarkerOutline,
    mdiCheck,
    mdiMenuDown,
    mdiTrashCanOutline,
    mdiFullscreen,
    mdiDownload,
    mdiPaperclip,
    mdiFileDocumentMultipleOutline,
    mdiFileDocumentPlusOutline,
    mdiImageBrokenVariant
} from '@mdi/js';

export interface TransactionEditOptions extends SetTransactionOptions {
    id?: string;
    templateType?: number;
    template?: TransactionTemplate;
    currentTransaction?: Transaction;
    currentTemplate?: TransactionTemplate;
    autoUploadPicture?: File;
    autoRecognizeClipboardText?: string;
    noTransactionDraft?: boolean;
}

interface TransactionEditResponse {
    message: string;
}

type MapViewType = InstanceType<typeof MapView>;
type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type CategoryEditDialogType = InstanceType<typeof CategoryEditDialog>;

const props = defineProps<{
    type: TransactionEditPageType;
    persistent?: boolean;
    show?: boolean;
}>();

const { tt, formatCoordinate, formatNumberToLocalizedNumerals } = useI18n();

const {
    mode,
    isSupportGeoLocation,
    editId,
    addByTemplateId,
    duplicateFromId,
    clientSessionId,
    loading,
    recognizing,
    submitting,
    submitted,
    uploadingPicture,
    geoLocationStatus,
    setGeoLocationByClickMap,
    transaction,
    defaultCurrency,
    allTimezones,
    allVisibleAccounts,
    allVisibleCategorizedAccounts,
    allCategories,
    firstVisibleAccountId,
    hasVisibleExpenseCategories,
    hasVisibleIncomeCategories,
    hasVisibleTransferCategories,
    canAddTransactionPicture,
    title,
    saveButtonTitle,
    cancelButtonTitle,
    sourceAmountName,
    sourceAmountTitle,
    sourceAccountTitle,
    transferInAmountTitle,
    sourceAccountName,
    destinationAccountName,
    sourceAccountCurrency,
    destinationAccountCurrency,
    transactionDisplayTimezone,
    transactionTimezoneTimeDifference,
    geoLocationStatusInfo,
    transactionDescriptionTitle,
    inputEmptyProblemMessage,
    inputIsEmpty,
    createNewTransactionModel,
    setTransactionModel,
    updateTransactionModelFromRecognizedResponse,
    updateTransactionModelByAfterSaveAction,
    updateTransactionTime,
    updateTransactionTimezone,
    swapTransactionData,
    getTransactionPictureUrl,
    getTransactionPictureKind,
    getTransactionPictureExtension,
    getTransactionPictureUploadFile
} = useTransactionEditPageBase(props.type);

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const transactionTagsStore = useTransactionTagsStore();
const transactionsStore = useTransactionsStore();
const transactionTemplatesStore = useTransactionTemplatesStore();
const desktopPageStore = useDesktopPageStore();

const map = useTemplateRef<MapViewType>('map');
const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const categoryEditDialog = useTemplateRef<CategoryEditDialogType>('categoryEditDialog');
const pictureInput = useTemplateRef<HTMLInputElement>('pictureInput');

let resolveFunc: ((response?: TransactionEditResponse) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);
const showPasteTextDialog = ref<boolean>(false);
const activeTab = ref<string>('basicInfo');
const initTransaction = ref<Transaction | null>(null);
const initTemplate = ref<TransactionTemplate | null>(null);
const originalTransactionEditable = ref<boolean>(false);
const noTransactionDraft = ref<boolean>(false);
const geoMenuState = ref<boolean>(false);
const removingPictureId = ref<string>('');
const pastedText = ref<string>('');
const attachDragOver = ref<boolean>(false);

const initOptions = ref<TransactionEditOptions | undefined>(undefined);

const sourceAmountColor = computed<string | undefined>(() => {
    if (transaction.value.type === TransactionType.Expense) {
        return 'expense';
    } else if (transaction.value.type === TransactionType.Income) {
        return 'income';
    } else if (transaction.value.type === TransactionType.Transfer) {
        return 'primary';
    }

    return undefined;
});

interface TransactionEditFooterStatus {
    kind: 'progress' | 'warning' | 'modified';
    text: string;
}

const footerStatus = computed<TransactionEditFooterStatus | null>(() => {
    if (loading.value || (mode.value !== TransactionEditPageMode.Add && mode.value !== TransactionEditPageMode.Edit)) {
        return null;
    }

    if (uploadingPicture.value) {
        return { kind: 'progress', text: tt('Uploading...') };
    }

    if (recognizing.value) {
        return { kind: 'progress', text: tt('Recognizing') };
    }

    if (submitting.value && !removingPictureId.value) {
        return { kind: 'progress', text: tt('Saving...') };
    }

    if (inputIsEmpty.value && inputEmptyProblemMessage.value) {
        return { kind: 'warning', text: tt(inputEmptyProblemMessage.value) };
    }

    if (isTransactionModified.value) {
        return { kind: 'modified', text: tt('Unsaved changes') };
    }

    return null;
});

const canAddNewCategory = computed<boolean>(() => mode.value === TransactionEditPageMode.Add || mode.value === TransactionEditPageMode.Edit);

const isTransactionModified = computed<boolean>(() => {
    if (mode.value === TransactionEditPageMode.Add) {
        if (props.type === TransactionEditPageType.Transaction) {
            return transactionsStore.isTransactionDraftModified(transaction.value, initOptions.value?.amount, initOptions.value?.categoryId, initOptions.value?.accountId, initOptions.value?.tagIds, firstVisibleAccountId.value);
        } else if (props.type === TransactionEditPageType.Template && transaction.value instanceof TransactionTemplate) {
            const template = transaction.value as TransactionTemplate;
            return !!initTemplate.value && !isEquals(template.toTemplateCreateRequest(clientSessionId.value), initTemplate.value.toTemplateCreateRequest(clientSessionId.value));
        } else {
            return true;
        }
    } else if (mode.value === TransactionEditPageMode.Edit) {
        if (props.type === TransactionEditPageType.Transaction) {
            return !!initTransaction.value && !isEquals(transaction.value.toModifyRequest(), initTransaction.value.toModifyRequest());
        } else if (props.type === TransactionEditPageType.Template && transaction.value instanceof TransactionTemplate) {
            const template = transaction.value as TransactionTemplate;
            return !!initTemplate.value && !isEquals(template.toTemplateModifyRequest(), initTemplate.value.toTemplateModifyRequest());
        } else {
            return true;
        }
    } else {
        return false;
    }
});

function open(options: TransactionEditOptions): Promise<TransactionEditResponse | undefined> {
    addByTemplateId.value = null;
    duplicateFromId.value = null;
    showState.value = true;
    activeTab.value = 'basicInfo';
    loading.value = true;
    submitting.value = false;
    submitted.value = false;
    geoLocationStatus.value = null;
    setGeoLocationByClickMap.value = false;
    originalTransactionEditable.value = false;
    noTransactionDraft.value = options.noTransactionDraft || false;

    initOptions.value = options;

    const newTransaction = createNewTransactionModel(options.type);
    setTransactionModel(newTransaction, options, true);
    initTransaction.value = Transaction.of(transaction.value);

    const promises: Promise<unknown>[] = [
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false }),
        transactionTagsStore.loadAllTags({ force: false })
    ];

    if (props.type === TransactionEditPageType.Transaction) {
        if (options && options.id) {
            if (options.currentTransaction) {
                setTransactionModel(options.currentTransaction, options, true);
            }

            mode.value = TransactionEditPageMode.View;
            editId.value = options.id;

            promises.push(transactionsStore.getTransaction({ transactionId: editId.value }));
        } else {
            mode.value = TransactionEditPageMode.Add;
            editId.value = null;

            if (options.template) {
                setTransactionModel(options.template, options, false);
                initTransaction.value = Transaction.of(transaction.value);
                addByTemplateId.value = options.template.id;
            } else if (!options.noTransactionDraft && (settingsStore.appSettings.autoSaveTransactionDraft === 'enabled' || settingsStore.appSettings.autoSaveTransactionDraft === 'confirmation') && transactionsStore.transactionDraft) {
                setTransactionModel(Transaction.ofDraft(transactionsStore.transactionDraft), options, false);
            }

            if (settingsStore.appSettings.autoGetCurrentGeoLocation
                && !geoLocationStatus.value && !transaction.value.geoLocation) {
                updateGeoLocation(false);
            }
        }
    } else if (props.type === TransactionEditPageType.Template) {
        initTemplate.value = TransactionTemplate.createNewTransactionTemplate(transaction.value);
        initTemplate.value.name = '';

        if (options && options.templateType) {
            initTemplate.value.templateType = options.templateType;
        }

        if (initTemplate.value.templateType === TemplateType.Schedule.type) {
            initTemplate.value.scheduledFrequencyType = ScheduledTemplateFrequencyType.Disabled.type;
            initTemplate.value.scheduledFrequency = '';
        }

        transaction.value = TransactionTemplate.ofTemplate(initTemplate.value);

        if (options && options.id) {
            if (options.currentTemplate) {
                setTransactionModel(options.currentTemplate, options, false);
                (transaction.value as TransactionTemplate).fillFrom(options.currentTemplate);
            }

            mode.value = TransactionEditPageMode.Edit;
            editId.value = options.id;
            transaction.value.id = options.id;

            promises.push(transactionTemplatesStore.getTemplate({ templateId: editId.value }));
        } else {
            mode.value = TransactionEditPageMode.Add;
            editId.value = null;
            transaction.value.id = '';
        }
    }

    if (options.type &&
        options.type >= TransactionType.Income &&
        options.type <= TransactionType.Transfer) {
        transaction.value.type = options.type;
    }

    if (mode.value === TransactionEditPageMode.Add) {
        clientSessionId.value = generateRandomUUID();
    }

    Promise.all(promises).then(function (responses) {
        if (editId.value && !responses[3]) {
            if (rejectFunc) {
                if (props.type === TransactionEditPageType.Transaction) {
                    rejectFunc('Unable to retrieve transaction');
                } else if (props.type === TransactionEditPageType.Template) {
                    rejectFunc('Unable to retrieve template');
                }
            }

            return;
        }

        if (props.type === TransactionEditPageType.Transaction && options && options.id && responses[3] && responses[3] instanceof Transaction) {
            const transaction: Transaction = responses[3];
            setTransactionModel(transaction, options, true);
            initTransaction.value = Transaction.of(transaction);
            originalTransactionEditable.value = transaction.editable;
        } else if (props.type === TransactionEditPageType.Template && options && options.id && responses[3] && responses[3] instanceof TransactionTemplate) {
            const template: TransactionTemplate = responses[3];
            setTransactionModel(template, options, false);

            if (!(transaction.value instanceof TransactionTemplate)) {
                transaction.value = TransactionTemplate.createNewTransactionTemplate(transaction.value);
            }

            (transaction.value as TransactionTemplate).fillFrom(template);
            initTemplate.value = TransactionTemplate.ofTemplate(template);
        } else {
            setTransactionModel(null, options, true);

            if (props.type === TransactionEditPageType.Transaction) {
                initTransaction.value = Transaction.of(transaction.value);
            } else if (props.type === TransactionEditPageType.Template && transaction.value instanceof TransactionTemplate) {
                initTemplate.value = TransactionTemplate.ofTemplate(transaction.value);
            }
        }

        if (options.autoUploadPicture) {
            uploadPicture(options.autoUploadPicture);
        }

        loading.value = false;

        if (isDefined(options.autoRecognizeClipboardText)) {
            pastedText.value = options.autoRecognizeClipboardText;

            if (pastedText.value && !settingsStore.appSettings.alwaysRequireConfirmationOfClipboardContentBeforeSubmission) {
                recognizeText(pastedText.value);
            } else {
                showPasteTextDialog.value = true;
            }
        }
    }).catch(error => {
        logger.error('failed to load essential data for editing transaction', error);

        loading.value = false;
        showState.value = false;

        if (!error.processed) {
            if (rejectFunc) {
                rejectFunc(error);
            }
        }
    });

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function save(afterAction: AfterSaveAction): void {
    const problemMessage = inputEmptyProblemMessage.value;

    if (problemMessage) {
        desktopPageStore.addToast(problemMessage);
        return;
    }

    if (props.type === TransactionEditPageType.Transaction && (mode.value === TransactionEditPageMode.Add || mode.value === TransactionEditPageMode.Edit)) {
        const doSubmit = function () {
            submitting.value = true;

            transactionsStore.saveTransaction({
                transaction: transaction.value as Transaction,
                defaultCurrency: defaultCurrency.value,
                isEdit: mode.value === TransactionEditPageMode.Edit,
                clientSessionId: clientSessionId.value
            }).then(() => {
                submitting.value = false;
                submitted.value = true;

                if (mode.value === TransactionEditPageMode.Add && !noTransactionDraft.value && !addByTemplateId.value && !duplicateFromId.value) {
                    transactionsStore.clearTransactionDraft();
                }

                if (mode.value === TransactionEditPageMode.Add && (afterAction === AfterSaveAction.StayWithNewTransaction || afterAction === AfterSaveAction.StayWithCurrentTransaction)) {
                    desktopPageStore.addToast('You have added a new transaction');
                    updateTransactionModelByAfterSaveAction(afterAction, initOptions.value);
                    clientSessionId.value = generateRandomUUID();
                } else {
                    if (resolveFunc) {
                        if (mode.value === TransactionEditPageMode.Add) {
                            resolveFunc({
                                message: 'You have added a new transaction'
                            });
                        } else if (mode.value === TransactionEditPageMode.Edit) {
                            resolveFunc({
                                message: 'You have saved this transaction'
                            });
                        }
                    }

                    showState.value = false;
                }
            }).catch(error => {
                submitting.value = false;

                if (error.error && (error.error.errorCode === KnownErrorCode.TransactionCannotCreateInThisTime || error.error.errorCode === KnownErrorCode.TransactionCannotModifyInThisTime)) {
                    confirmDialog.value?.open('You have set this time range to prevent editing transactions. Would you like to change the editable transaction range to All?', { color: 'primary' }).then(() => {
                        submitting.value = true;

                        userStore.updateUserTransactionEditScope({
                            transactionEditScope: TransactionEditScopeType.All.type
                        }).then(() => {
                            submitting.value = false;

                            desktopPageStore.addToast('Your editable transaction range has been set to All');
                        }).catch(error => {
                            submitting.value = false;

                            if (!error.processed) {
                                desktopPageStore.addErrorToast(error);
                            }
                        });
                    });
                } else if (!error.processed) {
                    desktopPageStore.addErrorToast(error);
                }
            });
        };

        if (transaction.value.sourceAmount === 0) {
            confirmDialog.value?.open('Are you sure you want to save this transaction with a zero amount?', { color: 'primary' }).then(() => {
                doSubmit();
            });
        } else {
            doSubmit();
        }
    } else if (props.type === TransactionEditPageType.Template && (mode.value === TransactionEditPageMode.Add || mode.value === TransactionEditPageMode.Edit)) {
        submitting.value = true;

        transactionTemplatesStore.saveTemplateContent({
            template: transaction.value as TransactionTemplate,
            isEdit: mode.value === TransactionEditPageMode.Edit,
            clientSessionId: clientSessionId.value
        }).then(() => {
            submitting.value = false;

            if (resolveFunc) {
                if (mode.value === TransactionEditPageMode.Add) {
                    resolveFunc({
                        message: 'You have added a new template'
                    });
                } else if (mode.value === TransactionEditPageMode.Edit) {
                    resolveFunc({
                        message: 'You have saved this template'
                    });
                }
            }

            showState.value = false;
        }).catch(error => {
            submitting.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    }
}

function recognizeText(text: string): void {
    if (recognizing.value || loading.value || submitting.value) {
        return;
    }

    if (!text || !text.trim()) {
        return;
    }

    recognizing.value = true;

    transactionsStore.recognizeTransactionText({ text }).then(response => {
        updateTransactionModelFromRecognizedResponse(response);
        recognizing.value = false;
    }).catch(error => {
        recognizing.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function recognizeFromClipboard(): void {
    if (recognizing.value || loading.value || submitting.value) {
        return;
    }

    pastedText.value = '';

    navigator.clipboard.readText().then(text => {
        pastedText.value = text && text.trim() ? text.trim() : '';

        if (pastedText.value && !settingsStore.appSettings.alwaysRequireConfirmationOfClipboardContentBeforeSubmission) {
            recognizeText(pastedText.value);
        } else {
            showPasteTextDialog.value = true;
        }
    }).catch(error => {
        logger.error('failed to read clipboard', error);
        showPasteTextDialog.value = true;
    });
}

function duplicate(withTime?: boolean, withGeoLocation?: boolean): void {
    if (props.type !== TransactionEditPageType.Transaction || mode.value !== TransactionEditPageMode.View) {
        return;
    }

    editId.value = null;
    duplicateFromId.value = transaction.value.id;
    clientSessionId.value = generateRandomUUID();
    submitted.value = false;
    activeTab.value = 'basicInfo';
    transaction.value.id = '';

    if (!withTime) {
        transaction.value.time = getCurrentUnixTime();
        transaction.value.timeZone = settingsStore.appSettings.timeZone;
        transaction.value.utcOffset = getTimezoneOffsetMinutes(transaction.value.time, transaction.value.timeZone);
    }

    if (!withGeoLocation) {
        transaction.value.removeGeoLocation();
    }

    transaction.value.clearPictures();
    mode.value = TransactionEditPageMode.Add;
}

function edit(): void {
    if (props.type !== TransactionEditPageType.Transaction || mode.value !== TransactionEditPageMode.View) {
        return;
    }

    mode.value = TransactionEditPageMode.Edit;
}

function editAndAddDocument(): void {
    if (!originalTransactionEditable.value) {
        return;
    }

    edit();
    activeTab.value = 'pictures';

    // open the file picker synchronously so that the browser still treats it as part of the user click
    showOpenPictureDialog();
}

function remove(): void {
    if (props.type !== TransactionEditPageType.Transaction || mode.value !== TransactionEditPageMode.View) {
        return;
    }

    confirmDialog.value?.open('Are you sure you want to delete this transaction?', { color: 'error' }).then(() => {
        submitting.value = true;

        transactionsStore.deleteTransaction({
            transaction: transaction.value as Transaction,
            defaultCurrency: defaultCurrency.value
        }).then(() => {
            if (resolveFunc) {
                resolveFunc();
            }

            submitting.value = false;
            showState.value = false;
        }).catch(error => {
            submitting.value = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function cancel(): void {
    const doClose = function () {
        if (props.type === TransactionEditPageType.Transaction && mode.value === TransactionEditPageMode.Add && submitted.value && resolveFunc) {
            resolveFunc({
                message: 'You have added a new transaction'
            });
        } else if (rejectFunc) {
            rejectFunc();
        }

        showState.value = false;
    };

    if (props.type !== TransactionEditPageType.Transaction || mode.value !== TransactionEditPageMode.Add || noTransactionDraft.value || addByTemplateId.value || duplicateFromId.value) {
        doClose();
        return;
    }

    if (settingsStore.appSettings.autoSaveTransactionDraft === 'confirmation') {
        if (transactionsStore.isTransactionDraftModified(transaction.value, initOptions.value?.amount, initOptions.value?.categoryId, initOptions.value?.accountId, initOptions.value?.tagIds, firstVisibleAccountId.value)) {
            confirmDialog.value?.open('Do you want to save this transaction draft?', { color: 'primary' }).then(() => {
                transactionsStore.saveTransactionDraft(transaction.value, initOptions.value?.amount, initOptions.value?.categoryId, initOptions.value?.accountId, initOptions.value?.tagIds, firstVisibleAccountId.value);
                doClose();
            }).catch(() => {
                transactionsStore.clearTransactionDraft();
                doClose();
            });
        } else {
            transactionsStore.clearTransactionDraft();
            doClose();
        }
    } else if (settingsStore.appSettings.autoSaveTransactionDraft === 'enabled') {
        transactionsStore.saveTransactionDraft(transaction.value, initOptions.value?.amount, initOptions.value?.categoryId, initOptions.value?.accountId, initOptions.value?.tagIds, firstVisibleAccountId.value);
        doClose();
    } else {
        doClose();
    }
}

function updateGeoLocation(forceUpdate: boolean): void {
    geoMenuState.value = false;

    if (!isSupportGeoLocation) {
        logger.warn('this browser does not support geo location');

        if (forceUpdate) {
            desktopPageStore.addToast('Unable to retrieve current position');
        }
        return;
    }

    navigator.geolocation.getCurrentPosition(function (position) {
        if (!position || !position.coords) {
            logger.error('current position is null');
            geoLocationStatus.value = GeoLocationStatus.Error;

            if (forceUpdate) {
                desktopPageStore.addToast('Unable to retrieve current position');
            }

            return;
        }

        geoLocationStatus.value = GeoLocationStatus.Success;

        transaction.value.setLatitudeAndLongitude(position.coords.latitude, position.coords.longitude);
    }, function (err) {
        logger.error('cannot retrieve current position', err);
        geoLocationStatus.value = GeoLocationStatus.Error;

        if (forceUpdate) {
            desktopPageStore.addToast('Unable to retrieve current position');
        }
    });

    geoLocationStatus.value = GeoLocationStatus.Getting;
}

function updateSpecifiedGeoLocation(coordinate: Coordinate): void {
    if (isSupportGetGeoLocationByClick() && setGeoLocationByClickMap.value) {
        transaction.value.setLatitudeAndLongitude(coordinate.latitude, coordinate.longitude);
        map.value?.setMarkerPosition(transaction.value.geoLocation);
    }
}

function clearGeoLocation(): void {
    geoMenuState.value = false;
    geoLocationStatus.value = null;
    transaction.value.removeGeoLocation();
}

function showOpenPictureDialog(): void {
    if (!canAddTransactionPicture.value || submitting.value) {
        return;
    }

    pictureInput.value?.click();
}

function uploadPicture(file: File): void {
    if (!file) {
        return;
    }

    uploadingPicture.value = true;
    submitting.value = true;

    getTransactionPictureUploadFile(file).then(pictureFile => {
        return transactionsStore.uploadTransactionPicture({
            pictureFile: pictureFile
        });
    }).then(response => {
        transaction.value.addPicture(response);
        uploadingPicture.value = false;
        submitting.value = false;
    }).catch(error => {
        uploadingPicture.value = false;
        submitting.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function getPictureDisplayName(pictureInfo: TransactionPictureInfoBasicResponse): string {
    return getTransactionAttachmentDisplayName(pictureInfo.originalFileName, getTransactionPictureUrl(pictureInfo), tt);
}

function getPictureTileActionLabel(pictureInfo: TransactionPictureInfoBasicResponse): string {
    const name = getTransactionPictureKind(pictureInfo) !== 'image' ? `: ${getPictureDisplayName(pictureInfo)}` : '';

    if (mode.value === TransactionEditPageMode.Add || mode.value === TransactionEditPageMode.Edit) {
        return tt('Remove Document') + name;
    } else if (getTransactionPictureKind(pictureInfo) === 'document') {
        return tt('Download') + name;
    } else {
        return tt('View') + name;
    }
}

function viewOrRemovePicture(pictureInfo: TransactionPictureInfoBasicResponse): void {
    if (mode.value !== TransactionEditPageMode.Add && mode.value !== TransactionEditPageMode.Edit) {
        if (getTransactionPictureKind(pictureInfo) === 'document') {
            // office files, text files and heic / heif images cannot be previewed by browsers
            downloadPicture(pictureInfo);
        } else {
            window.open(getTransactionPictureUrl(pictureInfo), '_blank');
        }

        return;
    }

    confirmDialog.value?.open('Are you sure you want to remove this transaction document?', { color: 'error' }).then(() => {
        removingPictureId.value = pictureInfo.pictureId;
        submitting.value = true;

        transactionsStore.removeUnusedTransactionPicture({ pictureInfo }).then(response => {
            if (response) {
                transaction.value.removePicture(pictureInfo);
            }

            removingPictureId.value = '';
            submitting.value = false;
        }).catch(error => {
            if (error.error && error.error.errorCode === KnownErrorCode.TransactionPictureNotFound) {
                transaction.value.removePicture(pictureInfo);
            } else if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }

            removingPictureId.value = '';
            submitting.value = false;
        });
    });
}

function downloadPicture(pictureInfo: TransactionPictureInfoBasicResponse): void {
    const downloadUrl = getFileDownloadUrl(getTransactionPictureUrl(pictureInfo));

    if (!downloadUrl) {
        return;
    }

    window.open(downloadUrl, '_blank');
}

function setTransactionCategoryId(categoryType: CategoryType, categoryId: string): void {
    if (categoryType === CategoryType.Expense) {
        transaction.value.expenseCategoryId = categoryId;
    } else if (categoryType === CategoryType.Income) {
        transaction.value.incomeCategoryId = categoryId;
    } else if (categoryType === CategoryType.Transfer) {
        transaction.value.transferCategoryId = categoryId;
    }
}

function addNewCategory(categoryType: CategoryType, primaryCategory: unknown, categoryName?: string): void {
    const parentCategory = primaryCategory instanceof TransactionCategory ? primaryCategory : undefined;

    categoryEditDialog.value?.open({
        type: categoryType,
        parentId: parentCategory ? parentCategory.id : '0',
        color: parentCategory?.color,
        icon: parentCategory?.icon,
        name: categoryName
    }).then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }

        const newCategory = result?.category;

        if (!newCategory || !newCategory.id) {
            return;
        }

        if (newCategory.parentId && newCategory.parentId !== '0') {
            setTransactionCategoryId(categoryType, newCategory.id);
        } else {
            // transactions can only use secondary categories, so offer to add a secondary category under the new primary category
            confirmDialog.value?.open('Transactions require a secondary category. Do you want to add a secondary category under this primary category now?', { color: 'primary' }).then(() => {
                addNewCategory(categoryType, newCategory);
            });
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function onSavingTag(state: boolean): void {
    submitting.value = state;
}

function onUploadPicture(event: Event): void {
    if (!event || !event.target) {
        return;
    }

    const el = event.target as HTMLInputElement;

    if (!el.files || !el.files.length || !el.files[0]) {
        return;
    }

    const pictureFile = el.files[0] as File;

    el.value = '';
    uploadPicture(pictureFile);
}

function onAttachDragOver(event: DragEvent): void {
    if (!canAddTransactionPicture.value || submitting.value || uploadingPicture.value) {
        return;
    }

    if (event.dataTransfer) {
        event.dataTransfer.dropEffect = 'copy';
    }

    attachDragOver.value = true;
}

function onAttachDrop(event: DragEvent): void {
    attachDragOver.value = false;

    if (!canAddTransactionPicture.value || submitting.value || uploadingPicture.value) {
        return;
    }

    const file = event.dataTransfer?.files?.[0];

    if (!file) {
        return;
    }

    if (!isSupportedTransactionAttachmentFile(file)) {
        desktopPageStore.addErrorToast('Unsupported file type');
        return;
    }

    uploadPicture(file);
}

function onShowDateTimeError(error: string): void {
    desktopPageStore.addErrorToast(error);
}

watch(activeTab, (newValue) => {
    if (newValue === 'map') {
        nextTick(() => {
            map.value?.initMapView();
        });
    }
});

defineExpose({
    open
});
</script>

<style>
/* Type segmented control (guidelines §6.2) */
.transaction-edit-type-bar {
    padding-block: var(--ebk-space-4) 0;
}

.transaction-edit-type-tabs.v-tabs {
    display: inline-flex;
    max-width: 100%;
    padding: 3px;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
    height: auto;

    .v-tab.v-btn {
        min-width: 6.5rem;
        height: 34px;
        border-radius: calc(var(--ebk-radius-control) - 2px);
    }
}

/* Large amount field: the most prominent element of the editor (guidelines §4.4) */
.transaction-edit-amount .v-field__prepend-inner,
.transaction-edit-amount .v-field__append-inner,
.transaction-edit-amount .v-field__field > input {
    font-size: 1.75rem;
    font-weight: 600;
    line-height: 1.2;
}

.transaction-edit-amount .v-field__field > input {
    min-height: 64px;
}

.transaction-edit-amount .v-field__append-inner .v-icon {
    font-size: 1.5rem;
}

.transaction-edit-timezone.v-input input::placeholder {
    color: rgba(var(--v-theme-on-background), var(--v-high-emphasis-opacity)) !important;
    opacity: unset;
}

/* Attach receipt drop zone */
.transaction-edit-attach {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-2);
}

.transaction-edit-attach-dropzone {
    display: flex;
    flex: 1 1 18rem;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-2);
    min-height: 56px;
    padding: var(--ebk-space-3) var(--ebk-space-4);
    border: 1px dashed rgba(var(--ebk-color-hairline-rgb), calc(var(--ebk-color-hairline-opacity, 0.14) * 3));
    border-radius: var(--ebk-radius-control);
    background-color: rgba(var(--ebk-color-surface-sunken-rgb), 0.6);
    color: var(--ebk-color-text-secondary);
    font-size: 0.875rem;
    cursor: pointer;
    transition: border-color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard),
                background-color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);

    &:hover:not(:disabled),
    &.transaction-edit-attach-dropzone--over {
        border-color: var(--ebk-color-primary);
        background-color: var(--ebk-color-selected-background);
        color: var(--ebk-color-primary-text);
    }

    &:disabled {
        cursor: default;
        opacity: 0.6;
    }
}

.transaction-edit-attach-text {
    overflow-wrap: anywhere;
}

/* Footer: status, Cancel, Save */
.transaction-edit-footer {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-3);
    width: 100%;
}

.transaction-edit-footer-status {
    display: flex;
    flex: 1 1 12rem;
    align-items: center;
    gap: var(--ebk-space-2);
    min-width: 0;
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
    overflow-wrap: anywhere;
}

.transaction-edit-footer-status-dot {
    flex: 0 0 auto;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--ebk-color-copper-line);

    &.transaction-edit-footer-status-dot--progress {
        background-color: var(--ebk-color-primary);
    }

    &.transaction-edit-footer-status-dot--warning {
        background-color: var(--ebk-color-warning);
    }

    &.transaction-edit-footer-status-dot--modified {
        border: 1px solid var(--ebk-color-primary);
        background-color: transparent;
    }
}

/* Map */
.transaction-edit-map-view {
    height: 289px;
}

@media (min-height: 620px) {
    .transaction-edit-map-view {
        height: 415px;
    }

    @media (min-width: 960px) {
        .transaction-pictures {
            min-height: 416px;
        }
    }
}

@media (min-height: 700px) {
    .transaction-edit-map-view {
        height: 513px;
    }

    @media (min-width: 960px) {
        .transaction-pictures {
            min-height: 514px;
        }
    }
}

/* Transaction documents (images, pdf and other documents) */
.transaction-pictures {
    width: 100%;
    padding-block: var(--ebk-space-4);
}

.transaction-pictures--empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-2);
    text-align: center;
    color: var(--ebk-color-text-secondary);
}

.transaction-pictures-empty-icon {
    opacity: 0.6;
}

.transaction-pictures-empty-title {
    font-size: 1rem;
    font-weight: 500;
    color: var(--ebk-color-text-primary);
}

.transaction-pictures-empty-hint {
    max-width: 360px;
    font-size: 0.875rem;
}

.transaction-pictures-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: var(--ebk-space-3);
}

.transaction-picture-tile {
    position: relative;
    aspect-ratio: 1;
    overflow: hidden;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-panel);
    background-color: var(--ebk-color-surface-sunken);
}

.transaction-picture-tile-preview {
    display: block;
    width: 100%;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    color: inherit;
    cursor: pointer;

    &:focus-visible {
        outline: var(--ebk-focus-outline);
        outline-offset: -2px;
    }
}

.transaction-picture-tile-image {
    width: 100%;
    height: 100%;
}

.transaction-picture-tile-document,
.transaction-picture-tile-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-1);
    width: 100%;
    height: 100%;
    padding: var(--ebk-space-2);
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
    text-align: center;
}

.transaction-picture-tile-document > .v-icon {
    color: var(--ebk-color-primary-text);
}

.transaction-picture-tile-document-name {
    display: -webkit-box;
    max-width: 100%;
    overflow: hidden;
    overflow-wrap: anywhere;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
}

.transaction-picture-tile-document-type {
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    opacity: 0.8;
}

.transaction-picture-tile-placeholder--error {
    overflow-wrap: anywhere;
}

.transaction-picture-tile-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: rgba(0, 0, 0, 0.5);
    color: #fff;
    opacity: 0;
    transition: opacity var(--ebk-motion-duration-sm) var(--ebk-motion-ease-standard);
    pointer-events: none;
}

.transaction-picture-tile-preview:hover .transaction-picture-tile-overlay,
.transaction-picture-tile-preview:focus-visible .transaction-picture-tile-overlay,
.transaction-picture-tile-overlay.transaction-picture-tile-overlay--visible {
    opacity: 1;
}

.transaction-picture-tile-action {
    display: inline-flex;
    flex-direction: column;
    align-items: center;
    gap: var(--ebk-space-1);
    font-size: 0.8125rem;
    font-weight: 500;
}

.transaction-picture-download-button {
    position: absolute !important;
    inset-block-start: var(--ebk-space-2);
    inset-inline-end: var(--ebk-space-2);
}

.transaction-picture-add {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-2);
    padding: var(--ebk-space-3);
    border-style: dashed;
    border-color: rgba(var(--ebk-color-hairline-rgb), calc(var(--ebk-color-hairline-opacity, 0.14) * 3));
    background-color: rgba(var(--ebk-color-surface-sunken-rgb), 0.6);
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
    text-align: center;
    cursor: pointer;
    transition: border-color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);

    &:hover:not(:disabled),
    &.transaction-picture-add--over {
        border-color: var(--ebk-color-primary);
        background-color: var(--ebk-color-selected-background);
        color: var(--ebk-color-primary-text);
    }

    &:disabled {
        cursor: default;
    }
}

@media (forced-colors: active) {
    .transaction-edit-attach-dropzone,
    .transaction-picture-tile {
        border-color: CanvasText;
    }

    .transaction-edit-footer-status-dot {
        forced-color-adjust: none;
    }
}
</style>

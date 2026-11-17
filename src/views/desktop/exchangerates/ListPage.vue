<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="ebk-mgmt-panel" :aria-busy="loading ? 'true' : 'false'">
                <v-layout class="page-with-navigation-drawer">
                    <v-navigation-drawer class="ebk-mgmt-subnav" :permanent="alwaysShowNav" v-model="showNav">
                        <div class="ebk-mgmt-subnav-section">
                            <div class="ebk-mgmt-kv">
                                <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Data source') }}</span>
                                <span class="ebk-mgmt-kv-value exchange-rates-source">
                                    <a target="_blank" rel="noopener" :href="exchangeRatesData.referenceUrl" v-if="!loading && exchangeRatesData && !isUserCustomExchangeRates && exchangeRatesData.referenceUrl">{{ exchangeRatesData.dataSource }}</a>
                                    <span v-else-if="!loading && exchangeRatesData && !isUserCustomExchangeRates && !exchangeRatesData.referenceUrl">{{ exchangeRatesData.dataSource }}</span>
                                    <span v-else-if="!loading && exchangeRatesData && isUserCustomExchangeRates">{{ tt('User Custom') }}</span>
                                    <span v-else-if="!loading && !exchangeRatesData">{{ tt('None') }}</span>
                                    <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="140" :loading="true" v-else-if="loading"></v-skeleton-loader>
                                </span>
                            </div>
                            <div class="ebk-mgmt-kv" v-if="exchangeRatesDataUpdateTime || loading">
                                <span class="ebk-mgmt-kv-label ebk-technical-heading">{{ tt('Last Updated') }}</span>
                                <span class="ebk-mgmt-kv-value" v-if="!loading">{{ exchangeRatesDataUpdateTime }}</span>
                                <v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="120" :loading="true" v-else></v-skeleton-loader>
                            </div>
                        </div>
                        <div class="ebk-mgmt-subnav-section">
                            <label class="ebk-mgmt-kv-label ebk-technical-heading d-block mb-2" for="exchange-rates-base-amount">{{ tt('Base Amount') }}</label>
                            <amount-input id="exchange-rates-base-amount" density="compact"
                                          :currency="baseCurrency"
                                          :disabled="loading || !exchangeRatesData || !exchangeRatesData.exchangeRates || !exchangeRatesData.exchangeRates.length"
                                          v-model="baseAmount"/>
                        </div>
                        <span class="ebk-mgmt-subnav-title ebk-technical-heading">{{ tt('Base Currency') }}</span>
                        <v-tabs show-arrows class="ebk-mgmt-subnav-tabs mb-4" direction="vertical"
                                :disabled="loading" v-model="baseCurrency">
                            <template v-if="exchangeRatesData && exchangeRatesData.exchangeRates && exchangeRatesData.exchangeRates.length">
                                <v-tab class="tab-text-truncate" :key="exchangeRate.currencyCode" :value="exchangeRate.currencyCode"
                                       v-for="exchangeRate in availableExchangeRates">
                                    <span class="ebk-mgmt-subnav-tab-content w-100">
                                        <span class="text-truncate">{{ exchangeRate.currencyDisplayName }}</span>
                                        <span class="ebk-mgmt-subnav-tab-count ebk-font-mono">{{ exchangeRate.currencyCode }}</span>
                                    </span>
                                </v-tab>
                            </template>
                            <template v-else-if="!exchangeRatesData || !exchangeRatesData.exchangeRates || !exchangeRatesData.exchangeRates.length">
                                <span class="mx-4 mt-h1 mb-3 text-medium-emphasis" v-if="!loading">{{ tt('None') }}</span>
                                <template v-else-if="loading">
                                    <v-skeleton-loader class="skeleton-no-margin mx-4 mt-h1 py-3" type="text"
                                                       :key="itemIdx" :loading="loading"
                                                       v-for="itemIdx in [ 1, 2, 3, 4, 5, 6, 7, 8 ]"></v-skeleton-loader>
                                </template>
                            </template>
                        </v-tabs>
                    </v-navigation-drawer>
                    <v-main>
                        <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                            <v-window-item value="exchangeRatesPage">
                                <v-card variant="flat" class="exchange-rates-content" min-height="680">
                                    <header class="ebk-panel-header-strip ebk-mgmt-header">
                                        <v-btn class="ebk-mgmt-header-menu-toggle d-lg-none" density="comfortable" color="default" variant="text"
                                               :aria-label="tt('Open Menu')" :icon="true" @click="showNav = !showNav">
                                            <v-icon :icon="mdiMenu" size="22" />
                                        </v-btn>
                                        <h2 class="ebk-mgmt-header-label">{{ tt('Exchange Rates Data') }}</h2>
                                        <span class="ebk-mgmt-header-meta ebk-numeric" v-if="!loading && availableExchangeRates.length">{{ formatNumberToLocalizedNumerals(availableExchangeRates.length) }}</span>
                                        <div class="ebk-panel-header-actions ebk-mgmt-header-actions">
                                            <v-btn color="default" variant="outlined" :prepend-icon="mdiPencilOutline"
                                                   :disabled="loading" @click="update"
                                                   v-if="isUserCustomExchangeRates">{{ tt('Update') }}</v-btn>
                                            <v-btn density="comfortable" color="default" variant="text"
                                                   :aria-label="tt('Refresh')" :icon="true" :loading="loading" @click="reload(true)">
                                                <template #loader>
                                                    <v-progress-circular indeterminate size="20"/>
                                                </template>
                                                <v-icon :icon="mdiRefresh" size="22" />
                                                <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                                            </v-btn>
                                        </div>
                                    </header>

                                    <div class="ebk-mgmt-summary" v-if="baseCurrency && availableExchangeRates.length">
                                        <span class="ebk-mgmt-summary-item">
                                            <span class="ebk-mgmt-summary-label ebk-technical-heading">{{ tt('Base Currency') }}</span>
                                            <span class="ebk-mgmt-summary-value">{{ baseCurrencyDisplayName }}</span>
                                            <span class="ebk-font-mono text-medium-emphasis">{{ baseCurrency }}</span>
                                        </span>
                                        <span class="ebk-mgmt-summary-separator" aria-hidden="true">·</span>
                                        <span class="ebk-mgmt-summary-item">
                                            <span class="ebk-mgmt-summary-label ebk-technical-heading">{{ tt('Base Amount') }}</span>
                                            <span class="ebk-mgmt-summary-value">{{ getBaseAmountDisplay() }}</span>
                                        </span>
                                    </div>

                                    <v-table class="ebk-mgmt-table exchange-rates-table" density="default" :hover="!loading">
                                        <thead>
                                        <tr>
                                            <th class="ebk-mgmt-column-name">{{ tt('Currency') }}</th>
                                            <th class="ebk-mgmt-column-amount">{{ tt('Amount') }}</th>
                                            <th class="ebk-mgmt-column-operation">
                                                <span class="ebk-visually-hidden">{{ tt('Operation') }}</span>
                                            </th>
                                        </tr>
                                        </thead>

                                        <tbody aria-hidden="true" v-if="loading && (!exchangeRatesData || !exchangeRatesData.exchangeRates || exchangeRatesData.exchangeRates.length < 1)">
                                        <tr class="ebk-mgmt-row" :key="itemIdx" v-for="itemIdx in [ 1, 2, 3, 4, 5, 6, 7, 8 ]">
                                            <td><v-skeleton-loader class="ebk-mgmt-skeleton" type="text" width="180" :loading="true"></v-skeleton-loader></td>
                                            <td><v-skeleton-loader class="ebk-mgmt-skeleton ms-auto" type="text" width="80" :loading="true"></v-skeleton-loader></td>
                                            <td></td>
                                        </tr>
                                        </tbody>

                                        <tbody v-if="!loading && (!exchangeRatesData || !exchangeRatesData.exchangeRates || !exchangeRatesData.exchangeRates.length)">
                                        <tr class="no-hover">
                                            <td class="ebk-mgmt-column-state" colspan="3">
                                                <state-panel type="error" :title="tt('Unable to retrieve exchange rates data')" v-if="loadError">
                                                    <template #actions>
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiRefresh" @click="reload(true)">{{ tt('Retry') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                                <state-panel type="empty" :title="tt('No exchange rates data')" v-else>
                                                    <template #actions v-if="isUserCustomExchangeRates">
                                                        <v-btn color="default" variant="outlined" :prepend-icon="mdiPencilOutline" @click="update">{{ tt('Update') }}</v-btn>
                                                    </template>
                                                </state-panel>
                                            </td>
                                        </tr>
                                        </tbody>

                                        <tbody v-if="availableExchangeRates.length">
                                        <tr class="ebk-mgmt-row exchange-rates-table-row-data"
                                            :class="{ 'ebk-row-selected': exchangeRate.currencyCode === baseCurrency }"
                                            :key="exchangeRate.currencyCode"
                                            v-for="exchangeRate in availableExchangeRates">
                                            <td class="ebk-mgmt-column-name">
                                                <div class="ebk-mgmt-row-content">
                                                    <span class="exchange-rates-code ebk-font-mono">{{ exchangeRate.currencyCode }}</span>
                                                    <span class="ebk-mgmt-name-text ms-3">{{ exchangeRate.currencyDisplayName }}</span>
                                                    <span class="ebk-mgmt-hidden-chip exchange-rates-base-chip" v-if="exchangeRate.currencyCode === baseCurrency">{{ tt('Base Currency') }}</span>
                                                </div>
                                            </td>
                                            <td class="ebk-mgmt-column-amount">
                                                <span class="ebk-mgmt-amount">{{ getFinalConvertedAmount(exchangeRate, true) }}</span>
                                            </td>
                                            <td class="ebk-mgmt-column-operation">
                                                <div class="ebk-mgmt-actions">
                                                    <v-btn class="ebk-mgmt-reveal" color="default" density="comfortable" variant="text"
                                                           :disabled="loading || updating"
                                                           v-if="exchangeRate.currencyCode !== baseCurrency"
                                                           @click="setAsBaseline(exchangeRate.currencyCode, getFinalConvertedAmount(exchangeRate, false))">
                                                        {{ tt('Set as Base') }}
                                                    </v-btn>
                                                    <v-btn color="error" density="comfortable" variant="text" :icon="true"
                                                           :aria-label="tt('Delete')"
                                                           :loading="customExchangeRateRemoving[exchangeRate.currencyCode]"
                                                           :disabled="loading || updating"
                                                           v-if="exchangeRate.currencyCode !== defaultCurrency && isUserCustomExchangeRates"
                                                           @click="remove(exchangeRate.currencyCode)">
                                                        <template #loader>
                                                            <v-progress-circular indeterminate size="20" width="2"/>
                                                        </template>
                                                        <v-icon :icon="mdiDeleteOutline" size="20" />
                                                        <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
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

    <update-dialog ref="updateDialog" />

    <confirm-dialog ref="confirmDialog"/>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import StatePanel from '@/components/common/StatePanel.vue';
import UpdateDialog from './list/dialogs/UpdateDialog.vue';

import { ref, computed, useTemplateRef, watch } from 'vue';
import { useDisplay } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { useExchangeRatesPageBase } from '@/views/base/ExchangeRatesPageBase.ts';

import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { type BigDecimal, NumeralSystem } from '@/core/numeral.ts';
import { AMOUNT_FACTOR } from '@/consts/numeral.ts';

import type { LocalizedLatestExchangeRate } from '@/models/exchange_rate.ts';

import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import logger from '@/lib/logger.ts';

import {
    mdiRefresh,
    mdiMenu,
    mdiDeleteOutline,
    mdiPencilOutline
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type UpdateDialogType = InstanceType<typeof UpdateDialog>;

const { lgAndUp } = useDisplay();

const { tt, getCurrentNumeralSystemType, formatExchangeRateAmountToWesternArabicNumerals, formatNumberToLocalizedNumerals } = useI18n();
const {
    baseCurrency,
    baseAmount,
    defaultCurrency,
    exchangeRatesData,
    isUserCustomExchangeRates,
    exchangeRatesDataUpdateTime,
    availableExchangeRates,
    getConvertedAmount,
    setAsBaseline
} = useExchangeRatesPageBase();

const exchangeRatesStore = useExchangeRatesStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const updateDialog = useTemplateRef<UpdateDialogType>('updateDialog');

const activeTab = ref<string>('exchangeRatesPage');
const loading = ref<boolean>(true);
const updating = ref<boolean>(false);
const loadError = ref<boolean>(false);
const customExchangeRateRemoving = ref<Record<string, boolean>>({});
const alwaysShowNav = ref<boolean>(lgAndUp.value);
const showNav = ref<boolean>(lgAndUp.value);

const numeralSystem = computed<NumeralSystem>(() => getCurrentNumeralSystemType());
const baseCurrencyDisplayName = computed<string>(() => availableExchangeRates.value.find(exchangeRate => exchangeRate.currencyCode === baseCurrency.value)?.currencyDisplayName ?? baseCurrency.value);

function getBaseAmountDisplay(): string {
    const baseExchangeRate = availableExchangeRates.value.find(exchangeRate => exchangeRate.currencyCode === baseCurrency.value);
    return baseExchangeRate ? getFinalConvertedAmount(baseExchangeRate, true) : '';
}

function reload(force: boolean): void {
    loading.value = true;

    exchangeRatesStore.getLatestExchangeRates({
        silent: false,
        force: force
    }).then(() => {
        loading.value = false;
        loadError.value = false;

        if (exchangeRatesData.value && exchangeRatesData.value.exchangeRates) {
            const exchangeRates = exchangeRatesData.value.exchangeRates;
            let foundDefaultCurrency = false;

            for (const exchangeRate of exchangeRates) {
                if (exchangeRate.currency === baseCurrency.value) {
                    foundDefaultCurrency = true;
                    break;
                }
            }

            if (force) {
                desktopPageStore.addToast('Exchange rates data has been updated');
            } else if (!foundDefaultCurrency) {
                desktopPageStore.addToast('There is no exchange rates data for your default currency');
            }
        }
    }).catch(error => {
        loading.value = false;
        loadError.value = !exchangeRatesData.value || !exchangeRatesData.value.exchangeRates || !exchangeRatesData.value.exchangeRates.length;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function update(): void {
    updateDialog.value?.open().then(result => {
        if (result && result.message) {
            desktopPageStore.addToast(result.message);
        }
    }).catch(error => {
        if (error) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function remove(currency: string): void {
    confirmDialog.value?.open('Are you sure you want to delete this user custom exchange rate?', { color: 'error' }).then(() => {
        updating.value = true;
        customExchangeRateRemoving.value[currency] = true;

        exchangeRatesStore.deleteUserCustomExchangeRate({
            currency: currency
        }).then(() => {
            if (currency === baseCurrency.value) {
                baseCurrency.value = defaultCurrency.value;
            }

            updating.value = false;
            customExchangeRateRemoving.value[currency] = false;
        }).catch(error => {
            updating.value = false;
            customExchangeRateRemoving.value[currency] = false;

            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}

function getFinalConvertedAmount(toExchangeRate: LocalizedLatestExchangeRate, displayLocalizedDigits: boolean): string {
    if (!baseCurrency.value) {
        if (displayLocalizedDigits) {
            return numeralSystem.value.digitZero;
        } else {
            return NumeralSystem.WesternArabicNumerals.digitZero;
        }
    }

    const fromExchangeRate = exchangeRatesStore.latestExchangeRateMap[baseCurrency.value];
    let exchangeRateAmount: BigDecimal | '' | null = BIG_DECIMAL_ZERO;

    try {
        exchangeRateAmount = getConvertedAmount(parseBigDecimal(baseAmount.value).divide(AMOUNT_FACTOR), fromExchangeRate, toExchangeRate);
    } catch (ex) {
        exchangeRateAmount = BIG_DECIMAL_ZERO;
        logger.warn('failed to convert amount by exchange rates, original base amount is ' + baseAmount.value, ex)
    }

    if (!exchangeRateAmount) {
        if (displayLocalizedDigits) {
            return numeralSystem.value.digitZero;
        } else {
            return NumeralSystem.WesternArabicNumerals.digitZero;
        }
    }

    let ret = formatExchangeRateAmountToWesternArabicNumerals(exchangeRateAmount);

    if (displayLocalizedDigits) {
        ret = numeralSystem.value.replaceWesternArabicDigitsToLocalizedDigits(ret);
    }

    return ret;
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
.exchange-rates-content {
    border-radius: 0;
}

.exchange-rates-source a {
    color: var(--ebk-color-primary-text);
}

.exchange-rates-code {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    min-width: 3.25rem;
    height: 28px;
    padding-inline: var(--ebk-space-2);
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--ebk-color-text-primary);
    font-size: 0.8125rem;
    font-weight: 600;
}

.exchange-rates-base-chip {
    border-color: var(--ebk-color-copper-line);
    color: var(--ebk-color-primary-text);
}
</style>

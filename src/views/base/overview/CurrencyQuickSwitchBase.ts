import { ref, computed } from 'vue';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useRootStore } from '@/stores/index.ts';

import { parseQuickSwitchCurrencies } from '@/lib/currency.ts';
import logger from '@/lib/logger.ts';

export function useCurrencyQuickSwitchBase() {
    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const exchangeRatesStore = useExchangeRatesStore();
    const rootStore = useRootStore();

    const switchingCurrency = ref<string | null>(null);

    const quickSwitchCurrencies = computed<string[]>({
        get: () => parseQuickSwitchCurrencies(settingsStore.appSettings.quickSwitchCurrenciesInHomePage),
        set: (value: string[]) => settingsStore.setQuickSwitchCurrenciesInHomePage(parseQuickSwitchCurrencies(value.join(',')).join(','))
    });

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);

    // the currency shown as selected: the requested one while switching (optimistic), otherwise the saved default currency,
    // so a failed request rolls the selection back to the saved default currency
    const selectedCurrency = computed<string>(() => switchingCurrency.value || defaultCurrency.value);

    // the chip row only makes sense when there are at least two currencies to switch between
    const showQuickSwitch = computed<boolean>(() => quickSwitchCurrencies.value.length >= 2);

    /**
     * Changes the user's default currency, every overview amount is converted from it on the client side.
     * Resolves true when the default currency has been changed, false when nothing was done.
     * Rejects (with the selection rolled back) when the profile update fails.
     */
    function switchCurrency(currency: string): Promise<boolean> {
        if (switchingCurrency.value || currency === defaultCurrency.value) {
            return Promise.resolve(false);
        }

        switchingCurrency.value = currency;

        return rootStore.updateUserProfile({ defaultCurrency: currency })
            // the profile update clears the cached exchange rates when the default currency changes, so reload them.
            // the default currency has already been saved at this point, so a failure here only leaves the rates to be loaded later
            .then(() => exchangeRatesStore.getLatestExchangeRates({ silent: true, force: false }).catch(error => {
                logger.warn('failed to reload exchange rates after switching the default currency', error);
            }))
            .then(() => true)
            .finally(() => {
                switchingCurrency.value = null;
            });
    }

    return {
        // states
        switchingCurrency,
        // computed states
        quickSwitchCurrencies,
        defaultCurrency,
        selectedCurrency,
        showQuickSwitch,
        // functions
        switchCurrency
    };
}

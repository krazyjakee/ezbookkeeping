<template>
    <div class="ebk-currency-quick-switch" :class="{ 'ebk-currency-quick-switch--touch': touch }">
        <span class="ebk-currency-quick-switch-label ebk-technical-heading" :id="labelId">{{ tt('Default Currency') }}</span>
        <div ref="options" class="ebk-currency-quick-switch-options" role="group"
             :aria-labelledby="labelId" :aria-busy="switchingCurrency ? 'true' : 'false'"
             @keydown="onKeydown">
            <button type="button" class="ebk-currency-quick-switch-option"
                    :class="{
                        'ebk-currency-quick-switch-option--selected': currency === selectedCurrency,
                        'ebk-currency-quick-switch-option--switching': currency === switchingCurrency
                    }"
                    :key="currency"
                    :data-currency="currency"
                    :tabindex="currency === focusableCurrency ? 0 : -1"
                    :title="getCurrencyName(currency)"
                    :aria-label="`${getCurrencyName(currency)} (${currency})`"
                    :aria-pressed="currency === selectedCurrency ? 'true' : 'false'"
                    :aria-disabled="isDisabled ? 'true' : undefined"
                    @focus="focusedCurrency = currency"
                    @click="onClick(currency)"
                    v-for="currency in currencies">
                <span class="ebk-currency-quick-switch-code">{{ currency }}</span>
                <span class="ebk-currency-quick-switch-spinner ebk-motion-exempt" aria-hidden="true"
                      v-if="currency === switchingCurrency"></span>
            </button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, useId, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

/**
 * Quick switch row for the user's default currency on the overview page, shared by desktop and mobile.
 * It is a group of toggle buttons (aria-pressed) with a roving tab stop: Tab enters the group on the selected currency,
 * the arrow keys / Home / End move the focus, and Enter / Space switch to the focused currency.
 * Switching is not triggered by moving the focus, because every switch saves the user profile.
 */
const props = defineProps<{
    currencies: string[];
    selectedCurrency: string;
    switchingCurrency?: string | null;
    disabled?: boolean;
    touch?: boolean;
}>();

const emit = defineEmits<{
    (e: 'switch', currency: string): void;
}>();

const { tt, getCurrencyName } = useI18n();

const labelId = `ebk-currency-quick-switch-label-${useId()}`;
const options = useTemplateRef<HTMLElement>('options');
const focusedCurrency = ref<string | null>(null);

const isDisabled = computed<boolean>(() => !!props.disabled || !!props.switchingCurrency);

const focusableCurrency = computed<string | undefined>(() => {
    if (focusedCurrency.value && props.currencies.includes(focusedCurrency.value)) {
        return focusedCurrency.value;
    }

    if (props.currencies.includes(props.selectedCurrency)) {
        return props.selectedCurrency;
    }

    return props.currencies[0];
});

function onClick(currency: string): void {
    if (isDisabled.value || currency === props.selectedCurrency) {
        return;
    }

    emit('switch', currency);
}

function onKeydown(event: KeyboardEvent): void {
    if (!options.value || props.currencies.length < 1) {
        return;
    }

    const isRtl = getComputedStyle(options.value).direction === 'rtl';
    const currentIndex = Math.max(0, props.currencies.indexOf(focusableCurrency.value || ''));
    let nextIndex: number;

    switch (event.key) {
        case 'ArrowRight':
            nextIndex = currentIndex + (isRtl ? -1 : 1);
            break;
        case 'ArrowLeft':
            nextIndex = currentIndex + (isRtl ? 1 : -1);
            break;
        case 'ArrowDown':
            nextIndex = currentIndex + 1;
            break;
        case 'ArrowUp':
            nextIndex = currentIndex - 1;
            break;
        case 'Home':
            nextIndex = 0;
            break;
        case 'End':
            nextIndex = props.currencies.length - 1;
            break;
        default:
            return;
    }

    event.preventDefault();

    const count = props.currencies.length;
    const nextCurrency = props.currencies[(nextIndex + count) % count] as string;
    focusedCurrency.value = nextCurrency;
    options.value.querySelector<HTMLElement>(`[data-currency="${nextCurrency}"]`)?.focus();
}
</script>

<style>
.ebk-currency-quick-switch {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-2) var(--ebk-space-3);
    min-width: 0;
}

.ebk-currency-quick-switch-label {
    color: var(--ebk-color-text-secondary);
    overflow-wrap: anywhere;
}

.ebk-currency-quick-switch-options {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-2);
    min-width: 0;
}

.ebk-currency-quick-switch-option {
    /* Framework7 makes bare buttons full width blocks */
    box-sizing: border-box;
    flex: 0 0 auto;
    width: auto;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-2);
    min-width: 56px;
    min-height: 32px;
    margin: 0;
    padding-block: 0;
    padding-inline: var(--ebk-space-3);
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-pill);
    background-color: var(--ebk-color-surface);
    color: var(--ebk-color-text-primary);
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    font-variant-numeric: var(--ebk-font-numeric);
    line-height: 1.2;
    cursor: pointer;
    /* only opacity is animated (motion.md); colour changes are instant */
    transition: opacity var(--ebk-motion-duration-sm) var(--ebk-motion-ease-standard);
    -webkit-tap-highlight-color: transparent;
}

.ebk-currency-quick-switch-option:hover:not([aria-disabled="true"], .ebk-currency-quick-switch-option--selected) {
    border-color: var(--ebk-color-copper-line);
    background-color: var(--ebk-color-selected-background);
}

.ebk-currency-quick-switch-option:focus-visible {
    outline: var(--ebk-focus-outline);
    outline-offset: var(--ebk-focus-ring-offset);
}

.ebk-currency-quick-switch-option--selected {
    border-color: var(--ebk-color-primary);
    background-color: var(--ebk-color-primary);
    color: var(--ebk-color-on-primary);
    cursor: default;
}

.ebk-currency-quick-switch-option[aria-disabled="true"] {
    cursor: progress;
}

.ebk-currency-quick-switch-option[aria-disabled="true"]:not(.ebk-currency-quick-switch-option--selected) {
    opacity: 0.55;
}

.ebk-currency-quick-switch-spinner {
    flex: 0 0 auto;
    width: 12px;
    height: 12px;
    border: 2px solid currentColor;
    border-inline-end-color: transparent;
    border-radius: 50%;
    animation: ebk-currency-quick-switch-spin 0.8s linear infinite;
}

@keyframes ebk-currency-quick-switch-spin {
    to {
        transform: rotate(360deg);
    }
}

/* mobile: 44px touch targets (README §3.4) */
.ebk-currency-quick-switch--touch .ebk-currency-quick-switch-option {
    min-width: 64px;
    min-height: var(--ebk-touch-target-min);
    font-size: 0.875rem;
}

@media (forced-colors: active) {
    .ebk-currency-quick-switch-option {
        border-color: ButtonText;
    }

    .ebk-currency-quick-switch-option--selected {
        border-color: Highlight;
        background-color: Highlight;
        color: HighlightText;
        forced-color-adjust: none;
    }
}
</style>

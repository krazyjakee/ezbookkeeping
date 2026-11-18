<template>
    <span class="ebk-signed-amount" :class="colorClass">
        <span class="ebk-signed-amount-value">{{ sign }}{{ text }}</span>
        <span class="ebk-signed-amount-type" v-if="showTypeLabel && typeLabel">{{ typeLabel }}</span>
        <span class="ebk-visually-hidden" v-else-if="typeLabel">{{ typeLabel }}</span>
    </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

/**
 * Displays an already formatted amount with an explicit sign, the user's amount colour and a type label
 * (design guidelines D1 / D2). Callers pass the formatted amount WITHOUT a sign, e.g. "£5,684.70".
 */
const props = defineProps<{
    text: string;
    type: 'income' | 'expense' | 'transfer' | 'none';
    showTypeLabel?: boolean;
    hideSign?: boolean;
    noColor?: boolean;
}>();

const { tt } = useI18n();

const sign = computed<string>(() => {
    if (props.hideSign || !props.text) {
        return '';
    }

    if (props.type === 'income') {
        return '+';
    } else if (props.type === 'expense') {
        return '−';
    }

    return '';
});

const colorClass = computed<string>(() => {
    if (props.noColor) {
        return '';
    }

    if (props.type === 'income') {
        return 'text-income';
    } else if (props.type === 'expense') {
        return 'text-expense';
    }

    return '';
});

const typeLabel = computed<string>(() => {
    if (props.type === 'income') {
        return tt('Income');
    } else if (props.type === 'expense') {
        return tt('Expense');
    } else if (props.type === 'transfer') {
        return tt('Transfer');
    }

    return '';
});
</script>

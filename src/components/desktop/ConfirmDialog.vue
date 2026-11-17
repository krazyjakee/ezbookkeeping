<template>
    <v-dialog persistent min-width="320" max-width="500" width="auto"
              :content-props="{ 'aria-labelledby': titleId }" v-model="showState" @after-enter="focusSafestAction">
        <v-card class="dialog-layout-card ebk-confirm-dialog" :class="{ 'ebk-confirm-dialog--destructive': isDestructive }">
            <div class="ebk-dialog-title-strip">
                <v-icon class="ebk-dialog-title-icon" size="20" :color="finalColor"
                        :icon="isDestructive ? mdiAlertOutline : mdiHelpCircleOutline" />
                <span class="text-title-medium text-wrap" :id="titleId">{{ titleContent }}</span>
            </div>
            <v-divider />
            <v-card-text class="text-body-large pa-4" v-if="textContent">{{ textContent }}</v-card-text>
            <v-card-actions class="ebk-dialog-actions px-4 pb-4">
                <v-spacer></v-spacer>
                <v-btn ref="cancelButton" color="secondary" variant="tonal" @click="cancel">{{ tt('Cancel') }}</v-btn>
                <v-btn :color="finalColor" @click="confirm">{{ tt('OK') }}</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, useId, useTemplateRef } from 'vue';
import type { ComponentPublicInstance } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { isString, isObject } from '@/lib/common.ts';

import {
    mdiAlertOutline,
    mdiHelpCircleOutline
} from '@mdi/js';

const props = defineProps<{
    show?: boolean;
    color?: string;
    title?: string;
    text?: string;
}>();

const emit = defineEmits<{
    (e: 'update:show', value: boolean): void;
}>();

const { tt } = useI18n();

const showState = ref<boolean>(false);
const titleContent = ref<string>(props.title || tt('global.app.title'));
const textContent = ref<string>(props.text || '');
const finalColor = ref<string>(props.color || 'primary');
const titleId = `ebk-confirm-dialog-title-${useId()}`;
const cancelButton = useTemplateRef<ComponentPublicInstance>('cancelButton');

const isDestructive = computed<boolean>(() => finalColor.value === 'error');

let resolveFunc: ((value?: unknown) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

function open(titleOrText: string, textOrOptions?: string | Record<string, unknown>, options?: Record<string, unknown>): Promise<unknown> {
    showState.value = true;

    if (!textOrOptions || isObject(textOrOptions)) { // only one parameter or second parameter is options
        titleContent.value = tt('global.app.title');

        if (!textOrOptions) {
            textContent.value = tt(titleOrText);
        } else {
            const actualOptions = textOrOptions as Record<string, unknown>;
            textContent.value = tt(titleOrText, actualOptions);
        }
    } else if (isString(textOrOptions)) { // second parameter is text
        if (!options) {
            titleContent.value = tt(titleOrText);
            textContent.value = tt(textOrOptions);
        } else {
            titleContent.value = tt(titleOrText, options);
            textContent.value = tt(textOrOptions, options);
        }
    }

    if (textOrOptions && isObject(textOrOptions) && isString(textOrOptions['color'])){
        finalColor.value = (textOrOptions['color'] as string) || 'primary';
    } else if (options && isString(options['color'])) {
        finalColor.value = (options['color'] as string) || 'primary';
    } else {
        finalColor.value = props.color || 'primary';
    }

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function confirm(): void {
    if (resolveFunc) {
        resolveFunc();
    }

    showState.value = false;
    emit('update:show', false);
}

function cancel(): void {
    if (rejectFunc) {
        rejectFunc();
    }

    showState.value = false;
    emit('update:show', false);
}

// the safest action (cancel) always gets the initial focus, so pressing enter never confirms by accident
function focusSafestAction(): void {
    const element = cancelButton.value?.$el as HTMLElement | undefined;
    element?.focus();
}

watch(showState, newValue => {
    emit('update:show', newValue);
});

defineExpose({
    open
});
</script>

<template>
    <f7-sheet swipe-to-close swipe-handler=".swipe-handler" style="height:auto"
              :opened="show" @sheet:open="onSheetOpen" @sheet:closed="onSheetClosed">
        <f7-toolbar class="toolbar-with-swipe-handler">
            <div class="swipe-handler"></div>
            <div class="left">
                <f7-link icon-f7="xmark" :class="{ 'disabled': loading || recognizing }"
                         :aria-label="tt('Cancel')"
                         @click="cancel"></f7-link>
            </div>
            <div class="right">
                <f7-button round fill icon-f7="checkmark_alt"
                           :class="{ 'disabled': loading || recognizing || !imageFile }"
                           :aria-label="tt('Confirm')"
                           @click="confirm"></f7-button>
            </div>
        </f7-toolbar>
        <f7-page-content class="no-margin-vertical no-padding-vertical">
            <div class="image-container display-flex justify-content-center"
                 :class="{ 'image-container-empty': !imageSrc, 'image-container-busy': loading || recognizing }"
                 :aria-busy="loading || recognizing" @click="showOpenImage">
                <img :src="imageSrc" v-if="imageSrc" />
                <div class="image-container-background display-flex justify-content-center align-items-center text-align-center padding-horizontal" v-if="!imageSrc">
                    <div class="display-inline-flex flex-direction-column align-items-center" v-if="!loading">
                        <span class="image-container-icon" aria-hidden="true">
                            <f7-icon f7="doc_text_viewfinder"></f7-icon>
                        </span>
                        <span>{{ tt('Click here to select a receipt or transaction image') }}</span>
                        <small class="margin-top-half image-container-hint">{{ tt('Uploaded image and personal data will be sent to the large language model, please be aware of potential privacy risks.') }}</small>
                    </div>
                    <div class="display-inline-flex flex-direction-column align-items-center" role="status" v-else-if="loading">
                        <f7-preloader :size="28" />
                        <span class="margin-top-half">{{ tt('Loading image...') }}</span>
                    </div>
                </div>
            </div>
            <div class="ai-image-recognition-state" v-if="recognitionFailed && !recognizing && !loading">
                <state-panel type="error" compact :title="tt('Unable to recognize image')">
                    <template #actions>
                        <f7-button small tonal :text="tt('Retry')" @click="confirm"></f7-button>
                    </template>
                </state-panel>
            </div>
        </f7-page-content>

        <input ref="imageInput" type="file" style="display: none" :accept="`${SUPPORTED_IMAGE_EXTENSIONS};capture=camera`" @change="openImage($event)" />
    </f7-sheet>
</template>


<script setup lang="ts">
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, closeAllDialog } from '@/lib/ui/mobile.ts';

import { useTransactionsStore } from '@/stores/transaction.ts';

import { ImageUploadQualityType } from '@/core/image.ts';
import { KnownFileType } from '@/core/file.ts';
import { SUPPORTED_IMAGE_EXTENSIONS } from '@/consts/file.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { generateRandomUUID } from '@/lib/misc.ts';
import { compressJpgImageByQuality } from '@/lib/ui/common.ts';
import logger from '@/lib/logger.ts';

export interface AIImageRecognitionResult {
    response: RecognizedTransactionResponse;
    imageFile: File;
}

defineProps<{
    show: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:show', value: boolean): void;
    (e: 'recognition:change', value: AIImageRecognitionResult): void;
}>();

const { tt } = useI18n();
const { showCancelableLoading, showToast } = useI18nUIComponents();

const transactionsStore = useTransactionsStore();

const imageInput = useTemplateRef<HTMLInputElement>('imageInput');

const loading = ref<boolean>(false);
const recognizing = ref<boolean>(false);
const cancelRecognizingUuid = ref<string | undefined>(undefined);
const imageFile = ref<File | null>(null);
const imageSrc = ref<string | undefined>(undefined);
const recognitionFailed = ref<boolean>(false);

function loadImage(image: Blob): void {
    loading.value = true;
    recognitionFailed.value = false;
    imageFile.value = null;
    imageSrc.value = undefined;

    compressJpgImageByQuality(image, ImageUploadQualityType.HD720P).then(blob => {
        imageFile.value = KnownFileType.JPG.createFileFromBlob(blob, "image");
        imageSrc.value = URL.createObjectURL(blob);
        loading.value = false;
    }).catch(error => {
        imageFile.value = null;
        imageSrc.value = undefined;
        loading.value = false;
        logger.error('failed to compress image', error);
        showToast('Unable to load image');
    });
}

function showOpenImage(): void {
    if (loading.value || recognizing.value) {
        return;
    }

    imageInput.value?.click();
}

function openImage(event: Event): void {
    if (!event || !event.target) {
        return;
    }

    const el = event.target as HTMLInputElement;

    if (!el.files || !el.files.length || !el.files[0]) {
        return;
    }

    const image = el.files[0] as File;

    el.value = '';

    loadImage(image);
}

function confirm(): void {
    if (loading.value || recognizing.value || !imageFile.value) {
        return;
    }

    const currentImageFile = imageFile.value;
    cancelRecognizingUuid.value = generateRandomUUID();
    recognizing.value = true;
    recognitionFailed.value = false;
    showCancelableLoading('Recognizing', 'AI can make mistakes. Check important info.', 'Cancel Recognition', cancelRecognize);

    transactionsStore.recognizeReceiptImage({
        imageFile: imageFile.value,
        cancelableUuid: cancelRecognizingUuid.value
    }).then(response => {
        recognizing.value = false;
        cancelRecognizingUuid.value = undefined;
        closeAllDialog();
        emit('update:show', false);
        emit('recognition:change', { response: response, imageFile: currentImageFile });
    }).catch(error => {
        if (error.canceled) {
            return;
        }

        recognizing.value = false;
        cancelRecognizingUuid.value = undefined;
        recognitionFailed.value = true;
        closeAllDialog();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function cancelRecognize(): void {
    if (!cancelRecognizingUuid.value) {
        return;
    }

    transactionsStore.cancelRecognizeReceiptImage(cancelRecognizingUuid.value);
    recognizing.value = false;
    cancelRecognizingUuid.value = undefined;
    closeAllDialog();

    showToast('User Canceled');
}

function cancel(): void {
    close();
}

function close(): void {
    emit('update:show', false);
    recognitionFailed.value = false;
    loading.value = false;
    recognizing.value = false;
    cancelRecognizingUuid.value = undefined;
    imageFile.value = null;
    imageSrc.value = undefined;
}

function onSheetOpen(): void {
    recognitionFailed.value = false;
    loading.value = false;
    recognizing.value = false;
    cancelRecognizingUuid.value = undefined;
    imageFile.value = null;
    imageSrc.value = undefined;
}

function onSheetClosed(): void {
    close();
}

defineExpose({
    loadImage
});
</script>

<style>
.image-container {
    --ebk-ai-image-recognition-height: 310px;
    height: var(--ebk-ai-image-recognition-height);
    border: 1px solid var(--ebk-color-hairline);
    background-color: var(--ebk-color-surface-sunken);
    cursor: pointer;

    &.image-container-empty {
        border: 1px dashed var(--ebk-color-copper-line);
    }

    &.image-container-busy {
        cursor: progress;
    }

    > img {
        height: var(--ebk-ai-image-recognition-height);
    }

    @media (min-height: 630px) {
        --ebk-ai-image-recognition-height: 525px;
    }
}

.image-container-background {
    width: 100%;
    height: 100%;
    color: var(--ebk-color-text-primary);

    > div {
        font-size: var(--f7-input-font-size);
    }
}

.image-container-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    margin-bottom: var(--ebk-space-3);
    border: 1px solid var(--ebk-color-copper-line);
    border-radius: var(--ebk-radius-pill);
    color: var(--ebk-color-primary-text);

    > .icon {
        font-size: 24px;
    }
}

.image-container-hint {
    color: var(--ebk-color-text-secondary);
}

.ai-image-recognition-state {
    padding: var(--ebk-space-3);
}
</style>

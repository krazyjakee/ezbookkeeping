<template>
    <v-dialog width="800" :persistent="loading || recognizing || !!imageFile" v-model="showState" @paste="onPaste">
        <one-column-dialog-layout content-class="pa-0" content-style="height: 500px"
                                  :disabled="loading || recognizing" :loading="loading || recognizing"
                                  :title="tt('AI Image Recognition')"
                                  :cancel-button-title="tt('Cancel')"
                                  @cancel="cancel">
            <template #toolbar>
                <v-btn class="mx-2" density="comfortable" variant="outlined"
                       :disabled="loading || recognizing || !imageFile" @click="recognize"
                       v-if="!recognizing">{{ tt('Recognize') }}
                </v-btn>
                <v-btn class="mx-2" density="comfortable" variant="outlined"
                       :disabled="loading"
                       @click="cancelRecognize" v-if="recognizing && cancelRecognizingUuid">{{ tt('Cancel Recognition') }}</v-btn>
            </template>

            <template #content>
                <div class="ai-recognition-stage"
                     :class="{ 'ai-recognition-stage--over': isDragOver, 'ai-recognition-stage--has-image': !!imageSrc }"
                     :aria-busy="loading || recognizing ? 'true' : 'false'"
                     @dragenter.prevent="onDragEnter"
                     @dragover.prevent
                     @dragleave.prevent="onDragLeave"
                     @drop.prevent="onDrop">
                    <button type="button" class="ai-recognition-image-button"
                            :aria-label="tt('You can drag and drop, paste or click to select a receipt or transaction image')"
                            :disabled="loading || recognizing"
                            v-if="imageSrc"
                            @click="showOpenImageDialog">
                        <v-img class="ai-recognition-image" :src="imageSrc">
                            <template #placeholder>
                                <div class="ai-recognition-image-placeholder"></div>
                            </template>
                        </v-img>
                    </button>

                    <button type="button" class="ai-recognition-empty"
                            :disabled="loading || recognizing"
                            v-else-if="!loading"
                            @click="showOpenImageDialog">
                        <span class="ai-recognition-empty-icon" aria-hidden="true">
                            <v-icon :icon="mdiImagePlusOutline" size="28" />
                        </span>
                        <span class="ai-recognition-empty-title">{{ isDragOver ? tt('Release to load image') : tt('You can drag and drop, paste or click to select a receipt or transaction image') }}</span>
                        <span class="ai-recognition-empty-text">{{ tt('Uploaded image and personal data will be sent to the large language model, please be aware of potential privacy risks.') }}</span>
                    </button>

                    <div class="ai-recognition-overlay" role="status" v-if="loading">
                        <v-progress-circular indeterminate size="32" />
                        <span class="ai-recognition-overlay-title">{{ tt('Loading image...') }}</span>
                    </div>

                    <div class="ai-recognition-overlay" role="status" v-else-if="isDragOver && imageSrc">
                        <span class="ai-recognition-overlay-title">{{ tt('Release to load image') }}</span>
                    </div>

                    <div class="ai-recognition-status" v-if="recognizing">
                        <v-progress-linear indeterminate color="primary" height="3" />
                        <div class="ai-recognition-status-body">
                            <v-icon :icon="mdiMagicStaff" size="20" />
                            <div class="ai-recognition-status-text">
                                <span class="ai-recognition-status-title">{{ tt('Recognizing') }}</span>
                                <span class="ai-recognition-status-hint">{{ tt('AI can make mistakes. Check important info.') }}</span>
                            </div>
                        </div>
                    </div>

                    <div class="ai-recognition-failure" v-else-if="recognizeFailed && !loading">
                        <state-panel type="error" compact :title="tt('Unable to recognize image')">
                            <template #actions>
                                <v-btn color="default" variant="outlined" density="comfortable" :prepend-icon="mdiRefresh"
                                       :disabled="!imageFile" @click="recognize">{{ tt('Retry') }}</v-btn>
                            </template>
                        </state-panel>
                    </div>
                    <span class="ebk-visually-hidden" role="status" aria-live="polite">{{ recognizing ? tt('Recognizing') : '' }}</span>
                </div>
            </template>

        </one-column-dialog-layout>
    </v-dialog>

    <input ref="imageInput" type="file" style="display: none" :accept="SUPPORTED_IMAGE_EXTENSIONS" @change="openImage($event)" />
</template>

<script setup lang="ts">
import StatePanel from '@/components/common/StatePanel.vue';

import { ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useTransactionsStore } from '@/stores/transaction.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { ImageUploadQualityType } from '@/core/image.ts';
import { KnownFileType } from '@/core/file.ts';
import { SUPPORTED_IMAGE_EXTENSIONS } from '@/consts/file.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

export interface AIImageRecognitionResult {
    response: RecognizedTransactionResponse;
    imageFile: File;
}

import { generateRandomUUID } from '@/lib/misc.ts';
import { compressJpgImageByQuality } from '@/lib/ui/common.ts';
import logger from '@/lib/logger.ts';

import {
    mdiImagePlusOutline,
    mdiMagicStaff,
    mdiRefresh
} from '@mdi/js';

const { tt } = useI18n();

const transactionsStore = useTransactionsStore();
const desktopPageStore = useDesktopPageStore();

const imageInput = useTemplateRef<HTMLInputElement>('imageInput');

let resolveFunc: ((result: AIImageRecognitionResult) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);
const loading = ref<boolean>(false);
const recognizing = ref<boolean>(false);
const cancelRecognizingUuid = ref<string | undefined>(undefined);
const imageFile = ref<File | null>(null);
const imageSrc = ref<string | undefined>(undefined);
const isDragOver = ref<boolean>(false);
const recognizeFailed = ref<boolean>(false);

// increased whenever a new image is loaded or the dialog is reset, so that a slower earlier image load cannot overwrite the current state
let imageLoadSequence: number = 0;

function loadImage(file: File): void {
    const currentImageLoadSequence = ++imageLoadSequence;

    loading.value = true;
    recognizeFailed.value = false;
    imageFile.value = null;
    imageSrc.value = undefined;

    compressJpgImageByQuality(file, ImageUploadQualityType.HD720P).then(blob => {
        if (currentImageLoadSequence !== imageLoadSequence) {
            return;
        }

        imageFile.value = KnownFileType.JPG.createFileFromBlob(blob, "image");
        imageSrc.value = URL.createObjectURL(blob);
        loading.value = false;
    }).catch(error => {
        if (currentImageLoadSequence !== imageLoadSequence) {
            return;
        }

        imageFile.value = null;
        imageSrc.value = undefined;
        loading.value = false;
        logger.error('failed to compress image', error);
        desktopPageStore.addErrorToast('Unable to load image');
    });
}

function open(): Promise<AIImageRecognitionResult> {
    imageLoadSequence++;
    showState.value = true;
    loading.value = false;
    recognizing.value = false;
    recognizeFailed.value = false;
    cancelRecognizingUuid.value = undefined;
    imageFile.value = null;
    imageSrc.value = undefined;

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function showOpenImageDialog(): void {
    if (loading.value || recognizing.value || isDragOver.value) {
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

function recognize(): void {
    if (loading.value || recognizing.value || !imageFile.value) {
        return;
    }

    const currentImageFile = imageFile.value;
    const currentRecognizingUuid = generateRandomUUID();
    cancelRecognizingUuid.value = currentRecognizingUuid;
    recognizing.value = true;
    recognizeFailed.value = false;

    transactionsStore.recognizeReceiptImage({
        imageFile: currentImageFile,
        cancelableUuid: currentRecognizingUuid
    }).then(response => {
        // ignore the result of a recognition which has been canceled or superseded
        if (cancelRecognizingUuid.value !== currentRecognizingUuid) {
            return;
        }

        resolveFunc?.({ response: response, imageFile: currentImageFile });
        showState.value = false;
        recognizing.value = false;
        cancelRecognizingUuid.value = undefined;
    }).catch(error => {
        if (error.canceled || cancelRecognizingUuid.value !== currentRecognizingUuid) {
            return;
        }

        recognizing.value = false;
        recognizeFailed.value = true;
        cancelRecognizingUuid.value = undefined;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
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

    desktopPageStore.addToast('User Canceled');
}

function cancel(): void {
    imageLoadSequence++;

    if (cancelRecognizingUuid.value) {
        transactionsStore.cancelRecognizeReceiptImage(cancelRecognizingUuid.value);
    }

    rejectFunc?.();
    showState.value = false;
    loading.value = false;
    recognizing.value = false;
    recognizeFailed.value = false;
    cancelRecognizingUuid.value = undefined;
    imageFile.value = null;
    imageSrc.value = undefined;
}

function onDragEnter(): void {
    if (loading.value || recognizing.value) {
        return;
    }

    isDragOver.value = true;
}

function onDragLeave(): void {
    isDragOver.value = false;
}

function onDrop(event: DragEvent): void {
    if (loading.value || recognizing.value) {
        return;
    }

    isDragOver.value = false;

    if (event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files.length && event.dataTransfer.files[0]) {
        loadImage(event.dataTransfer.files[0] as File);
    }
}

function onPaste(event: ClipboardEvent) {
    if (!event.clipboardData) {
        event.preventDefault();
        return;
    }

    for (let i = 0; i < event.clipboardData.items.length; i++) {
        const item = event.clipboardData.items[i];

        if (item && item.type.startsWith('image/')) {
            const file = item.getAsFile();

            if (file) {
                loadImage(file);
                event.preventDefault();
                return;
            }
        }
    }
}

defineExpose({
    open
});
</script>

<style>
.ai-recognition-stage {
    position: relative;
    display: flex;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background-color: var(--ebk-color-surface-sunken);
}

.ai-recognition-stage--over {
    box-shadow: inset 0 0 0 2px var(--ebk-color-primary);
}

.ai-recognition-image-button {
    display: block;
    width: 100%;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    cursor: pointer;

    &:disabled {
        cursor: default;
    }

    &:focus-visible {
        outline: var(--ebk-focus-outline);
        outline-offset: -4px;
    }
}

.ai-recognition-image {
    width: 100%;
    height: 100%;
}

.ai-recognition-image-placeholder {
    width: 100%;
    height: 100%;
    background-color: var(--ebk-color-surface-sunken);
}

.ai-recognition-empty {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-3);
    margin: var(--ebk-space-4);
    padding: var(--ebk-space-6);
    border: 1px dashed rgba(var(--ebk-color-hairline-rgb), calc(var(--ebk-color-hairline-opacity) * 3));
    border-radius: var(--ebk-radius-panel);
    background: transparent;
    color: var(--ebk-color-text-secondary);
    text-align: center;
    cursor: pointer;
    transition: border-color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);

    &:hover:not(:disabled),
    .ai-recognition-stage--over > & {
        border-color: var(--ebk-color-primary);
        background-color: var(--ebk-color-selected-background);
    }
}

.ai-recognition-empty-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
    border: 1px solid var(--ebk-color-copper-line);
    border-radius: var(--ebk-radius-pill);
    color: var(--ebk-color-primary-text);
}

.ai-recognition-empty-title {
    max-width: 32rem;
    color: var(--ebk-color-text-primary);
    font-size: 1rem;
    font-weight: 600;
    overflow-wrap: anywhere;
}

.ai-recognition-empty-text {
    max-width: 32rem;
    font-size: 0.875rem;
    overflow-wrap: anywhere;
}

.ai-recognition-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--ebk-space-3);
    background-color: rgba(var(--ebk-color-surface-rgb), 0.82);
    color: var(--ebk-color-text-primary);
}

.ai-recognition-overlay-title {
    font-size: 1rem;
    font-weight: 600;
}

.ai-recognition-status,
.ai-recognition-failure {
    position: absolute;
    inset-inline: var(--ebk-space-4);
    inset-block-end: var(--ebk-space-4);
    overflow: hidden;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-panel);
    background-color: var(--ebk-color-surface-raised);
    box-shadow: var(--ebk-shadow-md);
}

.ai-recognition-failure {
    border: 0;

    > .ebk-state-panel {
        background-color: var(--ebk-color-surface-raised);
    }
}

.ai-recognition-status-body {
    display: flex;
    align-items: center;
    gap: var(--ebk-space-3);
    padding: var(--ebk-space-3) var(--ebk-space-4);

    > .v-icon {
        color: var(--ebk-color-primary-text);
    }
}

.ai-recognition-status-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
}

.ai-recognition-status-title {
    color: var(--ebk-color-text-primary);
    font-weight: 600;
}

.ai-recognition-status-hint {
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
    overflow-wrap: anywhere;
}
</style>

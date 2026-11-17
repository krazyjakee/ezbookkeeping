<template>
    <div class="layout-wrapper auth-page ebk-canvas-grid" :class="{ 'auth-page--wide': wide }">
        <div class="auth-layout">
            <section class="auth-brand" :aria-label="tt('global.app.title')">
                <router-link class="auth-wordmark" to="/">
                    <img alt="" class="auth-wordmark-logo" :src="APPLICATION_LOGO_PATH" />
                    <span class="auth-wordmark-title">{{ tt('global.app.title') }}</span>
                </router-link>
                <p class="auth-claim">{{ tt('Your business, clearly balanced.') }}</p>
                <div class="auth-brand-illustration">
                    <auth-illustration :variant="illustration" :verified="verified" />
                </div>
            </section>

            <main class="auth-access">
                <router-link class="auth-wordmark auth-wordmark--compact" to="/">
                    <img alt="" class="auth-wordmark-logo" :src="APPLICATION_LOGO_PATH" />
                    <span class="auth-wordmark-title">{{ tt('global.app.title') }}</span>
                </router-link>

                <section class="ebk-panel ebk-panel--bracketed auth-access-panel" :aria-labelledby="titleId" :aria-busy="busy ? 'true' : 'false'">
                    <header class="ebk-panel-header-strip auth-access-panel-header">
                        <span class="auth-access-panel-label">{{ label }}</span>
                        <div class="ebk-panel-header-actions auth-access-panel-meta">
                            <slot name="meta"></slot>
                            <span class="auth-access-panel-live" :class="{ 'auth-access-panel-live--busy': busy }" aria-hidden="true"></span>
                        </div>
                    </header>
                    <v-progress-linear class="auth-access-panel-progress" color="primary" height="2" indeterminate :active="busy" />

                    <div class="auth-access-panel-body">
                        <h1 class="auth-access-title" :id="titleId">{{ title }}</h1>
                        <div class="auth-access-message">
                            <slot name="message"></slot>
                        </div>

                        <transition name="ebk-fade">
                            <div class="auth-access-status" :class="`auth-access-status--${statusType}`"
                                 :role="statusType === 'error' ? 'alert' : 'status'"
                                 v-if="statusText">
                                <v-icon class="auth-access-status-icon" size="20" :icon="statusIcon" />
                                <span class="auth-access-status-text">{{ statusText }}</span>
                            </div>
                        </transition>

                        <slot></slot>
                    </div>
                </section>

                <footer class="auth-access-footer" v-if="showFooter">
                    <language-select-button :disabled="busy || languageDisabled" @error="desktopPageStore.addErrorToast" />
                    <div class="auth-powered-by">
                        <span>Powered by </span>
                        <a href="https://github.com/krazyjakee/ezbookkeeping" target="_blank" rel="noopener">ezrbookkeeping</a>&nbsp;<span class="ebk-numeric">{{ version }}</span>
                    </div>
                </footer>
            </main>
        </div>

        <slot name="dialogs"></slot>
    </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';

import {
    mdiAlertCircleOutline,
    mdiCheckCircleOutline,
    mdiInformationOutline
} from '@mdi/js';

type AuthIllustrationVariant = 'login' | 'signup' | 'verify' | 'unlock' | 'recovery' | 'connect';
type AuthStatusType = 'error' | 'success' | 'info';

const props = withDefaults(defineProps<{
    illustration: AuthIllustrationVariant;
    label: string;
    title: string;
    version?: string;
    verified?: boolean;
    wide?: boolean;
    busy?: boolean;
    languageDisabled?: boolean;
    showFooter?: boolean;
    statusType?: AuthStatusType;
    statusText?: string | null;
}>(), {
    version: '',
    verified: false,
    wide: false,
    busy: false,
    languageDisabled: false,
    showFooter: true,
    statusType: 'error',
    statusText: null
});

const { tt } = useI18n();

const desktopPageStore = useDesktopPageStore();

const titleId = `auth-access-title-${useId()}`;

const statusIcon = computed<string>(() => {
    if (props.statusType === 'success') {
        return mdiCheckCircleOutline;
    } else if (props.statusType === 'info') {
        return mdiInformationOutline;
    } else {
        return mdiAlertCircleOutline;
    }
});
</script>

<template>
    <f7-page no-navbar no-swipeback login-screen hide-toolbar-on-scroll class="login-page unlock-page">
        <div class="ebk-auth-wordmark">
            <img alt="" class="ebk-auth-wordmark-logo" :src="APPLICATION_LOGO_PATH" />
            <span class="ebk-auth-wordmark-title">{{ tt('global.app.title') }}</span>
        </div>

        <section class="ebk-panel ebk-panel--bracketed ebk-auth-panel" aria-labelledby="mobile-unlock-title"
                 :aria-busy="unlocking ? 'true' : 'false'">
            <header class="ebk-panel-header-strip ebk-auth-panel-header">
                <span>{{ tt('Application Lock') }}</span>
                <span class="ebk-panel-header-actions ebk-auth-panel-live" :class="{ 'ebk-auth-panel-live--busy': unlocking }" aria-hidden="true"></span>
            </header>

            <div class="ebk-auth-panel-intro">
                <h1 class="ebk-auth-title" id="mobile-unlock-title">{{ tt('Unlock Application') }}</h1>
                <p class="ebk-auth-message" v-if="isWebAuthnAvailable">{{ tt('Please enter your PIN code or use WebAuthn to unlock application') }}</p>
                <p class="ebk-auth-message" v-else>{{ tt('Please enter your PIN code to unlock application') }}</p>
            </div>

            <f7-list form class="login-page-form no-margin-top">
                <f7-list-item class="list-item-pincode-input margin-bottom">
                    <pin-code-input :disabled="unlocking" :secure="true" :length="6" :auto-confirm="true" v-model="pinCode" @pincode:confirm="unlockByPin" />
                </f7-list-item>

                <f7-list-button class="login-page-primary-action" :class="{ 'disabled': unlocking || !isPinCodeValid(pinCode) }"
                                :text="tt('Unlock with PIN Code')" @click="unlockByPin"></f7-list-button>
                <f7-list-button class="login-page-secondary-action margin-top" :class="{ 'disabled': unlocking }" v-if="isWebAuthnAvailable"
                                :text="tt('Unlock with WebAuthn')" @click="unlockByWebAuthn"></f7-list-button>
            </f7-list>

            <div class="ebk-auth-panel-secondary">
                <span style="margin-inline-end: 0.3rem;">{{ tt('Can\'t Unlock?') }}</span>
                <f7-link :class="{ 'disabled': unlocking }" :text="tt('Re-login')" @click="relogin"></f7-link>
            </div>
        </section>

        <f7-list class="login-page-bottom">
            <f7-block-footer>
                <language-select-button :disabled="unlocking" @error="showToast" />

                <div class="login-page-powered-by margin-top-half">
                    <span>Powered by</span>
                    <f7-link @click="openExternalUrl('https://github.com/krazyjakee/ezbookkeeping')" target="_blank">ezrbookkeeping</f7-link>
                    <span>{{ version }}</span>
                </div>
            </f7-block-footer>
        </f7-list>

        <f7-toolbar class="login-page-fixed-bottom" tabbar bottom :outline="false">
            <language-select-button :disabled="unlocking" @error="showToast" />

            <div class="login-page-powered-by margin-top-half">
                <span>Powered by</span>
                <f7-link @click="openExternalUrl('https://github.com/krazyjakee/ezbookkeeping')" target="_blank">ezrbookkeeping</f7-link>
                <span>{{ version }}</span>
            </div>
        </f7-toolbar>
    </f7-page>
</template>

<script setup lang="ts">
import type { Router } from 'framework7/types';
import { ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useUnlockPageBase } from '@/views/base/UnlockPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';
import {
    isWebAuthnSupported,
    verifyWebAuthnCredential
} from '@/lib/webauthn.ts';
import {
    unlockTokenByWebAuthn,
    unlockTokenByPinCode,
    hasWebAuthnConfig,
    getWebAuthnCredentialId
} from '@/lib/userstate.ts';
import { isModalShowing } from '@/lib/ui/mobile.ts';
import logger from '@/lib/logger.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, showConfirm, openExternalUrl } = useI18nUIComponents();
const { version, pinCode, isWebAuthnAvailable, isPinCodeValid, doAfterUnlocked, doRelogin } = useUnlockPageBase();

const settingsStore = useSettingsStore();
const userStore = useUserStore();

const unlocking = ref<boolean>(false);

function unlockByWebAuthn(): void {
    if (unlocking.value) {
        return;
    }

    const router = props.f7router;
    const webAuthnCredentialId = getWebAuthnCredentialId();

    if (!userStore.currentUserBasicInfo || !webAuthnCredentialId) {
        showToast('An error occurred');
        return;
    }

    if (!settingsStore.appSettings.applicationLockWebAuthn || !hasWebAuthnConfig()) {
        showToast('WebAuthn is not enabled');
        return;
    }

    if (!isWebAuthnSupported()) {
        showToast('WebAuth is not supported on this device');
        return;
    }

    showLoading();

    verifyWebAuthnCredential(
        userStore.currentUserBasicInfo,
        webAuthnCredentialId
    ).then(({ id, userName, userSecret }) => {
        if (unlockTokenByWebAuthn(id, userName, userSecret)) {
            unlocking.value = true;

            doAfterUnlocked().then(unlockHint => {
                if (unlockHint) {
                    showToast(unlockHint);
                }

                router.refreshPage();
            }).catch(error => {
                logger.error('failed to set language after unlocking', error);

                if (!error.processed) {
                    showToast(error.message || error);
                }
            }).finally(() => {
                unlocking.value = false;
                hideLoading();
            });
        } else {
            hideLoading();
            logger.error('failed to unlock with webauthn');
            showToast('Failed to authenticate with WebAuthn');
        }
    }).catch(error => {
        hideLoading();
        logger.error('failed to use webauthn to verify', error);

        if (error.notSupported) {
            showToast('WebAuth is not supported on this device');
        } else if (error.name === 'NotAllowedError') {
            showToast('User has canceled authentication');
        } else if (error.invalid) {
            showToast('Failed to authenticate with WebAuthn');
        } else {
            showToast('User has canceled or this device does not support WebAuthn');
        }
    });
}

function unlockByPin(pinCode: string): void {
    if (unlocking.value || !isPinCodeValid(pinCode)) {
        return;
    }

    if (isModalShowing()) {
        return;
    }

    const router = props.f7router;
    const user = userStore.currentUserBasicInfo;

    if (!user || !user.username) {
        showToast('An error occurred');
        return;
    }

    try {
        if (unlockTokenByPinCode(user.username, pinCode)) {
            unlocking.value = true;
            showLoading(() => unlocking.value);

            doAfterUnlocked().then(unlockHint => {
                if (unlockHint) {
                    showToast(unlockHint);
                }

                router.refreshPage();
            }).catch(error => {
                logger.error('failed to set language after unlocking', error);

                if (!error.processed) {
                    showToast(error.message || error);
                }
            }).finally(() => {
                unlocking.value = false;
                hideLoading();
            });
        } else {
            logger.error('failed to unlock with pin code');
            showToast('Incorrect PIN code');
        }
    } catch (ex) {
        logger.error('failed to unlock with pin code', ex);
        showToast('Incorrect PIN code');
    }
}

function relogin(): void {
    const router = props.f7router;

    showConfirm('Are you sure you want to re-login?', () => {
        doRelogin().then(() => {
            router.navigate('/login', {
                clearPreviousHistory: true
            });
        }).catch(error => {
            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    });
}
</script>

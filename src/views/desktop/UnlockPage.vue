<template>
    <auth-page-layout illustration="unlock"
                      :label="tt('Application Lock')"
                      :title="tt('Unlock Application')"
                      :version="version"
                      :busy="unlocking || verifyingByWebAuthn"
                      :status-text="errorMessage">
        <template #message>
            <p class="ma-0" v-if="isWebAuthnAvailable">{{ tt('Please enter your PIN code or use WebAuthn to unlock application') }}</p>
            <p class="ma-0" v-else>{{ tt('Please enter your PIN code to unlock application') }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent>
            <div>
                <span class="auth-access-field-label ebk-technical-heading">{{ tt('PIN Code') }}</span>
                <pin-code-input :disabled="unlocking || verifyingByWebAuthn" :autofocus="true"
                                :secure="true" :length="6" :auto-confirm="true"
                                v-model="pinCode" @pincode:confirm="unlockByPin" />
            </div>

            <v-btn block size="large" color="primary"
                   :loading="unlocking && !verifyingByWebAuthn"
                   :disabled="unlocking || verifyingByWebAuthn || !isPinCodeValid(pinCode)"
                   @click="unlockByPin(pinCode)">
                {{ tt('Unlock with PIN Code') }}
            </v-btn>

            <div class="auth-access-divider" v-if="isWebAuthnAvailable">{{ tt('or') }}</div>

            <v-btn block size="large" variant="outlined" color="default"
                   :loading="verifyingByWebAuthn"
                   :disabled="unlocking || verifyingByWebAuthn"
                   @click="unlockByWebAuthn" v-if="isWebAuthnAvailable">
                {{ tt('Unlock with WebAuthn') }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary auth-links">
            <div>
                <span class="me-1">{{ tt('Can\'t Unlock?') }}</span>
                <a href="javascript:void(0);" :class="{ 'disabled': unlocking || verifyingByWebAuthn }" @click="relogin">{{ tt('Re-login') }}</a>
            </div>
        </div>

        <template #dialogs>
            <confirm-dialog ref="confirmDialog"/>
        </template>
    </auth-page-layout>
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref, useTemplateRef } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';
import { useUnlockPageBase } from '@/views/base/UnlockPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

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
import logger from '@/lib/logger.ts';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;

const router = useRouter();

const { tt } = useI18n();
const { version, pinCode, isWebAuthnAvailable, isPinCodeValid, doAfterUnlocked, doRelogin } = useUnlockPageBase();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const desktopPageStore = useDesktopPageStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');

const unlocking = ref<boolean>(false);
const verifyingByWebAuthn = ref<boolean>(false);
const errorMessage = ref<string | null>(null);

function unlockByWebAuthn(): void {
    if (unlocking.value) {
        return;
    }

    const webAuthnCredentialId = getWebAuthnCredentialId();

    if (!userStore.currentUserBasicInfo || !webAuthnCredentialId) {
        errorMessage.value = tt('An error occurred');
        return;
    }

    if (!settingsStore.appSettings.applicationLockWebAuthn || !hasWebAuthnConfig()) {
        errorMessage.value = tt('WebAuthn is not enabled');
        return;
    }

    if (!isWebAuthnSupported()) {
        errorMessage.value = tt('WebAuth is not supported on this device');
        return;
    }

    verifyingByWebAuthn.value = true;
    errorMessage.value = null;

    verifyWebAuthnCredential(
        userStore.currentUserBasicInfo,
        webAuthnCredentialId
    ).then(({ id, userName, userSecret }) => {
        verifyingByWebAuthn.value = false;

        if (unlockTokenByWebAuthn(id, userName, userSecret)) {
            unlocking.value = true;

            doAfterUnlocked().then(unlockHint => {
                desktopPageStore.addToast(unlockHint);
                router.replace('/');
            }).catch(error => {
                logger.error('failed to set language after unlocking', error);

                if (!error.processed) {
                    desktopPageStore.addErrorToast(error);
                }
            }).finally(() => {
                unlocking.value = false;
            });
        } else {
            logger.error('failed to unlock with webauthn');
            errorMessage.value = tt('Failed to authenticate with WebAuthn');
        }
    }).catch(error => {
        verifyingByWebAuthn.value = false;
        logger.error('failed to use webauthn to verify', error);

        if (error.notSupported) {
            errorMessage.value = tt('WebAuth is not supported on this device');
        } else if (error.name === 'NotAllowedError') {
            errorMessage.value = tt('User has canceled authentication');
        } else if (error.invalid) {
            errorMessage.value = tt('Failed to authenticate with WebAuthn');
        } else {
            errorMessage.value = tt('User has canceled or this device does not support WebAuthn');
        }
    });
}

function unlockByPin(pinCode: string): void {
    if (unlocking.value || !isPinCodeValid(pinCode)) {
        return;
    }

    errorMessage.value = null;

    const user = userStore.currentUserBasicInfo;

    if (!user || !user.username) {
        errorMessage.value = tt('An error occurred');
        return;
    }

    try {
        if (unlockTokenByPinCode(user.username, pinCode)) {
            unlocking.value = true;

            doAfterUnlocked().then(unlockHint => {
                desktopPageStore.addToast(unlockHint);
                router.replace('/');
            }).catch(error => {
                logger.error('failed to set language after unlocking', error);

                if (!error.processed) {
                    desktopPageStore.addErrorToast(error);
                }
            }).finally(() => {
                unlocking.value = false;
            });
        } else {
            logger.error('failed to unlock with pin code');
            errorMessage.value = tt('Incorrect PIN code');
        }
    } catch (ex) {
        logger.error('failed to unlock with pin code', ex);
        errorMessage.value = tt('Incorrect PIN code');
    }
}

function relogin(): void {
    confirmDialog.value?.open('Are you sure you want to re-login?').then(() => {
        doRelogin().then(() => {
            router.replace('/login');
        }).catch(error => {
            if (!error.processed) {
                desktopPageStore.addErrorToast(error);
            }
        });
    });
}
</script>

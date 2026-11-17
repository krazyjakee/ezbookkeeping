<template>
    <v-row>
        <v-col cols="12">
            <v-card class="ebk-settings-panel" :class="{ 'disabled': loading }">
                <header class="ebk-panel-header-strip ebk-settings-panel-header">
                    <h2 class="ebk-settings-panel-title">{{ tt('Two-Factor Authentication') }}</h2>
                    <div class="ebk-panel-header-actions">
                        <v-progress-circular indeterminate size="18" v-if="loading"></v-progress-circular>
                        <span class="ebk-settings-status" :class="{ 'ebk-settings-status--on': status === true }" v-if="!loading && status !== null && status !== undefined">
                            <v-icon size="16" :icon="status === true ? mdiShieldCheckOutline : mdiShieldOffOutline" />
                            <span>{{ status === true ? tt('Enabled') : tt('Disabled') }}</span>
                        </span>
                    </div>
                </header>

                <v-card-text class="mt-2">
                    <v-skeleton-loader class="skeleton-no-margin pt-1" type="text" style="width: 150px" :loading="true" v-if="loading"></v-skeleton-loader>
                    <span class="text-body-large" v-if="!loading && !new2FAQRCode">
                        {{ status === true ? tt('Two-factor authentication is already enabled.') : tt('Two-factor authentication is not enabled yet.') }}
                    </span>
                    <span class="text-body-large" v-if="!loading && new2FAQRCode">
                        {{ tt('Please use a two-factor authentication app to scan the qrcode below and enter the current passcode.') }}
                    </span>
                    <span class="text-body-large" v-if="!loading && status === true">
                        {{ tt('Your current password is required to disable two-factor authentication or regenerate backup codes for two-factor authentication. If you regenerate backup codes, the previous ones will become invalid.') }}
                    </span>
                </v-card-text>

                <v-card-text class="pt-0 pb-4" v-if="status === false && new2FAQRCode">
                    <div class="ebk-settings-qrcode-frame">
                        <v-img alt="qrcode" class="img-qrcode" :src="new2FAQRCode" />
                    </div>
                    <v-row class="mt-2">
                        <v-col cols="12" md="4">
                            <v-text-field
                                type="number"
                                name="passcode"
                                autocomplete="one-time-code"
                                inputmode="numeric"
                                :autofocus="true"
                                :disabled="loading || enabling || enableConfirming || disabling"
                                :label="tt('Passcode')"
                                :placeholder="tt('Passcode')"
                                v-model="currentPasscode"
                                @keyup.enter="enableConfirm"
                            />
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-text class="pt-0 pb-4" v-if="status === true">
                    <v-row>
                        <v-col cols="12" md="6">
                            <v-text-field
                                name="current-password"
                                autocomplete="current-password"
                                type="password"
                                :disabled="loading || enabling || enableConfirming || disabling"
                                :label="tt('Current Password')"
                                :placeholder="tt('Current Password')"
                                v-model="currentPassword"
                            />
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-text class="ebk-settings-panel-footer" v-if="!loading">
                    <v-btn color="error" variant="tonal" :loading="disabling" :disabled="!currentPassword || loading || disabling " v-if="status === true" @click="disable">
                        {{ tt('Disable Two-Factor Authentication') }}
                    </v-btn>
                    <v-btn color="default" variant="outlined" :loading="regenerating" :disabled="!currentPassword || loading || regenerating" v-if="status === true" @click="regenerateBackupCode()">
                        {{ tt('Regenerate Backup Codes') }}
                    </v-btn>
                    <v-btn color="primary" :loading="enabling" :disabled="loading || enabling" v-if="status === false && !new2FAQRCode" @click="enable">
                        {{ tt('Enable Two-Factor Authentication') }}
                    </v-btn>
                    <v-btn color="primary" :loading="enableConfirming" :disabled="!currentPasscode || loading || enableConfirming" v-if="status === false && new2FAQRCode" @click="enableConfirm">
                        {{ tt('Continue') }}
                    </v-btn>
                </v-card-text>
            </v-card>
        </v-col>

        <v-col cols="12">
            <v-card class="ebk-settings-panel" v-if="currentBackupCode">
                <header class="ebk-panel-header-strip ebk-settings-panel-header">
                    <h2 class="ebk-settings-panel-title">{{ tt('Backup Code') }}</h2>
                    <div class="ebk-panel-header-actions">
                        <v-btn density="compact" color="default" variant="text" size="28"
                               :aria-label="tt('Copy')" :icon="true" @click="copyBackupCodes">
                            <v-icon :icon="mdiContentCopy" size="18" />
                            <v-tooltip activator="parent">{{ tt('Copy') }}</v-tooltip>
                        </v-btn>
                    </div>
                </header>

                <v-card-text class="mt-2">
                    <span class="text-body-large" v-if="status === true">
                        {{ tt('Please copy these backup codes to safe place, the following backup codes will be displayed only once. If these codes were lost, you can regenerate them at any time.') }}
                    </span>
                    <v-textarea no-resize class="backup-code ebk-font-mono mt-4" :readonly="true" :rows="10" :aria-label="tt('Backup Code')" :value="currentBackupCode"/>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>
</template>

<script setup lang="ts">
import { ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useTwoFactorAuthStore } from '@/stores/twoFactorAuth.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { KnownErrorCode } from '@/consts/api.ts';
import { copyTextToClipboard } from '@/lib/ui/common.ts';

import {
    mdiContentCopy,
    mdiShieldCheckOutline,
    mdiShieldOffOutline
} from '@mdi/js';

const { tt } = useI18n();

const twoFactorAuthStore = useTwoFactorAuthStore();
const desktopPageStore = useDesktopPageStore();

const status = ref<boolean | null>(null);
const loading = ref<boolean>(true);
const new2FASecret = ref<string>('');
const new2FAQRCode = ref<string>('');
const currentPassword = ref<string>('');
const currentPasscode = ref<string>('');
const currentBackupCode = ref<string>('');
const enabling = ref<boolean>(false);
const enableConfirming = ref<boolean>(false);
const disabling = ref<boolean>(false);
const regenerating = ref<boolean>(false);

function init(): void {
    loading.value = true;

    twoFactorAuthStore.get2FAStatus().then(response => {
        status.value = response.enable;
        loading.value = false;
    }).catch(error => {
        loading.value = false;

        if (error.error && error.error.errorCode === KnownErrorCode.ApiNotFound) {
            status.value = null;
        } else if (!error.processed) {
            status.value = null;
            desktopPageStore.addErrorToast(error);
        }
    });
}

function enable(): void {
    new2FAQRCode.value = '';
    new2FASecret.value = '';
    currentBackupCode.value = '';

    enabling.value = true;

    twoFactorAuthStore.enable2FA().then(response => {
        enabling.value = false;

        new2FAQRCode.value = response.qrcode;
        new2FASecret.value = response.secret;
    }).catch(error => {
        enabling.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function enableConfirm(): void {
    if (!currentPasscode.value) {
        desktopPageStore.addToast('Passcode cannot be blank');
        return;
    }

    if (enableConfirming.value) {
        return;
    }

    const password = currentPasscode.value;

    currentBackupCode.value = '';
    currentPasscode.value = '';

    enableConfirming.value = true;

    twoFactorAuthStore.confirmEnable2FA({
        secret: new2FASecret.value,
        passcode: password
    }).then(response => {
        enableConfirming.value = false;

        new2FAQRCode.value = '';
        new2FASecret.value = '';

        status.value = true;

        if (response.recoveryCodes && response.recoveryCodes.length) {
            currentBackupCode.value = response.recoveryCodes.join('\n');
        }
    }).catch(error => {
        enableConfirming.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function disable(): void {
    if (!currentPassword.value) {
        desktopPageStore.addToast('Current password cannot be blank');
        return;
    }

    if (disabling.value) {
        return;
    }

    const password = currentPassword.value;

    currentBackupCode.value = '';
    currentPassword.value = '';

    disabling.value = true;

    twoFactorAuthStore.disable2FA({
        password: password
    }).then(() => {
        disabling.value = false;

        status.value = false;
        desktopPageStore.addToast('Two-factor authentication has been disabled');
    }).catch(error => {
        disabling.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function regenerateBackupCode(): void {
    if (!currentPassword.value) {
        desktopPageStore.addToast('Current password cannot be blank');
        return;
    }

    if (regenerating.value) {
        return;
    }

    const password = currentPassword.value;

    currentBackupCode.value = '';
    currentPassword.value = '';

    regenerating.value = true;

    twoFactorAuthStore.regenerate2FARecoveryCode({
        password: password
    }).then(response => {
        regenerating.value = false;

        currentBackupCode.value = response.recoveryCodes.join('\n');
    }).catch(error => {
        regenerating.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function reset(): void {
    new2FASecret.value = '';
    new2FAQRCode.value = '';
    currentPassword.value = '';
    currentPasscode.value = '';
    currentBackupCode.value = '';
    enabling.value = false;
    enableConfirming.value = false;
    disabling.value = false;
    regenerating.value = false;
}

function copyBackupCodes(): void {
    copyTextToClipboard(currentBackupCode.value);
    desktopPageStore.addToast('Backup codes copied');
}

defineExpose({
    reset
});

init();
</script>

<style>
.img-qrcode {
    width: 240px;
    height: 240px
}

.backup-code {
    font-family: monospace;
}
</style>

<template>
    <auth-page-layout illustration="login"
                      :label="tt('Account Access')"
                      :title="show2faInput ? tt('Two-Factor Authentication') : tt('Welcome to ezrbookkeeping')"
                      :version="version"
                      :busy="loggingInByPassword || loggingInByOAuth2 || verifying"
                      :status-text="errorMessage">
        <template #message>
            <p class="ma-0" v-if="!show2faInput && isInternalAuthEnabled()">{{ tt('Please log in with your ezrbookkeeping account') }}</p>
            <p class="ma-0" v-if="show2faInput && twoFAVerifyType === 'passcode'">{{ tt('Enter the passcode from your two-factor authentication app.') }}</p>
            <p class="ma-0" v-if="show2faInput && twoFAVerifyType === 'backupcode'">{{ tt('Enter one of your two-factor authentication backup codes.') }}</p>
            <p class="ma-0" v-if="tips">{{ tips }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent>
            <v-text-field
                type="text"
                name="username"
                autocomplete="username"
                autocapitalize="none"
                autocorrect="off"
                spellcheck="false"
                inputmode="email"
                :autofocus="true"
                :disabled="show2faInput || loggingInByPassword || loggingInByOAuth2 || verifying"
                :label="tt('Username')"
                :placeholder="tt('Your username or email')"
                v-model.trim="username"
                @input="tempToken = ''; errorMessage = null"
                @keyup.enter="passwordInput?.focus()"
                v-if="isInternalAuthEnabled()"
            />

            <v-text-field
                name="password"
                autocomplete="current-password"
                ref="passwordInput"
                :type="showPassword ? 'text' : 'password'"
                :disabled="show2faInput || loggingInByPassword || loggingInByOAuth2 || verifying"
                :label="tt('Password')"
                :placeholder="tt('Your password')"
                v-model="password"
                @input="tempToken = ''; errorMessage = null"
                @keyup.enter="login"
                v-if="isInternalAuthEnabled()"
            >
                <template #append-inner>
                    <v-btn class="auth-password-toggle" density="comfortable" variant="text" size="small" color="default"
                           :icon="true"
                           :aria-label="showPassword ? tt('Hide Password') : tt('Show Password')"
                           :aria-pressed="showPassword ? 'true' : 'false'"
                           :disabled="show2faInput || loggingInByPassword || loggingInByOAuth2 || verifying"
                           @click="showPassword = !showPassword">
                        <v-icon :icon="showPassword ? mdiEyeOffOutline : mdiEyeOutline" size="20" />
                        <v-tooltip activator="parent" location="top">{{ showPassword ? tt('Hide Password') : tt('Show Password') }}</v-tooltip>
                    </v-btn>
                </template>
            </v-text-field>

            <div v-show="show2faInput">
                <v-text-field
                    type="number"
                    name="passcode"
                    autocomplete="one-time-code"
                    inputmode="numeric"
                    ref="passcodeInput"
                    :disabled="loggingInByPassword || loggingInByOAuth2 || verifying"
                    :label="tt('Passcode')"
                    :placeholder="tt('Passcode')"
                    v-model="passcode"
                    @input="errorMessage = null"
                    @keyup.enter="verify"
                    v-if="twoFAVerifyType === 'passcode'"
                />
                <v-text-field
                    type="text"
                    name="backupcode"
                    autocomplete="off"
                    ref="backupCodeInput"
                    :disabled="loggingInByPassword || loggingInByOAuth2 || verifying"
                    :label="tt('Backup Code')"
                    :placeholder="tt('Backup Code')"
                    v-model="backupCode"
                    @input="errorMessage = null"
                    @keyup.enter="verify"
                    v-if="twoFAVerifyType === 'backupcode'"
                />
                <div class="auth-access-row auth-links mt-2">
                    <a href="javascript:void(0);"
                       :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 || verifying }"
                       @click="switch2FAVerifyType">
                        {{ twoFAVerifyType === 'passcode' ? tt('Use Backup Code') : tt('Use Passcode') }}
                    </a>
                </div>
            </div>

            <div class="auth-access-row auth-links" v-if="isInternalAuthEnabled() && !show2faInput">
                <a href="javascript:void(0);"
                   :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 || verifying }"
                   @click="showMobileQrCode = true">{{ tt('Use on Mobile Device') }}</a>
                <router-link to="/forgetpassword"
                             :class="{ 'disabled': !isUserForgetPasswordEnabled() || loggingInByPassword || loggingInByOAuth2 || verifying }">
                    {{ tt('Forget Password?') }}
                </router-link>
            </div>

            <v-btn block size="large" color="primary"
                   :loading="loggingInByPassword"
                   :disabled="inputIsEmpty || loggingInByPassword || loggingInByOAuth2 || verifying"
                   @click="login" v-if="isInternalAuthEnabled() && !show2faInput">
                {{ tt('Log In') }}
            </v-btn>
            <v-btn block size="large" color="primary"
                   :loading="verifying"
                   :disabled="twoFAInputIsEmpty || loggingInByPassword || loggingInByOAuth2 || verifying"
                   @click="verify" v-else-if="isInternalAuthEnabled() && show2faInput">
                {{ tt('Continue') }}
            </v-btn>

            <div class="auth-access-divider" v-if="isInternalAuthEnabled() && isOAuth2Enabled()">{{ tt('or') }}</div>

            <v-btn block size="large" variant="outlined" color="default"
                   :loading="loggingInByOAuth2"
                   :disabled="show2faInput || loggingInByPassword || loggingInByOAuth2 || verifying" :href="oauth2LoginUrl"
                   @click="loggingInByOAuth2 = true" v-if="isOAuth2Enabled()">
                {{ oauth2LoginDisplayName }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary auth-links">
            <div v-if="show2faInput">
                <a href="javascript:void(0);"
                   :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 || verifying }"
                   @click="cancel2FA">{{ tt('Back to login page') }}</a>
            </div>
            <div v-if="!isInternalAuthEnabled()">
                <a href="javascript:void(0);"
                   :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 || verifying }"
                   @click="showMobileQrCode = true">{{ tt('Use on Mobile Device') }}</a>
            </div>
            <div v-if="isInternalAuthEnabled() && !show2faInput">
                <span class="me-1">{{ tt('Don\'t have an account?') }}</span>
                <router-link to="/signup"
                             :class="{ 'disabled': !isUserRegistrationEnabled() || loggingInByPassword || loggingInByOAuth2 || verifying }">
                    {{ tt('Create an account') }}
                </router-link>
            </div>
        </div>

        <template #dialogs>
            <switch-to-mobile-dialog v-model:show="showMobileQrCode" />
        </template>
    </auth-page-layout>
</template>

<script setup lang="ts">
import { VTextField } from 'vuetify/components/VTextField';
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref, useTemplateRef, nextTick } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';
import { useLoginPageBase } from '@/views/base/LoginPageBase.ts';

import { useRootStore } from '@/stores/index.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { KnownErrorCode } from '@/consts/api.ts';

import { generateRandomUUID } from '@/lib/misc.ts';
import {
    isUserRegistrationEnabled,
    isUserForgetPasswordEnabled,
    isUserVerifyEmailEnabled,
    isInternalAuthEnabled,
    isOAuth2Enabled
} from '@/lib/server_settings.ts';

import {
    mdiEyeOutline,
    mdiEyeOffOutline
} from '@mdi/js';

const router = useRouter();

const { tt, te } = useI18n();

const rootStore = useRootStore();
const desktopPageStore = useDesktopPageStore();

const {
    version,
    username,
    password,
    passcode,
    backupCode,
    tempToken,
    twoFAVerifyType,
    oauth2ClientSessionId,
    loggingInByPassword,
    loggingInByOAuth2,
    verifying,
    inputIsEmpty,
    twoFAInputIsEmpty,
    oauth2LoginUrl,
    oauth2LoginDisplayName,
    tips,
    doAfterLogin
} = useLoginPageBase('desktop');

const passwordInput = useTemplateRef<VTextField>('passwordInput');
const passcodeInput = useTemplateRef<VTextField>('passcodeInput');
const backupCodeInput = useTemplateRef<VTextField>('backupCodeInput');

const show2faInput = ref<boolean>(false);
const showMobileQrCode = ref<boolean>(false);
const showPassword = ref<boolean>(false);
const errorMessage = ref<string | null>(null);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function showError(error: any): void {
    if (error && error.message) {
        errorMessage.value = te(error.message);
    } else {
        errorMessage.value = te(error) || tt('An error occurred');
    }
}

function switch2FAVerifyType(): void {
    twoFAVerifyType.value = twoFAVerifyType.value === 'passcode' ? 'backupcode' : 'passcode';
    errorMessage.value = null;

    nextTick(() => {
        if (twoFAVerifyType.value === 'passcode') {
            passcodeInput.value?.focus();
        } else {
            backupCodeInput.value?.focus();
        }
    });
}

function cancel2FA(): void {
    show2faInput.value = false;
    tempToken.value = '';
    passcode.value = '';
    backupCode.value = '';
    twoFAVerifyType.value = 'passcode';
    errorMessage.value = null;

    nextTick(() => {
        passwordInput.value?.focus();
    });
}

function login(): void {
    if (!username.value) {
        errorMessage.value = tt('Username cannot be blank');
        return;
    }

    if (!password.value) {
        errorMessage.value = tt('Password cannot be blank');
        return;
    }

    if (tempToken.value) {
        show2faInput.value = true;
        return;
    }

    if (loggingInByPassword.value) {
        return;
    }

    loggingInByPassword.value = true;
    errorMessage.value = null;
    showPassword.value = false;

    rootStore.authorize({
        loginName: username.value,
        password: password.value
    }).then(authResponse => {
        if (authResponse.need2FA) {
            loggingInByPassword.value = false;
            tempToken.value = authResponse.token;
            show2faInput.value = true;

            nextTick(() => {
                if (passcodeInput.value) {
                    passcodeInput.value.focus();
                    passcodeInput.value.select();
                }
            });

            return;
        }

        doAfterLogin(authResponse).then(loginHint => {
            desktopPageStore.addToast(loginHint);
            router.replace('/');
        }).finally(() => {
            loggingInByOAuth2.value = false;
        });
    }).catch(error => {
        loggingInByPassword.value = false;

        if (isUserVerifyEmailEnabled() && error.error && error.error.errorCode === KnownErrorCode.UserEmailNotVerified && error.error.context && error.error.context.email) {
            router.push(`/verify_email?email=${encodeURIComponent(error.error.context.email)}&emailSent=${error.error.context.hasValidEmailVerifyToken || false}`);
            return;
        }

        if (!error.processed) {
            showError(error);
        }
    });
}

function verify(): void {
    if (twoFAInputIsEmpty.value || verifying.value) {
        return;
    }

    if (twoFAVerifyType.value === 'passcode' && !passcode.value) {
        errorMessage.value = tt('Passcode cannot be blank');
        return;
    } else if (twoFAVerifyType.value === 'backupcode' && !backupCode.value) {
        errorMessage.value = tt('Backup code cannot be blank');
        return;
    }

    verifying.value = true;
    errorMessage.value = null;

    rootStore.authorize2FA({
        token: tempToken.value,
        passcode: twoFAVerifyType.value === 'passcode' ? passcode.value : null,
        recoveryCode: twoFAVerifyType.value === 'backupcode' ? backupCode.value : null
    }).then(authResponse => {
        doAfterLogin(authResponse).then(loginHint => {
            desktopPageStore.addToast(loginHint);
            router.replace('/');
        }).finally(() => {
            verifying.value = false;
        });
    }).catch(error => {
        verifying.value = false;

        if (!error.processed) {
            showError(error);
        }
    });
}

oauth2ClientSessionId.value = generateRandomUUID();
</script>

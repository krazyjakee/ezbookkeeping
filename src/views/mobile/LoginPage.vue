<template>
    <f7-page no-navbar no-swipeback login-screen hide-toolbar-on-scroll class="login-page">
        <div class="ebk-auth-wordmark">
            <img alt="" class="ebk-auth-wordmark-logo" :src="APPLICATION_LOGO_PATH" />
            <span class="ebk-auth-wordmark-title">{{ tt('global.app.title') }}</span>
        </div>

        <section class="ebk-panel ebk-panel--bracketed ebk-auth-panel" aria-labelledby="mobile-login-title"
                 :aria-busy="loggingInByPassword || loggingInByOAuth2 ? 'true' : 'false'">
            <header class="ebk-panel-header-strip ebk-auth-panel-header">
                <span>{{ tt('Account Access') }}</span>
                <span class="ebk-panel-header-actions ebk-auth-panel-live" :class="{ 'ebk-auth-panel-live--busy': loggingInByPassword || loggingInByOAuth2 }" aria-hidden="true"></span>
            </header>

            <div class="ebk-auth-panel-intro">
                <h1 class="ebk-auth-title" id="mobile-login-title">{{ tt('Welcome to ezrbookkeeping') }}</h1>
                <p class="ebk-auth-message" v-if="isInternalAuthEnabled()">{{ tt('Please log in with your ezrbookkeeping account') }}</p>
                <p class="ebk-auth-message" v-if="tips">{{ tips }}</p>
                <div class="ebk-auth-status ebk-auth-status--error" role="alert" v-if="errorMessage">
                    <f7-icon f7="exclamationmark_circle" aria-hidden="true"></f7-icon>
                    <span>{{ errorMessage }}</span>
                </div>
            </div>

            <f7-list form class="login-page-form no-margin-top margin-bottom-half">
                <f7-list-input
                    type="text"
                    name="username"
                    autocomplete="username"
                    autocapitalize="none"
                    autocorrect="off"
                    spellcheck="false"
                    inputmode="email"
                    clear-button
                    :disabled="loggingInByPassword || loggingInByOAuth2"
                    :label="tt('Username')"
                    :placeholder="tt('Your username or email')"
                    v-model:value.trim="username"
                    @input="tempToken = ''; errorMessage = null"
                    v-if="isInternalAuthEnabled()"
                ></f7-list-input>

                <f7-list-input
                    name="password"
                    autocomplete="current-password"
                    :type="showPassword ? 'text' : 'password'"
                    :disabled="loggingInByPassword || loggingInByOAuth2"
                    :label="tt('Password')"
                    :placeholder="tt('Your password')"
                    v-model:value="password"
                    @input="tempToken = ''; errorMessage = null"
                    @keyup.enter="loginByPressEnter"
                    v-if="isInternalAuthEnabled()"
                >
                    <template #content-end>
                        <f7-link class="ebk-auth-password-toggle" role="button"
                                 :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 }"
                                 :aria-label="showPassword ? tt('Hide Password') : tt('Show Password')"
                                 :aria-pressed="showPassword ? 'true' : 'false'"
                                 :icon-f7="showPassword ? 'eye_slash' : 'eye'"
                                 @click="showPassword = !showPassword"></f7-link>
                    </template>
                </f7-list-input>

                <f7-list-item class="login-page-utilities" v-if="isInternalAuthEnabled()">
                    <template #title>
                        <small>
                            <f7-link :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 }" @click="switchToDesktopVersion">{{ tt('Switch to Desktop Version') }}</f7-link>
                        </small>
                    </template>
                    <template #after>
                        <small>
                            <f7-link :class="{ 'disabled': !isUserForgetPasswordEnabled() || loggingInByPassword || loggingInByOAuth2 }" @click="forgetPasswordEmail = ''; showForgetPasswordSheet = true">{{ tt('Forget Password?') }}</f7-link>
                        </small>
                    </template>
                </f7-list-item>

                <f7-list-button class="login-page-primary-action" :class="{ 'disabled': inputIsEmpty || loggingInByPassword || loggingInByOAuth2 }" :text="tt('Log In')"
                                @click="login" v-if="isInternalAuthEnabled()"></f7-list-button>

                <f7-list-item class="login-divider margin-vertical-half display-flex align-items-center" v-if="isInternalAuthEnabled() && isOAuth2Enabled()">
                    <hr class="margin-inline-end-half" />
                    <small>{{ tt('or') }}</small>
                    <hr class="margin-inline-start-half" />
                </f7-list-item>

                <f7-list-button external class="login-page-secondary-action"
                                :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2, 'margin-top': !isInternalAuthEnabled() }"
                                :href="oauth2LoginUrl" :text="oauth2LoginDisplayName"
                                @click="loginByOAuth2" v-if="isOAuth2Enabled()"></f7-list-button>

                <f7-list-item class="login-page-utilities login-page-utilities-centered" v-if="!isInternalAuthEnabled()">
                    <template #title>
                        <small>
                            <f7-link :class="{ 'disabled': loggingInByPassword || loggingInByOAuth2 }" @click="switchToDesktopVersion">{{ tt('Switch to Desktop Version') }}</f7-link>
                        </small>
                    </template>
                </f7-list-item>
            </f7-list>

            <div class="ebk-auth-panel-secondary" v-if="isInternalAuthEnabled()">
                <span style="margin-inline-end: 0.3rem;">{{ tt('Don\'t have an account?') }}</span>
                <f7-link :class="{ 'disabled': !isUserRegistrationEnabled() || loggingInByPassword || loggingInByOAuth2 }" href="/signup" :text="tt('Create an account')"></f7-link>
            </div>
        </section>

        <f7-list class="login-page-bottom">
            <f7-block-footer>
                <language-select-button :disabled="loggingInByPassword || loggingInByOAuth2" @error="showToast" />

                <div class="login-page-powered-by margin-top-half">
                    <span>Powered by</span>
                    <f7-link @click="openExternalUrl('https://github.com/krazyjakee/ezbookkeeping')" target="_blank">ezrbookkeeping</f7-link>
                    <span>{{ version }}</span>
                </div>
            </f7-block-footer>
        </f7-list>

        <f7-toolbar class="login-page-fixed-bottom" tabbar bottom :outline="false">
            <language-select-button :disabled="loggingInByPassword || loggingInByOAuth2" @error="showToast" />

            <div class="login-page-powered-by margin-top-half">
                <span>Powered by</span>
                <f7-link @click="openExternalUrl('https://github.com/krazyjakee/ezbookkeeping')" target="_blank">ezrbookkeeping</f7-link>
                <span>{{ version }}</span>
            </div>
        </f7-toolbar>

        <f7-sheet
            style="height:auto"
            :opened="show2faSheet" @sheet:closed="show2faSheet = false"
        >
            <f7-page-content>
                <div class="display-flex padding-horizontal padding-top justify-content-space-between align-items-center">
                    <div class="ebk-sheet-title"><b>{{ tt('Two-Factor Authentication') }}</b></div>
                </div>
                <div class="padding-horizontal padding-bottom">
                    <p class="ebk-auth-sheet-message">
                        {{ twoFAVerifyType === 'passcode' ? tt('Enter the passcode from your two-factor authentication app.') : tt('Enter one of your two-factor authentication backup codes.') }}
                    </p>
                    <f7-list class="no-margin">
                        <f7-list-input
                            type="number"
                            name="passcode"
                            inputmode="numeric"
                            autocomplete="one-time-code"
                            outline
                            clear-button
                            class="no-margin no-padding-bottom"
                            v-if="twoFAVerifyType === 'passcode'"
                            :aria-label="tt('Passcode')"
                            :placeholder="tt('Passcode')"
                            v-model:value="passcode"
                            @keyup.enter="verify"
                        ></f7-list-input>
                        <f7-list-input
                            autocomplete="off"
                            outline
                            clear-button
                            class="no-margin no-padding-bottom"
                            v-if="twoFAVerifyType === 'backupcode'"
                            :aria-label="tt('Backup Code')"
                            :placeholder="tt('Backup Code')"
                            v-model:value="backupCode"
                            @keyup.enter="verify"
                        ></f7-list-input>
                    </f7-list>
                    <f7-button large fill :class="{ 'disabled': twoFAInputIsEmpty || verifying }" :text="tt('Verify')" @click="verify"></f7-button>
                    <div class="margin-top text-align-center">
                        <f7-link @click="switch2FAVerifyType" :text="tt(twoFAVerifyTypeSwitchName)"></f7-link>
                    </div>
                </div>
            </f7-page-content>
        </f7-sheet>

        <f7-sheet swipe-to-close swipe-handler=".swipe-handler"
            style="height:auto"
            :opened="showForgetPasswordSheet" @sheet:closed="showForgetPasswordSheet = false"
        >
            <div class="swipe-handler"></div>
            <f7-page-content>
                <div class="display-flex padding justify-content-space-between align-items-center">
                    <div class="ebk-sheet-title"><b>{{ tt('Forget Password?') }}</b></div>
                </div>
                <div class="padding-horizontal padding-bottom">
                    <p class="no-margin">
                        <span>{{ tt('Please enter your email address used for registration and we\'ll send you an email with a reset password link') }}</span>
                    </p>
                    <f7-list class="no-margin">
                        <f7-list-input
                            type="email"
                            autocomplete="email"
                            outline
                            clear-button
                            class="no-margin no-padding-bottom"
                            :placeholder="tt('Your email address')"
                            v-model:value="forgetPasswordEmail"
                            @keyup.enter="requestResetPassword"
                        ></f7-list-input>
                    </f7-list>
                    <f7-button large fill :class="{ 'disabled': !forgetPasswordEmail || requestingForgetPassword }" :text="tt('Send Reset Link')" @click="requestResetPassword"></f7-button>
                    <div class="margin-top text-align-center">
                        <f7-link :class="{ 'disabled': requestingForgetPassword }" @click="showForgetPasswordSheet = false" :text="tt('Cancel')"></f7-link>
                    </div>
                </div>
            </f7-page-content>
        </f7-sheet>

        <password-input-sheet :title="tt('Verify your email')"
                              :hint="tt(hasValidEmailVerifyToken ? 'format.misc.accountActivationAndResendValidationEmailTip' : 'format.misc.resendValidationEmailTip', { email: resendVerifyEmail })"
                              :confirm-disabled="requestingResendVerifyEmail"
                              :cancel-disabled="requestingResendVerifyEmail"
                              v-model:show="showVerifyEmailSheet"
                              v-model="currentPasswordForResendVerifyEmail"
                              @password:confirm="requestResendVerifyEmail">
        </password-input-sheet>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useLoginPageBase } from '@/views/base/LoginPageBase.ts';

import { useRootStore } from '@/stores/index.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';
import { KnownErrorCode } from '@/consts/api.ts';

import { generateRandomUUID } from '@/lib/misc.ts';
import {
    isUserRegistrationEnabled,
    isUserForgetPasswordEnabled,
    isUserVerifyEmailEnabled,
    isInternalAuthEnabled,
    isOAuth2Enabled
} from '@/lib/server_settings.ts';
import { getDesktopVersionPath } from '@/lib/version.ts';
import { useI18nUIComponents, showLoading, hideLoading, isModalShowing } from '@/lib/ui/mobile.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt, te } = useI18n();
const { showAlert, showConfirm, showToast, openExternalUrl } = useI18nUIComponents();

const rootStore = useRootStore();

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
} = useLoginPageBase('mobile');

const forgetPasswordEmail = ref<string>('');
const resendVerifyEmail = ref<string>('');
const hasValidEmailVerifyToken = ref<boolean>(false);
const currentPasswordForResendVerifyEmail = ref<string>('');
const requestingForgetPassword = ref<boolean>(false);
const requestingResendVerifyEmail = ref<boolean>(false);
const show2faSheet = ref<boolean>(false);
const showPassword = ref<boolean>(false);
const errorMessage = ref<string | null>(null);
const showForgetPasswordSheet = ref<boolean>(false);
const showVerifyEmailSheet = ref<boolean>(false);

const twoFAVerifyTypeSwitchName = computed<string>(() => {
    if (twoFAVerifyType.value === 'backupcode') {
        return 'Use Passcode';
    } else {
        return 'Use Backup Code';
    }
});

function switchToDesktopVersion(): void {
    showConfirm('Are you sure you want to switch to desktop version?', () => {
        window.location.replace(getDesktopVersionPath());
    });
}

function login(): void {
    const router = props.f7router;

    if (!username.value) {
        showAlert('Username cannot be blank');
        return;
    }

    if (!password.value) {
        showAlert('Password cannot be blank');
        return;
    }

    if (tempToken.value) {
        show2faSheet.value = true;
        return;
    }

    loggingInByPassword.value = true;
    errorMessage.value = null;
    showPassword.value = false;
    resendVerifyEmail.value = '';
    hasValidEmailVerifyToken.value = false;
    currentPasswordForResendVerifyEmail.value = '';
    showLoading(() => loggingInByPassword.value);

    rootStore.authorize({
        loginName: username.value,
        password: password.value
    }).then(authResponse => {
        if (authResponse.need2FA) {
            loggingInByPassword.value = false;
            hideLoading();

            tempToken.value = authResponse.token;
            show2faSheet.value = true;
            return;
        }

        doAfterLogin(authResponse).then(loginHint => {
            if (loginHint) {
                showToast(loginHint);
            }

            router.refreshPage();
        }).finally(() => {
            loggingInByPassword.value = false;
            hideLoading();
        });
    }).catch(error => {
        loggingInByPassword.value = false;
        hideLoading();

        if (isUserVerifyEmailEnabled() && error.error && error.error.errorCode === KnownErrorCode.UserEmailNotVerified && error.error.context && error.error.context.email) {
            resendVerifyEmail.value = error.error.context.email;
            hasValidEmailVerifyToken.value = error.error.context.hasValidEmailVerifyToken || false;
            currentPasswordForResendVerifyEmail.value = '';
            showVerifyEmailSheet.value = true;
            return;
        }

        if (!error.processed) {
            errorMessage.value = error.message ? te(error.message) : (te(error) || tt('An error occurred'));
        }
    });
}

function loginByPressEnter(): void {
    if (isModalShowing()) {
        return;
    }

    return login();
}

function loginByOAuth2(): void {
    loggingInByOAuth2.value = true;
    showLoading();
}

function verify(): void {
    const router = props.f7router;

    if (twoFAInputIsEmpty.value || verifying.value) {
        return;
    }

    if (twoFAVerifyType.value === 'passcode' && !passcode.value) {
        showAlert('Passcode cannot be blank');
        return;
    } else if (twoFAVerifyType.value === 'backupcode' && !backupCode.value) {
        showAlert('Backup code cannot be blank');
        return;
    }

    verifying.value = true;
    showLoading(() => verifying.value);

    rootStore.authorize2FA({
        token: tempToken.value,
        passcode: twoFAVerifyType.value === 'passcode' ? passcode.value : null,
        recoveryCode: twoFAVerifyType.value === 'backupcode' ? backupCode.value : null
    }).then(authResponse => {
        doAfterLogin(authResponse).then(loginHint => {
            if (loginHint) {
                showToast(loginHint);
            }

            router.refreshPage();
        }).finally(() => {
            verifying.value = false;
            hideLoading();
            show2faSheet.value = false;
        });
    }).catch(error => {
        verifying.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function requestResetPassword(): void {
    if (!forgetPasswordEmail.value) {
        showAlert('Email address cannot be blank');
        return;
    }

    requestingForgetPassword.value = true;
    showLoading(() => requestingForgetPassword.value);

    rootStore.requestResetPassword({
        email: forgetPasswordEmail.value
    }).then(() => {
        requestingForgetPassword.value = false;
        hideLoading();

        showToast('Password reset email has been sent');
        showForgetPasswordSheet.value = false;
    }).catch(error => {
        requestingForgetPassword.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function requestResendVerifyEmail(): void {
    if (!currentPasswordForResendVerifyEmail.value) {
        showToast('Current password cannot be blank');
        return;
    }

    requestingResendVerifyEmail.value = true;
    showLoading(() => requestingResendVerifyEmail.value);

    rootStore.resendVerifyEmailByUnloginUser({
        email: resendVerifyEmail.value,
        password: currentPasswordForResendVerifyEmail.value
    }).then(() => {
        requestingResendVerifyEmail.value = false;
        hideLoading();

        showToast('Validation email has been sent');
        showVerifyEmailSheet.value = false;
    }).catch(error => {
        requestingResendVerifyEmail.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function switch2FAVerifyType(): void {
    if (twoFAVerifyType.value === 'passcode') {
        twoFAVerifyType.value = 'backupcode';
    } else {
        twoFAVerifyType.value = 'passcode';
    }
}

oauth2ClientSessionId.value = generateRandomUUID();
</script>

<style>
.login-divider > .item-content {
    width: 100%;
    min-height: 0;
    white-space: nowrap;

    > .item-inner {
        padding-top: 0;
        padding-bottom: 0;
        min-height: 0;

        > small {
            opacity: 0.7;
        }

        > hr {display: block;
            flex: 1 1 100%;
            height: 0;
            max-height: 0;
            border-style: solid;
            border-width: thin 0 0 0;
            opacity: 0.12;
            transition: inherit;
        }
    }
}
</style>

<template>
    <auth-page-layout illustration="connect"
                      :label="tt('Third-Party Login')"
                      :title="oauth2LoginDisplayName"
                      :version="version"
                      :busy="loggingInByOAuth2"
                      :status-type="currentStatus.type"
                      :status-text="currentStatus.text">
        <template #message>
            <p class="ma-0" v-if="!error && !errorMessage && platform && token && !userName">{{ tt('Logging in...') }}</p>
            <p class="ma-0" v-else-if="!error && !errorMessage && userName">{{ tt('format.misc.oauth2bindTip', { providerName: oauth2ProviderDisplayName, userName: userName }) }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent v-if="!error && userName">
            <input type="text" name="username" autocomplete="username" class="d-none" aria-hidden="true" tabindex="-1" :value="userName" readonly />
            <v-text-field
                type="password"
                name="password"
                autocomplete="current-password"
                :autofocus="true"
                :disabled="show2faInput || loggingInByOAuth2"
                :label="tt('Password')"
                :placeholder="tt('Your password')"
                v-model="password"
                @input="loginErrorMessage = null"
                @keyup.enter="verifyAndLogin"
            />

            <v-text-field
                type="number"
                name="passcode"
                autocomplete="one-time-code"
                inputmode="numeric"
                ref="passcodeInput"
                :disabled="loggingInByOAuth2"
                :label="tt('Passcode')"
                :placeholder="tt('Passcode')"
                v-model="passcode"
                @input="loginErrorMessage = null"
                @keyup.enter="verifyAndLogin"
                v-show="show2faInput"
            />

            <v-btn block size="large" color="primary" :loading="loggingInByOAuth2" :disabled="!password || loggingInByOAuth2" @click="verifyAndLogin">
                {{ tt('Continue') }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary">
            <router-link class="auth-back-link" to="/login" :class="{ 'disabled': loggingInByOAuth2 }">
                <v-icon class="icon-with-direction" size="20" :icon="mdiChevronLeft"/>
                <span>{{ tt('Back to login page') }}</span>
            </router-link>
        </div>
    </auth-page-layout>
</template>

<script setup lang="ts">
import { VTextField } from 'vuetify/components/VTextField';
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref, computed, useTemplateRef, nextTick } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';
import { useLoginPageBase } from '@/views/base/LoginPageBase.ts';

import { useRootStore } from '@/stores/index.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { type ErrorResponse, buildErrorResponse } from '@/core/api.ts';
import { KnownErrorCode } from '@/consts/api.ts';

import { navigateToHomePage } from '@/lib/web.ts';
import {
    isUserVerifyEmailEnabled,
    getOIDCCustomDisplayNames
} from '@/lib/server_settings.ts';

import {
    mdiChevronLeft
} from '@mdi/js';

const props = defineProps<{
    token?: string;
    provider?: string;
    platform?: string;
    userName?: string;
    errorCode?: string;
    errorMessage?: string;
}>();

const router = useRouter();

const {
    tt,
    te,
    getLocalizedOAuth2ProviderName,
    getLocalizedOAuth2LoginText
} = useI18n();

const rootStore = useRootStore();
const desktopPageStore = useDesktopPageStore();

const {
    version,
    password,
    loggingInByOAuth2,
    doAfterLogin
} = useLoginPageBase('desktop');

const passcodeInput = useTemplateRef<VTextField>('passcodeInput');

const passcode = ref<string>('');
const show2faInput = ref<boolean>(false);
const loginErrorMessage = ref<string | null>(null);

const oauth2ProviderDisplayName = computed<string>(() => getLocalizedOAuth2ProviderName(props.provider ?? '', getOIDCCustomDisplayNames()));
const oauth2LoginDisplayName = computed<string>(() => getLocalizedOAuth2LoginText(props.provider ?? '', getOIDCCustomDisplayNames()));

const error = computed<ErrorResponse | undefined>(() => {
    if (props.errorCode && props.errorMessage) {
        return buildErrorResponse(parseInt(props.errorCode), props.errorMessage);
    } else {
        return undefined;
    }
});

const currentStatus = computed<{ type: 'error' | 'info', text: string | null }>(() => {
    if (error.value) {
        return { type: 'error', text: te({ error: error.value }) };
    } else if (props.errorMessage) {
        return { type: 'error', text: props.errorMessage };
    } else if (loginErrorMessage.value) {
        return { type: 'error', text: loginErrorMessage.value };
    } else if (!(props.platform && props.token) && !props.userName) {
        return { type: 'error', text: tt('An error occurred') };
    } else {
        return { type: 'info', text: null };
    }
});

const inputProblemMessage = computed<string | null>(() => {
    if (!password.value) {
        return 'Password cannot be blank';
    } else {
        return null;
    }
});

function navigateToHome(): void {
    if (props.platform === 'desktop') {
        navigateToHomePage('desktop');
    } else if (props.platform === 'mobile') {
        navigateToHomePage('mobile');
    } else {
        router.replace('/');
    }
}

function verifyAndLogin(): void  {
    const problemMessage = inputProblemMessage.value;

    if (problemMessage) {
        loginErrorMessage.value = tt(problemMessage);
        return;
    }

    loggingInByOAuth2.value = true;
    loginErrorMessage.value = null;

    rootStore.authorizeOAuth2({
        password: password.value,
        passcode: passcode.value,
        callbackToken: props.token || ''
    }).then(authResponse => {
        doAfterLogin(authResponse).then(loginHint => {
            desktopPageStore.addToast(loginHint);
            navigateToHome();
        }).finally(() => {
            loggingInByOAuth2.value = false;
        });
    }).catch(error => {
        loggingInByOAuth2.value = false;

        if (isUserVerifyEmailEnabled() && error.error && error.error.errorCode === KnownErrorCode.UserEmailNotVerified && error.error.context && error.error.context.email) {
            router.push(`/verify_email?email=${encodeURIComponent(error.error.context.email)}&emailSent=${error.error.context.hasValidEmailVerifyToken || false}`);
            return;
        } else if (error.error && error.error.errorCode === KnownErrorCode.TwoFactorAuthorizationPasscodeEmpty) {
            show2faInput.value = true;

            nextTick(() => {
                if (passcodeInput.value) {
                    passcodeInput.value.focus();
                    passcodeInput.value.select();
                }
            });

            return;
        }

        if (!error.processed) {
            loginErrorMessage.value = error.message ? te(error.message) : (te(error) || tt('An error occurred'));
        }
    });
}

if (!error.value && props.platform && props.token && !props.userName) {
    loggingInByOAuth2.value = true;

    rootStore.authorizeOAuth2({
        callbackToken: props.token
    }).then(authResponse => {
        doAfterLogin(authResponse).then(loginHint => {
            desktopPageStore.addToast(loginHint);
            navigateToHome();
        }).finally(() => {
            loggingInByOAuth2.value = false;
        });
    }).catch(error => {
        loggingInByOAuth2.value = false;

        if (isUserVerifyEmailEnabled() && error.error && error.error.errorCode === KnownErrorCode.UserEmailNotVerified && error.error.context && error.error.context.email) {
            router.push(`/verify_email?email=${encodeURIComponent(error.error.context.email)}&emailSent=${error.error.context.hasValidEmailVerifyToken || false}`);
            return;
        }

        if (!error.processed) {
            loginErrorMessage.value = error.message ? te(error.message) : (te(error) || tt('An error occurred'));
        }
    });
}
</script>

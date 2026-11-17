<template>
    <auth-page-layout illustration="verify"
                      :verified="verified"
                      :label="tt('Account Access')"
                      :title="tt('Verify your email')"
                      :version="version"
                      :busy="loading || resending"
                      :status-type="currentStatus.type"
                      :status-text="currentStatus.text">
        <template #message>
            <p class="ma-0" v-if="token && loading">{{ tt('Verifying...') }}</p>
            <p class="ma-0" v-if="!token && email">{{ tt(hasValidEmailVerifyToken ? 'format.misc.accountActivationAndResendValidationEmailTip' : 'format.misc.resendValidationEmailTip', { email: email }) }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent v-if="!loading && !token && email && isUserVerifyEmailEnabled()">
            <input type="email" name="email" autocomplete="username" class="d-none" aria-hidden="true" tabindex="-1" :value="email" readonly />
            <v-text-field
                name="password"
                autocomplete="current-password"
                type="password"
                :disabled="loading || resending"
                :label="tt('Password')"
                :placeholder="tt('Your password')"
                v-model="password"
                @input="resendStatus = null"
                @keyup.enter="resendEmail"
            />

            <v-btn block size="large" color="primary" :loading="resending" :disabled="loading || resending || !password" @click="resendEmail">
                {{ tt('Resend Validation Email') }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary">
            <router-link class="auth-back-link"
                         :class="{ 'disabled': loading || resending }"
                         :to="verified && isUserLogined() ? '/' : '/login'">
                <v-icon class="icon-with-direction" size="20" :icon="mdiChevronLeft"/>
                <span v-if="!verified || !isUserLogined()">{{ tt('Back to login page') }}</span>
                <span v-else-if="verified">{{ tt('Back to home page') }}</span>
            </router-link>
        </div>

        <template #dialogs>
            <snack-bar ref="snackbar" @update:show="onSnackbarShowStateChanged" />
        </template>
    </auth-page-layout>
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref, computed, useTemplateRef } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';

import { isUserVerifyEmailEnabled } from '@/lib/server_settings.ts';
import { isUserLogined } from '@/lib/userstate.ts';
import { getClientDisplayVersion } from '@/lib/version.ts';

import {
    mdiChevronLeft
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const props = defineProps<{
    email: string;
    token?: string;
    hasValidEmailVerifyToken: boolean;
}>();

const router = useRouter();

const { tt, te } = useI18n();

const rootStore = useRootStore();

const version = `${getClientDisplayVersion()}`;

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const password = ref<string>('');
const loading = ref<boolean>(true);
const resending = ref<boolean>(false);
const verified = ref<boolean>(false);
const errorMessage = ref<string>('');
const resendStatus = ref<{ type: 'error' | 'success', text: string } | null>(null);

const currentStatus = computed<{ type: 'error' | 'success', text: string | null }>(() => {
    if (props.token && verified.value) {
        return { type: 'success', text: tt('Email address is verified') };
    } else if (props.token && !verified.value && errorMessage.value) {
        return { type: 'error', text: errorMessage.value };
    } else if (!props.token && !props.email) {
        return { type: 'error', text: tt('Parameter Invalid') };
    } else if (resendStatus.value) {
        return resendStatus.value;
    } else {
        return { type: 'error', text: null };
    }
});

function init(): void {
    verified.value = false;
    loading.value = true;

    if (!props.token) {
        loading.value = false;
        return;
    }

    rootStore.verifyEmail(props.token).then(() => {
        loading.value = false;
        verified.value = true;
        snackbar.value?.showMessage('Email address is verified');
    }).catch(error => {
        loading.value = false;
        verified.value = false;

        if (!error.processed) {
            errorMessage.value = te(error.message || error);
        }
    });
}

function resendEmail(): void {
    resending.value = true;
    resendStatus.value = null;

    rootStore.resendVerifyEmailByUnloginUser({
        email: props.email,
        password: password.value
    }).then(() => {
        resending.value = false;
        resendStatus.value = { type: 'success', text: tt('Validation email has been sent') };
    }).catch(error => {
        resending.value = false;

        if (!error.processed) {
            resendStatus.value = { type: 'error', text: error.message ? te(error.message) : (te(error) || tt('An error occurred')) };
        }
    });
}

function onSnackbarShowStateChanged(newValue: boolean): void {
    if (!newValue && verified.value && isUserLogined()) {
        router.replace('/');
    }
}

init();
</script>

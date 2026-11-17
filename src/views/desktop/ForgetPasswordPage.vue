<template>
    <auth-page-layout illustration="recovery"
                      :label="tt('Account Access')"
                      :title="tt('Forget Password?')"
                      :version="version"
                      :busy="requesting"
                      :status-type="statusType"
                      :status-text="statusMessage">
        <template #message>
            <p class="ma-0">{{ tt('Please enter your email address used for registration and we\'ll send you an email with a reset password link') }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent>
            <v-text-field
                type="email"
                name="email"
                autocomplete="email"
                :autofocus="true"
                :disabled="requesting"
                :label="tt('E-mail')"
                :placeholder="tt('Your email address')"
                v-model="email"
                @input="statusMessage = null"
                @keyup.enter="requestResetPassword"
            />

            <v-btn block size="large" color="primary" :loading="requesting" :disabled="!email || requesting" @click="requestResetPassword">
                {{ tt('Send Reset Link') }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary">
            <router-link class="auth-back-link" to="/login" :class="{ 'disabled': requesting }">
                <v-icon class="icon-with-direction" size="20" :icon="mdiChevronLeft"/>
                <span>{{ tt('Back to login page') }}</span>
            </router-link>
        </div>
    </auth-page-layout>
</template>

<script setup lang="ts">
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';

import { getClientDisplayVersion } from '@/lib/version.ts';

import {
    mdiChevronLeft
} from '@mdi/js';

const { tt, te } = useI18n();

const rootStore = useRootStore();

const version = `${getClientDisplayVersion()}`;

const email = ref<string>('');
const requesting = ref<boolean>(false);
const statusType = ref<'error' | 'success'>('error');
const statusMessage = ref<string | null>(null);

function requestResetPassword(): void {
    if (!email.value) {
        statusType.value = 'error';
        statusMessage.value = tt('Email address cannot be blank');
        return;
    }

    requesting.value = true;
    statusMessage.value = null;

    rootStore.requestResetPassword({
        email: email.value
    }).then(() => {
        requesting.value = false;
        statusType.value = 'success';
        statusMessage.value = tt('Password reset email has been sent');
    }).catch(error => {
        requesting.value = false;

        if (!error.processed) {
            statusType.value = 'error';
            statusMessage.value = error.message ? te(error.message) : (te(error) || tt('An error occurred'));
        }
    });
}
</script>

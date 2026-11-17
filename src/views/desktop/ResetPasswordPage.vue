<template>
    <auth-page-layout illustration="recovery"
                      :label="tt('Account Access')"
                      :title="tt('Reset Password')"
                      :version="version"
                      :busy="updating"
                      :status-type="passwordChanged ? 'success' : 'error'"
                      :status-text="statusMessage">
        <template #message>
            <p class="ma-0">{{ tt('Please re-enter your email address, and then enter a new password.') }}</p>
        </template>

        <v-form class="auth-access-form" @submit.prevent>
            <v-text-field
                type="email"
                name="email"
                autocomplete="email"
                :autofocus="true"
                :disabled="updating"
                :label="tt('E-mail')"
                :placeholder="tt('Your email address')"
                v-model="email"
                @input="statusMessage = null"
                @keyup.enter="passwordInput?.focus()"
            />

            <v-text-field
                name="new-password"
                autocomplete="new-password"
                ref="passwordInput"
                type="password"
                :disabled="updating"
                :label="tt('Password')"
                :placeholder="tt('Your password')"
                v-model="newPassword"
                @input="statusMessage = null"
                @keyup.enter="confirmPasswordInput?.focus()"
            />

            <v-text-field
                name="confirm-password"
                autocomplete="new-password"
                ref="confirmPasswordInput"
                type="password"
                :disabled="updating"
                :label="tt('Confirm Password')"
                :placeholder="tt('Re-enter the password')"
                v-model="confirmPassword"
                @input="statusMessage = null"
                @keyup.enter="resetPassword"
            />

            <v-btn block size="large" color="primary" :loading="updating" :disabled="!email || !newPassword || !confirmPassword || updating" @click="resetPassword">
                {{ tt('Update Password') }}
            </v-btn>
        </v-form>

        <div class="auth-access-secondary">
            <router-link class="auth-back-link" to="/login" :class="{ 'disabled': updating }">
                <v-icon class="icon-with-direction" size="20" :icon="mdiChevronLeft"/>
                <span>{{ tt('Back to login page') }}</span>
            </router-link>
        </div>

        <template #dialogs>
            <snack-bar ref="snackbar" @update:show="onSnackbarShowStateChanged" />
        </template>
    </auth-page-layout>
</template>

<script setup lang="ts">
import { VTextField } from 'vuetify/components/VTextField';
import SnackBar from '@/components/desktop/SnackBar.vue';
import AuthPageLayout from '@/components/desktop/AuthPageLayout.vue';

import { ref, computed, useTemplateRef } from 'vue';
import { useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';

import { getClientDisplayVersion } from '@/lib/version.ts';

import {
    mdiChevronLeft
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const props = defineProps<{
    token: string;
}>();

const router = useRouter();

const { tt, te } = useI18n();

const rootStore = useRootStore();

const version = `${getClientDisplayVersion()}`;

const passwordInput = useTemplateRef<VTextField>('passwordInput');
const confirmPasswordInput = useTemplateRef<VTextField>('confirmPasswordInput');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const email = ref<string>('');
const newPassword = ref<string>('');
const confirmPassword = ref<string>('');
const updating = ref<boolean>(false);
const passwordChanged = ref<boolean>(false);
const statusMessage = ref<string | null>(null);

const inputProblemMessage = computed<string | null>(() => {
    if (!email.value) {
        return 'Email address cannot be blank';
    } else if (!newPassword.value && !confirmPassword.value) {
        return 'Nothing has been modified';
    } else if (!newPassword.value && confirmPassword.value) {
        return 'New password cannot be blank';
    } else if (newPassword.value && !confirmPassword.value) {
        return 'Password confirmation cannot be blank';
    } else if (newPassword.value && confirmPassword.value && newPassword.value !== confirmPassword.value) {
        return 'Password and password confirmation do not match';
    } else {
        return null;
    }
});

function onSnackbarShowStateChanged(newValue: boolean): void {
    if (!newValue && passwordChanged.value) {
        router.replace('/login');
    }
}

function resetPassword(): void  {
    passwordChanged.value = false;

    const problemMessage = inputProblemMessage.value;

    if (problemMessage) {
        statusMessage.value = tt(problemMessage);
        return;
    }

    updating.value = true;
    statusMessage.value = null;

    rootStore.resetPassword({
        token: props.token,
        email: email.value,
        password: newPassword.value
    }).then(() => {
        updating.value = false;
        passwordChanged.value = true;
        statusMessage.value = tt('Password has been updated');
        snackbar.value?.showMessage('Password has been updated');
    }).catch(error => {
        updating.value = false;
        passwordChanged.value = false;

        if (!error.processed) {
            statusMessage.value = error.message ? te(error.message) : (te(error) || tt('An error occurred'));
        }
    });
}
</script>

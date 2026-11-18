import { defineStore } from 'pinia';

import type {
    UserExternalAuthUnlinkRequest,
    UserExternalAuthInfoResponse,
    OAuth2LinkRequest
} from '@/models/user_external_auth.ts';

import logger from '@/lib/logger.ts';
import services, { toStoreError } from '@/lib/services.ts';

export const useUserExternalAuthStore = defineStore('userExternalAUth', () => {
    function getExternalAuths(): Promise<UserExternalAuthInfoResponse[]> {
        return new Promise((resolve, reject) => {
            services.getExternalAuths().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve third-party login list' });
                    return;
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to load third-party login list', error);

                reject(toStoreError(error, 'Unable to retrieve third-party login list'));
            });
        });
    }

    function unlinkExternalAuth(req: UserExternalAuthUnlinkRequest): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.unlinkExternalAuth(req).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to unlink third-party login' });
                    return;
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to revoke token', error);

                reject(toStoreError(error, 'Unable to unlink third-party login'));
            });
        });
    }

    function linkExternalOAuth2(req: OAuth2LinkRequest): Promise<string> {
        return new Promise((resolve, reject) => {
            services.linkExternalOAuth2(req).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'An error occurred' });
                    return;
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to get external auth link url', error);

                reject(toStoreError(error, 'An error occurred'));
            });
        });
    }

    return {
        // functions
        getExternalAuths,
        unlinkExternalAuth,
        linkExternalOAuth2
    };
});

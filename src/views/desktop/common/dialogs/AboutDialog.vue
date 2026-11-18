<template>
    <v-dialog width="800" v-model="showState">
        <one-column-dialog-layout content-class="pt-4"
                                  :title="tt('About')" :cancel-button-title="tt('Close')"
                                  @cancel="showState = false">
            <template #content>
                <div class="about-panels">
                    <section class="ebk-panel about-identity" aria-labelledby="about-identity-title">
                        <img alt="" class="about-identity-logo" :src="APPLICATION_LOGO_PATH" />
                        <div class="about-identity-text">
                            <h3 class="about-identity-title" id="about-identity-title">{{ tt('global.app.title') }}</h3>
                            <div class="about-identity-attribution">
                                <span>Based on ezBookkeeping, originally created by </span>
                                <a target="_blank" rel="noopener" href="https://github.com/mayswind">MaysWind</a>
                                <span> · Released under the MIT License</span>
                            </div>
                        </div>
                        <v-btn class="about-identity-refresh" density="comfortable" color="default" variant="outlined"
                               :prepend-icon="mdiWebRefresh" @click="refreshBrowserCache"
                               v-if="!clientVersionMatchServerVersion">
                            {{ tt('Refresh Browser Cache') }}
                        </v-btn>
                    </section>

                    <section class="ebk-panel" aria-labelledby="about-application-title">
                        <header class="ebk-panel-header-strip">
                            <h3 class="about-panel-title" id="about-application-title">{{ tt('global.app.title') }}</h3>
                        </header>
                        <dl class="about-rows">
                            <div class="about-row">
                                <dt>{{ tt('Version') }}</dt>
                                <dd class="ebk-numeric">{{ clientVersion }}</dd>
                            </div>
                            <div class="about-row" v-if="clientBuildTime">
                                <dt>{{ tt('Build Time') }}</dt>
                                <dd class="ebk-numeric">{{ clientBuildTime }}</dd>
                            </div>
                            <div class="about-row">
                                <dt>{{ tt('Official Website') }}</dt>
                                <dd><a class="link-no-color" href="https://github.com/krazyjakee/ezbookkeeping" target="_blank" rel="noopener">https://github.com/krazyjakee/ezbookkeeping</a></dd>
                            </div>
                            <div class="about-row">
                                <dt>{{ tt('Report Issue') }}</dt>
                                <dd><a class="link-no-color" href="https://github.com/krazyjakee/ezbookkeeping/issues" target="_blank" rel="noopener">https://github.com/krazyjakee/ezbookkeeping/issues</a></dd>
                            </div>
                            <div class="about-row">
                                <dt>{{ tt('Getting help') }}</dt>
                                <dd><a class="link-no-color" href="https://krazyjakee.github.io/ezbookkeeping/" target="_blank" rel="noopener">https://krazyjakee.github.io/ezbookkeeping/</a></dd>
                            </div>
                        </dl>
                    </section>

                    <section class="ebk-panel" aria-labelledby="about-exchange-rates-title" v-if="exchangeRatesData && !isUserCustomExchangeRates">
                        <header class="ebk-panel-header-strip">
                            <h3 class="about-panel-title" id="about-exchange-rates-title">{{ tt('Exchange Rates Data') }}</h3>
                        </header>
                        <dl class="about-rows">
                            <div class="about-row">
                                <dt>{{ tt('Provider') }}</dt>
                                <dd>
                                    <a class="link-no-color" :href="exchangeRatesData.referenceUrl" target="_blank" rel="noopener"
                                       v-if="exchangeRatesData.referenceUrl">{{ exchangeRatesData.dataSource }}</a>
                                    <span v-if="!exchangeRatesData.referenceUrl">{{ exchangeRatesData.dataSource }}</span>
                                </dd>
                            </div>
                        </dl>
                    </section>

                    <section class="ebk-panel" aria-labelledby="about-map-title" v-if="mapProviderName">
                        <header class="ebk-panel-header-strip">
                            <h3 class="about-panel-title" id="about-map-title">{{ tt('Map') }}</h3>
                        </header>
                        <dl class="about-rows">
                            <div class="about-row">
                                <dt>{{ tt('Provider') }}</dt>
                                <dd>
                                    <a class="link-no-color" :href="mapProviderWebsite" target="_blank" rel="noopener"
                                       v-if="mapProviderWebsite">{{ mapProviderName }}</a>
                                    <span v-if="!mapProviderWebsite">{{ mapProviderName }}</span>
                                </dd>
                            </div>
                        </dl>
                    </section>

                    <section class="ebk-panel" aria-labelledby="about-license-title">
                        <header class="ebk-panel-header-strip">
                            <h3 class="about-panel-title" id="about-license-title">{{ tt('License') }}</h3>
                        </header>
                        <div class="about-license text-body-medium">
                            <div class="about-license-text">
                                <span :key="num" v-for="(line, num) in licenseLines"
                                      :style="{ 'display': line ? 'initial' : 'block', 'padding' : line ? '0' : '0 0 1em 0' }">
                                    {{ `${line} ` }}
                                </span>
                            </div>
                            <div>
                                <span>ezrbookkeeping is a fork of ezBookkeeping by MaysWind. ezBookkeeping's codebase and localization translation rely on contributions from the community. The following people have contributed to ezBookkeeping:</span>
                            </div>
                            <div class="mt-4">
                                <strong>Original Project Maintainer</strong>
                                <div class="mt-2">
                                    <a target="_blank" href="https://github.com/mayswind">@mayswind</a>
                                </div>
                            </div>
                            <div class="mt-4">
                                <strong>Code Contributors</strong>
                            </div>
                            <table class="contributors-table mt-2">
                                <thead>
                                <tr>
                                    <th>Contributor</th>
                                </tr>
                                </thead>
                                <tbody>
                                <tr :key="index" v-for="(contributor, index) in contributors.code">
                                    <td>
                                        <a target="_blank" :href="`https://github.com/${contributor}`">
                                            @{{ contributor }}
                                        </a>
                                    </td>
                                </tr>
                                </tbody>
                            </table>
                            <div class="mt-4">
                                <strong>Translation Contributors</strong>
                            </div>
                            <table class="contributors-table mt-2">
                                <thead>
                                <tr>
                                    <th>Tag</th>
                                    <th>Language</th>
                                    <th>Contributors</th>
                                </tr>
                                </thead>
                                <tbody>
                                <tr :key="lang.languageTag" v-for="lang in allLanguages">
                                    <td>{{ lang.languageTag }}</td>
                                    <td>{{ lang.nativeDisplayName }}</td>
                                    <td>
                                        <template :key="contributor"
                                                  v-for="(contributor, index) in contributors.translators[lang.languageTag] ?? []">
                                            <span v-if="index > 0">, </span>
                                            <a target="_blank" :href="`https://github.com/${contributor}`">
                                                @{{ contributor }}
                                            </a>
                                        </template>
                                        <span v-if="!contributors.translators[lang.languageTag] || !contributors.translators[lang.languageTag]?.length">/</span>
                                    </td>
                                </tr>
                                </tbody>
                            </table>

                            <div class="my-4">
                                <span>ezrbookkeeping also contains additional third party software and illustration.</span><br/>
                                <span>All the third party software / illustration included or linked is redistributed under the terms and conditions of their original licenses.</span>
                            </div>
                            <div class="py-2" :key="licenseInfo.name" v-for="licenseInfo in thirdPartyLicenses">
                                <strong>{{ licenseInfo.name }}</strong>
                                <br v-if="licenseInfo.copyright"/><span v-if="licenseInfo.copyright">{{ licenseInfo.copyright }}</span>
                                <br v-if="licenseInfo.licenseUrl"/><span v-if="licenseInfo.licenseUrl">{{ licenseInfo.license || 'License' }}: </span>
                                <a target="_blank" :href="licenseInfo.licenseUrl">{{ licenseInfo.licenseUrl }}</a>
                                <br v-if="licenseInfo.url"/><a target="_blank" :href="licenseInfo.url" v-if="licenseInfo.url">{{ licenseInfo.url }}</a>
                            </div>
                        </div>
                    </section>
                </div>
            </template>
        </one-column-dialog-layout>
    </v-dialog>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAboutPageBase } from '@/views/base/AboutPageBase.ts';

import type { LanguageOption } from '@/core/language.ts';
import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';

import {
    mdiWebRefresh
} from '@mdi/js';

const { tt, getAllLanguageOptions } = useI18n();
const {
    clientVersion,
    clientVersionMatchServerVersion,
    clientBuildTime,
    exchangeRatesData,
    isUserCustomExchangeRates,
    mapProviderName,
    mapProviderWebsite,
    contributors,
    licenseLines,
    thirdPartyLicenses,
    refreshBrowserCache,
    init
} = useAboutPageBase();

const props = defineProps<{
    show: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:show', value: boolean): void;
}>();

const showState = computed<boolean>({
    get: () => {
        return props.show;
    },
    set: value => {
        emit('update:show', value);
    }
});

const allLanguages = computed<LanguageOption[]>(() => getAllLanguageOptions(false));

watch(() => props.show, (newValue) => {
    if (newValue) {
        init();
    }
});
</script>

<style>
.about-panels {
    display: flex;
    flex-direction: column;
    gap: var(--ebk-space-4);
}

.about-identity {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--ebk-space-4);
    padding: var(--ebk-space-4);

    .about-identity-logo {
        width: 3rem;
        height: 3rem;
        border-radius: var(--ebk-radius-md);
    }

    .about-identity-text {
        flex: 1 1 16rem;
        min-width: 0;
    }

    .about-identity-title {
        margin: 0;
        font-size: 1.25rem;
        font-weight: 600;
        line-height: 1.3;
        color: var(--ebk-color-text-primary);
    }

    .about-identity-attribution {
        margin-block-start: var(--ebk-space-1);
        color: var(--ebk-color-text-secondary);
        font-size: 0.8125rem;
        overflow-wrap: anywhere;

        a {
            color: var(--ebk-color-primary-text);
        }
    }
}

.about-panel-title {
    margin: 0;
    font: inherit;
    letter-spacing: inherit;
    color: inherit;
}

.about-rows {
    margin: 0;
    padding: 0;
}

.about-row {
    display: grid;
    grid-template-columns: minmax(8rem, 12rem) minmax(0, 1fr);
    gap: var(--ebk-space-3);
    align-items: baseline;
    min-height: 44px;
    padding: var(--ebk-space-3) var(--ebk-space-4);
    border-block-end: 1px solid var(--ebk-color-hairline);

    &:last-child {
        border-block-end: 0;
    }

    > dt {
        color: var(--ebk-color-text-secondary);
        font-size: 0.875rem;
    }

    > dd {
        margin: 0;
        color: var(--ebk-color-text-primary);
        overflow-wrap: anywhere;
    }
}

.about-license {
    padding: var(--ebk-space-4);
}

.about-license-text {
    margin-block-end: var(--ebk-space-4);
    padding: var(--ebk-space-3) var(--ebk-space-4);
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-control);
    background-color: var(--ebk-color-surface-sunken);
    color: var(--ebk-color-text-secondary);
    font-size: 0.8125rem;
}

@media (max-width: 599px) {
    .about-row {
        grid-template-columns: minmax(0, 1fr);
        gap: var(--ebk-space-1);
    }
}

.contributors-table {
    border-collapse: collapse;

    > thead > tr {
        > th:not(:first-child) {
            padding-inline-start: 10px;
        }

        > th:not(:last-child) {
            padding-inline-end: 10px;
        }
    }

    > thead > tr > th,
    > tbody > tr > td {
        padding: 4px 8px;
        border: 1px solid var(--ebk-color-hairline);
        text-align: start;
    }
}
</style>

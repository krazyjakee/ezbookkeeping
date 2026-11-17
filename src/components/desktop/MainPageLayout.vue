<template>
    <div class="layout-wrapper layout-nav-type-vertical ebk-shell ebk-canvas-grid"
         :class="{
             'layout-overlay-nav': isOverlayNav,
             'ebk-shell--overlay-nav': isOverlayNav,
             'ebk-shell--with-context': hasContextNav,
             'ebk-shell--context-open': hasContextNav && isOverlayNav && showVerticalOverlayMenu
         }">
        <button type="button" class="ebk-shell-skip-link" @click="skipToContent">{{ tt('Skip to content') }}</button>

        <nav class="ebk-shell-rail" :aria-label="tt('Main Navigation')" @keydown="onRailKeydown">
            <router-link to="/" class="ebk-shell-rail-logo app-logo" :aria-label="tt('global.app.title')">
                <img alt="" class="main-logo" :src="APPLICATION_LOGO_PATH" />
            </router-link>

            <ul class="ebk-shell-rail-items" v-if="!hasTopToolbar">
                <li :key="item.key" v-for="item in primaryNavItems">
                    <router-link class="ebk-shell-rail-item" data-ebk-rail-item
                                 :class="{ 'ebk-shell-rail-item--active': item.active }"
                                 :to="item.to"
                                 :aria-current="item.active ? 'page' : undefined">
                        <span class="ebk-shell-rail-item-indicator ebk-selection-indicator" aria-hidden="true"></span>
                        <v-icon class="ebk-shell-rail-item-icon" :icon="item.active ? item.activeIcon : item.icon" size="22" />
                        <span class="ebk-shell-rail-item-label">{{ item.label }}</span>
                        <v-tooltip activator="parent" location="end" :open-delay="300">{{ item.title }}</v-tooltip>
                    </router-link>
                </li>
            </ul>

            <ul class="ebk-shell-rail-items ebk-shell-rail-items--footer" v-if="!hasTopToolbar">
                <li>
                    <router-link class="ebk-shell-rail-item" data-ebk-rail-item
                                 to="/settings/preferences"
                                 :class="{ 'ebk-shell-rail-item--active': isSettingsActive }"
                                 :aria-current="isSettingsActive ? 'page' : undefined">
                        <span class="ebk-shell-rail-item-indicator ebk-selection-indicator" aria-hidden="true"></span>
                        <v-icon class="ebk-shell-rail-item-icon" :icon="isSettingsActive ? mdiCog : mdiCogOutline" size="22" />
                        <span class="ebk-shell-rail-item-label">{{ tt('Settings') }}</span>
                        <v-tooltip activator="parent" location="end" :open-delay="300">{{ tt('Settings') }}</v-tooltip>
                    </router-link>
                </li>
            </ul>
        </nav>

        <div class="layout-content-wrapper">
            <header class="layout-navbar navbar-blur ebk-shell-header">
                <div class="navbar-content-container">
                    <div class="navbar-content d-flex h-100 align-center">
                        <v-btn ref="contextNavToggle" class="ebk-shell-context-toggle me-1" color="default" variant="text"
                               :icon="true" :aria-label="tt('Open Menu')"
                               :aria-expanded="showVerticalOverlayMenu ? 'true' : 'false'"
                               aria-controls="ebk-shell-context-nav"
                               @click="openContextNav"
                               v-if="hasContextNav && isOverlayNav">
                            <v-icon :icon="mdiMenu" size="22" />
                            <v-tooltip activator="parent">{{ tt('Open Menu') }}</v-tooltip>
                        </v-btn>

                        <div class="ebk-shell-title">
                            <nav class="ebk-shell-breadcrumb" :aria-label="tt('Breadcrumb')">
                                <ol>
                                    <li>
                                        <router-link to="/">{{ tt('global.app.title') }}</router-link>
                                    </li>
                                    <li :key="idx" v-for="(crumb, idx) in breadcrumbItems">
                                        <span class="ebk-shell-breadcrumb-separator" aria-hidden="true">/</span>
                                        <span>{{ crumb }}</span>
                                    </li>
                                </ol>
                            </nav>
                            <h1 ref="pageTitleElement" class="ebk-shell-page-title" @mouseenter="updateTitleTruncation">
                                {{ currentPageTitle }}
                                <v-tooltip activator="parent" location="bottom start" :disabled="!isTitleTruncated">{{ currentPageTitle }}</v-tooltip>
                            </h1>
                        </div>

                        <div class="app-top-toolbar ebk-shell-toolbar d-inline-flex" v-if="hasTopToolbar">
                            <slot name="top-toolbar" />
                        </div>

                        <v-spacer />

                        <div class="ebk-shell-actions d-flex align-center">
                            <v-btn class="ebk-shell-add-button" color="primary" variant="flat"
                                   :class="{ 'ebk-shell-add-button--compact': isCompactHeader }"
                                   :aria-label="isCompactHeader ? tt('Add Transaction') : undefined"
                                   @click="showAddDialogInTransactionListPage"
                                   v-if="showAddTransactionButtonInDesktopNavbar && !hasTopToolbar">
                                <v-icon :icon="mdiPlus" size="20" />
                                <span class="ebk-shell-add-button-label" v-if="!isCompactHeader">{{ tt('Add Transaction') }}</span>
                                <v-tooltip activator="parent" :disabled="!isCompactHeader">{{ tt('Add Transaction') }}</v-tooltip>
                            </v-btn>

                            <span class="ebk-shell-actions-divider" aria-hidden="true"
                                  v-if="showAddTransactionButtonInDesktopNavbar && !hasTopToolbar"></span>

                            <v-btn class="ebk-shell-action-button" color="default" variant="text" density="comfortable"
                                   :aria-label="tt('Use on Mobile Device')" :icon="true"
                                   @click="showMobileQrCode = true">
                                <v-icon :icon="mdiCellphone" size="20" />
                                <v-tooltip activator="parent">{{ tt('Use on Mobile Device') }}</v-tooltip>
                            </v-btn>
                            <v-btn class="ebk-shell-action-button" color="default" variant="text" density="comfortable"
                                   :aria-label="`${tt('Theme')}: ${currentThemeName}`" :icon="true"
                                   @click="switchToNextTheme">
                                <v-icon :icon="(currentTheme === 'light' ? mdiWeatherSunny : (currentTheme === 'dark' ? mdiWeatherNight : mdiThemeLightDark))" size="20" />
                                <v-tooltip activator="parent">{{ `${tt('Theme')}: ${currentThemeName}` }}</v-tooltip>
                            </v-btn>
                            <v-btn class="ebk-shell-user-button ms-1" variant="text" :icon="true"
                                   :aria-label="currentNickName">
                                <v-avatar size="32" variant="tonal" :color="currentUserAvatar ? 'rgba(0,0,0,0)' : 'primary'">
                                    <v-icon :color="currentUserAvatar ? 'primary' : undefined" :icon="mdiAccount" size="20"/>
                                    <span class="user-avatar-image" :style="currentUserAvatarStyle" v-if="currentUserAvatar"></span>
                                </v-avatar>
                                <v-menu activator="parent" width="230" location="bottom end" offset="10px">
                                    <v-list>
                                        <v-list-item>
                                            <template #prepend>
                                                <v-list-item-action>
                                                    <v-avatar variant="tonal"
                                                              :color="currentUserAvatar ? 'rgba(0,0,0,0)' : 'primary'">
                                                        <v-img :src="currentUserAvatar" v-if="currentUserAvatar">
                                                            <template #placeholder>
                                                                <div class="d-flex align-center justify-center bg-light-primary">
                                                                    <v-icon color="primary" :icon="mdiAccount"/>
                                                                </div>
                                                            </template>
                                                        </v-img>
                                                        <v-icon :icon="mdiAccount" v-else-if="!currentUserAvatar"/>
                                                    </v-avatar>
                                                </v-list-item-action>
                                            </template>
                                            <v-list-item-title class="ms-1">
                                                {{ currentNickName }}
                                            </v-list-item-title>
                                        </v-list-item>
                                        <v-divider class="my-1"/>
                                        <v-list-item :prepend-icon="mdiCogOutline"
                                                     :title="tt('Settings')"
                                                     to="/settings/preferences"></v-list-item>
                                        <v-list-item :prepend-icon="mdiInformationOutline"
                                                     :title="tt('About')"
                                                     @click="showAboutDialog = true"></v-list-item>
                                        <v-divider class="my-1"/>
                                        <v-list-item :prepend-icon="mdiLockOutline"
                                                     :title="tt('Lock Application')"
                                                     v-if="isEnableApplicationLock"
                                                     @click="lock"></v-list-item>
                                        <v-list-item :disabled="logouting"
                                                     :prepend-icon="mdiLogout"
                                                     :title="tt('Log Out')"
                                                     @click="logout"></v-list-item>
                                    </v-list>
                                </v-menu>
                            </v-btn>
                        </div>
                    </div>
                </div>
            </header>

            <div class="ebk-shell-body">
                <aside id="ebk-shell-context-nav" ref="contextNav" class="layout-vertical-nav ebk-shell-context"
                       :class="{ 'visible': showVerticalOverlayMenu, 'overlay-nav': isOverlayNav }"
                       :role="isOverlayNav && showVerticalOverlayMenu ? 'dialog' : undefined"
                       :aria-modal="isOverlayNav && showVerticalOverlayMenu ? 'true' : undefined"
                       :aria-label="currentContextLabel"
                       @keydown="onContextNavKeydown"
                       @click="onContextNavClick"
                       v-if="hasContextNav">
                    <div class="nav-header" v-if="isOverlayNav">
                        <span class="ebk-shell-context-title ebk-technical-heading">{{ currentContextLabel }}</span>
                        <v-btn ref="contextNavCloseButton" color="default" variant="text" density="comfortable"
                               :icon="true" :aria-label="tt('Close')" @click="closeContextNav(true)">
                            <v-icon :icon="mdiClose" size="20" />
                        </v-btn>
                    </div>
                    <nav ref="contextNavItems" class="ebk-shell-context-nav" :aria-label="currentContextLabel">
                        <ul class="nav-items" :class="navItemsClass">
                            <slot name="nav-items" />
                        </ul>
                    </nav>
                </aside>

                <main id="ebk-shell-main-content" ref="mainContent" class="layout-page-content" tabindex="-1">
                    <transition name="ebk-route" appear>
                        <div class="page-content-container">
                            <slot name="content" />
                        </div>
                    </transition>
                </main>
            </div>
        </div>

        <div class="layout-overlay" :class="{ 'visible': showVerticalOverlayMenu && isOverlayNav }"
             aria-hidden="true" @click="closeContextNav(true)"></div>

        <v-overlay class="justify-center align-center" :persistent="true" v-model="showLoading">
            <v-progress-circular indeterminate></v-progress-circular>
        </v-overlay>

        <switch-to-mobile-dialog v-model:show="showMobileQrCode" />
        <about-dialog v-model:show="showAboutDialog" />
    </div>
</template>

<script setup lang="ts">
import AboutDialog from '@/views/desktop/common/dialogs/AboutDialog.vue';

import { ref, computed, useSlots, useTemplateRef, watch, nextTick, onMounted, onBeforeUnmount } from 'vue';

import { useDisplay, useTheme } from 'vuetify';
import { useRoute, useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';
import { ThemeType } from '@/core/theme.ts';

import { getSystemTheme, setExpenseAndIncomeAmountColor } from '@/lib/ui/common.ts';

import {
    mdiMenu,
    mdiClose,
    mdiHome,
    mdiHomeOutline,
    mdiListBox,
    mdiListBoxOutline,
    mdiCreditCard,
    mdiCreditCardOutline,
    mdiChartPie,
    mdiChartPieOutline,
    mdiCompass,
    mdiCompassOutline,
    mdiPlus,
    mdiCellphone,
    mdiThemeLightDark,
    mdiWeatherSunny,
    mdiWeatherNight,
    mdiAccount,
    mdiCog,
    mdiCogOutline,
    mdiInformationOutline,
    mdiLockOutline,
    mdiLogout
} from '@mdi/js';

interface PrimaryNavItem {
    key: string;
    path: string;
    to: string;
    icon: string;
    activeIcon: string;
    label: string;
    title: string;
    active: boolean;
}

type ComponentWithElement = { $el?: HTMLElement } | null;

const FOCUSABLE_SELECTOR = 'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
const CONTEXT_NAV_LINK_SELECTOR = '.nav-link > a';

const props = defineProps<{
    navItemsClass?: string;
    noNavbar?: boolean;
    pageTitle?: string;
    breadcrumbs?: string[];
    contextLabel?: string;
    activeNavKey?: string;
}>();

const slots = useSlots();
const { mdAndDown, width: displayWidth } = useDisplay();
const theme = useTheme();
const route = useRoute();
const router = useRouter();

const { tt, initLocale } = useI18n();

const rootStore = useRootStore();
const settingsStore = useSettingsStore();
const userStore = useUserStore();
const desktopPageStore = useDesktopPageStore();

const contextNav = useTemplateRef<HTMLElement>('contextNav');
const contextNavItems = useTemplateRef<HTMLElement>('contextNavItems');
const contextNavToggle = useTemplateRef<ComponentWithElement>('contextNavToggle');
const contextNavCloseButton = useTemplateRef<ComponentWithElement>('contextNavCloseButton');
const mainContent = useTemplateRef<HTMLElement>('mainContent');
const pageTitleElement = useTemplateRef<HTMLElement>('pageTitleElement');

const logouting = ref<boolean>(false);
const showVerticalOverlayMenu = ref<boolean>(false);
const showLoading = ref<boolean>(false);
const showMobileQrCode = ref<boolean>(false);
const showAboutDialog = ref<boolean>(false);
const isTitleTruncated = ref<boolean>(false);

let focusBeforeContextNavOpened: HTMLElement | null = null;
let contextNavObserver: MutationObserver | null = null;

const hasContextNav = computed<boolean>(() => !props.noNavbar);
const hasTopToolbar = computed<boolean>(() => !!slots['top-toolbar']);
const isOverlayNav = computed<boolean>(() => mdAndDown.value);
const isCompactHeader = computed<boolean>(() => displayWidth.value < 960);

const primaryNavItems = computed<PrimaryNavItem[]>(() => [
    {
        key: 'overview',
        path: '/',
        to: '/',
        icon: mdiHomeOutline,
        activeIcon: mdiHome,
        label: tt('Overview'),
        title: tt('Overview')
    },
    {
        key: 'transactions',
        path: '/transaction/list',
        to: '/transaction/list?pageType=0&dateType=7',
        icon: mdiListBoxOutline,
        activeIcon: mdiListBox,
        label: tt('Transactions'),
        title: tt('Transaction Details')
    },
    {
        key: 'statistics',
        path: '/statistics/transaction',
        to: '/statistics/transaction',
        icon: mdiChartPieOutline,
        activeIcon: mdiChartPie,
        label: tt('Statistics'),
        title: tt('Statistics & Analysis')
    },
    {
        key: 'insights',
        path: '/insights/explorer',
        to: '/insights/explorer',
        icon: mdiCompassOutline,
        activeIcon: mdiCompass,
        label: tt('Insights Explorer'),
        title: tt('Insights Explorer')
    },
    {
        key: 'accounts',
        path: '/account/list',
        to: '/account/list',
        icon: mdiCreditCardOutline,
        activeIcon: mdiCreditCard,
        label: tt('Accounts'),
        title: tt('Accounts')
    }
].map(item => ({
    ...item,
    active: props.activeNavKey ? props.activeNavKey === item.key : route.path === item.path
})));

const isSettingsActive = computed<boolean>(() => props.activeNavKey === 'settings');
const currentPrimaryNavItem = computed<PrimaryNavItem | undefined>(() => primaryNavItems.value.find(item => item.active));

const currentPageTitle = computed<string>(() => {
    if (props.pageTitle) {
        return props.pageTitle;
    }

    if (currentPrimaryNavItem.value) {
        return currentPrimaryNavItem.value.title;
    }

    if (route.path === '/overview/edit') {
        return tt('Home Page Layout');
    }

    return tt('global.app.title');
});

const breadcrumbItems = computed<string[]>(() => {
    if (props.breadcrumbs) {
        return props.breadcrumbs;
    }

    if (route.path === '/overview/edit') {
        return [tt('Overview')];
    }

    return [];
});

const currentContextLabel = computed<string>(() => props.contextLabel || currentPageTitle.value);

const currentNickName = computed<string>(() => userStore.currentUserNickname || tt('User'));
const currentUserAvatar = computed<string | null>(() => userStore.currentUserAvatar);
const currentUserAvatarStyle = computed<Record<string, string> | undefined>(() => currentUserAvatar.value ? {
    backgroundImage: `url("${currentUserAvatar.value}")`
} : undefined);

const currentTheme = computed<string>({
    get: () => {
        return settingsStore.appSettings.theme;
    },
    set: (value: string) => {
        if (value !== settingsStore.appSettings.theme) {
            settingsStore.setTheme(value);

            if (value === ThemeType.Light || value === ThemeType.Dark) {
                theme.change(value);
            } else {
                theme.change(getSystemTheme());
            }
        }
    }
});

const currentThemeName = computed<string>(() => {
    if (currentTheme.value === ThemeType.Light) {
        return tt('Light');
    } else if (currentTheme.value === ThemeType.Dark) {
        return tt('Dark');
    } else {
        return tt('System Default');
    }
});

const showAddTransactionButtonInDesktopNavbar = computed<boolean>(() => settingsStore.appSettings.showAddTransactionButtonInDesktopNavbar);
const isEnableApplicationLock = computed<boolean>(() => settingsStore.appSettings.applicationLock);

function getElement(component: ComponentWithElement): HTMLElement | null {
    return component?.$el instanceof HTMLElement ? component.$el : null;
}

function isElementVisible(element: HTMLElement): boolean {
    return element.getClientRects().length > 0 && getComputedStyle(element).visibility !== 'hidden';
}

function getFocusableElements(container: HTMLElement | null): HTMLElement[] {
    if (!container) {
        return [];
    }

    return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
        .filter(element => !element.closest('[aria-hidden="true"]') && isElementVisible(element));
}

function moveFocusInList(elements: HTMLElement[], current: HTMLElement, key: string): boolean {
    const index = elements.indexOf(current);

    if (index < 0 || !elements.length) {
        return false;
    }

    let nextIndex = index;

    if (key === 'ArrowDown' || key === 'ArrowRight') {
        nextIndex = (index + 1) % elements.length;
    } else if (key === 'ArrowUp' || key === 'ArrowLeft') {
        nextIndex = (index - 1 + elements.length) % elements.length;
    } else if (key === 'Home') {
        nextIndex = 0;
    } else if (key === 'End') {
        nextIndex = elements.length - 1;
    } else {
        return false;
    }

    elements[nextIndex]?.focus();
    return true;
}

function skipToContent(): void {
    mainContent.value?.focus();
    mainContent.value?.scrollIntoView({ block: 'start' });
}

function onRailKeydown(e: KeyboardEvent): void {
    const target = e.target as HTMLElement | null;

    if (!target || !target.matches('[data-ebk-rail-item]') || e.altKey || e.ctrlKey || e.metaKey) {
        return;
    }

    const items = Array.from((e.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[data-ebk-rail-item]'));

    if (moveFocusInList(items, target, e.key)) {
        e.preventDefault();
    }
}

function openContextNav(): void {
    if (!isOverlayNav.value) {
        return;
    }

    focusBeforeContextNavOpened = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    showVerticalOverlayMenu.value = true;

    nextTick(() => {
        const activeLink = contextNavItems.value?.querySelector<HTMLElement>('.nav-link > a.router-link-exact-active');
        const target = activeLink && isElementVisible(activeLink) ? activeLink : (getElement(contextNavCloseButton.value) || getFocusableElements(contextNav.value)[0]);
        target?.focus({ preventScroll: true });
    });
}

function closeContextNav(restoreFocus: boolean): void {
    if (!showVerticalOverlayMenu.value) {
        return;
    }

    showVerticalOverlayMenu.value = false;

    if (restoreFocus) {
        const target = getElement(contextNavToggle.value) || focusBeforeContextNavOpened;
        target?.focus({ preventScroll: true });
    }

    focusBeforeContextNavOpened = null;
}

function onContextNavKeydown(e: KeyboardEvent): void {
    const target = e.target as HTMLElement | null;

    if (!target) {
        return;
    }

    if (e.key === 'Escape' && isOverlayNav.value && showVerticalOverlayMenu.value) {
        e.preventDefault();
        e.stopPropagation();
        closeContextNav(true);
        return;
    }

    if (e.key === 'Tab' && isOverlayNav.value && showVerticalOverlayMenu.value) {
        const focusable = getFocusableElements(contextNav.value);
        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (!first || !last) {
            e.preventDefault();
            return;
        }

        if (e.shiftKey && (target === first || !contextNav.value?.contains(target))) {
            e.preventDefault();
            last.focus();
        } else if (!e.shiftKey && target === last) {
            e.preventDefault();
            first.focus();
        }

        return;
    }

    if (!target.matches(CONTEXT_NAV_LINK_SELECTOR) || e.altKey || e.ctrlKey || e.metaKey) {
        return;
    }

    if ((e.key === 'Enter' || e.key === ' ') && !target.hasAttribute('href')) {
        e.preventDefault();
        target.click();
        return;
    }

    const links = Array.from(contextNavItems.value?.querySelectorAll<HTMLElement>(CONTEXT_NAV_LINK_SELECTOR) || [])
        .filter(link => isElementVisible(link));

    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight' && moveFocusInList(links, target, e.key)) {
        e.preventDefault();
    }
}

function onContextNavClick(e: MouseEvent): void {
    if (!isOverlayNav.value || !showVerticalOverlayMenu.value) {
        return;
    }

    const target = e.target as HTMLElement | null;
    const link = target?.closest<HTMLElement>(CONTEXT_NAV_LINK_SELECTOR);

    if (link && !link.classList.contains('disabled')) {
        closeContextNav(true);
    }
}

// page-provided context nav items may use anchors without href as selectable items, make them keyboard accessible
function enhanceContextNavItems(): void {
    const container = contextNavItems.value;

    if (!container) {
        return;
    }

    container.querySelectorAll<HTMLElement>(CONTEXT_NAV_LINK_SELECTOR).forEach(link => {
        const active = link.classList.contains('router-link-exact-active');
        const disabled = link.classList.contains('disabled');

        if (!link.hasAttribute('href')) {
            if (link.getAttribute('tabindex') !== '0') {
                link.setAttribute('tabindex', '0');
            }

            if (link.getAttribute('role') !== 'button') {
                link.setAttribute('role', 'button');
            }

            if (active && link.getAttribute('aria-current') !== 'true') {
                link.setAttribute('aria-current', 'true');
            } else if (!active && link.hasAttribute('aria-current')) {
                link.removeAttribute('aria-current');
            }
        }

        if (disabled && link.getAttribute('aria-disabled') !== 'true') {
            link.setAttribute('aria-disabled', 'true');
        } else if (!disabled && link.hasAttribute('aria-disabled')) {
            link.removeAttribute('aria-disabled');
        }
    });
}

function updateTitleTruncation(): void {
    const element = pageTitleElement.value;
    isTitleTruncated.value = !!element && element.scrollWidth > element.clientWidth + 1;
}

function switchToNextTheme(): void {
    if (currentTheme.value === 'light') {
        currentTheme.value = 'dark';
    } else if (currentTheme.value === 'dark') {
        currentTheme.value = 'auto';
    } else {
        currentTheme.value = 'light';
    }
}

function lock(): void {
    rootStore.lock();
    router.replace('/unlock');
}

function logout(): void {
    logouting.value = true;
    showLoading.value = true;

    rootStore.logout().then(() => {
        settingsStore.clearAppSettings();

        initLocale(userStore.currentUserLanguage, settingsStore.appSettings.timeZone).then(localeDefaultSettings => {
            settingsStore.updateLocalizedDefaultSettings(localeDefaultSettings);
        }).finally(() => {
            logouting.value = false;
            showLoading.value = false;

            setExpenseAndIncomeAmountColor(userStore.currentUserExpenseAmountColor, userStore.currentUserIncomeAmountColor);
            router.replace('/login');
        });
    }).catch(error => {
        logouting.value = false;
        showLoading.value = false;

        if (!error.processed) {
            desktopPageStore.addErrorToast(error);
        }
    });
}

function showAddDialogInTransactionListPage(): void {
    desktopPageStore.setShowAddTransactionDialogInTransactionList();
    router.push('/transaction/list?pageType=0&dateType=7');
}

watch(isOverlayNav, newValue => {
    if (!newValue) {
        showVerticalOverlayMenu.value = false;
        focusBeforeContextNavOpened = null;
    }
});

watch(() => route.fullPath, () => {
    closeContextNav(false);
});

watch(currentPageTitle, () => {
    nextTick(updateTitleTruncation);
});

onMounted(() => {
    enhanceContextNavItems();

    if (contextNavItems.value && typeof MutationObserver !== 'undefined') {
        contextNavObserver = new MutationObserver(() => enhanceContextNavItems());
        contextNavObserver.observe(contextNavItems.value, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['class']
        });
    }
});

onBeforeUnmount(() => {
    contextNavObserver?.disconnect();
    contextNavObserver = null;
});
</script>

<style>
.top-navigation-button.v-btn {
    width: 40px;
    min-width: 40px;
    border-radius: var(--ebk-radius-control);
}

.top-navigation-button.v-btn:not(.v-btn--active) {
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity)) !important;
}

.user-avatar-image {
    position: absolute;
    z-index: 1;
    background-position: center;
    background-repeat: no-repeat;
    background-size: cover;
    inset: 0;
    pointer-events: none;
}
</style>

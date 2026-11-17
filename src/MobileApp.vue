<template>
    <f7-app v-bind="f7params">
        <f7-view id="main-view" class="safe-areas" main url="/"></f7-view>
        <Transition name="ebk-fade">
            <div ref="offlineBanner" class="ebk-offline-banner" role="status" aria-live="polite" v-if="isOffline">
                <i class="icon f7-icons" aria-hidden="true">wifi_slash</i>
                <span>{{ tt('You are offline. Changes cannot be saved until the connection is restored.') }}</span>
            </div>
        </Transition>
        <!-- framework7 toasts and notifications have no live region, their text is mirrored here so it is announced (#36, WCAG 4.1.3) -->
        <div class="ebk-visually-hidden" role="status" aria-live="polite" aria-atomic="true">{{ liveMessage }}</div>
    </f7-app>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, useTemplateRef } from 'vue';

import type { Framework7Parameters, Notification, Actions, Dialog, Popover, Popup, Router, Sheet } from 'framework7/types';
import { f7ready } from 'framework7-vue';
import routes from './router/mobile.ts';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useEnvironmentsStore } from '@/stores/environment.ts';
import { useUserStore } from '@/stores/user.ts';
import { useTokensStore } from '@/stores/token.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';
import { ThemeType } from '@/core/theme.ts';

import { isFunction } from '@/lib/common.ts';
import { isProduction } from '@/lib/version.ts';
import { getTheme, isEnableSwipeBack, isEnableAnimate } from '@/lib/settings.ts';
import { isReducedMotion, useReducedMotion } from '@/lib/motion.ts';
import { initMapProvider } from '@/lib/map/index.ts';
import { isUserLogined, isUserUnlocked } from '@/lib/userstate.ts';
import { updateMapCacheExpiration, updateTransactionPictureCacheExpiration } from '@/lib/cache.ts';
import { setExpenseAndIncomeAmountColor } from '@/lib/ui/common.ts';
import { isiOSHomeScreenMode, isModalShowing, setAppFontSize, useI18nUIComponents } from '@/lib/ui/mobile.ts';
import logger from '@/lib/logger.ts';

const { tt, getCurrentLanguageInfo, setLanguage, initLocale } = useI18n();
const { showToast } = useI18nUIComponents();

const rootStore = useRootStore();
const settingsStore = useSettingsStore();
const environmentsStore = useEnvironmentsStore();
const userStore = useUserStore();
const tokensStore = useTokensStore();
const exchangeRatesStore = useExchangeRatesStore();

function isEnableFramework7Animate(): boolean {
    return isEnableAnimate() && !isReducedMotion();
}

const f7params = ref<Framework7Parameters>({
    name: 'ezrbookkeeping',
    theme: 'ios',
    colors: {
        primary: '#c67e48'
    },
    routes: routes,
    darkMode: (() => {
        let darkMode: boolean | string = 'auto';

        if (getTheme() === ThemeType.Light) {
            darkMode = false;
        } else if (getTheme() === ThemeType.Dark) {
            darkMode = true;
        }

        return darkMode;
    })(),
    touch: {
        disableContextMenu: true,
        tapHold: true
    },
    serviceWorker: {
        path: isProduction() ? './sw.js' : undefined,
        scope: './',
    },
    actions: {
        animate: isEnableFramework7Animate(),
        backdrop: true,
        closeOnEscape: true
    },
    dialog: {
        // @ts-expect-error there is an "animate" field in dialog parameters, but it is not declared in the type definition file
        animate: isEnableFramework7Animate(),
        backdrop: true
    },
    popover: {
        animate: isEnableFramework7Animate(),
        backdrop: true,
        closeOnEscape: true
    },
    popup: {
        animate: isEnableFramework7Animate(),
        backdrop: true,
        closeOnEscape: true,
        swipeToClose: true
    },
    sheet: {
        animate: isEnableFramework7Animate(),
        backdrop: true,
        closeOnEscape: true
    },
    input: {
        // keep a focused input away from the on-screen keyboard and the bottom toolbar (android scrolls by default, ios scrolls natively)
        scrollIntoViewCentered: true
    },
    smartSelect: {
        routableModals: false
    },
    view: {
        animate: isEnableFramework7Animate(),
        browserHistory: !isiOSHomeScreenMode(),
        browserHistoryInitialMatch: true,
        browserHistoryAnimate: false,
        iosSwipeBack: isEnableSwipeBack(),
        iosSwipeBackAnimateShadow: false,
        mdSwipeBack: isEnableSwipeBack(),
        mdSwipeBackAnimateShadow: false
    }
});

const offlineBanner = useTemplateRef<HTMLElement>('offlineBanner');
const liveMessage = ref<string>('');

function announce(element: HTMLElement | undefined | null): void {
    const text = element?.textContent?.replace(/\s+/g, ' ').trim();

    if (!text) {
        return;
    }

    // clear first, so the same message repeated is announced again
    liveMessage.value = '';
    window.setTimeout(() => {
        liveMessage.value = text;
    }, 50);
}

const notification = ref<Notification.Notification | null>(null);
const isOffline = ref<boolean>(typeof navigator !== 'undefined' && navigator.onLine === false);

const hasPushPopupBackdrop = ref<boolean | undefined>(undefined);
const hasBackdrop = ref<boolean | undefined>(undefined);
const currentNotificationContent = computed<string | null>(() => rootStore.currentNotification);

function setThemeColorMeta(darkMode: boolean | undefined): void {
    // keep the root theme attribute (used by the shared design tokens and set before first paint in mobile.html) in sync with framework7
    document.documentElement.setAttribute('data-ebk-theme', darkMode ? ThemeType.Dark : ThemeType.Light);

    if (hasPushPopupBackdrop.value) {
        document.querySelector('meta[name=theme-color]')?.setAttribute('content', '#000');
        return;
    }

    if (darkMode) {
        if (hasBackdrop.value) {
            document.querySelector('meta[name=theme-color]')?.setAttribute('content', '#0b0b0b');
        } else {
            document.querySelector('meta[name=theme-color]')?.setAttribute('content', '#0b0a09');
        }
    } else {
        if (hasBackdrop.value) {
            document.querySelector('meta[name=theme-color]')?.setAttribute('content', '#949495');
        } else {
            document.querySelector('meta[name=theme-color]')?.setAttribute('content', '#f7f4ef');
        }
    }
}

function updateOnlineState(): void {
    isOffline.value = navigator.onLine === false;
}

let offlineBannerResizeObserver: ResizeObserver | null = null;

// the offline banner sits below the navbar, page contents move down by its height instead of being covered by it
function updateOfflineBannerSpace(): void {
    const root = document.documentElement;
    const banner = offlineBanner.value;

    if (isOffline.value && banner) {
        root.classList.add('ebk-offline');
        root.style.setProperty('--ebk-offline-banner-space', `${Math.ceil(banner.getBoundingClientRect().height) + 8}px`);
    } else {
        root.classList.remove('ebk-offline');
        root.style.removeProperty('--ebk-offline-banner-space');
    }
}

function setupDialogAccessibility(dialog: Dialog.Dialog): void {
    const dialogElement = dialog.el as HTMLElement | undefined;

    if (!dialogElement) {
        return;
    }

    const titleElement = dialogElement.querySelector<HTMLElement>('.dialog-title');

    if (!dialogElement.hasAttribute('role')) {
        dialogElement.setAttribute('role', dialogElement.querySelector('.dialog-buttons') ? 'alertdialog' : 'dialog');
    }

    dialogElement.setAttribute('aria-modal', 'true');

    if (titleElement) {
        if (!titleElement.id) {
            titleElement.id = `ebk-dialog-title-${Date.now().toString(36)}`;
        }

        dialogElement.setAttribute('aria-labelledby', titleElement.id);
    }

    // dialogs with inputs (prompts) focus their input, the others focus the safe action:
    // the first non-primary button (cancel), or the only button of an alert
    if (dialogElement.querySelector('input, textarea, select') || dialogElement.contains(document.activeElement)) {
        return;
    }

    const buttons = Array.from(dialogElement.querySelectorAll<HTMLButtonElement>('.dialog-buttons > .dialog-button'));
    const safeButton = buttons.length > 1 ? buttons.find(button => !button.classList.contains('button-fill')) : buttons[0];

    safeButton?.focus({ preventScroll: true });
}

// the home page navbar shows the wordmark instead of a page title (D5, docs/design/README.md §5.2)
function markHomePageNavbar(page: Router.Page | undefined): void {
    const navbarEl = page?.navbarEl || page?.el?.querySelector<HTMLElement>(':scope > .navbar');

    if (!navbarEl || page?.route?.path !== '/' || navbarEl.classList.contains('ebk-navbar-home')) {
        return;
    }

    navbarEl.classList.add('ebk-navbar-home');
    navbarEl.style.setProperty('--ebk-app-logo', `url("${APPLICATION_LOGO_PATH}")`);

    // the logo changes the title width, let framework7 center the title again
    page?.app?.navbar?.size(navbarEl);
}

const DISABLEABLE_CONTROL_SELECTOR = 'a.link, a.button, a.list-button, a.item-link, a.actions-button, .actions-button, a.tab-link';
let disabledStateObserver: MutationObserver | null = null;
let hiddenPagesObserver: MutationObserver | null = null;

function syncDisabledControlState(element: Element): void {
    const disabled = element.classList.contains('disabled') || !!element.parentElement?.classList.contains('disabled');

    if (disabled && element.getAttribute('aria-disabled') !== 'true') {
        element.setAttribute('aria-disabled', 'true');
    } else if (!disabled && element.getAttribute('aria-disabled') === 'true') {
        element.removeAttribute('aria-disabled');
    }
}

function syncDisabledControlStates(root: Element, changedClassOnly?: boolean): void {
    if (root.matches(DISABLEABLE_CONTROL_SELECTOR)) {
        syncDisabledControlState(root);
    }

    // a "disabled" class on a list item applies to its list button, other subtrees only change with added nodes
    const scope = changedClassOnly ? Array.from(root.children) : Array.from(root.querySelectorAll(DISABLEABLE_CONTROL_SELECTOR));

    for (const element of scope) {
        if (element.matches(DISABLEABLE_CONTROL_SELECTOR)) {
            syncDisabledControlState(element);
        }
    }
}

// pages (and their navbars) kept in the stack behind the current page stay in the DOM, take them out of the tab order
// and the accessibility tree while they are hidden (#36, WCAG 2.4.3)
function syncHiddenPagesInert(): void {
    document.querySelectorAll<HTMLElement>('.view .page, .view .navbar').forEach(element => {
        const hidden = element.classList.contains('page-previous') || element.classList.contains('navbar-previous');

        if (hidden && !element.hasAttribute('inert')) {
            element.setAttribute('inert', '');
        } else if (!hidden && element.hasAttribute('inert')) {
            element.removeAttribute('inert');
        }
    });
}

// framework7 renders the navbar back link as an icon-only link without text, give it an accessible name, and expose the
// page heading and main region (#36, WCAG 1.3.1 / 4.1.2)
function labelNavbarBackLinks(page: Router.Page | undefined): void {
    const navbarEl = page?.navbarEl || page?.el?.querySelector<HTMLElement>(':scope > .navbar');

    // page structure for assistive technology: the navbar title is the page heading, the page content is the main region
    const titleEl = navbarEl?.querySelector<HTMLElement>('.title');

    if (titleEl && !titleEl.hasAttribute('role')) {
        titleEl.setAttribute('role', 'heading');
        titleEl.setAttribute('aria-level', '1');
    }

    const pageContentEl = page?.el?.querySelector<HTMLElement>(':scope > .page-content');

    if (pageContentEl && !pageContentEl.hasAttribute('role')) {
        pageContentEl.setAttribute('role', 'main');
    }

    navbarEl?.querySelectorAll<HTMLElement>('.link.back').forEach(link => {
        if (!link.textContent?.trim() && (!link.hasAttribute('aria-label') || link.dataset['ebkBackLabel'])) {
            link.setAttribute('aria-label', tt('Back'));
            link.dataset['ebkBackLabel'] = 'true';
        }
    });
}

function onBackdropChanged(element: { push?: boolean, opened?: boolean }): void {
    if (element.push) {
        hasPushPopupBackdrop.value = element.opened;
    } else {
        hasBackdrop.value = element.opened;
    }

    setThemeColorMeta(environmentsStore.framework7DarkMode);
}

onMounted(() => {
    setAppFontSize(settingsStore.appSettings.fontSize);

    f7ready((f7) => {
        environmentsStore.framework7DarkMode = f7.darkMode;
        setThemeColorMeta(f7.darkMode);

        f7.on('actionsOpen', (actions: Actions.Actions) => onBackdropChanged(actions));
        f7.on('actionsClose', (actions: Actions.Actions) => onBackdropChanged(actions));
        f7.on('dialogOpen', (dialog: Dialog.Dialog) => onBackdropChanged(dialog));
        f7.on('dialogClose', (dialog: Dialog.Dialog) => onBackdropChanged(dialog));
        f7.on('dialogOpened', (dialog: Dialog.Dialog) => setupDialogAccessibility(dialog));
        f7.on('toastOpen', (toast: { el?: HTMLElement }) => announce(toast?.el));
        f7.on('notificationOpen', (notification: { el?: HTMLElement }) => announce(notification?.el));
        f7.on('popoverOpen', (popover: Popover.Popover) => onBackdropChanged(popover));
        f7.on('popoverClose', (popover: Popover.Popover) => onBackdropChanged(popover));
        f7.on('popupOpen', (popup: Popup.Popup) => onBackdropChanged(popup));
        f7.on('popupClose', (popup: Popup.Popup) => onBackdropChanged(popup));
        f7.on('sheetOpen', (sheet: Sheet.Sheet) => onBackdropChanged(sheet));
        f7.on('sheetClose', (sheet: Sheet.Sheet) => onBackdropChanged(sheet));

        f7.on('pageInit', (page: Router.Page) => {
            markHomePageNavbar(page);
            labelNavbarBackLinks(page);
        });
        f7.on('pageBeforeIn', (page: Router.Page) => {
            markHomePageNavbar(page);
            labelNavbarBackLinks(page);
            page?.el?.removeAttribute('inert');
            page?.navbarEl?.removeAttribute('inert');
        });

        // the page stack also changes without page events (initial browser history match), so follow the page classes
        const mainViewEl = document.getElementById('main-view');
        let hiddenPagesSyncScheduled = false;

        if (mainViewEl) {
            hiddenPagesObserver = new MutationObserver(mutations => {
                if (hiddenPagesSyncScheduled || !mutations.some(mutation => (mutation.target as Element).matches?.('.page, .navbar') || mutation.type === 'childList')) {
                    return;
                }

                hiddenPagesSyncScheduled = true;
                requestAnimationFrame(() => {
                    hiddenPagesSyncScheduled = false;
                    syncHiddenPagesInert();
                });
            });
            hiddenPagesObserver.observe(mainViewEl, { subtree: true, childList: true, attributes: true, attributeFilter: ['class'] });
        }

        const currentPageEl = document.querySelector<HTMLElement & { f7Page?: Router.Page }>('#main-view > .page-current');
        markHomePageNavbar(currentPageEl?.f7Page);
        labelNavbarBackLinks(currentPageEl?.f7Page);
        syncHiddenPagesInert();

        f7.on('pageBeforeOut', () => {
            if (isModalShowing()) {
                f7.actions.close('.actions-modal.modal-in', false);
                f7.dialog.close('.dialog.modal-in', false);
                f7.popover.close('.popover.modal-in', false);
                f7.popup.close('.popup.modal-in', false);
                f7.sheet.close('.sheet-modal.modal-in', false);
            }
        });

        f7.on('darkModeChange', (darkMode) => {
            environmentsStore.framework7DarkMode = darkMode;
            setThemeColorMeta(darkMode);
        });

        // reduced motion can change at runtime (in-app setting or system preference), new pages and modals follow it,
        // and transitions of modals which are already created are collapsed by the reduced motion css rules
        watch(useReducedMotion(), () => {
            const animate = isEnableFramework7Animate();
            const appParams = f7.params as unknown as Record<string, { animate?: boolean } | undefined>;

            for (const paramsKey of ['actions', 'dialog', 'popover', 'popup', 'sheet', 'view']) {
                const params = appParams[paramsKey];

                if (params) {
                    params.animate = animate;
                }
            }

            const mainView = f7.views.main;

            if (mainView) {
                mainView.params.animate = animate;

                if (mainView.router) {
                    mainView.router.params.animate = animate;
                }
            }
        });
    });

    if (typeof ResizeObserver !== 'undefined') {
        offlineBannerResizeObserver = new ResizeObserver(() => updateOfflineBannerSpace());
    }

    window.addEventListener('online', updateOnlineState);
    window.addEventListener('offline', updateOnlineState);
    updateOnlineState();

    document.addEventListener('DOMContentLoaded', () => {
        const languageInfo = getCurrentLanguageInfo();
        initMapProvider(languageInfo?.alternativeLanguageTag);
    });

    // framework7 marks disabled links and buttons with a "disabled" class only, expose the state to assistive technology (#36, WCAG 4.1.2)
    syncDisabledControlStates(document.body);
    disabledStateObserver = new MutationObserver(mutations => {
        for (const mutation of mutations) {
            if (mutation.type === 'attributes') {
                syncDisabledControlStates(mutation.target as Element, true);
            } else {
                mutation.addedNodes.forEach(node => {
                    if (node.nodeType === Node.ELEMENT_NODE) {
                        syncDisabledControlStates(node as Element);
                    }
                });
            }
        }
    });
    disabledStateObserver.observe(document.body, { subtree: true, childList: true, attributes: true, attributeFilter: ['class'] });

    document.addEventListener('dragstart', (e) => {
        if (!e.target || !('closest' in e.target) || !isFunction(e.target.closest) || !e.target.closest('.dragenabled')) {
            e.preventDefault();
        }
    }, true);
});

watch(offlineBanner, (newElement, oldElement) => {
    if (oldElement) {
        offlineBannerResizeObserver?.unobserve(oldElement);
    }

    if (newElement) {
        offlineBannerResizeObserver?.observe(newElement);
    }

    updateOfflineBannerSpace();
}, { flush: 'post' });

watch(isOffline, () => updateOfflineBannerSpace(), { flush: 'post' });

onBeforeUnmount(() => {
    disabledStateObserver?.disconnect();
    disabledStateObserver = null;
    hiddenPagesObserver?.disconnect();
    hiddenPagesObserver = null;
    offlineBannerResizeObserver?.disconnect();
    offlineBannerResizeObserver = null;
});

watch(currentNotificationContent, (newValue) => {
    if (notification.value) {
        notification.value.close();
        // @ts-expect-error there is an "destroy" function in the Notification component, but it is not defined in the type definition file
        // see https://framework7.io/docs/notification
        notification.value.destroy();
        notification.value = null;
    }

    if (newValue) {
        f7ready((f7) => {
            notification.value = f7.notification.create({
                icon: `<img alt="logo" src="${APPLICATION_LOGO_PATH}" />`,
                title: tt('global.app.title'),
                text: newValue,
                closeOnClick: true,
                on: {
                    close() {
                        rootStore.setNotificationContent(null);
                    }
                }
            }).open();
        });
    }
});

initLocale(userStore.currentUserLanguage, settingsStore.appSettings.timeZone).then(localeDefaultSettings => {
    settingsStore.updateLocalizedDefaultSettings(localeDefaultSettings);
}).catch(error => {
    logger.error('failed to initialize locale', error);

    if (!error.processed) {
        showToast(error.message || error);
    }
});

setExpenseAndIncomeAmountColor(userStore.currentUserExpenseAmountColor, userStore.currentUserIncomeAmountColor);

if (isUserLogined()) {
    if (!settingsStore.appSettings.applicationLock || isUserUnlocked()) {
        // refresh token if user is logined
        tokensStore.refreshTokenAndRevokeOldToken().then(response => {
            const user = response.user;
            const languageSettings = user ? setLanguage(user.language).catch(error => {
                if (!error.processed) {
                    showToast(error.message || error);
                }

                return Promise.reject(error);
            }) : Promise.reject();

            languageSettings.then(localeDefaultSettings => {
                settingsStore.updateLocalizedDefaultSettings(localeDefaultSettings);
            }).finally(() => {
                if (user) {
                    setExpenseAndIncomeAmountColor(user.expenseAmountColor, user.incomeAmountColor);

                    if (response.notificationContent) {
                        rootStore.setNotificationContent(response.notificationContent);
                    }
                }

                updateMapCacheExpiration(settingsStore.appSettings.mapCacheExpiration);
                updateTransactionPictureCacheExpiration(settingsStore.appSettings.transactionPictureCacheExpiration);
                exchangeRatesStore.removeExpiredExchangeRates(true);
                exchangeRatesStore.autoUpdateExchangeRatesData();
            });
        }).catch(error => {
            logger.error('failed to refresh token', error);
        });
    }
}
</script>

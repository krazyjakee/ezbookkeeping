import { ref, computed, watch, type ComputedRef } from 'vue';

import { MotionPreference } from '@/core/setting.ts';
import { getApplicationSettings } from '@/lib/settings.ts';

export const REDUCED_MOTION_ROOT_CLASS: string = 'ebk-reduce-motion';
export const FULL_MOTION_ROOT_CLASS: string = 'ebk-motion-full';

// keep in sync with the motion tokens in src/styles/common/_tokens.scss
export const MOTION_DURATION_XS: number = 90;
export const MOTION_DURATION_SM: number = 160;
export const MOTION_DURATION_MD: number = 240;
export const MOTION_DURATION_LG: number = 360;
export const MOTION_DURATION_REDUCED: number = 120;
export const MOTION_STAGGER_STEP: number = 30;
export const MOTION_STAGGER_MAX_ITEMS: number = 6;

const REDUCED_MOTION_MEDIA_QUERY: string = '(prefers-reduced-motion: reduce)';

export interface ChartAnimationOptions {
    readonly animation: boolean;
    readonly animationDuration: number;
    readonly animationDurationUpdate: number;
    readonly animationEasing: string;
    readonly animationEasingUpdate: string;
}

function normalizeMotionPreference(value: unknown): MotionPreference {
    if (value === MotionPreference.Reduce || value === MotionPreference.Full) {
        return value;
    }

    return MotionPreference.System;
}

function getReducedMotionMediaQueryList(): MediaQueryList | null {
    if (typeof window === 'undefined' || !window.matchMedia) {
        return null;
    }

    return window.matchMedia(REDUCED_MOTION_MEDIA_QUERY);
}

const reducedMotionMediaQueryList: MediaQueryList | null = getReducedMotionMediaQueryList();
const motionPreference = ref<MotionPreference>(normalizeMotionPreference(getApplicationSettings().motionPreference));
const systemPrefersReducedMotion = ref<boolean>(!!reducedMotionMediaQueryList?.matches);

reducedMotionMediaQueryList?.addEventListener('change', (e: MediaQueryListEvent) => {
    systemPrefersReducedMotion.value = e.matches;
});

const reducedMotion = computed<boolean>(() => {
    if (motionPreference.value === MotionPreference.Reduce) {
        return true;
    } else if (motionPreference.value === MotionPreference.Full) {
        return false;
    } else {
        return systemPrefersReducedMotion.value;
    }
});

let motionRootClassWatcherInitialized: boolean = false;

function applyMotionRootClasses(): void {
    if (typeof document === 'undefined') {
        return;
    }

    const root = document.documentElement;
    root.classList.toggle(REDUCED_MOTION_ROOT_CLASS, reducedMotion.value);
    root.classList.toggle(FULL_MOTION_ROOT_CLASS, motionPreference.value === MotionPreference.Full);
}

/**
 * Starts keeping the motion classes of the root element (`ebk-reduce-motion` / `ebk-motion-full`)
 * in sync with the motion preference and the system reduced motion preference.
 */
export function initMotionPreference(): void {
    if (motionRootClassWatcherInitialized) {
        return;
    }

    motionRootClassWatcherInitialized = true;
    watch([motionPreference, reducedMotion], () => applyMotionRootClasses(), { immediate: true });
}

export function getMotionPreference(): MotionPreference {
    return motionPreference.value;
}

export function setMotionPreference(value: unknown): void {
    motionPreference.value = normalizeMotionPreference(value);
}

/**
 * Returns whether motion should be reduced, reading this inside a computed / watcher / render function is reactive
 */
export function isReducedMotion(): boolean {
    return reducedMotion.value;
}

export function useReducedMotion(): ComputedRef<boolean> {
    return reducedMotion;
}

export function getMotionDuration(duration: number): number {
    return reducedMotion.value ? Math.min(duration, MOTION_DURATION_REDUCED) : duration;
}

export function getStaggerIndex(index: number): number {
    return Math.min(Math.max(index, 0), MOTION_STAGGER_MAX_ITEMS - 1);
}

export function getStaggerStyle(index: number): Record<string, string> {
    return {
        '--ebk-motion-index': getStaggerIndex(index).toString(10)
    };
}

/**
 * Returns the echarts animation options, chart entrance animations are disabled when motion is reduced
 * and bounded to the large motion duration (360ms) otherwise
 */
export function getChartAnimationOptions(enabled?: boolean): ChartAnimationOptions {
    return {
        animation: (enabled ?? true) && !reducedMotion.value,
        animationDuration: MOTION_DURATION_LG,
        animationDurationUpdate: MOTION_DURATION_MD,
        animationEasing: 'cubicOut',
        animationEasingUpdate: 'cubicOut'
    };
}

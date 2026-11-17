import { type ComputedRef, type InjectionKey, computed, inject, provide } from 'vue';

export interface OverviewWidgetState {
    loadFailed: ComputedRef<boolean>;
    retry: () => void;
}

const OVERVIEW_WIDGET_STATE_KEY: InjectionKey<OverviewWidgetState> = Symbol('overviewWidgetState');

/**
 * Shares the dashboard load state with the widget panels, so that every widget without data
 * can show the error state with a retry action instead of an empty panel
 */
export function provideOverviewWidgetState(state: OverviewWidgetState): void {
    provide(OVERVIEW_WIDGET_STATE_KEY, state);
}

export function useOverviewWidgetState(): OverviewWidgetState {
    return inject(OVERVIEW_WIDGET_STATE_KEY, {
        loadFailed: computed<boolean>(() => false),
        retry: () => {}
    });
}

# Motion system

This page documents the shared motion primitives introduced by #26. Motion in ezbookkeeping2 should be responsive and purposeful. It confirms what happened and never holds the user back.

- **Tokens:** `src/styles/common/_tokens.scss`
- **Transition presets and classes:** `src/styles/common/_motion.scss`
- **Runtime state:** `src/lib/motion.ts` (`isReducedMotion()`, `useReducedMotion()`, `getChartAnimationOptions()`, stagger helpers)
- **Setting:** `motionPreference` (`'system' | 'reduce' | 'full'`, default `'system'`), shown as **Motion** in the Preferences settings page on desktop and mobile

## Tokens

| Token | Value | Reduced motion | Use |
| --- | --- | --- | --- |
| `--ebk-motion-duration-xs` | 90ms | 90ms | Hover and press feedback |
| `--ebk-motion-duration-sm` | 160ms | 120ms | Selection indicators, small reveals, exits |
| `--ebk-motion-duration-md` | 240ms | 120ms | Dialogs, sheets, panel reveals |
| `--ebk-motion-duration-lg` | 360ms | 120ms | Route transitions, chart entrances |
| `--ebk-motion-duration-reduced` | 120ms | 120ms | Maximum duration of any reduced motion change |
| `--ebk-motion-ease-standard` | `cubic-bezier(.2,0,0,1)` | | Default |
| `--ebk-motion-ease-emphasized` | `cubic-bezier(.3,0,0,1)` | | Entrances |
| `--ebk-motion-ease-exit` | `cubic-bezier(.4,0,1,1)` | | Exits |
| `--ebk-motion-distance-sm` / `-md` | 4px / 12px | 0 | Translate distances |
| `--ebk-motion-scale-from` | .96 | 1 | Scale-in start |
| `--ebk-motion-stagger` | 30ms | 0ms | Stagger step per item |
| `--ebk-motion-stagger-max-index` | 5 | | Stagger at most 6 items, later items share the last delay |

TypeScript code can use the same values: `MOTION_DURATION_XS/SM/MD/LG/REDUCED`, `MOTION_STAGGER_STEP`, and `MOTION_STAGGER_MAX_ITEMS` from `src/lib/motion.ts`.

Rules: animate only `transform` and `opacity`. Don't run continuous animation in bookkeeping workspaces; loading indicators are the only continuous motion, and they stop when hidden. Copper edge illumination and staggered reveals are for overview widgets, selection, and newly loaded content.

## Reusable transitions

### Vue `<Transition>` presets

| Name | Enter | Leave | Use |
| --- | --- | --- | --- |
| `ebk-fade` | opacity, `sm`, standard | opacity, `sm`, exit | Hover-level content, notifications, inline swaps |
| `ebk-rise` | opacity + 12px rise, `md`, emphasized | opacity + 4px lift, `sm`, exit | Panels, sheets, newly loaded sections |
| `ebk-scale` | opacity + scale from .96, `md`, emphasized | same, `sm`, exit | Custom dialogs, menus, popovers |
| `ebk-reveal` | opacity + 4px rise, `md`, staggered by `--ebk-motion-index` | opacity, `sm`, exit | Widget and list reveals (`<TransitionGroup>`) |
| `ebk-route` | opacity + 4px rise, `lg` | none, removed immediately | Route content |

```vue
<Transition name="ebk-rise">
    <section class="ebk-panel" v-if="loaded">...</section>
</Transition>

<TransitionGroup name="ebk-reveal" tag="div">
    <overview-widget v-for="(widget, index) in widgets" :key="widget.id" :style="getStaggerStyle(index)" />
</TransitionGroup>

<router-view v-slot="{ Component }">
    <Transition name="ebk-route">
        <component :is="Component" />
    </Transition>
</router-view>
```

`getStaggerStyle(index)` (from `src/lib/motion.ts`) returns `{ '--ebk-motion-index': '<0..5>' }`.

### CSS classes (no `<Transition>` needed)

| Class | Effect |
| --- | --- |
| `.ebk-animate-fade-in` | One-shot fade in on mount (`sm`) |
| `.ebk-animate-rise-in` | One-shot fade plus 12px rise on mount (`md`) |
| `.ebk-animate-reveal` | One-shot staggered reveal, uses `--ebk-motion-index` |
| `.ebk-press-feedback` | Scales to .98 while pressed (`xs`) |
| `.ebk-selection-indicator` | Transitions the indicator's `transform` / `opacity` (`sm`) |
| `.ebk-edge-illumination` + `--active` | Copper line on the leading edge that fades and grows in when active. RTL-aware, hidden in forced-colors mode |
| `.ebk-motion-exempt` | Opts a loading indicator out of the reduced motion collapse (Vuetify progress, Framework7 preloader and progressbar are already exempt) |

## Interruption and cancellation

- **A new navigation cancels the running one.** Vue `<Transition>` cancels an in-progress enter when its element leaves. Don't use `mode="out-in"` for routes: it makes the new route wait for the old one to leave. Use the enter-only `ebk-route` preset instead, which removes the old route at once so the new one is interactive right away.
- **Input is never blocked.** Transitions never set `pointer-events: none` or `inert` on content that's entering. Focus moves to the new content right away, and scroll position and form state are kept, because the transitions are pure transform/opacity changes on existing elements.
- **Saves never wait for animation.** Start requests (save, delete, reorder) when the user acts, never in `after-leave` / `transitionend` callbacks. Close dialogs and sheets in parallel with the request, or after it, depending on the existing UX. Don't chain a request behind a close animation.
- **Repeated actions:** a second press during an exit restarts from the current state. Durations are short (`sm` for exits), so the controls are reachable again within 160ms, or 120ms with reduced motion.
- **Hidden content:** keyframe classes run once. Ambient or looping animation must pause when the element is hidden (`v-if` / `display: none` / `visibilitychange`).
- **Framework transitions:** Framework7 page and modal transitions and Vuetify dialog and menu transitions keep their own cancellation behavior. Framework7 router transitions are not queued: a new navigation while one is animating is ignored by Framework7, so don't retry it automatically.

## Reduced motion

The effective state is `reduce`, or `system` while `prefers-reduced-motion: reduce` matches. `src/lib/motion.ts` exposes it reactively: `isReducedMotion()` is reactive when called inside `computed`, watchers, or render functions, and `useReducedMotion()` returns the computed ref. It updates when the setting changes and when the system preference changes.

How it is applied:

| Layer | Mechanism |
| --- | --- |
| Root classes | `<html>` gets `ebk-reduce-motion` when the state is reduced, and `ebk-motion-full` when the user chose **Full**. The pre-paint script in `src/*.html` sets them before the first paint, then `initMotionPreference()` (called in `desktop-main.ts` and `mobile-main.ts`) keeps them in sync |
| Tokens | Under the class (or the `prefers-reduced-motion` media query unless `ebk-motion-full` is set), durations drop to 120ms or less, distances become 0, scale-from becomes 1, and stagger becomes 0 |
| ezbookkeeping2 presets | Become opacity-only, linear, 120ms, with no delay |
| Other CSS transitions and animations | Collapse to 1ms with one iteration (finished at once, but `transitionend` / `animationend` still fire, so the frameworks don't hang). Loading indicators are exempt |
| Vuetify | Ripples are hidden. `VDialog`, `VMenu`, `VSelect` and `VAutocomplete` default to the CSS `fade-transition` instead of the Web Animations dialog transition (Vuetify global defaults, updated reactively in `desktop-main.ts`) |
| Framework7 | `animate: false` for views/router, actions, dialog, popover, popup and sheet, both in the startup parameters and at runtime (`src/MobileApp.vue`). Modals that already exist are covered by the CSS collapse |
| ECharts | `getChartAnimationOptions(enabled)` returns `animation: false` when reduced. Otherwise it returns entrance 360ms and update 240ms with `cubicOut`. All desktop chart components spread it into their series options. User-defined custom charts get `animation: false` |

The legacy mobile **Enable Animation** switch still turns Framework7 animations off on its own.

## Checklist for new motion

1. Use a preset or a token. Never hard-code durations or easings.
2. Only `transform` and `opacity`, and use logical directions for horizontal movement so RTL mirrors it.
3. Check the reduced-motion result: it should still communicate the change with an opacity change of 120ms or less, or with no motion at all.
4. Check that nothing waits on the animation: navigation, saves, focus and scroll.

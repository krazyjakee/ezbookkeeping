# Design tokens

This page documents the shared design tokens introduced by #25 for the "Copper Instrument Panel" direction (see [README.md](README.md)). Desktop (Vuetify) and mobile (Framework7) both use the same tokens. Each framework keeps its own components and behavior.

- **Source:** `src/styles/common/_tokens.scss` (tokens and global wiring) and `src/styles/common/_motion.scss` (motion primitives, see [motion.md](motion.md)).
- **Entry points:** both `src/styles/desktop/index.scss` and `src/styles/mobile/index.scss` load the two files first (`@use '../common/tokens'; @use '../common/motion';`).
- **Naming:** every token is a CSS custom property with the `--ebk-` prefix. Colors come in two forms: a ready-to-use color (`--ebk-color-surface`) and an RGB triplet (`--ebk-color-surface-rgb`) that you can combine with alpha (`rgba(var(--ebk-color-surface-rgb), 0.8)`).

## Theme scopes

Theme-dependent tokens are declared on every theme scope. This matters because a custom property that references another custom property is resolved where it is declared, not where it is used.

| Theme | Selectors |
| --- | --- |
| Light (default) | `:root`, `.v-theme--light` |
| Dark | `:root[data-ebk-theme="dark"]`, `:root.dark` (Framework7), `.v-theme--dark` (Vuetify) |

`data-ebk-theme` is set on `<html>` before the first paint (see "No theme flash" below). It is kept in sync by `src/DesktopApp.vue`, which watches the Vuetify theme, and by `src/MobileApp.vue` (`setThemeColorMeta`, which runs on every Framework7 `darkModeChange`).

## Token reference

### Color

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--ebk-copper-50/100/300/500/600/700/900` | ramp | same | Copper ramp. Tints are for selected backgrounds |
| `--ebk-color-primary` | `#C67E48` | same | Brand / primary (unchanged) |
| `--ebk-color-primary-hover` / `-pressed` | copper 600 / 700 | same | Hover and pressed states of primary actions |
| `--ebk-color-primary-text` | copper 700 | copper 500 | Copper text that must pass AA |
| `--ebk-color-canvas` | `#F7F4EF` | `#0B0A09` | Page background |
| `--ebk-color-surface` | `#FFFFFF` | `#161412` | Panels, cards, dialogs, sheets |
| `--ebk-color-surface-raised` | `#FFFFFF` | `#1E1B18` | Menus, popovers, sticky headers |
| `--ebk-color-surface-sunken` | `#F1ECE5` | `#100E0C` | Inputs, table headers, wells |
| `--ebk-color-text-primary` | `#2B2421` | `#F4EBE1` | Primary text |
| `--ebk-color-text-secondary` | `#6B605A` | `#B9ADA2` | Secondary text (≥ 4.5:1 on surface) |
| `--ebk-color-hairline` | `rgba(54,47,42,.14)` | `rgba(236,226,217,.12)` | Panel and divider borders |
| `--ebk-color-copper-line` | `rgba(198,126,72,.45)` | `rgba(198,126,72,.55)` | Active edge, corner brackets |
| `--ebk-color-selected-background` | copper at 10% | copper at 16% | Selected rows and items |
| `--ebk-color-success/info/warning/error` | `#4CD964` / `#2196F3` / `#FF9500` / `#FF3B30` | same | Semantic status colors (unchanged) |

The theme never remaps the financial colors. The income and expense amount colors (`src/styles/*/amount-color.scss`, user preference), the category, account and tag colors chosen by users, and the chart color schemes (`src/consts/color.ts`) are all rendered as they are.

### Focus, glow and elevation

| Token | Value |
| --- | --- |
| `--ebk-focus-ring-color` | copper 600 (light), copper 500 (dark). At least 3:1 against canvas and surface |
| `--ebk-focus-ring-width` / `--ebk-focus-ring-offset` | `2px` / `2px` |
| `--ebk-focus-outline` | `2px solid var(--ebk-focus-ring-color)` |
| `--ebk-glow` | `0 0 0 1px copper-line, 0 0 12px rgba(copper, .25)` |
| `--ebk-shadow-sm/md/lg` (= `--ebk-elevation-1/2/3`) | Theme-aware shadows. Dark theme relies more on surface steps |

### Spacing, radius, density and layers

| Token | Value |
| --- | --- |
| `--ebk-space-0…6` | `0`, `.25rem`, `.5rem`, `.75rem`, `1rem`, `1.25rem`, `1.5rem` (4px scale) |
| `--ebk-radius-sm/md/lg/xl` | `.375rem` / `.625rem` / `.75rem` / `1rem` |
| `--ebk-radius-panel` / `-control` / `-pill` | `10px` (= md) / `8px` / `999px` |
| `--ebk-density-table-row-height` | `44px` (desktop transaction tables, maximum) |
| `--ebk-touch-target-min` | `44px` (mobile) |
| `--ebk-header-strip-height` | `40px` desktop, `36px` mobile (overridden in `src/styles/mobile/_variable.scss`) |
| `--ebk-bracket-size` / `--ebk-grid-size` / `--ebk-grid-opacity` | `10px` / `24px` / `.035` |
| `--ebk-z-base/raised/sticky/navigation/overlay/snackbar/tooltip/toast` | `0` / `1` / `5` / `1000` / `2400` / `9000` / `9000` / `10000` |

### Typography

| Token | Value |
| --- | --- |
| `--ebk-font-body` | Inter, then system UI fonts, then CJK (PingFang, Hiragino, YaHei, Noto Sans CJK, Malgun Gothic), Indic (Nirmala UI, Noto Sans Devanagari/Bengali/Tamil/Kannada), and Thai (Leelawadee UI, Thonburi, Noto Sans Thai) fallbacks. No web fonts are downloaded |
| `--ebk-font-mono` | `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace` |
| `--ebk-font-numeric` | `tabular-nums` (value for `font-variant-numeric`) |
| `--ebk-font-technical-size/weight/letter-spacing` | `.75rem` / `600` / `.08em` |

Desktop maps `--v-font-body` to `--ebk-font-body`. Mobile keeps the native Framework7 iOS system font stack.

### Motion

See [motion.md](motion.md). The tokens are `--ebk-motion-duration-xs/sm/md/lg/reduced`, `--ebk-motion-ease-standard/emphasized/exit`, `--ebk-motion-distance-sm/md`, `--ebk-motion-scale-from`, `--ebk-motion-stagger` and `--ebk-motion-stagger-max-index`.

### Legacy aliases

Existing names keep working. They are now defined in the shared tokens.

| Legacy | Now |
| --- | --- |
| `--ebk-primary-color` | RGB triplet of copper 500 |
| `--ebk-surface` / `--ebk-surface-muted` / `--ebk-surface-raised` | `--ebk-color-surface-rgb` / `--ebk-color-surface-sunken-rgb` / `--ebk-color-surface-raised-rgb` |
| `--ebk-border` | `--ebk-color-hairline-rgb` (the components choose the alpha) |
| `--ebk-shadow-color`, `--ebk-shadow-sm/md/lg` | Theme-aware shadow tokens |
| `--ebk-focus-ring` | The soft 3px copper halo (unchanged) |
| `--ebk-transition-fast` / `--ebk-transition-base` | `duration-sm` / `duration-md` with `ease-standard` |
| `--ebk-auth-background` (desktop) | `--ebk-color-canvas-rgb` |

## Framework adapters

### Vuetify (desktop)

`src/desktop-main.ts` maps the tokens into the light and dark Vuetify themes:

| Vuetify color / variable | Light | Dark |
| --- | --- | --- |
| `background` / `on-background` | canvas / text primary | canvas / text primary |
| `surface` / `on-surface` | surface / text primary | surface / text primary |
| `surface-raised`, `surface-sunken` (+ `on-*`) | raised / sunken | raised / sunken |
| `primary`, `primary-darken-1` | `#c67e48`, copper 600 | same |
| `success`, `info`, `warning`, `error` | unchanged | unchanged |
| `border-color` / `border-opacity` | hairline RGB / `.14` | hairline RGB / `.12` |
| `table-header-background` | surface sunken | surface sunken |

The `bg-surface-raised`, `bg-surface-sunken`, `text-surface-raised` and similar helper classes are generated by Vuetify. `src/styles/desktop/_variable.scss` keeps only the desktop-specific variables (layout sizes, Vuetify component variables). `<html>` paints `--ebk-color-canvas` before Vuetify mounts.

### Framework7 (mobile)

`src/styles/mobile/_variable.scss` maps the tokens into the Framework7 iOS theme variables on `:root.ios` and `:root.ios.dark`. Listing `:root.ios.dark` lets these values win over Framework7's own `.ios.dark` rules.

| Framework7 variable | Token |
| --- | --- |
| `--f7-page-bg-color` | canvas |
| `--f7-bars-bg-color` (+ `-rgb`) | canvas (50% alpha in dark, which keeps the translucent bars) |
| `--f7-block-strong-bg-color`, `--f7-list-strong-bg-color` | surface |
| `--f7-list-group-title-bg-color` | surface sunken |
| `--f7-text-color` | text primary |
| `--f7-block-title-text-color`, `--f7-block-footer-text-color`, `--f7-list-item-footer-text-color`, `--f7-list-item-after-text-color` | text secondary |
| `--f7-list-outline-border-color` | hairline |
| `--f7-theme-color` | `#c67e48`, from the `colors.primary` app parameter in `src/MobileApp.vue` |

## No theme flash

`src/desktop.html` and `src/mobile.html` each run a small inline script before the first paint. The script reads the `ebk_app_settings` local storage entry (the same key `src/lib/settings.ts` uses) and then:

1. resolves `theme` (`light`, `dark`, or `auto`, which follows `prefers-color-scheme`) and sets `data-ebk-theme` on `<html>`. On mobile it also sets the `dark` class that Framework7 uses;
2. resolves `motionPreference` and sets `ebk-reduce-motion` / `ebk-motion-full` on `<html>` (see [motion.md](motion.md));
3. sets `<meta name="theme-color">` to the dark canvas color when the theme is dark.

The token stylesheet sets `color-scheme` and the canvas background on `<html>`, so the first frame already uses the correct theme. Afterwards the apps take over: Vuetify `theme.change()` plus a watcher on desktop, and Framework7 `darkMode` on mobile. Both follow system theme changes while the theme setting is `auto`.

## Global wiring

The token stylesheet also adds these app-wide rules. Pages and components get no other restyling here. That work belongs to #27 and later issues.

- **Focus ring:** `:focus-visible` shows a 2px copper outline at a 2px offset. Text inputs (the framework field styles their focus), `contenteditable`, and programmatic focus targets (`tabindex="-1"`) are excluded. In forced-colors mode the ring uses `Highlight`.
- **Tabular numerals:** `font-variant-numeric: var(--ebk-font-numeric)` applies to `.ebk-numeric`, `.text-income`, `.text-expense`, and any element whose class contains `amount` or `balance` (for example `.transaction-amount`, `.account-balance`, `.overview-widget-primary-amount`). For other numeric columns such as dates and counters, add `.ebk-numeric`.
- **Canvas color:** `<html>` (and on desktop the Vuetify `background`) uses `--ebk-color-canvas`.

## Utility classes

| Class | What it does |
| --- | --- |
| `.ebk-canvas-grid` | Canvas color with a 24px grid at 3.5%. Removed in forced-colors mode and when printing. Use it on the page canvas only, never under dense tables |
| `.ebk-panel` | Surface panel: 1px hairline border, 10px radius, elevation 1 |
| `.ebk-panel--active` / `.ebk-panel--selected` | Copper line on the leading (inline-start) edge, mirrored in RTL |
| `.ebk-panel--bracketed` | Copper L-brackets at the start-start and end-end corners via `::before` / `::after`. They use logical properties, so RTL is mirrored. Hidden in forced-colors mode. Use only on overview widgets and dialog headers, and don't put `overflow: hidden` on the panel |
| `.ebk-panel-header-strip` | 40px (desktop) or 36px (mobile) header strip: technical type, hairline underneath, `.ebk-panel-header-actions` pushed to the end |
| `.ebk-technical-heading` | Technical heading type only (0.75rem, 600, `.08em`). Uppercase only for Latin-script languages via `:lang()` (en, de, es, fr, it, nl, pl, pt, ro, sk, sl, tr, vi) |
| `.ebk-glow` / `.ebk-glow-focus` | Copper glow, always on or only while focused. Only for focused controls, primary button hover, the active navigation item and live indicators |
| `.ebk-numeric` | Tabular numerals |
| `.ebk-font-mono` | Monospace stack for IDs, tokens and code |

## Examples

```html
<!-- overview widget -->
<section class="ebk-panel ebk-panel--bracketed">
    <header class="ebk-panel-header-strip">
        <span>{{ tt('Net Assets') }}</span>
        <div class="ebk-panel-header-actions"><v-btn icon variant="text" ... /></div>
    </header>
    <div class="pa-4"><span class="ebk-numeric">{{ displayAmount }}</span></div>
</section>
```

```scss
// component styles: always use tokens instead of hard-coded colors
.my-list-row {
    min-height: var(--ebk-density-table-row-height);
    border-block-end: 1px solid var(--ebk-color-hairline);

    &.selected {
        background-color: var(--ebk-color-selected-background);
        border-inline-start: 1px solid var(--ebk-color-copper-line);
    }
}
```

```ts
// vuetify custom theme colors generated from the tokens
<v-sheet color="surface-sunken">...</v-sheet>
```

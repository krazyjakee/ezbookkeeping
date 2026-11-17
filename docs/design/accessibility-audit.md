# Accessibility, localization and reduced motion audit (#36)

This audit checks the finished redesign (#24–#35) against [README.md](README.md) §7 and WCAG 2.2 AA. It covers the desktop app, the mobile app and the documentation site. Every finding below is either fixed in this change or explicitly justified. Checks that need a person, a real screen reader or a physical device are listed under [Not covered](#not-covered).

- **Date:** 2026-11-17
- **Code:** `main` after #35 (commit `2e6a6e07`) plus the fixes listed here
- **Evidence:** scripts, JSON results, logs and screenshots are kept outside the repository in `/tmp/ezb-a11y/` (`scan.mjs`, `keyboard.mjs`, `motion.mjs`, `visual.mjs`, `semantics.mjs`, `scan-docs.mjs`, `docs-kbd.mjs`, and `out/`). No images are committed.

## Contrast decision

White text on copper `#C67E48` reaches only 3.2:1. This fails WCAG AA (1.4.3) for normal text on the primary buttons, the mobile add key, selected segmented controls, date picker selections and every other copper fill.

**Decision:** copper fills keep brand copper `#C67E48`, which the milestone requires us to retain. Text and icons on copper fills use dark ink `#140F0D` (`--ebk-color-on-primary`) in both themes:

| On | Ink `#140F0D` | White |
| --- | --- | --- |
| Copper 500 `#C67E48` | 5.9:1 | 3.2:1 |
| Copper 600 `#B67443` (hover, pressed) | 5.0:1 | 3.8:1 |
| Copper 500 + Vuetify press overlay (14% ink) | 4.6:1 | n/a |

Hover and pressed fills stop at copper 600. Copper 700 under ink would drop to 3.4:1.

The ink is applied everywhere a copper fill carries text or icons:

- Vuetify `on-primary`
- Framework7 filled buttons and the active segment of plain segmented controls
- the mobile add key and the mobile Log In button
- the settings segmented controls and selected pill tabs
- vue-datepicker selections
- the documentation site brand buttons

The decision is recorded in [README.md](README.md) §3.1 and [tokens.md](tokens.md).

## Tested matrix

| Dimension | Values |
| --- | --- |
| Browser engine | Chromium: Chrome for Testing headless shell 156.0.8078.4, driven by playwright-core 1.65.1 |
| Libraries under test | Vue 3.5.44, Vuetify 4.3.1, Framework7 / framework7-vue 9.3.0, @vuepic/vue-datepicker 14.0.0, ECharts 6.1.0, VitePress 2.0.0-alpha.15 |
| Automated rules | axe-core 4.14.1, tags `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa`. A supplementary DOM contrast check computes the blended background for every text node, including the nodes axe reports as "incomplete" |
| App targets | Dev server `http://127.0.0.1:8081`, demo account. Desktop: 33 states (every route in §8 plus the add/detail transaction dialogs, import dialog, confirm dialog, calendar view). Mobile: 33 states (§8 routes plus the number pad sheet and the transaction list action sheet) |
| Docs target | `website/.vitepress/dist` served by `vitepress preview` on port 8096. 40 pages (all English pages plus the zh_Hans home, FAQ and installation pages). The current `custom.css` was injected over the existing build to verify the theme fixes; the build was not rerun |
| Themes | light and dark (`prefers-color-scheme` emulation, app theme setting "auto"); docs `vitepress-theme-appearance` light and dark |
| Viewports | desktop 1440×900, 1024×768, 640×400 (1280×800 at 200% zoom), 320×800 (400% zoom reflow); mobile 390×844 at DPR 2, 320×640 |
| Text size | mobile app font size `xxx-large` (5) and `xxxx-large` (6), also at 320 px wide |
| Emulations | `prefers-reduced-motion: reduce`; in-app Settings → Preferences → Motion = Reduce (`motionPreference: 'reduce'`); `forced-colors: active` (light and dark); forced `dir="rtl"`; offline (`context.setOffline` plus the `offline` event) |
| Languages | en, de, ru, ta, bn, applied as the user language through the token refresh and profile responses. RTL layout was checked with forced `dir="rtl"` because the build ships no RTL language |
| Input | scripted keyboard: Tab, Shift+Tab, Enter, Space, Escape, arrow keys |

## Results summary

| Pass | Desktop + mobile axe violations (nodes) | Notes |
| --- | --- | --- |
| Baseline | 2,000+ across 13 rules | color-contrast, label, meta-viewport, link-name, aria-allowed-attr, aria-prohibited-attr, aria-required-children, nested-interactive, list, listitem, link-in-text-block, label-content-name-mismatch, aria-tooltip-name |
| Final (pass 5) | aria-tooltip-name 922, label-content-name-mismatch 60, color-contrast 6 | All three are justified below (J1, J2, J3). The supplementary contrast check only reports skeleton placeholders, icon glyphs and color swatches (J3, J4) |
| Docs baseline | nested-interactive on all pages, link-name 2, color-contrast up to 136 per page | |
| Docs final | nested-interactive (VitePress sidebar, J5) | Contrast and image alt text fixed |

Keyboard (final): every focus stop has a visible indicator on desktop login, the shell, the add transaction dialog, filters, the confirm dialog and the mobile shell. The skip link is the first Tab stop. Arrow keys move through the rail. The add dialog traps focus (40 Tab presses stayed inside). Table filter menus open with Enter, arrow keys move through the options and Enter applies one. Closing the confirm and transaction dialogs returns focus to the control that opened them. Hidden mobile pages are out of the tab order.

Findings by severity (axe impact scale):

| Severity | Fixed | Justified or tracked |
| --- | --- | --- |
| Serious | 16 | 0 |
| Moderate | 20 | 1 tracked (T1) |
| Minor | 3 | 7 justified (J1–J7), 1 tracked (T2) |

## Findings

WCAG references are WCAG 2.2 success criteria. File paths are relative to the repository root.

### Contrast and use of color

| ID | Page / surface | Issue | WCAG | Severity | Status and files |
| --- | --- | --- | --- | --- | --- |
| A01 | All copper fills, desktop and mobile | White text on copper, 3.2:1 | 1.4.3 | Serious | Fixed: ink on copper. `src/styles/common/_tokens.scss` (`--ebk-color-on-primary`, pressed = copper 600), `src/desktop-main.ts` (`on-primary`), `src/styles/desktop/override/_vuetify.scss`, `src/styles/desktop/components/_setting-items.scss`, `src/styles/desktop/override/_vue-datepicker.scss`, `src/styles/mobile/override/_vue-datepicker.scss`, `src/styles/mobile/override/_framework7-instrument.scss`, `src/styles/mobile/override/_framework7-shell.scss`, `src/styles/mobile/components/_auth.scss`, `website/.vitepress/theme/custom.css` |
| A02 | Desktop links and `text-primary`; mobile links, text buttons, floating Save key, tonal Add/Retry buttons; signup link | Copper 500 text on light surfaces, 3.2:1 (2.5:1 on the tonal tint) | 1.4.3 | Serious | Fixed: copper 700 for copper text in light. `src/styles/desktop/base/_global.scss`, `src/styles/desktop/override/_vuetify.scss`, `src/styles/mobile/override/_framework7-instrument.scss` (Framework7 theme color copper 700 in light, tonal label copper 900) |
| A03 | Every income/expense amount (overview, lists, tables, statistics, calendar, amount field label) | User amount colors fail: green 3.1–3.7:1, red 3.9–4.2:1 (4.0:1 in dark), yellow 1.9:1 | 1.4.3 | Serious | Fixed: per-theme AA text shade of the same hue. `src/styles/desktop/amount-color.scss`, `src/styles/mobile/amount-color.scss`, `src/components/desktop/AmountInput.vue` (label at full opacity); documented in `docs/design/tokens.md` |
| A04 | Mobile destructive list buttons, danger section titles; desktop danger panels, field errors, import issue count | Error red `#FF3B30` as text, 3.2–3.5:1 | 1.4.3 | Serious | Fixed: `--ebk-color-error-text` (`#C4161C` in light). `_tokens.scss`, `_vuetify.scss`, `src/styles/desktop/components/_management.scss`, `src/views/desktop/transactions/import/tabs/ImportTransactionCheckDataTab.vue`, `_framework7-instrument.scss`, `src/styles/mobile/components/_settings.scss` |
| A05 | Desktop secondary (Cancel, Clear) buttons; mobile list descriptions, sessions, group titles, transaction day and weekday, statistics percentages, hidden-item dimming | Gray text 3.3–4.1:1 | 1.4.3 | Serious | Fixed. `src/desktop-main.ts` (`secondary` `#6b605a`), `src/styles/mobile/_variable.scss`, `src/views/mobile/transactions/ListPage.vue`, `src/styles/mobile/components/_statistics-list.scss`, `src/styles/mobile/components/_management.scss` (dimming 0.7) |
| A06 | Mobile statistics pie legend | Value and percentage drawn in the series color, 4.1–4.4:1 | 1.4.3 | Moderate | Fixed: text tokens, color kept as a swatch and chip border. `src/components/mobile/PieChart.vue` |
| A07 | Desktop calendar (overview, transaction calendar) | Selected or today day number in copper on a copper tint, 3.0:1 | 1.4.3 | Moderate | Fixed. `src/styles/desktop/components/_transaction-calendar.scss` |
| A08 | Docs home, code blocks, dark inline code | Hero wordmark 2.95:1 (large text); Shiki tokens `#22863A`, `#6A737D`, `#D73A49`, `#E36209` at 3.0–4.1:1 on the code well; dark inline code at 4.5:1 | 1.4.3 | Moderate | Fixed. `website/.vitepress/theme/custom.css` |
| A09 | Mobile disclosure chevrons, picker chevrons, page-title dropdown icons, category separator | Icons at 1.6–2.7:1 | 1.4.11 | Moderate | Fixed: chevron color at 50% black or white (at least 3.3:1), opacity removed. `src/styles/mobile/_variable.scss`, `src/views/mobile/accounts/ListPage.vue`, `src/views/mobile/statistics/TransactionPage.vue`, `src/views/mobile/tags/ListPage.vue`, `src/views/mobile/transactions/ListPage.vue`, `src/views/mobile/accounts/ReconciliationStatementPage.vue`, `src/views/mobile/transactions/EditPage.vue` |
| A10 | Desktop transaction calendar view, mobile calendar | Daily income and expense amounts and dots told apart by color only. The ↑/↓ markers were only on the overview widget | 1.4.1 | Moderate | Fixed: ↑/↓ markers for every calendar, filled dot for income and ring for expense. `src/components/common/TransactionCalendar.vue` |
| A11 | Desktop signup | "Click here to log in" link identified by color only (1.5–1.9:1 against the surrounding text) | 1.4.1 | Moderate | Fixed: links in running text are underlined. `src/styles/desktop/base/_global.scss` |
| A12 | Mobile filled controls in forced colors | Filled buttons, the add key and Log In lose their background and have no boundary | 1.4.11 | Minor | Fixed: `ButtonText` border in forced colors. `_framework7-instrument.scss` |

### Keyboard, focus and dialogs

| ID | Page / surface | Issue | WCAG | Severity | Status and files |
| --- | --- | --- | --- | --- | --- |
| K01 | Desktop transaction table headers (Time, Category, Account, Tag, Amount filters) | Menu activators were `div`s: not focusable, and they exposed `aria-expanded` on a div | 2.1.1, 4.1.2 | Serious | Fixed: native buttons. `src/views/desktop/transactions/ListPage.vue`, `src/styles/desktop/components/_shared-controls.scss` |
| K02 | 18 desktop menu items (transaction list filters, statistics and explorer date ranges, the language selector, the transaction editor geolocation menu) | Click handlers on `v-list-item-title`, so Enter or Space on the focused item did nothing | 2.1.1 | Serious | Fixed: the handler moved to the list item. `src/views/desktop/transactions/ListPage.vue`, `src/views/desktop/statistics/TransactionPage.vue`, `src/views/desktop/insights/ExplorerPage.vue`, `src/views/desktop/transactions/list/dialogs/EditDialog.vue`, `src/components/desktop/LanguageSelectButton.vue` |
| K03 | Desktop insights explorer side panel | "New Exploration" and saved explorations were anchors without `href` or `tabindex`; invalid list structure (`div`, `hr` in `ul`) | 2.1.1, 1.3.1 | Serious | Fixed. `src/views/desktop/insights/ExplorerPage.vue` |
| K04 | Desktop account settings | Avatar menu opened from a `div` | 2.1.1 | Moderate | Fixed: button with a name, edit overlay also on focus. `src/views/desktop/users/UserBasicSettingPage.vue` |
| K05 | Mobile accounts | Sub-account expand toggle was not focusable, had no expanded state and a target under 24 px | 2.1.1, 4.1.2, 2.5.8 | Moderate | Fixed. `src/views/mobile/accounts/ListPage.vue` |
| K06 | Desktop confirm dialogs | Escape was ignored (persistent dialog) | 2.1.1 | Moderate | Fixed: Escape cancels. `src/components/desktop/ConfirmDialog.vue` |
| K07 | Desktop confirm, add/edit transaction and other dialogs opened from code | Focus fell to `body` after closing | 2.4.3 | Serious | Fixed: ConfirmDialog returns focus to its opener; `src/DesktopApp.vue` restores the last focused control outside overlays when a dialog closes and focus was lost. Verified: Cancel returns to "Add Transaction", Escape on a detail dialog returns to its table row |
| K08 | Mobile (every pushed page) | Pages and navbars kept behind the current page stayed in the tab order and accessibility tree | 2.4.3, 1.3.2 | Serious | Fixed: hidden stack pages and navbars are `inert`. `src/MobileApp.vue` |
| K09 | Desktop overview layout editor | `aria-label` on a non-interactive drag `div` | 4.1.2 | Minor | Fixed: hidden from assistive technology; move up/down buttons are the accessible alternative. `src/views/desktop/overview/OverviewDashboard.vue` |
| K10 | Desktop application settings sync | "More" button nested in the expansion panel title button; `hr` and nested lists as list children | 4.1.2, 1.3.1 | Moderate | Fixed: static header (the panel is always open and read-only), presentational dividers. `src/views/desktop/settings/ApplicationCloudSyncSettingsPage.vue` |

### Names, roles, states and live messages

| ID | Page / surface | Issue | WCAG | Severity | Status and files |
| --- | --- | --- | --- | --- | --- |
| N01 | Desktop Preferences (24), Account settings (22), Statistics settings (10) | Selects, switches and text fields had no accessible name (Vuetify pointed `aria-labelledby` at a label that does not exist) | 1.3.1, 4.1.2 | Serious | Fixed: named after the visible row label. `src/views/desktop/settings/PreferencesSettingsPage.vue`, `src/views/desktop/users/UserBasicSettingPage.vue`, `src/views/desktop/settings/StatisticsSettingPage.vue` |
| N02 | Desktop and mobile application lock / unlock | PIN digit inputs unlabelled | 1.3.1, 4.1.2 | Serious | Fixed: group name plus "PIN Code n/N" per digit. `src/components/common/PinCodeInput.vue` |
| N03 | Desktop exchange rates | Base amount input unlabelled (AmountInput dropped `id` and `aria-*` attributes) | 1.3.1 | Moderate | Fixed: attributes forwarded to the rendered field. `src/components/desktop/AmountInput.vue` |
| N04 | Mobile chart color settings | Color inputs unlabelled | 1.3.1 | Moderate | Fixed: visually hidden label "Color n: #hex". `src/views/mobile/settings/ChartColorSchemeSettingsPage.vue` |
| N05 | Mobile, every page with a back link; About | Icon-only back link without a name; empty placeholder link | 2.4.4, 4.1.2 | Serious | Fixed. `src/MobileApp.vue`, `src/views/mobile/AboutPage.vue` |
| N06 | Mobile | Disabled links and buttons (Framework7 `.disabled` class) exposed no disabled state | 4.1.2 | Moderate | Fixed: `aria-disabled` kept in sync. `src/MobileApp.vue` |
| N07 | Mobile | No main landmark and no page heading | 1.3.1, 2.4.6 | Moderate | Fixed: page content `role="main"`, navbar title heading level 1. `src/MobileApp.vue` |
| N08 | Mobile statistics | Pie chart previous/next arrows unnamed | 4.1.2 | Moderate | Fixed. `src/components/mobile/PieChart.vue` |
| N09 | Mobile accounts | `li` outside a list (sub-account wrapper) | 1.3.1 | Minor | Fixed (`role="none"`). `src/views/mobile/accounts/ListPage.vue` |
| N10 | Desktop dialogs | Vuetify sets `role="dialog"` on the overlay root but the label on the content, so dialogs had no name | 4.1.2 | Moderate | Fixed: the dialog is named by its title. `src/DesktopApp.vue`, `src/components/desktop/ConfirmDialog.vue`. Verified: "Add Transaction", `aria-modal="true"` |
| N11 | Desktop date and time fields | Calendar icon announced as the raw key "input.appendAction"; field exposed as a menu button with `aria-owns` pointing at an unrendered menu | 4.1.2, 3.1.2 | Moderate | Fixed: Vuetify keys mapped (`Open`), menu semantics removed from the anchor field. `src/desktop-main.ts`, `src/components/desktop/DateTimeSelect.vue` |
| N12 | Desktop calendars and date pickers | Day cells and month navigation labelled in English for every language; cells with amounts did not include them in the name | 3.1.2, 2.5.3 | Moderate | Fixed: localized long date plus the daily income and expense; Previous/Next Period. `src/components/common/TransactionCalendar.vue`, `src/components/common/DateTimePicker.vue`. Residual axe flag: J2 |
| N13 | Mobile toasts ("Settings saved", errors) and notifications | Framework7 toasts have no live region, so save results were not announced | 4.1.3 | Serious | Fixed: the text is mirrored into a polite status region. `src/MobileApp.vue`. Verified: desktop snackbars are `role="status"`, the mobile offline banner is `role="status"`, and the mobile live region carries "Settings saved" |
| N14 | Desktop overview chart widgets (income & expense trend, net assets trend, calendar heatmap) | Canvas without a name or description | 1.1.1 | Moderate | Fixed: the chart body is an image named by the panel title and described by the footer summary. `src/views/desktop/overview/widgets/OverviewWidgetPanel.vue` and the three widgets. A full data table on the overview is tracked as T1 |
| N15 | Docs export and import guide (en, zh_Hans) | 14 linked screenshots with empty `alt`, so the links had no name | 1.1.1, 2.4.4 | Serious | Fixed: alt text from the section heading. `website/docs/export_and_import/index.md`, `website/docs/zh_Hans/export_and_import/index.md` |

### Zoom, reflow, text size and localization

| ID | Page / surface | Issue | WCAG | Severity | Status and files |
| --- | --- | --- | --- | --- | --- |
| Z01 | `desktop.html`, `mobile.html`, `index.html` | `maximum-scale=1`, `user-scalable=no` blocked pinch zoom | 1.4.4 | Serious | Fixed. `src/desktop.html`, `src/mobile.html`, `src/index.html` |
| Z02 | Desktop at 320 CSS px (overview, transactions, statistics, preferences, accounts) | 24–41 px horizontal scroll: header actions, panel header actions, transaction search; calendar grid cut off | 1.4.10 | Moderate | Fixed: the header and panel header strips wrap, and the calendar grid scrolls inside its panel (two-dimensional content). `src/styles/desktop/layout/_layout.scss`, `src/styles/common/_tokens.scss`, `src/styles/desktop/components/_overview-widget.scss`. Final: 0 px overflow and no clipped text at 320 px and 200% on all checked pages |
| Z03 | All checked pages in de, ru, ta, bn; mobile xxx-large and xxxx-large; forced RTL | Long translations and large text | 1.4.4, 1.4.10, 1.4.12 | n/a | Pass: no horizontal overflow and no clipped text on the desktop shell, overview, transactions and add dialog (1440 and 1024 wide) or the mobile home and add pages (390, and 320 at xxx-large) |
| Z04 | Locale numbers | Amount formats | 3.1 | n/a | Pass: de `−5.684,70 £`, ru `−5 684,70 £`, ta and bn `−£ 5,684.70` (the configured defaults of those languages), with localized month names and dates |

### Motion, flashing and forced colors

| Check | Result |
| --- | --- |
| System `prefers-reduced-motion: reduce` | Pass. Desktop overview load, add dialog and mobile home → statistics: no animation or transition longer than 120 ms except loading spinners (allowed by motion.md) |
| In-app Motion = Reduce | Pass, same measurements |
| No preference (control) | Overview reveals 240 ms, route 360 ms, Framework7 page transitions 400 ms; all are stopped by both reduce modes |
| Charts | Every desktop ECharts component uses `getChartAnimationOptions()` or turns animation off when motion is reduced (`CustomChart.vue`). Mobile charts are SVG/CSS without animation. Frame comparison of the statistics canvas showed no change between 40 ms, 200 ms and 2 s in reduced modes |
| Flashing | Pass: no infinite animations outside progress indicators, and nothing flashes more than 3 times per second |
| Forced colors (light and dark) | Pass: canvas grid, corner brackets and start ticks are dropped; desktop primary buttons get a system border; mobile filled controls fixed in A12; focus ring uses `Highlight` |
| Docs | Reduced motion: no animation over 120 ms. Forced colors: grid dropped. The skip link is the first Tab stop, and every Tab stop has a visible ring |

### Justified and tracked items

| ID | Page / surface | Item | WCAG | Severity | Status |
| --- | --- | --- | --- | --- | --- |
| J1 | Desktop, all pages (922 nodes) | axe `aria-tooltip-name`: Vuetify 4.3.1 keeps closed tooltip overlays (`role="tooltip"`) in the DOM with their content `display: none` | 4.1.2 | Minor | Justified (third party). Each tooltip has its text when shown, and every activator has its own accessible name |
| J2 | Desktop overview calendar and transaction calendar (10 cells per month) | axe `label-content-name-mismatch` on day cells with amounts. The name contains the localized date and the daily amounts, but in a different order than the visible "1 ↑£ 2,344.95" (the arrow is decorative CSS) | 2.5.3 | Minor | Justified: the vue-datepicker grid cell owns the name; the visible day number and amount are both in it |
| J3 | Mobile home, statistics | axe color-contrast on chevron glyphs (3.1:1, 3.9:1): Framework7 icon-font glyphs are non-text and pass 3:1 | 1.4.11 | Minor | Justified (false positive: axe treats icon ligatures as text) |
| J4 | Mobile chart color settings; skeletons behind the mobile login | Color swatches (the swatch is the value; the hex is in the label); skeleton placeholder text while loading | 1.4.3 | Minor | Justified |
| J5 | Docs, every page | axe `nested-interactive`: VitePress collapsible sidebar groups (`role="button"`) contain the group link | 4.1.2 | Minor | Justified (third party, VitePress 2.0.0-alpha.15 default theme) |
| J6 | Desktop add/edit transaction dialog | Escape does not close the dialog while it is `persistent` (loading, saving or unsaved changes). This protects data; Cancel and the close button are in the tab order, and focus is trapped and restored | 2.1.1 | Minor | Justified (existing behavior) |
| J7 | Mobile shell | The Framework7 tab bar comes before the navbar and content in the tab order | 2.4.3 | Minor | Justified: Framework7 structure, consistent on every page |
| T1 | Desktop overview chart widgets | D7 asks for a data table or ranked list with exact values beside every chart. The overview trend and heatmap widgets give the panel name, the latest values and a link to Statistics, which has the full table and summary | 1.1.1 | Moderate | Tracked: adding tables to overview widgets is feature work beyond this audit |
| T2 | de, ru, ta | `format.misc.batchTransactionCount` is still English ("{count} transactions") in de.json, ru.json and ta.json | 3.1.2 | Minor | Tracked for translators (locale content is out of scope here) |

Observation: the desktop app has no global offline banner. Its data panels show an offline state, and the mobile app shows the banner (`role="status"`). README §5.2 only requires the banner on mobile.

## Not covered

These could not be done in this environment (headless Chromium on Linux, no assistive technology, no devices). A person should run them before the milestone closes:

1. **Screen readers.** Do the full bookkeeping workflows with each pairing below:

   | Screen reader | Browser | Surface |
   | --- | --- | --- |
   | NVDA 2025.x | Firefox and Chrome on Windows | Desktop app |
   | JAWS | Chrome | Desktop app |
   | VoiceOver | Safari on macOS | Desktop app and docs |
   | VoiceOver | Safari on iOS 18+ | Mobile app, Safari and home-screen PWA |
   | TalkBack | Chrome on Android 14+ | Mobile app |

   The workflows:
   - log in, including the 2FA and unlock PIN
   - add, edit and delete a transaction, including a validation error (empty amount) and recovery
   - filter the transaction list (table header menus, side panel)
   - open statistics and read the chart summary and table
   - change a preference and hear "Settings saved"
   - import a file up to the check step
   - go offline and hear the banner (mobile)
   - check the confirm dialogs and focus restoration after every dialog and sheet

   Watch for announcements of `inert` pages, `aria-disabled` links, the toast live region and the localized calendar cell names.
2. **Windows High Contrast / Contrast themes.** Check the desktop app in Edge and the mobile app in Chrome on Windows. Only Chromium `forced-colors` emulation was used here.
3. **Physical devices.** Check iOS and Android at the largest system text size and with OS display zoom. Check pinch zoom now that the viewport no longer blocks it. Check that inputs under 16 px do not auto-zoom on iOS after the viewport change (Framework7 iOS inputs are 17 px), and that the on-screen keyboard keeps focused inputs visible.
4. **Touch targets on device.** Measure targets with a real finger. Automated checks only covered axe `target-size` (24 px).
5. **Real browser zoom.** Use the browser's 200% and 400% zoom and Firefox's "zoom text only". Here, zoom was approximated with CSS-pixel viewports.
6. **Other engines.** Run the axe scan in Firefox (Gecko) and Safari (WebKit). Only Chromium was used.
7. **A real RTL language.** No RTL locale ships, so `dir="rtl"` was forced. Re-test when an RTL translation is added.
8. **Docs rebuild.** Rebuild the documentation site (`website`, `npm run docs:build`) and rerun `scan-docs.mjs` without CSS injection. This confirms the theme and alt text fixes in the published output.
9. **Cognitive and content review.** A person should review plain-language error messages and translation quality, especially for T2.

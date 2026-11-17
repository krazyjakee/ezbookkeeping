# ezrbookkeeping design guidelines: "Copper Instrument Panel"

**Version 1.0. Binding for the redesign milestone (#24–#37) and all later UI work.**

This document is the single source of truth for how ezrbookkeeping looks and behaves. Companion references give exact values and must agree with it:

- [tokens.md](tokens.md): design tokens and framework adapters (`src/styles/common/_tokens.scss`)
- [motion.md](motion.md): motion tokens, transitions and the reduced-motion policy (`src/styles/common/_motion.scss`, `src/lib/motion.ts`)
- [components.md](components.md): state tables for shared controls

If a companion document or the code disagrees with this one, this document wins: fix the other.

---

## 1. Principles

1. **A precision instrument for money.** Calm, technical and trustworthy. It is never a game UI: no neon, holograms, scanlines, HUD clutter or continuous ambient animation.
2. **Figures first.** Amounts, dates, accounts and statuses are the most prominent things on screen. Decoration always sits a step behind content.
3. **Copper is a signal.** Brand copper `#C67E48` marks the active location, selection, focus, the primary action and live state. Large surfaces stay warm neutral.
4. **Framed modules.** Content lives in panels with hairline borders and a labelled header strip, like modules in a console.
5. **Presentation only.** The redesign changes how things look and move, not what the app does. It adds no features, settings, data or business rules beyond what these guidelines name explicitly.
6. **One language, two frameworks.** Desktop (Vuetify) and mobile (Framework7) share tokens, rules and vocabulary. Each keeps its framework's native behavior: gestures, routing and component APIs.

## 2. Decisions

These resolve open questions from the concept exploration and are final.

| # | Decision |
| --- | --- |
| D1 | **Amounts are signed and labelled, as well as coloured.** In transaction lists, calendars, reports, rankings and overview tiles, income is shown as `+£8,017.20` and expense as `−£5,684.70` (U+2212 minus). Transfers have no sign and use a `→` between accounts. Where space allows (tables, rows), a small secondary type label (Income / Expense / Transfer) sits under or beside the amount. Account balances keep their natural sign. Inputs never show a sign prefix. |
| D2 | **Amount colours come from the user's preference** (the amount colour settings and the `amount-color.scss` classes). Never hard-code green or red for income or expense. |
| D3 | **Corner ornaments.** Standard panels have a single copper **start tick** (an L at the top start corner of the header strip). Two brackets (`.ebk-panel--bracketed`, start-start and end-end) are reserved for **hero panels** (the overview's primary position panel, the login access panel) and **dialogs**. Nothing else gets brackets. |
| D4 | **Desktop shell:** a labelled navigation rail, a sticky header, a contextual side panel and the content area. See §5.1. |
| D5 | **Mobile shell:** a Framework7 navbar plus a floating tab bar with a copper add key at the center. See §5.2. The home page navbar shows the `ezrbookkeeping` wordmark; other pages show their title with a back action. |
| D6 | **No new functionality in redesign issues.** No bulk selection, multi-select tables, new widgets, new filters or "illustrative" sample figures. Concept elements that need new features are out of scope. |
| D7 | **Charts:** series are distinguishable without colour. Use labelled legends with values; income lines are solid and expense lines dashed; donut segments carry percentage labels. Every chart panel has a data table or ranked list with exact values, plus a one-sentence **text summary** generated from the data (for example "Payroll accounts for 73.88% of expenses"). Pattern fills are used only in forced-colors / high-contrast mode. |
| D8 | **Login:** a split layout. The start side has the wordmark, the claim "Your business, clearly balanced." and the existing instrument-style illustration, restyled with no figures. The end side has the **ACCOUNT ACCESS** hero panel with the form. Mobile shows only the access panel. |
| D9 | **Primary action:** "Add Transaction" is always visible. It is the copper button in the desktop header and the center key in the mobile tab bar. Each view has exactly one filled copper button. |
| D10 | **Headings:** the page title is in sentence case (`Transactions`). Panel header labels are technical: small, letter-spaced and uppercase for Latin scripts only (`TRANSACTION LIST`). |

## 3. Foundations

### 3.1 Colour

| Role | Light | Dark |
| --- | --- | --- |
| Canvas (page) | `#F7F4EF` + grid | `#0B0A09` + grid |
| Panel surface | `#FFFFFF` | `#161412` |
| Raised (menus, popovers, sticky header) | `#FFFFFF` | `#1E1B18` |
| Sunken (inputs, table header strip, wells) | `#F1ECE5` | `#100E0C` |
| Text primary / secondary | `#2B2421` / `#6B605A` | `#F4EBE1` / `#B9ADA2` |
| Hairline | `rgba(54,47,42,.14)` | `rgba(236,226,217,.12)` |
| Copper line (active edge, ticks, brackets) | `rgba(198,126,72,.45)` | `rgba(198,126,72,.55)` |
| Copper | `#C67E48` (copper text on light surfaces uses copper-700 `#97592F`) | `#C67E48` |
| Text and icons on copper fills | ink `#140F0D` | ink `#140F0D` |
| Semantic | success `#4CD964`, info `#2196F3`, warning `#FF9500`, error `#FF3B30` (error text uses `#C4161C`) | same (error text `#FF3B30`) |

Copper fills are allowed only for the primary button, the mobile add key, the selected segmented control and progress. Selected rows and navigation items use a copper-tinted background (copper at 10–14%) plus a 2–3px copper leading edge.

**Text on copper fills is dark ink, never white.** White on copper `#C67E48` reaches only 3.2:1 and fails WCAG AA for normal text. Ink `#140F0D` reaches 5.9:1 on copper, 5.0:1 on the copper-600 hover and pressed state, and at least 4.6:1 under the framework hover and press overlays. The fill stays brand copper in both themes. This applies to the Vuetify `on-primary` color, Framework7 filled buttons and the active plain segmented control, the mobile add key, date picker selections and the documentation site buttons (`--ebk-color-on-primary`). Hover and pressed fills use copper-600; they never darken to copper-700, because ink on copper-700 drops below 4.5:1.

**Text colors pass AA in both themes.** Copper text uses `--ebk-color-primary-text` (copper-700 on light surfaces, copper-500 on dark). On mobile, Framework7's theme color is copper-700 in the light theme, so links, text buttons and selection controls follow it. Error text uses `--ebk-color-error-text`. Income and expense amounts keep the hue the user picked, with a text shade per theme that reaches 4.5:1 (see [tokens.md](tokens.md#amount-colors)). Chart series colors and user-chosen category, account and tag colors are graphics, not text. Text next to them uses the text tokens, and the color appears only as a swatch, border or mark.

### 3.2 Typography

- **Family:** the existing Inter stack with system fallbacks. No web-font downloads.
- **Numerals:** tabular figures everywhere numbers align (`--ebk-font-numeric`). Amount columns are end-aligned.

| Use | Size / weight |
| --- | --- |
| Page title | 1.5rem / 600 desktop; navbar title on mobile |
| Hero amount (overview position, KPI tiles) | 2rem / 600 desktop, 1.75rem mobile |
| Tile and row amount | 1rem–1.25rem / 600 |
| Body | 0.875rem–1rem / 400 |
| Panel header label | 0.75rem / 600, `.08em` tracking, uppercase for Latin scripts only |
| Secondary meta (dates under titles, type labels) | 0.75rem / 400, text-secondary |

### 3.3 Geometry and ornament

- **Radius:** panels 10px; controls 8px; dialogs, sheets and popups 16px; chips and pills 999px.
- **Borders:** 1px hairline on every panel. The selected panel adds a copper leading edge.
- **Header strip:** 40px desktop / 36px mobile, a technical label at the start, actions at the end, a hairline underneath, and the copper start tick (D3).
- **Grid texture:** 24px at 3.5% on the page canvas only. Never inside panels, tables or dialogs. Off in forced-colors and print.
- **Glow:** only on keyboard focus, primary button hover, the active navigation item and live indicators.
- **Elevation:** shadow tokens in light; surface steps in dark.

### 3.4 Spacing and density

- Use the 4px scale (`--ebk-space-1…6`).
- **Panel padding:** 16px desktop, 12px mobile.
- **Gaps:** 16px between panels.
- **Row heights:**
  - Desktop transaction rows: 44px.
  - Date group rows: 32px.
  - Mobile list rows: at least 52px.
  - Touch targets: at least 44×44px.

### 3.5 Motion

Follow [motion.md](motion.md).
- **Durations:** 90 / 160 / 240 / 360ms.
- **Animated properties:** transform and opacity only.
- **Staggered reveals:** for overview widgets only, at most 6 items.
- **Interruptible:** new navigation cancels running transitions.
- **Saves never wait** for an animation.
- **Reduced motion** (system, or Settings → Preferences → Motion) gives opacity changes of 120ms or less and no chart animation.

## 4. Content patterns

### 4.1 Panel

`header strip (start tick · LABEL · meta … actions)` / body / optional footer.
- **Meta:** period, count or currency, in secondary text, before the actions.
- **Actions:** icon buttons with tooltips, or a text button. Never a filled copper button inside a panel header.

### 4.2 KPI tile

Label (technical), hero amount (signed and coloured per D1/D2, with an arrow icon ↑ income / ↓ expense), and secondary meta (period or currency). Pairs such as income and expense sit side by side, divided by a hairline.

### 4.3 Data table and list

- **Header:** the sunken header strip with technical labels. Sortable columns show an arrow.
- **Date groups:** a 32px group row with the date and a weekday chip.
- **Rows:** a category icon in a 32px tile, the title, and the type label underneath. Amounts are end-aligned and signed. Account names show `A → B` for transfers. Tags appear as chips, with an em dash when there are none. Descriptions truncate with a tooltip.
- **Selected row:** a copper tint plus leading edge.
- **Hover:** a neutral tint.
- **Footer:** pagination and rows per page, at the end.

### 4.4 Forms and editors

- **Fields:** a label above (technical style in dialogs), a sunken input, a copper focus line, and an error message under the field.
- **Amount input:** the largest element in an editor. It shows the currency code and the type label, and its colour follows D2.
- **Paired fields:** related fields sit in two columns on desktop (Date | Time, Account | Category).
- **Footer:** Cancel (outlined) plus the primary save (copper fill), with a status text at the start where available.

### 4.5 Charts

- **Panel:** the chart sits in a panel with a header strip.
- **Lines and series:** legends sit above or beside the chart and carry labels and values. Income lines are solid and expense lines dashed (D7). Value labels mark the latest point.
- **Axes:** tabular numerals, compact currency ticks (`£8k`), and grid lines at hairline strength.
- **Donuts:** the center shows the total and its label.
- **Accompanying data:** every chart is paired with a ranked list or table and a text summary panel (D7).
- **Animation:** entrance animation of 360ms or less; none in reduced motion.

### 4.6 Feedback states

Every data panel defines **loading**, **empty**, **error** and **offline** states:
- **Loading:** a skeleton shaped like the content.
- **Empty and error:** the shared `StatePanel` component, with an icon, a one-line message and an action (Retry, or Add the first item).
- **Notifications:** snackbars and toasts are opaque raised panels with a copper or error leading edge.
- **Destructive confirmations:** the confirm button is error-coloured and focus starts on Cancel.

## 5. Layout system

### 5.1 Desktop shell (D4)

```
┌────────┬──────────────────────────────────────────────────────────────┐
│ ▣ ezr  │ Page title / breadcrumb      [+ Add Transaction] ◐  📱  (JD) │ ← sticky header 64px
│ ────── ├──────────────┬───────────────────────────────────────────────┤
│ ⌂ Over │ SIDE PANEL   │  CONTENT (panels on grid canvas)               │
│ ⇄ Tx   │ sub-nav      │                                                │
│ ◔ Stat │ ──────────   │                                                │
│ ⌖ Expl │ FILTERS /    │                                                │
│ ▤ Acct │ OPTIONS      │                                                │
│        │              │                                                │
│ ⚙ Sett │ [Reset]      │                                                │
└────────┴──────────────┴───────────────────────────────────────────────┘
  rail 88px  side panel 260px (only on pages with sub-nav/filters)
```

- **Rail:** icon above label. The active item has a copper tint, a leading edge and copper text. Settings sits at the bottom.
- **Side panel:** page sub-navigation (for example Transaction List / Calendar / Gallery) followed by the FILTERS or REPORT OPTIONS panel. Below 1145px it becomes a slide-over with a backdrop, focus trap and Escape to close.
- **Content:** maximum width 120rem, centered, 16px gutters.
- **Keyboard:** a skip link, arrow-key navigation in the rail and side panel, and focus restoration on closing overlays.

### 5.2 Mobile shell (D5)

- **Navbar:** wordmark on Home, title and back elsewhere. Actions use 44px targets.
- **Tab bar:** floating raised panel; Details, Accounts, **[+] copper key**, Statistics, Settings. The active tab gets copper text plus an indicator bar. Labels truncate with an ellipsis.
- **Pages:** panels sit on the grid canvas with 12px side margins. Sheets and popups are 16px panels with a hairline top edge.
- **Offline:** a banner below the navbar.
- **Safe areas and keyboard:** safe areas are respected on all edges; inputs scroll into view above the on-screen keyboard.

## 6. Screen specifications

### 6.1 Overview (#30)

- **Desktop**, row by row:
  1. **MONTHLY POSITION** hero panel (bracketed): the period expense as the hero figure, then Income and **Net cash flow** (income − expense, signed). Next to it, **ASSET SUMMARY**: total assets, total liabilities and net assets.
  2. Four period tiles: TODAY, THIS WEEK, THIS MONTH, THIS YEAR, each with income and expense (KPI pattern).
  3. **INCOME & EXPENSE TRENDS** chart beside the **TRANSACTION CALENDAR** (days with activity get ↑ / ↓ markers; the selected day shows its transaction summary below the calendar).
  4. **EXPENSE CATEGORY RANKING** with bars, amounts and percentages, and the calendar heatmap when enabled.

  Widgets keep the existing configuration, visibility and order from the layout editor; the editor offers move up/down buttons in addition to drag.
- **Mobile:** MONTHLY POSITION panel (net assets or available balance hero, an income/expense pair and a net line), then **PERIOD SUMMARY** as a table (Period | Income ↑ | Expense ↓ | ›), then the other enabled widgets.

### 6.2 Transactions (#31)

- **Desktop list:** side panel with sub-nav (List / Calendar / Gallery) and FILTERS (date range, type, account, category, per page, Reset filters). Content: a **TRANSACTION LIST** panel with a header strip (label, count, search, Import), a period summary line (`November 2026 · INCOME +£8,017.20 · EXPENSE −£5,684.70`), the data table (§4.3) and a pagination footer.
- **Editor dialog:** bracketed dialog with the **ADD / EDIT TRANSACTION** header and an Expense | Income | Transfer segmented control, followed by:
  - the large amount field with currency
  - Date | Time
  - Account | Category
  - Description
  - Tags (chips plus "+ Add tag")
  - an **Attach receipt — PDF or image** drop zone
  - a footer with status, Cancel and the copper Save button

  Existing sections such as location and pictures keep their current navigation.
- **Mobile entry:** segmented type control, then an **AMOUNT · GBP** panel with the large amount, then a grouped panel of Category / Account / Date / Description rows with chevrons, then Attach receipt, then the number pad sheet. Save stays where the app already has it.
- **Templates and scheduled transactions:** the same table pattern, with status badges per #14 (label plus colour).

### 6.3 Statistics, insights and reconciliation (#32)

- **Desktop:** side panel with REPORT VIEW (analysis types) and REPORT OPTIONS (group by, currency). Content:
  - a filter bar (period stepper, accounts, categories, search)
  - three KPI tiles: TOTAL INCOME, TOTAL EXPENSE, NET MOVEMENT
  - the chart panel or panels
  - a breakdown table with a Total row
  - a REPORT SUMMARY text panel
- **Mobile:** period stepper, then a Categories | Trends segmented control, then the chart panel, then the ranked list, then the income/expense summary row.
- **Insights explorer and reconciliation statements:** the same panel, table and summary patterns.

### 6.4 Management and data transfer (#33)

- **Lists** (accounts, categories, tags, templates, exchange rates, custom icons, import presets): panels with header strips. Nested children are indented with a hairline guide. Hidden items are dimmed and carry a "Hidden" chip. Drag handles stay visible on hover and focus, with non-drag move buttons available.
- **Import:** a stepper (Upload → Map → Check → Import) in a header strip. Validation issues are listed with counts and a jump-to-row action. Progress reports the real completion count.

### 6.5 Authentication, settings and utility (#34)

- **Login (D8):** canvas grid background.
  - **Access panel:** the hero "ACCOUNT ACCESS" panel with "Welcome to ezrbookkeeping", then the Username/email and Password fields (visibility toggle), Forgot password, the copper Log In button, Use on Mobile Device and Create an account.
  - **Other screens** (signup, 2FA, unlock, reset and verify) use the same access panel, titled for each step.
- **Settings:** sections are listed in the side panel. Each section is a panel with grouped rows. Theme and Motion selectors are in Preferences. Save feedback uses a snackbar.

### 6.6 Website and PWA (#35)

- **Documentation site:** uses the same palette, type rules, panels and callouts, in light and dark.
- **PWA and browser metadata:**
  - App icons and the manifest use copper on warm neutral.
  - Splash and theme colors follow the canvas colors.
  - The favicon is legible at 16px.

## 7. Accessibility and internationalisation

- **Contrast:** WCAG 2.2 AA. Text at least 4.5:1; UI and focus at least 3:1, in both themes.
- **Focus:** focus-visible is a 2px copper ring at 2px offset. It is never removed without a replacement.
- **Not colour alone:** status, type and series are never conveyed by colour alone (D1, D7).
- **RTL:** layouts use logical properties. Edges, ticks, brackets and transitions mirror in RTL.
- **Text size:** long translations (de, ru, ta, bn) and large text (mobile font-size up to xxx-large, 200% desktop zoom) wrap rather than clip.
- **Forced colors:** decoration is dropped and system colors are used.
- **Live regions:** snackbars, the offline banner and save results are announced.

The audit for #36 checks these rules against the whole redesign. Its results, the tested matrix and the checks that still need a person are in [accessibility-audit.md](accessibility-audit.md).

## 8. Route and state inventory

Every route covers the states in §4.6 that apply to it. **Owner** is the implementing issue.

### Desktop (`src/router/desktop.ts`)

| Route | Page | Owner |
| --- | --- | --- |
| `/` | Overview | #30 |
| `/overview/edit` | Overview layout editor | #30 |
| `/transaction/list` (list, calendar, gallery; add/edit/detail dialog; import dialog) | Transactions | #31 (import: #33) |
| `/template/list`, `/schedule/list` | Templates, scheduled transactions | #31 |
| `/statistics/transaction` | Statistics | #32 |
| `/insights/explorer` | Insights explorer | #32 |
| `/account/list` (+ reconciliation statement dialog) | Accounts | #33 (reconciliation: #32) |
| `/category/list`, `/tag/list`, `/custom_icon/list`, `/presets/import/list`, `/exchange_rate` | Management | #33 |
| `/settings/user/basic`, `/settings/user/security`, `/settings/user/two_factor`, `/settings/user/data_management` | Account settings | #34 (data management: #33) |
| `/settings/preferences`, `/settings/application_lock`, `/settings/statistics`, `/settings/cloud_sync`, `/settings/browser_cache`, `/settings/developer_tools` | Settings | #34 |
| `/login`, `/signup`, `/verify_email`, `/forgetpassword`, `/resetpassword`, `/oauth2_callback`, `/unlock` | Authentication | #34 |
| Shell, About dialog, snackbars, confirm dialogs | Shell and shared | #28 / #27 |

### Mobile (`src/router/mobile.ts`)

| Route | Owner |
| --- | --- |
| `/`, `/settings/overview_layout` | #30 |
| `/transaction/*`, `/template/*`, `/schedule/list` | #31 |
| `/statistic/*`, `/account/reconciliation_statements` | #32 |
| `/account/*`, `/category/*`, `/tag/*`, `/exchange_rates*`, `/settings/filter/*`, `/settings/account_category_display_order`, `/user/data/management` | #33 |
| `/login`, `/signup`, `/unlock`, `/settings*`, `/app_lock`, `/about`, `/user/*` | #34 |
| Shell: tab bar, navbar, sheets, popups, number pad, toasts | #29 / #27 |

## 9. Review checklist

1. Figures are the most prominent content: signed (D1), user-coloured (D2) and tabular.
2. Ornaments are used only as allowed by D3, and the grid only on the canvas.
3. Each view has one copper fill (D9), and Add Transaction is always reachable.
4. Contrast, focus, RTL, reduced motion and forced colors are all verified.
5. Loading, empty, error and offline states exist.
6. No new features (D6); existing behavior, saved layouts and APIs are unchanged.
7. Long translations and large text don't clip.
8. Light and dark are both checked on desktop and mobile.

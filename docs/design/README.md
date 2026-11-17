# ezrbookkeeping design brief: "Copper Instrument Panel"

This is the shared design reference for the sci-fi revamp milestone (#24–#37). Implementation issues cite it, and design review checks against it. The product name (`ezrbookkeeping`) and attribution follow #23.

## 1. Direction

ezrbookkeeping should feel like a **precision instrument panel for money**. It is calm, technical and trustworthy, not a game UI.

- **Information first.** Amounts, dates, accounts and states are the brightest things on screen. Decoration sits one step behind content.
- **Copper as signal, not paint.** The brand copper `#C67E48` marks focus, selection, primary actions and live state. Large surfaces stay warm neutral.
- **Framed panels.** Content sits in panels with hairline borders, small corner brackets and a labelled header strip, like modules in a console.
- **Restraint.** No continuous ambient animation in bookkeeping workspaces, and no glow behind body text. Grid texture appears only on the page canvas and never under dense tables.
- **One language, two frameworks.** Vuetify (desktop) and Framework7 (mobile) keep their component behavior. Both consume the same tokens (`src/styles/common/_tokens.scss`, introduced by #25).

## 2. Palette (retained meanings)

| Role | Light | Dark | Notes |
| --- | --- | --- | --- |
| Brand / primary | `#C67E48` | `#C67E48` | Unchanged. Hover/pressed use the copper ramp below |
| Copper ramp | 50 `#FBF3EC` · 100 `#F3DFCD` · 300 `#DFA97F` · 500 `#C67E48` · 600 `#B67443` · 700 `#97592F` · 900 `#5A3219` | same | Tints are for selected backgrounds; 700 is for copper text on light surfaces (AA) |
| Canvas | `#F7F4EF` | `#0B0A09` | Page background, optional grid texture |
| Surface (panel) | `#FFFFFF` | `#161412` | Cards, dialogs, sheets |
| Surface raised | `#FFFFFF` | `#1E1B18` | Menus, popovers, sticky headers |
| Surface sunken | `#F1ECE5` | `#100E0C` | Inputs, table header, wells |
| Text primary | `#2B2421` | `#F4EBE1` | Was `#413935` / `#fcf0e3`; slightly higher contrast |
| Text secondary | `#6B605A` | `#B9ADA2` | Must stay ≥ 4.5:1 on surface |
| Hairline | `rgba(54,47,42,.14)` | `rgba(236,226,217,.12)` | Panel and divider borders |
| Copper line | `rgba(198,126,72,.45)` | `rgba(198,126,72,.55)` | Active panel edge, corner brackets, focus |
| Success / Info / Warning / Error | `#4CD964` / `#2196F3` / `#FF9500` / `#FF3B30` | same | Unchanged semantic colors |

**Financial colors:** the income/expense amount colors (user-selectable in preferences, `src/styles/*/amount-color.scss`) are **never** remapped by the theme. Category, account and tag colors chosen by users, and chart color schemes (`src/consts/color.ts`), are rendered as-is. Status is never conveyed by color alone: use a label, icon or shape too.

## 3. Typography

- **Body:** the existing Inter stack (`--v-font-body`), with system fallbacks for CJK, Indic and Thai scripts. No new web-font downloads (the PWA must work offline).
- **Technical headings:** panel header strips use 0.75rem, weight 600, `letter-spacing: .08em`, uppercase **only for Latin scripts** (`:lang(en)` etc.). Other scripts use normal case at the same size.
- **Numerals:** every amount, balance, date column and counter uses `font-variant-numeric: tabular-nums` (token `--ebk-font-numeric`). Amounts align to the end of the cell.
- **Monospace:** IDs, tokens and code use `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`.
- Text-size preferences (mobile font-size classes, desktop browser zoom) must keep working. Never fix heights on text containers.

## 4. Panel geometry and ornament

- Radius: panels 10px (`--ebk-radius-md`), controls 8px, chips 999px. No clip-path chamfers on focusable containers, because they clip focus rings.
- Border: 1px hairline. The *active* or *selected* panel gets a 1px copper line on its leading edge (inline-start, so it works in RTL).
- **Corner brackets:** an optional `.ebk-panel--bracketed` modifier draws 10px L-brackets in the copper line color at two opposite corners using `::before`/`::after`. Use them on overview widgets and dialog headers only.
- **Header strip:** panel titles sit in a 40px (desktop) or 36px (mobile) strip, with a hairline underneath and optional actions at the end.
- **Grid texture:** a 24px grid at 3.5% opacity on the app canvas only (`.ebk-canvas-grid`). It is disabled in forced-colors mode and on printed pages.
- **Glow:** `0 0 0 1px copper-line, 0 0 12px rgba(copper,.25)`. Allowed on the focused control, the primary button hover state, the active navigation item and live indicators. Never on large panels or behind text.
- **Elevation:** use hairline plus shadow tokens (`--ebk-shadow-sm/md/lg`). Dark theme relies on surface steps more than shadows.

## 5. Density and spacing

Use the 4px spacing scale (`--ebk-space-1…6`). Desktop transaction tables keep the current row height (about 44px) or smaller; decoration must not add vertical whitespace to dense lists. Mobile touch targets are at least 44×44 CSS px.

## 6. Motion (implemented by #26)

| Token | Value | Use |
| --- | --- | --- |
| `--ebk-motion-duration-xs` | 90ms | Hover and press feedback |
| `--ebk-motion-duration-sm` | 160ms | Selection indicators, small reveals |
| `--ebk-motion-duration-md` | 240ms | Dialogs, sheets, panel reveals |
| `--ebk-motion-duration-lg` | 360ms | Route transitions, chart entrances |
| `--ebk-motion-ease-standard` | `cubic-bezier(.2,0,0,1)` | Default |
| `--ebk-motion-ease-emphasized` | `cubic-bezier(.3,0,0,1)` | Entrances |
| `--ebk-motion-ease-exit` | `cubic-bezier(.4,0,1,1)` | Exits |
| `--ebk-motion-distance-sm/md` | 4px / 12px | Translate distances |
| Stagger | 30ms per item, maximum 6 items | Widget and list reveals |

Rules:
- Animate only `transform` and `opacity`.
- Transitions can be interrupted: a new navigation cancels the running one.
- Saves never wait for an animation.
- Reduced motion (system preference, or the in-app setting: Follow system / Reduce / Full) replaces movement with an opacity change of 120ms or less, and turns off chart entrance animations.
- Loading indicators are the only continuous motion. They stop when hidden.

## 7. States every surface must define

Default, hover, focus-visible, pressed, selected, disabled, invalid, loading (skeleton or progress), empty, error with retry, and offline (PWA). Focus-visible is a 2px copper ring at 2px offset, at least 3:1 against the surface. In forced-colors mode, use system colors and drop the decoration.

## 8. Screen references

`docs/design/screenshots/before/` holds the pre-revamp baselines. `docs/design/screenshots/after/` is filled in as each implementation issue lands, and serves as the reviewed mockup reference for light/dark and desktop/mobile.

## 9. Route and state inventory

Every route below must cover the states in §7 that apply to it. **Owner** is the implementation issue responsible.

### Desktop (`src/router/desktop.ts`)

| Route | Page | Owner |
| --- | --- | --- |
| `/` | Overview (widgets) | #30 |
| `/overview/edit` | Overview layout editor | #30 |
| `/transaction/list` (+ calendar, gallery tabs; add/edit/detail dialog; import dialog) | Transactions | #31 (import: #33) |
| `/template/list`, `/schedule/list` | Templates, scheduled transactions | #31 |
| `/statistics/transaction` | Statistics and charts | #32 |
| `/insights/explorer` | Insights explorer | #32 |
| `/account/list` (+ reconciliation statement dialog) | Accounts | #33 (reconciliation: #32) |
| `/category/list`, `/tag/list`, `/custom_icon/list`, `/presets/import/list`, `/exchange_rate` | Management | #33 |
| `/settings/user/basic`, `/settings/user/security`, `/settings/user/two_factor`, `/settings/user/data_management` | Account settings | #34 (data management export/import: #33) |
| `/settings/preferences`, `/settings/application_lock`, `/settings/statistics`, `/settings/cloud_sync`, `/settings/browser_cache`, `/settings/developer_tools` | Settings | #34 |
| `/login`, `/signup`, `/verify_email`, `/forgetpassword`, `/resetpassword`, `/oauth2_callback`, `/unlock` | Authentication | #34 |
| App shell: top bar, navigation, settings layout, About dialog, global snackbars, confirm dialogs | Shell and shared | #28 / #27 |

### Mobile (`src/router/mobile.ts`)

| Route | Owner |
| --- | --- |
| `/` (overview), `/settings/overview_layout` | #30 |
| `/transaction/list`, `/transaction/add`, `/transaction/edit`, `/transaction/detail`, `/transaction/filter/amount`, `/template/*`, `/schedule/list` | #31 |
| `/statistic/transaction`, `/statistic/settings`, `/account/reconciliation_statements` | #32 |
| `/account/*`, `/category/*`, `/tag/*`, `/exchange_rates*`, `/settings/filter/*`, `/settings/account_category_display_order`, `/user/data/management` | #33 |
| `/login`, `/signup`, `/unlock`, `/settings*`, `/app_lock`, `/about`, `/user/profile`, `/user/2fa`, `/user/sessions` | #34 |
| Shell: tab bar, navbar, sheets, popups, number pad, toasts | #29 / #27 |

### Cross-cutting

- Design tokens: #25.
- Motion: #26.
- Shared controls: #27.
- Website and PWA assets: #35.
- Accessibility, localization and reduced-motion audit: #36.
- Visual baselines, performance and rollout: #37.

## 10. Review checklist

1. Amounts are readable at a glance, with tabular numerals and user amount colors intact.
2. Contrast is at least 4.5:1 for text and at least 3:1 for UI and focus, in both themes.
3. No decoration overlaps content or shrinks hit targets.
4. RTL mirrors brackets, edges and transitions.
5. Reduced motion is honored.
6. Long translations (de, ru, ta) don't clip.
7. Loading, empty and error states exist.
8. No new runtime dependencies or font downloads.

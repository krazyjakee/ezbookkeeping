# Shared controls

This page documents how the shared controls look and behave in the "Copper Instrument Panel" language (#27). Read it together with the [design brief](README.md), the [tokens](tokens.md) and the [motion system](motion.md). Pages inherit these styles automatically. Page-specific adoption belongs to #30–#34.

## Where the styles live

| Layer | Desktop (Vuetify) | Mobile (Framework7) |
| --- | --- | --- |
| Framework overrides | `src/styles/desktop/override/_vuetify.scss`. The block at the end is wrapped in Vuetify's own `@layer vuetify-overrides`, so it beats the Vuetify component styles but still loses to Vuetify utility classes (`rounded-*`, `bg-*`, `text-*`) and to page styles | `src/styles/mobile/override/_framework7-instrument.scss`. It sets Framework7 CSS variables on `:root.ios` and `:root.ios.dark`, plus a few structural rules |
| Date pickers | `src/styles/desktop/override/_vue-datepicker.scss` | `src/styles/mobile/override/_vue-datepicker.scss` |
| Shared component pieces | `src/styles/desktop/components/_shared-controls.scss` (dialog title strip, snackbar message, amount field numerals) | Component `<style>` blocks (`NumberPadSheet.vue`) |
| Empty, error and offline states | `src/components/common/StatePanel.vue`, used by both apps | same |

All colors come from the shared tokens. The user amount colors (`amount-color.scss`) and user category, account and tag colors are never remapped. Fields that use a custom `color` or `base-color`, such as `AmountInput`, keep that color for their outline and label.

## States at a glance

Focus-visible is always the global 2px copper ring at a 2px offset (from `_tokens.scss`). The "glow" is `--ebk-glow`. It appears only on focused fields (blur only, so it never draws a line through the floating label), on primary buttons while hovered or focused, and on the tab indicator.

| Control | Default | Hover | Focus-visible | Pressed | Selected | Disabled | Invalid | Loading |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Primary button (flat / elevated) | Copper fill, 8px radius | Copper glow + Vuetify overlay | Ring + glow | Scales to .98 (90ms). No scaling with reduced motion | n/a | Vuetify disabled colors. Dark theme: 50% opacity, desaturated | n/a | Vuetify loader replaces the label, no glow, no press scale |
| Tonal button | Tint + 1px hairline in its own color (22%) | Overlay | Ring | .98 | n/a | Dashed edge | n/a | Loader |
| Outlined / text button | Border at 45% of its color / no border | Overlay | Ring | .98 | n/a | Dashed edge (outlined) | n/a | Loader |
| Icon button | Round, unchanged | Overlay | Ring | .98 | `active` keeps Vuetify color | Vuetify | n/a | n/a |
| Text field / select / autocomplete / textarea (outlined) | Sunken well (`--ebk-color-surface-sunken`), outline at 28% | Outline at 60% | 2px copper outline, soft copper blur, label in AA copper text color | n/a | n/a | Transparent well, 38% opacity | Error outline, faint error tint in the well, bold message under the field (text, not only color) | Vuetify progress bar on the field |
| Date picker (vue-datepicker) | Transparent background (inherits panel), hairline borders, 8px cells, tabular numerals | Copper tint | Ring | n/a | Copper fill, bold | Library default | n/a | n/a |
| List item | Unchanged | Vuetify overlay | Ring | Overlay | Tint + 2px copper bar on the leading edge (mirrored in RTL). Navigation lists are left to the shell (#28) | Vuetify | n/a | n/a |
| Table | Sunken header strip, technical header type (uppercase for Latin scripts), hairline rows, tabular numerals in cells | Row hover tint | Ring on focusable cells | n/a | `tr.ebk-row-selected` or `tr.v-data-table__selected`: tint + 2px copper bar on the first cell (mirrored in RTL) | n/a | n/a | Skeleton rows |
| Chip | Pill. Tonal chips get a hairline in their color. Label chips keep 8px | Overlay | Ring | n/a | Copper hairline ring | Vuetify | n/a | n/a |
| Tabs | Unchanged | Overlay | Ring | n/a | Copper indicator with a soft glow. Pill tabs keep their copper fill | Vuetify | n/a | n/a |
| Pagination | Tabular numerals | Overlay | Ring | n/a | Current page: tint + 2px copper base line, bold | Vuetify | n/a | n/a |
| Tooltip | Inverted, 8px, hairline, tabular numerals | n/a | n/a | n/a | n/a | n/a | n/a | n/a |
| Menu / select dropdown / popover | Raised surface, hairline, `--ebk-shadow-md`, 10px | Item overlay | Ring | n/a | List item rule above | n/a | n/a | n/a |
| Dialog | Hairline panel, `--ebk-shadow-lg`, copper corner brackets (start-start and end-end, inside the edge), tinted header strip | n/a | Initial focus on the safest action | n/a | n/a | n/a | n/a | Spinner next to the title (dialog layouts) |
| Snackbar | Raised panel, hairline, 3px copper leading edge | n/a | Close button ring | n/a | n/a | n/a | Error messages (`showError`): error leading edge + alert icon | n/a |
| Alert (tonal) | Hairline in its own color, 8px | n/a | n/a | n/a | n/a | n/a | n/a | n/a |
| Progress / skeleton | Pill progress bars. Skeleton bones use a hairline tint, with a faint copper shimmer | n/a | n/a | n/a | n/a | n/a | n/a | Continuous motion is allowed for loaders only, and stops when hidden |

### Mobile (Framework7)

| Control | Treatment |
| --- | --- |
| Buttons | 8px radius. Filled buttons glow on hover (pointer devices) and on keyboard focus. Press scales to .98, except segmented, tab and number pad buttons. Tonal buttons get a hairline. Disabled outline buttons are dashed |
| Inputs | Outline inputs sit in a sunken well with an outline at 28%. Focus: copper outline, soft blur, label in the AA copper text color. In list inputs the row hairline turns copper while focused and red while invalid. Error messages wrap instead of clipping |
| Lists | Hairline rows, copper-tinted pressed state, tabular numerals in `item-after` and `item-footer`. `li.list-item-selected` gets a tint and a copper leading edge (mirrored in RTL) |
| Segmented controls | Sunken track. The active segment is raised, uses copper text and has a 2px copper base line |
| Chips | Pill shape, hairline outline, tabular numerals |
| Dialogs (alerts, confirms, prompts, loading) | Opaque raised panel with a 16px radius, hairline, large shadow and copper corner brackets. The title sits on a header strip with a hairline under it. Text and buttons wrap long translations. Buttons use the AA copper text color. `color-red` and `dialog-button-destructive` buttons use the error color |
| Popovers, action sheets | Opaque raised surface with a hairline |
| Sheets | Surface panel, 16px radius, hairline top edge. A leading toolbar gets a hairline header strip. The number pad sheet is excluded |
| Toasts, notifications | Opaque raised panel, hairline, 3px copper leading edge, tabular numerals |
| Number pad (`NumberPadSheet`) | Sunken display row with tabular numerals, hairline grid, sunken operator column, copper-tinted pressed keys, bold confirm key. Keys stay 60px tall |
| Progress / preloader / skeleton | Copper progress on a copper track. Secondary-text preloader. Hairline-tinted skeleton text |

## Confirmation and destructive dialogs

- **Desktop** (`src/components/desktop/ConfirmDialog.vue`, API unchanged): a neutral title strip with an icon replaces the colored toolbar. The icon is a question mark, or an alert icon in the error color when the dialog is destructive (`open(..., { color: 'error' })`). Cancel is a secondary tonal button. The confirm button uses the dialog color, so destructive confirms are red. Cancel receives focus as soon as the dialog has opened, so pressing Enter never confirms by accident. The dialog is labelled by its title (`aria-labelledby`).
- **Dialog layouts** (`OneColumnDialogLayout`, `TwoColumnDialogLayout`): the close "x" is now a real icon button. It is reachable with the keyboard, has an accessible name and keeps its tooltip.
- **Mobile** (`f7.dialog` via `src/lib/ui/mobile.ts`): styled by CSS as described above. Button order and the `strong` confirm button are unchanged.

## Snackbars, toasts and validation

- **Desktop `SnackBar`** (API unchanged): `showMessage` shows a neutral panel with a copper edge. `showError` adds the error edge and an alert icon. When a caller passes `color`, Vuetify's colored snackbar is used as before.
- **Mobile toasts**: an opaque raised panel replaces the glass background, so text contrast does not depend on the content behind it.
- **Validation**: messages sit under the field and wrap (`overflow-wrap: anywhere`). Error messages are medium weight. Status is carried by the message text and the outline, not by color alone.

## Empty, error and offline states

`StatePanel` (`src/components/common/StatePanel.vue`) is a presentational panel that works in both apps:

```vue
<state-panel type="empty" :title="tt('No available custom icons')" />
<state-panel type="error" :title="tt('Unable to retrieve transaction list')" :text="errorText">
    <template #actions><v-btn @click="reload">{{ tt('Refresh') }}</v-btn></template>
</state-panel>
<state-panel type="offline" compact :title="..." />
```

| Prop / slot | Meaning |
| --- | --- |
| `type` | `empty` (default: dashed hairline, copper tray icon, `role="status"`), `error` (error border and leading edge, alert icon, `role="alert"`) or `offline` (warning icon) |
| `title`, `text` | Translated strings supplied by the caller. The component has no strings of its own |
| `compact` | A one-line row for menus and sheets |
| `#icon`, default slot, `#actions` | Custom icon, extra content, and a retry or create button |

It is used today for the "No available custom icons" state in `IconSelect` (desktop) and `IconSelectionSheet` (mobile). Pages adopt it in #30–#34.

## Accessibility, RTL and large text

- Decoration never takes layout space or hit area. Brackets, leading edges and selection bars are absolutely positioned or drawn as inset shadows or gradients, with `pointer-events: none`. Mobile list rows and dialog buttons keep their 44px height.
- Leading edges, brackets and the selected row and list bars use logical properties, or have an explicit `[dir="rtl"]` mirror.
- No fixed heights were added to text containers. Long labels wrap in dialogs, snackbars, validation messages, state panels and toasts. Mobile font-size classes keep working because only Framework7 variables and colors were changed.
- Forced-colors mode drops the brackets, edges and glows.
- Reduced motion: the press scale is turned off under `:root.ebk-reduce-motion` and under `prefers-reduced-motion` (unless the user chose **Full**). Every other transition uses the motion tokens, and only `transform` and `opacity` are animated.

## Screenshots

These are in `screenshots/after/`. Dialogs, menus and snackbars could not be captured: the review browser ran in a background tab, where Vue and Vuetify enter transitions never finish.

| File | Shows |
| --- | --- |
| `components-desktop-table-light.jpg` | Header strip, hairline rows, chips, sunken selects, pill tabs, list selection edge (light) |
| `components-desktop-pagination-chips-light.jpg` | Current page cell, pill chips, tabular amounts (light) |
| `components-desktop-selects-switches-light.jpg` / `-dark.jpg` | Selects (sunken well), switches, outlined buttons in both themes |
| `components-desktop-fields-table-dark.jpg` | Text fields, disabled primary button, tonal error buttons (default and disabled), table header strip (dark) |
| `components-mobile-lists-dark.jpg` | Mobile inset lists with hairline rows and toggles (dark) |

# Release notes: the "Copper Instrument Panel" redesign

This release completes the redesign milestone (#24–#37). It also includes the rename to **ezrbookkeeping** (#23), full-instance backups (#38) and new MCP tools for bookkeeping agents (#22). The redesign changes how the app looks and moves. It does not change your data, your amounts or how bookkeeping works.

Validation evidence (builds, performance, cross-browser smoke tests, visual captures and the rollback test) is in [docs/design/rollout-validation.md](../design/rollout-validation.md).

## What's new

### A new look for desktop and mobile

- **One design language for both apps.** Warm neutral surfaces, framed panels with a labelled header strip, and the existing copper `#C67E48` accent. Copper marks the current location, the selection, focus and the primary action. Light and dark themes are both included, and the default still follows your system.
- **Desktop shell.** A labelled navigation rail, a sticky header with an always-visible **Add Transaction** button, a side panel for each section, and a skip link.
- **Mobile shell.** A floating tab bar with a copper add key in the center, the `ezrbookkeeping` wordmark on the home page, and an offline banner when the connection drops.
- **Figures first.** Amounts are signed and labelled as well as coloured: income `+£8,017.20`, expense `−£5,684.70`, and transfers with `→`. The income and expense colours you chose in your settings still apply.
- **Redesigned screens.** Overview dashboard and widgets, transaction list, calendar and gallery, transaction entry and attachments, recurring and template transactions, statistics, charts and the insights explorer, reconciliation, accounts, categories, tags, exchange rates, import and export, sign-in, settings and utility pages.
- **Charts you can read without colour.** Charts have labelled legends with values, solid income lines and dashed expense lines, a one-sentence summary, and a data table or ranked list with exact values.
- **Loading, empty, error and offline states** look the same everywhere and offer Retry or Reset filters where that makes sense.

### Motion

- Motion is short and purposeful. Panels, dialogs and newly loaded content fade and rise into place in 360 ms or less. Nothing animates while the app is idle.
- **New setting: Settings → Preferences → Motion** (System Default, Reduced, Full). **Reduced**, or the system "reduce motion" preference, replaces movement with fades of 120 ms or less and turns off chart animations. The older mobile **Enable Animation** switch still works.

### Accessibility

The redesign was audited against WCAG 2.2 AA ([accessibility-audit.md](../design/accessibility-audit.md)). Among the changes:

- Text on copper buttons uses dark ink for contrast. Income and expense amounts use a darker shade of your chosen colour.
- All controls work with the keyboard and show a visible focus ring. Dialogs keep focus inside, and return it when they close.
- Pinch zoom is allowed on mobile. Layouts reflow down to 320 px wide and work at large text sizes.
- Pages, dialogs, charts and calendar days have names for screen readers. Save results are announced on mobile.

### ezrbookkeeping (rename)

- The product, Docker image, release archives and documentation are now called **ezrbookkeeping**. The Docker image is `krazyjakee/ezrbookkeeping` (it was `krazyjakee/ezbookkeeping2`).
- These stay the same: the binary name `ezbookkeeping`, the configuration file `conf/ezbookkeeping.ini`, the `EBK_*` environment variables, the container paths under `/ezbookkeeping` and the database file name. You do not need to change your configuration.

### Full-instance backups

- New `ezbookkeeping backup create | list | verify | restore` commands. A backup is one archive (optionally encrypted) with the database, uploaded files and the configuration file.
- Optional scheduled backups with retention (`[backup]` section in `conf/ezbookkeeping.ini`, off by default).
- See the [Backup and Restore](https://krazyjakee.github.io/ezbookkeeping/backup/) documentation.

### MCP tools for bookkeeping agents

- Transaction references: supplier, invoice and receipt numbers, and external source IDs.
- Attachment evidence: original file name, size and SHA-256.
- Idempotency keys for writes, and an audit log of MCP writes (`query_audit_log`).
- Bulk import with dry run, mapping, duplicate detection and per-row results. Also export and reconciliation tools.
- See the [MCP](https://krazyjakee.github.io/ezbookkeeping/mcp/) documentation.

## Compatibility

- **Data:** no data is converted. The new database tables and columns (MCP references, idempotency records and the audit log, plus attachment metadata) are added automatically when the server starts. Existing tables are only extended. The rollback test showed that the previous release still runs against the upgraded database (see [Rollback](#rollback)).
- **Settings:** your existing app settings are kept. **Motion** defaults to *System Default*.
- **Browsers:** the same targets as before (Chrome/Edge 119+, Firefox 128+, Safari 16.4+). The smoke tests and accessibility scans passed in Chromium, Firefox and WebKit.
- **Installed PWA:** the service worker and caches are unchanged. An installed app updates the next time it starts while online.
- **API and MCP:** the existing HTTP API and MCP tools are unchanged. The new MCP tools and fields are additions.

## Known limitations

- On a low-powered device (4× CPU throttling), the desktop transaction list renders about 120–160 ms slower than before at 50 rows per page, because the rows have more content and styling. Scrolling stays at 60 fps. At the default 15 rows per page, the difference is smaller. This is tracked as P1 in [rollout-validation.md](../design/rollout-validation.md#regressions-and-budgets).
- The largest element on the desktop overview and statistics pages is now the hero amount, which appears once data has loaded. Because of that, the reported Largest Contentful Paint is later than before, but the figures themselves render only about 10% later.
- When offline, the mobile overview shows the offline banner and also an "Unable to retrieve transaction overview" error panel instead of an offline-specific panel.
- No language that ships with ezrbookkeeping is written right to left. The right-to-left layout was checked by forcing `dir="rtl"` only.
- Screen reader, Windows contrast theme and physical device checks are still open. See [accessibility-audit.md](../design/accessibility-audit.md#not-covered).
- Vuetify keeps closed tooltips in the page, which automated accessibility checkers report (`aria-tooltip-name`). This is a third-party behavior and doesn't affect users.

## Rollback

Take a backup first (`ezbookkeeping backup create`), then pin the previous Docker image tag or reinstall the previous release archive. The database changes are additive, so the previous version runs against the upgraded database. To restore a backup, use the same or a newer version than the one that created it. The full procedure is in [rollout-validation.md → Deployment rollback path](../design/rollout-validation.md#deployment-rollback-path).

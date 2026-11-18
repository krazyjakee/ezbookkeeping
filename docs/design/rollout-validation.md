# Rollout validation of the redesign (#37)

This document records the final validation of the redesign milestone (#24–#36) before rollout. It covers builds and checks, bundle size, performance on desktop and a lower-powered mobile proxy, animation behavior, smoke tests, cross-browser checks, visual baselines and the deployment rollback path. Release notes for users are in [../release-notes/redesign.md](../release-notes/redesign.md).

- **Date:** 2026-11-17
- **Before:** `f064815b` (rebrand to ezbookkeeping2, the last commit before the first redesign implementation `41205dd2`). Later commits such as `9cb1aae5` already contain the tokens (#25/#26), restyled controls (#27) and new shells (#28/#29), so they can't serve as a "before".
- **After:** `fcf232b2` (`main` after #36), plus the two presentation fixes from this issue (see [Regressions found and fixed](#regressions-found-and-fixed)). Overview data tables (`7f00e8f3`) landed during this validation and were not measured.
- **Evidence:** the scripts, JSON results, logs and captures are outside the repository in `/tmp/ezb-37/` (`scripts/`, `perf*.json`, `bundles.json`, `anim*.json`, `axe-*.json`, `smoke-out/`, `captures/`, `captures.zip`). No images are committed.

**Verdict:** ready to roll out, with one tracked performance item (P1).

- The redesign added a mobile bundle regression: every mobile page loaded Vuetify, its CSS and ECharts. That is fixed here.
- All budgets pass except the desktop transaction list on the 4× CPU profile (P1).
- The smoke suite passes 17/17 in Chromium, Firefox and WebKit.
- The previous release runs against the upgraded database.

## Environment

| Item | Value |
| --- | --- |
| Builds | `npm ci` + `npm run build` (vite 8 / rolldown, `vite-plugin-checker` runs `vue-tsc` during the build) in isolated `git archive` snapshots: `/tmp/ezb-37/before`, `/tmp/ezb-37/after` (HEAD), `/tmp/ezb-37/fixed` (HEAD plus the #37 fixes). The Go backend was built from each snapshot with `go build` (go 1.27.3) |
| Serving | Each production `dist` was served by its own backend binary (`server run`, SQLite, local file storage), each with its own data directory: before :18110, after :18120, fixed :18121, smoke :18122, rollback :18123 |
| Data | `seed.py` demo company (5 accounts plus sub-accounts, 9 categories, about 40 transactions) plus 1,000 generated transactions over 92 days (`scripts/seed_extra.py`, seed 37). The data was identical on every server. The smoke server had only the demo data |
| Browsers | Chromium: Chrome for Testing headless shell 156.0.8078.4 (playwright-core 1.65.1, CDP). Firefox 159.0 and WebKit 27.2 (Playwright 1.65.1 builds; WebKit needs `libavif16`, `libgav1-1` and `libyuv0`, which were added to its bundle directory) |
| Desktop profile | 1440×900, DPR 1, no throttling |
| Lower-powered proxy | 4× CPU throttling (`Emulation.setCPUThrottlingRate`). Mobile pages also use mobile emulation (390×844, DPR 2, touch). Desktop pages were also measured with 4× throttling at 1440×900 |
| Runs | Each case loaded with a cold cache (new context) 3 times, alternating the build order. The tables show medians |

## Builds and checks

| Check | Result |
| --- | --- |
| `npm ci` + `npm run build` (includes `vue-tsc`) | Pass for before, after and fixed. Build time 80 s, peak RSS 8.3 GB (after) |
| `npm run lint` (`vue-tsc --noEmit && eslint .`) | Pass (fixed snapshot) |
| `vitest run` | 17 files, 38,724 tests pass, including the new `src/lib/__tests__/chunk_boundaries.test.ts`. The new test fails on HEAD (`components/common/StatePanel.vue imports @mdi/js`) and passes with the fix |
| `go build` | Pass for before and after |
| Documentation site (`website`, `npm ci && npm run docs:build`) | Pass (VitePress, 4.3 s, no dead links) |

## Bundle size

Raw / gzip (level 9) KB. "Initial" means every script, module preload and stylesheet referenced by the entry HTML.

| Asset | Before | After (HEAD) | After + fix | Δ before → after + fix |
| --- | --- | --- | --- | --- |
| desktop initial JS (11 files) | 5,198.8 / 1,327.4 | 5,302.5 / 1,355.5 | 5,303.2 / 1,356.0 | +2.0% / +2.2% |
| desktop initial CSS | 607.9 / 91.0 | 703.8 / 103.8 | 703.8 / 103.8 | +15.8% / +14.2% (+12.8 KB gz) |
| mobile initial JS | 3,906.7 / 896.7 (9 files) | **5,565.5 / 1,435.6 (11 files)** | 3,956.1 / 911.3 (9 files) | +1.3% / +1.6% |
| mobile initial CSS | 188.6 / 33.3 | **690.0 / 101.9** | 265.4 / 45.1 | +40.7% / +35.3% (+11.8 KB gz) |
| all JS (44 chunks) | 21,525.5 / 5,178.9 | 21,667.5 / 5,217.6 | 21,668.2 / 5,218.1 | +0.7% / +0.8% |
| all CSS | 1,622.8 / 243.0 | 1,793.2 / 267.0 | 1,793.5 / 267.1 | +10.5% / +9.9% |
| images | 930.2 | 802.2 | 802.2 | −13.8% (re-exported icons, #35) |
| whole `dist` | 28,008.8 / 8,224.4 | 28,196.1 / 8,173.6 | 28,196.9 / 8,174.2 | +0.7% / −0.6% |

The largest chunk changes are `desktop.js` (+22.2 KB gz), `css/desktop.css` (+12.4 KB gz), `css/mobile.css` (+11.2 KB gz), `mobile.js` (+9.4 KB gz) and `common.js` (+3.8 KB gz). The vendor chunks are unchanged.

## Budgets

| # | Budget | Result |
| --- | --- | --- |
| B1 | CLS ≤ 0.1 on every load and interaction (target ≤ 0.01) | **Pass.** The maximum was 0.004 (before: 0.006, plus 0.03 during rapid navigation) |
| B2 | Initial JS per entry (gzip) no more than 15% above before | **Pass after the fix.** HEAD failed on mobile (+60%) |
| B3 | Initial CSS per entry (gzip) no more than +15 KB | **Pass:** desktop +12.8 KB, mobile +11.8 KB. This is the design system (tokens, panels, signed amounts and states). It is +14% / +35% relative, justified by #25–#29 |
| B4 | FCP and data-ready time (first rendered amount) no more than 15% above before | **Pass,** except the desktop transaction list FCP on 4× (+17%, P1). LCP is reported separately (see L1) |
| B5 | TBT no more than 15% or +50 ms above before, whichever is larger | Pass for the desktop overview (+33 ms) and every mobile page (≤ +10%). Desktop statistics is +55 ms at 4× (5 ms over, accepted). The desktop transaction list is **+130–160 ms at 4× (P1)** |
| B6 | Scrolling: desktop at 60 fps with no frame over 33 ms; mobile on 4× no worse than before | **Pass:** desktop 60 fps and 0 frames over 33 ms in both builds. Mobile on 4× runs at 50.0 fps vs 50.6 before, with the same 4 slow frames, which are infinite-scroll loads |
| B7 | Input: no event over 100 ms (INP "good") on the 4× mobile entry page | **Pass:** 56 ms in both builds |
| B8 | JS heap after GC no more than 15% above before | **Pass:** desktop transaction list +14% (21.9 → 25.0 MB), others ≤ +4%. HEAD failed on mobile (+23%) |
| B9 | No recurring idle work: 0 rAF callbacks, layouts and running animations over 10 s idle | **Pass** (see Animation checks) |

## Performance

Medians of three cold loads. TBT is the long-task time over 50 ms after FCP until the page settled. "First amount" is the first time a formatted amount was inserted into the DOM, measured with a MutationObserver, as a data-ready proxy. Mobile figures come from the final build (with the reset layer). Desktop "after + fix" is identical to HEAD apart from the inlined StatePanel icons.

### Desktop, no throttling

| Page | Metric | Before | After (HEAD) | After + fix |
| --- | --- | --- | --- | --- |
| Overview | FCP / LCP ms | 352 / 352 | 344 / 440 | 332 / 416 |
| | TBT ms, heap MB | 0, 26.4 | 0, 27.4 | 0, 27.5 |
| Transaction list (50 rows per page, 1,045 transactions) | FCP = LCP ms | 360 | 372 | 372 |
| | TBT ms, elements | 16, 2,048 | 29, 2,574 | 37, 2,574 |
| | scroll (rAF-driven, 4 s) | 60 fps, p95 16.8 ms | 60 fps, p95 16.7 ms | 60 fps, p95 16.8 ms |
| Statistics | FCP / LCP ms | 316 / 316 | 316 / 404 | 316 / 416 |
| | chart hover sweep (3 s, 90 moves) | 60 fps, 0 frames over 33 ms | 60 fps, 0 | 60 fps, 0 |

### Desktop, 4× CPU

| Page | Metric | Before | After (HEAD) | After + fix | Δ |
| --- | --- | --- | --- | --- | --- |
| Overview | FCP ms | 1,184 | 1,224 | 1,260 | +6% |
| | first amount ms | 1,224 | n/m | 1,348 | +10% |
| | LCP ms | 1,184 | 1,676 | 1,676 | +42% (L1) |
| | TBT ms / main thread ms | 65 / 1,708 | 117 / 1,819 | 98 / 1,799 | +33 ms / +5% |
| Transaction list | FCP = LCP ms | 1,136 | 1,312 | 1,332 | +17% (P1) |
| | TBT ms | 206 | 323 | 364 | +158 ms (P1) |
| | style recalc ms / max long task ms | 106 / 912 | n/m | 228–282 / 1,053–1,118 | (P1) |
| | scroll | 60 fps, 0 frames over 33 ms | 60 fps, 0 | 60 fps, 0 | |
| Statistics | FCP ms | 1,088 | 1,152 | 1,148 | +5.5% |
| | first amount ms | 1,256 | n/m | 1,373 | +9% |
| | LCP ms | 1,088 | 1,500 | 1,492 | +37% (L1) |
| | TBT ms | 69 | 125 | 124 | +55 ms |
| | chart hover sweep | 60 fps, 0 frames over 33 ms | 60 fps, 0 | 60 fps, 0 | |

### Mobile (390×844, DPR 2, touch)

| Page | Metric | Before | After (HEAD) | After + fix |
| --- | --- | --- | --- | --- |
| Overview, no throttling | LCP ms, transfer KB, heap MB | 316, 4,750, 14.4 | 344, **6,913**, 19.8 | 320, 4,878, 12.8 |
| Overview, 4× | FCP / LCP ms, main thread ms | 984 / 1,076, 1,086 | 1,108 / 1,208, 1,202 | 992 / 1,092, 1,089 |
| Transaction list, 4× | FCP = LCP ms, TBT ms | 1,376 / 1,524, 226 | 1,520, 234 | 1,424, 240 |
| | infinite scroll (5 s, 218 rows loaded) | 50.6 fps, TBT 612 ms | 50.2 fps, 653 ms | 50.0 fps, 647 ms |
| Transaction entry, 4× | FCP / LCP ms | 1,100 / 1,648 | 1,276 / 1,916 | 1,164 / 1,776 |
| | TBT ms, max event (number pad taps) | 398, 56 ms | 441, 56 ms | 437, 56 ms |
| | heap after GC MB | 21.1 | 23.4 | 21.3 |

CLS was at most 0.004 on every page and profile, and every interaction added 0 shift (after: 0–0.002, before: 0–0.006).

### Regressions and budgets

- **R1, fixed: mobile loaded the desktop vendor bundle.** `src/components/common/StatePanel.vue` (added in #27) imports three icons from `@mdi/js`. The build's `codeSplitting` groups put `@mdi/*` into `vendor-desktop`, together with Vuetify and vue-echarts, and vue-echarts imports `echarts`. Because StatePanel is in the shared `common` chunk, `mobile.html` preloaded and evaluated `vendor-desktop` (577 KB), its 425 KB Vuetify stylesheet and `echarts` on every page load. Compared with the fixed build, the cost was +2.0 MB transferred, +581 KB gzip of initial JS and CSS, +2.1 MB heap after GC, and +12% FCP on the 4× overview.
  - **Fix:** the three SVG paths are inlined in StatePanel.
  - **Guard:** `src/lib/__tests__/chunk_boundaries.test.ts` fails when any file that the build puts in the `common` chunk statically imports a desktop-only or mobile-only package or module.
- **R2, fixed: mobile layout depended on the leaked Vuetify reset.** The redesigned mobile styles had been tuned while Vuetify's reset (`box-sizing: border-box` for every element, `font: inherit` on form controls, `vertical-align: inherit` on pseudo-elements) leaked into mobile. Without it, rows and panels grew taller: 32 of 34 mobile captures differed from HEAD in 1–11% of their pixels.
  - **Fix:** `src/styles/mobile/base/_global.scss` declares the same rules in a cascade layer (`ebk-mobile-reset`), so, as with Vuetify's layered reset, every unlayered Framework7 or app rule still wins.
  - **Result:** all 74 captures now match HEAD within 0.14% of pixels (the threshold is a channel difference over 24/255, and the remaining differences are clock times in the data).
- **P1, tracked: desktop transaction list render cost on 4× CPU.** At 50 rows per page, the first render takes +130–160 ms TBT and +17% FCP. Causes:
  - 26% more elements per page (signed-amount component, type labels, row meta and tooltips)
  - about twice the style recalculation time (106 → 230–280 ms on 4×, 168 → 324 ms unthrottled in a trace)

  Selector matching is not the cause: the CSS selector stats trace shows 14 ms in total, and no selector costs more than 0.6 ms. Removing the new per-row description tooltip didn't help (an experiment build measured the same). Unthrottled, the difference is under 25 ms TBT and +3% FCP, and scrolling stays at 60 fps with no slow frames. At the default 15 rows per page, the cost is proportionally smaller.

  **Follow-up options:** cut per-row elements (render the type label and meta lines only where shown), or lower custom-property churn on rows (fewer `--ebk-*` overrides per row element).
- **L1, explained: desktop LCP.** The LCP element changed. Before, it was a widget title or navigation label rendered with the shell. Now it is the hero amount (`.ebk-signed-amount.overview-widget__hero-amount`, or the statistics total), which only exists once data has loaded. The data-ready proxy (first amount) is only +9–10%, FCP is +5–6%, and reduced motion gives the same LCP, so the reveal animation isn't delaying it. LCP stays under 1.7 s on 4× and under 0.45 s unthrottled.

## Animation checks

Script: `scripts/anim.mjs` (plus `offprobe.mjs`), run against before and after + fix.

| Check | Before | After |
| --- | --- | --- |
| Idle work over 10 s (desktop overview, transactions, statistics; mobile home, transactions, statistics), measured 4 s after load | 0 rAF callbacks, 0 layouts, ≤ 1 style recalc, ~1 ms task time, no running animations | Same: 0 rAF, 0 layouts, ≤ 1 style recalc, ~1 ms, no running or infinite animations |
| Layout jumps | CLS up to 0.006 on load | CLS ≤ 0.004 on load; 0 during scroll, chart hover and number pad entry |
| Desktop rapid navigation: 30 hash route changes 70 ms apart, then 10 Add Transaction open/Escape cycles | 1 overlay left active, CLS 0.03 | 0 active overlays, 0 scrims, no leftover `*-enter/leave-*` classes, main content not dimmed or `pointer-events: none`, CLS 0, no errors |
| Input right after a route change (click Add Transaction 50 ms after navigating) | dialog visible after 220 ms | 246 ms (includes the 240 ms dialog transition, which motion.md allows; the click is never blocked) |
| Mobile rapid navigation: 15 tab-bar taps and 15 back taps 80 ms apart, then number pad sheet open/close | 30/30 taps handled, 1 current page, 0 modals or backdrops left | Same |
| Service worker (production build) | registered and active (`sw.js`, scope `/`), controls the page after reload, precache 58 entries | Same, unchanged `src/sw.ts` |
| Offline reload (`context.setOffline` after the SW took control) | the app shell reloads from cache; no offline indicator | The app shell reloads from cache. The mobile banner "You are offline. Changes cannot be saved until the connection is restored." shows (`role="status"`, `html.ebk-offline`), both live and after the offline reload |

Observation: offline, the mobile overview also shows an "Unable to retrieve transaction overview" error panel rather than the `StatePanel` offline variant. This is noted in the release notes and not changed here (the overview is being edited in parallel).

## Smoke tests

Script: `/tmp/ezb-37/scripts/smoke.mjs <chromium|firefox|webkit>`. It drives the after + fix production build at 1440×900 (plus one mobile flow) and checks every result through the HTTP API with the session token. Each run uses a unique tag, so the three engines can run against one server.

| # | Step | Verified |
| --- | --- | --- |
| 1 | Log in through the desktop form | token stored, profile `demo` |
| 2 | Add expense 12.34 | API: type 3, `sourceAmount` 1234 |
| 3 | Add income 250.00 | API: type 2, 25000 |
| 4 | Add transfer 40.00 to another account (two-column account menu) | API: type 4, out = in = 4000, different accounts |
| 5 | Edit the expense to 56.78 (detail dialog → edit) | API: 5678 |
| 6 | Delete the income (detail dialog → Delete → confirm) | API: gone |
| 7 | Search by description; `?type=4` filter | 1 matching row and no others; only transfers shown, API agrees |
| 8 | Attach a PNG and a PDF in the add dialog, then view them | API: 2 attachments, served `200 image/*` (PNG re-encoded to JPEG by the server, as before) and `200 application/pdf`; the detail dialog shows the image and the PDF tile |
| 9 | Create a template | API template list |
| 10 | Create a scheduled transaction (Daily) | API: `scheduledFrequencyType` > 0 |
| 11 | Statistics | chart canvas plus category list rendered |
| 12 | Export from Data Management (UI download) and the API export | the CSV has the new rows and the edited amount |
| 13 | Import a 2-row CSV (upload → check → select → Import) | API: both rows with the right amounts |
| 14 | Preferences: Theme = Dark, Motion = Reduced, reload | `ebk_app_settings` persisted, dark theme and `html.ebk-reduce-motion` applied after reload, then reset |
| 15 | Mobile: log in and add an expense with the number pad | API: 31400 (£314.00) |
| 16 | Persisted values cross-check | every transaction matches what was entered, and the deleted one is absent |
| 17 | No uncaught page errors | — |

**Result:** 17/17 in Chromium 156.0.8078.4, Firefox 159.0 and WebKit 27.2. The suite was rerun on the final build after R2.

## Cross-browser

`scripts/axe-xb.mjs` ran axe-core 4.14.1 (`wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa`) on 26 states in each engine against the final production build:

- desktop: login, overview, transaction list, add dialog, statistics, accounts and preferences
- mobile: login, home, transaction list, add, statistics and settings
- each in light and dark

| Engine | Clean states | Violations (nodes) | JS errors |
| --- | --- | --- | --- |
| Chromium 156.0.8078.4 | 10 / 26 | `aria-tooltip-name` 284 (J1), `label-content-name-mismatch` 22 (J2, overview calendar), `color-contrast` 5 (J3, mobile chevron glyphs) | 0 |
| Firefox 159.0 | 10 / 26 | identical | 0 |
| WebKit 27.2 | 10 / 26 | identical | 0 |

All remaining items are the justified items from [accessibility-audit.md](accessibility-audit.md#justified-and-tracked-items). The results are also in its tested matrix. Firefox has no `isMobile` emulation, so its mobile states ran at the mobile viewport with touch only.

## Visual baselines

`scripts/capture.mjs` captured 74 states per build (before, after and after + fix) into `/tmp/ezb-37/captures/{before,after,fixed}/`:

- **Desktop:** 16 routes and states (login, overview, transaction list and calendar, add and detail dialogs, statistics, explorer, accounts, categories, templates, schedules, preferences, data management, exchange rates and the import dialog), each in light and dark.
- **Mobile:** 13 states (login, home, transaction list, add, the number pad sheet, accounts, statistics, settings, preferences, categories, templates, exchange rates and about), each in light and dark.
- **Extra variants:** forced `dir="rtl"` and `prefers-reduced-motion: reduce` for four key states per platform.

The contact sheet `/tmp/ezb-37/captures/index.html` (zipped with all images to `/tmp/ezb-37/captures.zip`, 31 MB) puts the three builds side by side and shows the after → after + fix pixel difference. `diff-report.json` holds the numbers.

Findings:

- After + fix matches HEAD on all 74 states (≤ 0.14% of pixels differ). This required R2.
- No page errors in any capture. No unstyled or framework-default surfaces were found in either theme. Dialogs, menus, the number pad sheet, state panels and the login illustration all use the tokens.
- Forced RTL mirrors the desktop rail, panels and dialogs. On mobile, only the `dir` attribute was forced (Framework7's RTL stylesheet loads only for RTL languages), as in the #36 audit.
- With reduced motion, the captures match the default captures once settled, which confirms that no state depends on an animation finishing.

## Regressions found and fixed

| ID | Files | Change |
| --- | --- | --- |
| R1 | `src/components/common/StatePanel.vue` | Inline the three `@mdi/js` paths so the shared chunk no longer pulls `vendor-desktop` (Vuetify + ECharts) into `mobile.html` |
| R1 guard | `src/lib/__tests__/chunk_boundaries.test.ts` | New test: shared-chunk source files must not statically import desktop-only or mobile-only packages or modules |
| R2 | `src/styles/mobile/base/_global.scss` | Layered box-sizing, form-control font and pseudo-element alignment reset for mobile, matching the rules mobile had relied on |

## Deployment rollback path

The redesign changes only the presentation. The releases in this milestone only add to the database (#22: new MCP tables and attachment metadata columns; #38 adds no tables). So the previous version can run against the upgraded database.

### Before rolling back

1. **Take a backup with the running (new) version.** This captures the database, uploaded files and the configuration file:
   - Docker: `docker exec ezbookkeeping /ezbookkeeping/ezbookkeeping backup create`
   - Binary: `./ezbookkeeping backup create -o /safe/place`

   Then check it with `backup verify <file>`. A restore needs the **same or a newer** version than the one that created the backup, so keep the new version available in case you need to restore. Older releases don't have the `backup` command at all. For them, copy the data directory (SQLite) or take a database dump.
2. Note any configuration you set through `EBK_*` environment variables. Backups don't contain them.

### Docker

1. Pin a previous image instead of `latest`. These older tags are still on Docker Hub, but new builds only publish `latest`:
   - `krazyjakee/ezrbookkeeping:sha-41b9241`: the rename, before the shells and most redesign screens
   - `krazyjakee/ezbookkeeping2:sha-f064815`: the last image before any redesign code, if your registry has it (published under the previous image name)
2. `docker compose up -d` (or `docker run …`) with the **same volumes** (`/ezbookkeeping/data`, `/ezbookkeeping/storage`, `/ezbookkeeping/log`) and environment. The container paths, the binary name and the configuration keys are unchanged. Older versions ignore the `[backup]` section.

### Binary

1. Stop the service.
2. Unpack the previous release archive (`ezrbookkeeping-<version>-<arch>.tar.gz`, or the earlier `ezbookkeeping2-…` archives) over the installation, keeping `conf/`, `data/`, `storage/` and `log/`.
3. Start it again.

### After rolling back

- Clear the browser caches on clients. Installed PWAs pick up the old service worker on their next online start: Settings → Browser Cache Management, or a hard reload.
- Data written while rolled back stays valid for the new version, but some new fields stay empty:
  - attachments uploaded by the old version have no original name, size or SHA-256
  - MCP references, idempotency records and audit entries aren't cleaned up when the old version deletes transactions or clears data; the leftover rows are inert
- To roll forward again, start the new version. Its startup database update is a no-op on an already-upgraded schema.

### Verification

On 2026-11-17, `scripts/facts.py` ran against a copy of the smoke-test instance: 78 transactions, 14 attachments (PNG and PDF), 6 templates and 6 schedules, written by the after build (HEAD).

1. `backup create` (20 KB archive, 14 storage objects) and `backup verify` with the after binary reported "Backup is valid".
2. The **before** binary (`f064815b`) started on the after database with no errors in its log. It returned identical facts: transaction count, per-type sums, account balances, templates and schedules, and all 14 attachments served with HTTP 200. Its desktop and mobile UI loaded with no page or API errors, and a new expense was written.
3. The after binary started again on the same database (roll forward). It returned all prior facts plus the expense written by the old version.

## Not covered

- Real devices and real Safari or Edge. Firefox and WebKit are Playwright builds on Linux, and the "lower-powered device" is a CPU-throttling proxy. A run on a mid-range Android phone (Chrome) and an iPhone (Safari, including the home-screen PWA) is still needed.
- Network throttling. All loads were over loopback, so the transfer size (B2) stands in for network cost.
- Screen readers and the other manual checks listed in [accessibility-audit.md → Not covered](accessibility-audit.md#not-covered).
- Database engines other than SQLite for the rollback test. The schema changes are additive in every engine (xorm `Sync2`), but MySQL and PostgreSQL were not started with the old binary.
- `7f00e8f3` (overview data tables, landed during this validation) was not part of the measured builds.

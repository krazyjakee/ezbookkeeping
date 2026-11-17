# Repository cleanup review — 2026-11-17

## Scope and starting state

Reviewed commit `41205dd28681b875fcec5f1218f210355a81593d` plus the existing working tree. Uncommitted MCP, transaction attachment, model, service and documentation work was already present. Additional MCP and UI edits appeared during the review. Those changes were preserved and are excluded from this cleanup's claims and metrics.

The starting tracked diff and untracked-file list were recorded before editing. Read-only audits covered architecture, code duplication, verification, security, dependencies, documentation, UI and repository hygiene. A fresh reviewer subsequently inspected the cleanup implementation and regression tests.

## Repository model

ezrbookkeeping is a self-hosted bookkeeping application. Accounts, categories, transactions, transfers, tags and attachments are its principal domain concepts.

- The Go executable starts in `ezbookkeeping.go`; `cmd` initializes configuration, persistence, background jobs and HTTP routes.
- Gin handlers in `pkg/api` translate HTTP requests into domain operations. `pkg/services` handles persistence and ownership checks. Datastore and object-storage boundaries support configured backends.
- Vue and TypeScript entrypoints select desktop Vuetify or mobile Framework7. Shared libraries, stores and base page logic serve both interfaces.
- npm and Vite build the frontend; Go and the platform build scripts produce the executable and release packages. Go tests, Vitest, vue-tsc, ESLint and Go vet provide verification.
- GitHub Actions and composite actions build release/snapshot packages and Docker images; VitePress builds the documentation website.

Public contracts retained include HTTP and MCP interfaces, database schema, import/export formats, CLI options, configuration keys, `EBK_` variables, storage paths, executable name and the upstream Go module path. The shared copper tokens, financial colors, reduced-motion support and separate desktop/mobile interfaces have documented rationale and were retained.

## Summary

Prioritized reproducible correctness and security issues over architectural or visual rewrites. Work was divided into five small batches: PostgreSQL connections; logging/upload resource handling; array comparison; development tooling; and POSIX build failure handling with documentation corrections.

## Removed

- Unused direct development dependencies `@jest/globals`, `@tsconfig/node26` and `ts-node`, plus their orphaned lockfile entries. Repository searches found no consumers; frontend tests use Vitest.
- Raw OAuth state and cached PKCE-containing remarks from error logs.
- Four scattered custom-icon file closes, replaced by one deferred close at the point where the handler opens the file.

## Simplified and fixed

- PostgreSQL TCP connections use standard URL user-information escaping and host/port joining. Passwords and usernames containing spaces now round-trip correctly, and IPv6 hosts retain brackets. Existing plain URLs, Unix-socket behavior and invalid-host errors remain covered.
- Array comparison groups by `Map` rather than an object whose inherited keys could cause crashes. Ordering independence, duplicate multiplicity and input preservation remain covered.
- Request logs redact `code`, `state`, `token`, `access_token` and `refresh_token`, including repeated, encoded and case-varied keys. Malformed queries produce an omission marker. The actual request URL remains unchanged; ordinary logged query ordering and encoding are normalized.
- Custom-icon files close on every handler return, including database errors and duplicate submissions. The service's existing successful-upload close remains compatible.
- POSIX builds stop when dependency installation or compilation fails. Packaging propagates backend/frontend failures rather than archiving stale output.
- `npm run lint` checks without modifying source; `npm run lint:fix` explicitly requests the previous autofix behavior.

## Verification added

Regression tests reproduced the original PostgreSQL, array-comparison, sensitive-logging and stale-package failures before the fixes.

- Seven PostgreSQL input cases cover spaces, plus signs, reserved characters, IPv6 and invalid host syntax, using independent URL parsing.
- Seven frontend cases cover special property names, reordered duplicates, different multiplicities, missing keys, lengths, empty inputs and immutability.
- Six request-log cases capture actual emitted logs, including malformed queries and unchanged request URLs. Capture uses a temporary file so missing output fails without blocking.
- Five POSIX build outcomes use failing command stubs and pre-existing output: dependency and compilation failures must not produce an archive; successful packaging must still work.

## Documentation changes

The README describes concrete functionality and compatibility commitments without claiming that renaming is the fork's only change. English and Chinese development guides now document separate verification commands, live third-party test controls and explicit lint fixes. Four composite action descriptions correctly identify `skip-tests` as a Go test-name regular expression.

## UI changes

None in this cleanup. The existing product design and concurrent UI work were preserved.

## Deferred findings

- **MCP:** an existing failing test and ongoing implementation are outside this cleanup. They require review together with their uncommitted feature work.
- **Windows build scripts:** similar native-command error propagation deserves platform-specific regression tests. This change is limited to the executable POSIX path available here.
- **Configuration logging:** its clone-error fallback returns the original configuration. The unsafe fallback is visible, but an actual clone-failure condition was not reproduced; a separate change should fail closed and test secret redaction and source immutability.
- **Map proxies:** forwarding of incoming authorization headers/cookies and ignored URL parse errors need investigation of intended upstream authentication and normal UI reachability before changing behavior.
- **Year-month validation:** permissive parsing should be tightened only after establishing supported formats and compatibility with persisted filters.
- **Financial integration coverage:** add real database ownership/rollback tests before refactoring financial writes. Existing service/provider boundaries have real purposes; file size alone does not justify restructuring them.
- Lower-value comment, helper, badge and visual changes were deferred to avoid unnecessary public API or product churn.

## Verification results

Baseline: backend and frontend builds, vue-tsc, and Go vet passed. Vitest passed 38,715 tests in 16 files. `rtk go test ./...` failed in `TestMCPWriteToolsAcceptControlArguments` and `TestExchangeRatesApiLatestExchangeRateHandler_BankOfIsraelDataSource`. ESLint reported only the local, Git-excluded `vite.local.config.ts` missing from its TypeScript project.

| Command | Result |
| --- | --- |
| `rtk go test ./cmd ./pkg/datastore ./pkg/middlewares ./pkg/api -count=1 -timeout=60s` | Passed: 56 tests, including subtests. |
| `rtk npm test` | Passed: 38,722 tests across 16 files. |
| `rtk npx vue-tsc --noEmit` | Passed at baseline; also checked by the final lint command. |
| `rtk npm run lint -- --ignore-pattern vite.local.config.ts` | Passed; excludes the existing Git-excluded local configuration. |
| `rtk proxy npx eslint . --format stylish` | Failed solely on that local configuration's missing TypeScript project membership. |
| `rtk go vet ./...` | Passed. |
| `rtk go build -o /tmp/ezbookkeeping-antislop-verified .` | Passed. |
| `rtk npm run build` | Passed, including service worker generation. Existing config-loader, bundle-size and build-option warnings remain. |
| `rtk npm run docs:build` from `website` | Passed. |
| `rtk sh -n build.sh` | Passed. |
| `rtk npm ci --dry-run --ignore-scripts --no-audit --no-fund` | Passed; checked the install plan without changing the installation. |
| `rtk env BUILD_PIPELINE=1 go test ./...` | Failed only in the existing `TestMCPWriteToolsAcceptControlArguments`, for `query_audit_log`. Other tested packages passed; live third-party checks were omitted through the existing CI setting. |
| `rtk git diff --check` | Passed. |

The fresh reviewer independently passed datastore, middleware, API, POSIX build-script and array-comparison tests and found no remaining blockers in the cleanup diff. Full cleanup diff review included all new tests, build behavior, action descriptions, documentation and a structural comparison of every lockfile change. No API, schema, migration, runtime dependency or authentication behavior was changed.

## Remaining risks

The entire working tree is not green: the baseline MCP failure remains, and live third-party tests depend on external services. The ignored local Vite configuration remains outside ESLint's project. Windows builds were not exercised. No live PostgreSQL server was required for the connection-string tests, and no dedicated file-descriptor lifecycle test was added for custom icons. Dependency checks used the existing installation plus lockfile/install-plan validation.

Other work continued in the shared repository during validation. Results apply to the tested working-tree snapshots; this cleanup does not validate or claim ownership of concurrent MCP or UI changes.

## Before / after metrics

| Measure | Before | After |
| --- | ---: | ---: |
| Root direct development dependencies | 25 | 22 |
| Non-root package-lock entries | 936 | 792 |
| Frontend test cases | 38,715 | 38,722 |
| Scattered custom-icon handler closes | 4 | 1 deferred close |

No runtime dependencies were removed. No lockfile entries were added, and retained package versions, source URLs and integrity hashes are unchanged. No files were deleted. Build size changes are not attributed because concurrent UI/MCP edits would confound that comparison.

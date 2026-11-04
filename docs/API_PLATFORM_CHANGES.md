# API Platform Stage 1: what changed, why, and what it affects

**Date:** 2026-11-04 · **Status:** not committed yet (two repos: `sereni-base` and `services/base-ui`)

This page lists every code change made for the workspace public API (Stage 1 of [API_PLATFORM_FOUNDATION.md](API_PLATFORM_FOUNDATION.md)), why each was needed, and how it touches features that already existed. Line numbers are as of the date above; links jump to them.

## In one paragraph

Each workspace can now get a permanent API address (`/api/v1/ws_xxxxxxxxxx`) from a new **Workspace API** card on the home page. Behind that address is a public REST API for reading and writing records, with paging, filters, bulk writes and safe retries. It has **no authentication yet**, so it only exists when the server runs with `PUBLIC_API_OPEN_ACCESS=true`, and the server refuses that setting in production. Most of the code is new files. The changes to existing files are small, but a few of them apply to the whole app, not only the new API. Those are listed under [Side effects](#side-effects-on-existing-features).

---

## 1. Database

| What | Why | Where |
|---|---|---|
| New table `workspace_api`: one row per workspace that turned the API on, holding its public ID, on/off flag and `schema_version` | The public address must be fixed forever and separate from the workspace slug | [workspace_api.go L17–66](../internal/models/tenant/workspace_api.go#L17-L66) |
| Two `CHECK` constraints on it (ID format, `enabled_at` required when on), added with raw SQL | go-postgres-rest only supports simple one-column checks | [workspace_api.go L60](../internal/models/tenant/workspace_api.go#L60), helper [create_db.go L47–56](../internal/scripts/create_db.go#L47-L56) |
| New table `api_idempotency_keys` | Stores responses so a retried POST isn't written twice | [api_idempotency_key.go L17–64](../internal/models/tenant/api_idempotency_key.go#L17-L64) |
| Both tables created at startup, right after `workspaces` | Same way every other table is created | [create_db.go L104–106](../internal/scripts/create_db.go#L104-L106) |
| Trigger function `workspace_api_bump_schema_version` | Bumps `schema_version` when a base, table or field changes, in the same transaction, so the API's name caches never serve stale names | [databaseConstant.go L1089–1122](../internal/constant/databaseConstant.go#L1089-L1122) |
| Six triggers on `bases`, `models`, `columns` (insert/delete, plus update only when a relevant column really changes) | Same reason; the `WHEN` clause skips no-op saves | [databaseConstant.go L1125–1154](../internal/constant/databaseConstant.go#L1125-L1154), installed by [schema_version.go L17](../internal/scripts/schema_version.go#L17), called from [create_db.go L118–121](../internal/scripts/create_db.go#L118-L121) |

## 2. Turning the API on from the app (internal API)

| What | Why | Where |
|---|---|---|
| `GET /api/v1/workspace/{id}/api`: status, for any workspace member | The Developer popup needs to know if the API is on and its address | handler [workspace_api.go L33](../internal/handlers/workspace_api.go#L33), route [router.go L302–310](../internal/router/router.go#L302-L310) |
| `POST /api/v1/workspace/{id}/api/enable`: owner and co-owner only (`workspace.update` permission) | Turning a workspace's data into an API is an admin decision | handler [workspace_api.go L56](../internal/handlers/workspace_api.go#L56), same routes |
| Public ID generator and enable logic: random `ws_` + 10 characters, retry on clash, same ID kept when turned on again | Spec section 1 | [platform/workspace.go L42](../internal/platform/workspace.go#L42) (`NewPublicID`), [L86](../internal/platform/workspace.go#L86) (`Enable`) |
| Wiring | | [app.go L230–240](../internal/app/app.go#L230-L240), [L253](../internal/app/app.go#L253), [L268–269](../internal/app/app.go#L268-L269) |

## 3. The public API (`internal/platform/`, all new)

| File | What it does |
|---|---|
| [api.go](../internal/platform/api.go) | Registers the routes ([L63](../internal/platform/api.go#L63)); request pipeline: workspace lookup ([L92](../internal/platform/api.go#L92)), auth placeholder ([L122](../internal/platform/api.go#L122)), per-workspace rate limit ([L127](../internal/platform/api.go#L127)) |
| [resolve.go](../internal/platform/resolve.go) | Workspace cache (10 s, misses cached too) ([L67](../internal/platform/resolve.go#L67)); bases, tables and fields cached per `schema_version` ([L113](../internal/platform/resolve.go#L113)); name-or-ID lookup with `409 ambiguous_name` ([L261](../internal/platform/resolve.go#L261)) |
| [schema.go](../internal/platform/schema.go) | `GET /bases`, `/tables`, one table with its fields ([L43](../internal/platform/schema.go#L43), [L57](../internal/platform/schema.go#L57), [L77](../internal/platform/schema.go#L77)) |
| [query.go](../internal/platform/query.go) | `limit`, `cursor`, `select`, `filter`, `sort`, `keys` ([L72](../internal/platform/query.go#L72)); builds the SQL ([L391](../internal/platform/query.go#L391)); cursor encode/decode ([L440–455](../internal/platform/query.go#L440-L455)) |
| [fields.go](../internal/platform/fields.go) | Field types; database → JSON ([L213](../internal/platform/fields.go#L213)) and JSON → database with validation ([L304](../internal/platform/fields.go#L304)) |
| [records.go](../internal/platform/records.go) | Record handlers: list ([L379](../internal/platform/records.go#L379)), get ([L440](../internal/platform/records.go#L440)), create ([L776](../internal/platform/records.go#L776)), update one/many ([L866](../internal/platform/records.go#L866), [L950](../internal/platform/records.go#L950)), delete one/many ([L1041](../internal/platform/records.go#L1041), [L1060](../internal/platform/records.go#L1060)); link and lookup resolution ([L182](../internal/platform/records.go#L182), [L264](../internal/platform/records.go#L264)); link writes ([L676](../internal/platform/records.go#L676)) |
| [idempotency.go](../internal/platform/idempotency.go) | `Idempotency-Key` handling ([L54](../internal/platform/idempotency.go#L54)) and the hourly cleanup ([L181](../internal/platform/idempotency.go#L181)) |
| [envelope.go](../internal/platform/envelope.go) | `{data, error, meta}` responses and the 8 error codes |
| [ids.go](../internal/platform/ids.go) | `bse_` / `tbl_` / `fld_` / `rec_` IDs |

**Why its own SQL instead of go-postgres-rest:** the library ANDs every filter, builds broken SQL for an unknown operator, silently drops the whole sort if one entry is bad, has no `jsonb` operators, and its bulk insert takes columns from the first record only (other records lose column defaults). Identifiers in our SQL come only from metadata and are quoted; values are always parameters.

### Link fields carry their lookups

A link field is returned as `[{"id": "rec_12", "<looked-up field>": value, ...}]`, one object per linked record, holding every lookup through the link. Before, links and lookups came back as parallel arrays (`["rec_12"]` and `["asha@example.com"]`) that clients had to zip together. Lookups no longer have keys of their own. The table schema lists them under the link field, as `lookups`. Writes accept record IDs, or the same objects. Code: `buildRecords`, `linkLookups`, `lookupValues` and `linkInput` in [records.go](../internal/platform/records.go), and `linkLookupFields` in [schema.go](../internal/platform/schema.go).

## 4. Record writes reuse the app's own rules (`internal/services/table/`)

| What | Why | Where |
|---|---|---|
| `CreateRecordsForAPI`, `UpdateRecordsForAPI`, `DeleteRecordsForAPI`: all records of a request in one transaction | Bulk writes must be all-or-nothing, and the primary-key, unique-key, check-constraint and auto-UUID checks are private methods of this service, so the code has to live here | [api_records.go L30](../internal/services/table/api_records.go#L30), [L77](../internal/services/table/api_records.go#L77), [L123](../internal/services/table/api_records.go#L123), [L171](../internal/services/table/api_records.go#L171) |
| Separate interface `APIRecordService` (not added to `TableManagementService`) | Existing mocks and fakes of `TableManagementService` stay untouched | [interfaces/api_records.go L18](../internal/services/interfaces/api_records.go#L18) |
| Link fields are written with the existing link service, one pair at a time, after the record is saved | Keeps both sides of a link and its one/many rules correct; the app has no transactional link write yet | [records.go L676](../internal/platform/records.go#L676) |

## 5. Shared middleware and router (these affect the whole app)

| What | Why | Where |
|---|---|---|
| Request IDs are now `req_` + 32 hex; a client's `X-Request-Id` is kept only if it is at most 64 characters of `A-Z a-z 0-9 _ -` | Spec section 7; stops request IDs from injecting text into logs | [request_id.go L18–32](../internal/middleware/request_id.go#L18-L32) |
| Log lines end with `request_id=…` and, for public API calls, `ws=…` | Trace a request and a workspace in logs | [logging.go L16–40](../internal/middleware/logging.go#L16-L40) |
| `RateLimiter` rewritten on top of a new `RateLimiterBy(key, limit, onLimit)` | The public API limits per workspace, not per IP, and must answer in its own format with `Retry-After` | [validation.go L60–107](../internal/middleware/validation.go#L60-L107) |
| CORS: paths starting `/api/v1/ws_` are open to every origin (no credentials) | Browser apps on any site must call the public API; it has no cookies, so `*` is safe | [cors.go L17–54](../internal/middleware/cors.go#L17-L54) |
| CORS: app routes now also expose `X-Request-Id, Retry-After` | Browser code can read the request ID | [cors.go L77–78](../internal/middleware/cors.go#L77-L78) |
| `UseRawPath = true` on the router | Table names with `/` (`In%2FOut`) must reach the handler | [router.go L68–70](../internal/router/router.go#L68-L70) |
| Public routes registered only when open access is on; a `NoRoute` handler gives `/api/v1/ws_…` paths a JSON 404 | Spec sections 2 and 7 | [router.go L134–144](../internal/router/router.go#L134-L144), field [L57–59](../internal/router/router.go#L57-L59) |

## 6. Config, env and startup

| What | Why | Where |
|---|---|---|
| `PUBLIC_API_OPEN_ACCESS` (default `false`), `PUBLIC_API_RATE_LIMIT_PER_MINUTE` (600), `PUBLIC_API_SEED_WORKSPACES` | Switch, limit and test workspaces | [config.go L30](../internal/config/config.go#L30), [L126–135](../internal/config/config.go#L126-L135), [L210–214](../internal/config/config.go#L210-L214), [L282–285](../internal/config/config.go#L282-L285) |
| Default `CORS_ALLOWED_HEADERS` also lists `Idempotency-Key,X-Request-Id` | App-side calls may send them | [config.go L279](../internal/config/config.go#L279) |
| Server refuses to start with open access on when `SERVER_ENV` is `prod`/`production` | An API without auth must never be open in production | [app.go L59–63](../internal/app/app.go#L59-L63) |
| Seed workspaces at startup (open access only) | Test workspaces without clicking | [app.go L372–375](../internal/app/app.go#L372-L375), [seed_api_workspaces.go L21](../internal/scripts/seed_api_workspaces.go#L21) |
| Hourly cleanup of idempotency keys (open access only) | Keys live 24 hours | [app.go L340–344](../internal/app/app.go#L340-L344) |
| `.env.example`: the three settings (open access `false`) and the CORS header additions | Fresh setups start with the API off | `.env.example` L123, L125–135 |

## 7. Errors and response codes

| What | Where |
|---|---|
| New sentinel `WorkspaceNotFound` (mapped to the existing `WSP_3006`) | [app_errors.go L147](../internal/app-errors/app_errors.go#L147), [errorMapping.go L95–96](../internal/utils/response/constants/errorMapping.go#L95-L96) |
| `WorkspaceAPIPublicIDFailed` and `RecordWriteError` (carries the failing record's position) | [app_errors.go L150–165](../internal/app-errors/app_errors.go#L150-L165) |
| New codes `WSA_SUCCESS_2001/2002`, `WSA_1001` | [constants/workspace_api.go](../internal/utils/response/constants/workspace_api.go), registered in [constants.go L54](../internal/utils/response/constants/constants.go#L54), [L70](../internal/utils/response/constants/constants.go#L70) |

## 8. Frontend (`services/base-ui`)

| What | Why | Where |
|---|---|---|
| Third home-page card **Workspace API** | Entry point to the feature | [HomePage.tsx L18](../services/base-ui/src/pages/HomePage.tsx#L18), [L502–503](../services/base-ui/src/pages/HomePage.tsx#L502-L503), [WorkspaceApiCard.tsx](../services/base-ui/src/components/platform/WorkspaceApiCard.tsx) |
| Developer popup: left nav (Overview active, the rest "Soon") | Matches the mockup; future stages fill the other sections | [WorkspaceApiModal.tsx](../services/base-ui/src/components/platform/WorkspaceApiModal.tsx) |
| Overview: "Enable API & app login?" card, then the Base URL, API ID and an example call with copy buttons | Spec section 1; owners/co-owners only | [DeveloperOverview.tsx](../services/base-ui/src/components/platform/DeveloperOverview.tsx) |
| Two API calls (status, enable) and `getApiOrigin()` | The SDK has no methods for these endpoints | [clientService.ts L673–702](../services/base-ui/src/service/clientService.ts#L673-L702), hooks in [useWorkspaceApi.ts](../services/base-ui/src/components/platform/useWorkspaceApi.ts) |
| **API docs page** at `/workspace/:id/developer/api-docs`: every public endpoint with request (cURL or JavaScript) and response examples, field value formats, safe retries, errors and limits. Examples use the workspace's real base URL once its API is on. | Developers need in-app help to use the endpoints | [ApiDocsPage.tsx](../services/base-ui/src/pages/ApiDocsPage.tsx), content and building blocks in [platform/docs/](../services/base-ui/src/components/platform/docs/), route in [App.tsx L37](../services/base-ui/src/App.tsx#L37), [L377–378](../services/base-ui/src/App.tsx#L377-L378) |
| **API docs** button in the Developer popup header, and a "Read the API docs" link on the enabled Overview; both open the page in a new tab, and the popup stays open | Ways to open the docs page | [WorkspaceApiModal.tsx](../services/base-ui/src/components/platform/WorkspaceApiModal.tsx), [DeveloperOverview.tsx](../services/base-ui/src/components/platform/DeveloperOverview.tsx) |

## 9. Tests and docs

- **New tests:**
  - `tests/platform/` (IDs, public ID service, the API over sqlmock, and `api_db_test.go`, which runs every endpoint against real Postgres in a throwaway schema when `PLATFORM_TEST_PG_PASSWORD` is set)
  - `tests/handlers/workspace_api_test.go` (with a generated mock)
  - `tests/router/public_api_routes_test.go`
  - `tests/middleware/rate_limiter_by_test.go`, `tests/middleware/cors_public_api_test.go`
  - `tests/app/public_api_test.go`
  - base-ui: `src/components/platform/__tests__/` and `src/pages/__tests__/ApiDocsPage.test.tsx`
- **Changed tests:**
  - `tests/middleware/request_id_test.go`: now expects the `req_` format, with 2 new cases.
  - Router tests: list the new routes and add the handler.
- **Docs:**
  - [API_PLATFORM_FOUNDATION.md](API_PLATFORM_FOUNDATION.md), updated to what was built.
  - [api-platform/README.md](api-platform/README.md) (quick start) and [api-platform/openapi.yaml](api-platform/openapi.yaml) (validated with Redocly).
  - [ENVIRONMENT_VARIABLES.md](ENVIRONMENT_VARIABLES.md).

---

## Side effects on existing features

Most of the work is new files that only run for `/api/v1/ws_…` paths or the new Developer popup. The items below are the places where existing behaviour changes. They're ordered by how much attention they deserve.

| # | Change | Who notices | Risk | What to do |
|---|---|---|---|---|
| 1 | **Schema triggers on `bases`, `models`, `columns`** ([databaseConstant.go L1089–1154](../internal/constant/databaseConstant.go#L1089-L1154)). Every insert/delete, and every update that really changes a title, type, options, order or delete flag, now runs a small trigger: for fields, one lookup in `bases` plus one `UPDATE workspace_api` (which matches nothing for workspaces without the API). | Creating, renaming and deleting bases, tables and fields; imports that create many fields | Low. A few milliseconds per changed row. Record (data) writes are not affected. For workspaces with the API on, two schema changes at the same moment in the same workspace briefly queue on one row. | After the rebuild, check the startup log shows `workspace_api` was created and has no `install schema version triggers` warning. If `workspace_api` were missing while the triggers exist, saving bases, tables or fields would fail. |
| 2 | **Request ID format** for every request, app API included ([request_id.go L18–32](../internal/middleware/request_id.go#L18-L32)). Generated IDs change from `3f2a9c1e-…` to `req_3f2a9c1e…`, and a client-sent `X-Request-Id` with characters outside `A-Z a-z 0-9 _ -` (or longer than 64) is replaced instead of kept. | Anything that parses request IDs as UUIDs, or sends IDs containing `.` `:` `/` | Low | Search clients and log tooling for UUID parsing of `X-Request-Id`; none found in this repo. |
| 3 | **Log line format** ([logging.go L16–40](../internal/middleware/logging.go#L16-L40)): ` request_id=…` (and ` ws=…`) appended to each request line | Log parsers or alerts that match the whole line | Low | Update any strict log patterns. |
| 4 | **Router matches on the raw path** ([router.go L68–70](../internal/router/router.go#L68-L70)). URLs containing encoded characters such as `%2F` are matched before decoding; path values are still decoded. | A route parameter containing an encoded `/`: it now reaches the handler instead of returning 404 | Very low; IDs in this app are UUIDs | None. Existing router tests pass. |
| 5 | **New JSON 404 for `/api/v1/ws_…`** ([router.go L134–144](../internal/router/router.go#L134-L144)). Before, there was no `NoRoute` handler. | Only `/api/v1/ws_…` paths; all other unknown paths keep gin's plain "404 page not found" | None | — |
| 6 | **CORS** ([cors.go](../internal/middleware/cors.go)). App routes now also send `Access-Control-Expose-Headers` (additive). `/api/v1/ws_…` ignores the `CORS_*` settings and allows every origin. | Browser apps | None for the app. For dev servers with open access on, any page in a developer's browser could call the open API (e.g. `localhost:8080`). | Accepted for development; production can't have it on. |
| 7 | **Auth rate limiter rewritten** ([validation.go L60–107](../internal/middleware/validation.go#L60-L107)): same 60-second window per IP and same 429 body; only internals moved into `RateLimiterBy`, plus a guard against a crash when the limit is 0 | Login, forgot/reset password, OTP | None | Existing limiter tests pass. |
| 8 | **New startup failure, by design** ([app.go L59–63](../internal/app/app.go#L59-L63)): `PUBLIC_API_OPEN_ACCESS=true` with `SERVER_ENV=prod` or `production` stops the server | Production deploys with a wrong `.env` | Intended | Keep it `false` outside dev. |
| 9 | **Default CORS headers** ([config.go L279](../internal/config/config.go#L279)) now include `Idempotency-Key,X-Request-Id`. This only applies when `CORS_ALLOWED_HEADERS` isn't set. | Setups without that variable | None (additive) | — |
| 10 | **Records created through the API have no "Created by"** (and updates don't change "Last modified by"), because there's no caller identity until API keys exist | People viewing those records in the app | Cosmetic | Fills in automatically once keys identify the caller. |
| 11 | **Home page layout**: a third 300 px card makes the greeting text narrower on wide screens | Everyone on the home page | Cosmetic | Narrow the cards if it looks cramped. |
| 12 | **Your local `.env`** (not committed) now has `PUBLIC_API_OPEN_ACCESS=true` and the CORS header additions | Only your machine | Your dev workspace data is readable and writable by anyone who can reach your server once you enable its API | Set it back to `false` when not testing. |

**Unchanged:**
- Every existing endpoint, handler and service method.
- The `TableManagementService` interface and its mocks.
- How records are stored and the existing triggers.
- The base-ui's existing API calls and routes (the docs page adds one new route).
- With `PUBLIC_API_OPEN_ACCESS=false` (the default in `.env.example`), none of the public routes exist. In that mode the only runtime effects of this change are items 1–9 above.

## Not part of this change

- **`/loginNew`:** the `Login` handler in `internal/handlers/auth.go` and its `/loginNew` route were in the working tree when this work started. They're gone now and were not removed by this work.
- **Other untracked files:** `clamav-db/` and the other untracked docs (`API_PLATFORM_ROADMAP_PLAN.md`, `FEATURE_STATUS.md`, `RELEASE_*`) are unrelated.

## After rebuilding: quick check

1. **Startup log:** shows `Table "public".workspace_api created successfully` and `api_idempotency_keys`, with no schema-trigger warning.
2. **App still works:** create a table and a field in the app, which exercises the new triggers.
3. **Enable the API:** go to **Home → Workspace API → Enable**, copy the Base URL, then run `curl <base-url>/bases`.
4. **Database test (optional):**
   ```
   make test-public-api PLATFORM_TEST_PG_PASSWORD=<postgres password>
   ```
   This takes about 25 s, because it waits out the 10 s workspace cache twice: once to check that turning the API off gives 404, once to check `ambiguous_name`. It also covers a bulk create that Postgres rejects partway through, and checks that nothing is written.

# API Platform — Stage 1: Workspace API Foundation

**Status:** Built (steps 1–29); open only in dev behind `PUBLIC_API_OPEN_ACCESS` · **Updated:** 2026-11-04

**API docs:** [quick start](api-platform/README.md) · [OpenAPI 3 spec](api-platform/openapi.yaml)

This document turns the Stage 1 API draft into a build spec, checked against the current SereniBase backend. It covers the public ID for each workspace, the endpoints, the request and response format, the list and write rules, the field type mapping and the request pipeline.

**Out of scope:** security (keys, login, rules), SDKs, and the per-workspace API docs.

**Subtask 1** is covered by sections 1 to 6. Still to do for it: the base OpenAPI spec and basic API docs written from these decisions.

---

## Can we build this?

Yes. It fits as a new package inside sereni-base (`internal/platform`) that reuses the existing table services and the go-postgres-rest library. It does not need a new service or any new dependency.

**Confirmed against the code:**

- **Routing:** `/api/v1/:ws/...` works next to every existing `/api/v1/<name>` group on our Gin version (v1.12.0), checked with a routing test. Existing routes always win. Anything else falls through to the workspace lookup, which returns `404 not_found` for an unknown workspace (for example `/api/v1/auth/bases` → workspace `auth` → 404).
- **Atomic bulk writes:** go-postgres-rest's `BulkInsert` and `BulkUpdate` each run in one transaction, and `BulkDelete` is a single `DELETE ... WHERE id IN (...)`. All-or-nothing bulk writes come from the library as is.
- **Cursor paging:** the library's filter groups can nest (`ComplexFilter.Groups`). In the build, record reads and writes use their own parameterized SQL instead (see [Decisions](#10-decisions), 7).
- **nginx:** `location /api/v1/` uses `proxy_pass` without a URI, so URL-encoded names reach the app unchanged.

**Where the draft had to change to fit the code:**

| Draft | What the code does today | Decision in this doc |
|---|---|---|
| `workspace_api` written as plain SQL with `TIMESTAMPTZ`, `created_at`/`updated_at` and `CHECK`s | All tables are created at startup from Go `TableSchema()` definitions in [create_db.go](../internal/scripts/create_db.go). They use `timestamp` (UTC) and `created_time`/`last_modified_time`. The pinned go-postgres-rest only accepts single-column `CHECK`s of the form `<column> <op> <literal>`, and its `CreateTable` rejects `bigint`. | Same table, defined as a `TableSchema()` and following the existing column conventions, with `schema_version` as `integer`. The `CHECK`s are added with raw SQL right after the table is created. |
| `schema_version` bumped "from the existing services" in the same transaction | Schema changes in the services don't share a transaction. | Bump it with Postgres triggers on `bases`, `models` and `columns`. A trigger always runs in the same transaction as the change, and the codebase already installs triggers this way ([link_keys.go](../internal/scripts/link_keys.go)). |
| Bases, tables and fields addressed by name | Names are not unique. There is no unique index, and `BaseAlreadyExists` is defined but never used. | A name that matches more than one item returns `409 ambiguous_name`. IDs always work. |
| Random-looking record IDs (`rec_9x2k...`) | Every user table's `id` column is a `BIGSERIAL`. | `rec_<id>`, e.g. `rec_1042` (see [Decisions](#10-decisions)). |
| Names in the URL are URL-encoded | `gin.Default()` matches routes on the decoded path, so a table named `In/Out` (`In%2FOut`) returns 404. Checked with a test. | Set `UseRawPath = true` on the Gin engine. The same test shows existing routes are unaffected. |
| Draft field type list | SereniBase's own list has no rollup or auto number, and adds rating, duration, year, time, user, JSON and UUID. | Mapping rewritten against the real types (section 6). |
| `req_...` request IDs | The global `RequestID` middleware already sets `X-Request-Id`, but as a plain UUID, and it accepts any value a client sends. | Change that middleware to generate `req_...` IDs and to check client-supplied values. |
| Open API in "dev and staging, never production" | `SERVER_ENV` only has `dev` and `prod`. | An explicit `PUBLIC_API_OPEN_ACCESS` flag. The server refuses to start with it on when `SERVER_ENV=prod`. |
| "The seed script" enables test workspaces | There is no seed script for test data, only `create_default_rbac` and `register_owner`. | Add one (step 10 of the [implementation guide](#9-implementation-guide)). |

---

## 1. Workspace ID

Each workspace gets a fixed public ID: `ws_` + 10 random characters from `0-9a-z`, e.g. `ws_8f3k2m9qx7`. It is created once, never changes, and is stored in a new `workspace_api` table. The existing workspace `slug` stays as it is.

### Table

Defined as `tenant.WorkspaceAPI` in `internal/models/tenant/workspace_api.go` and created in `CreateMasterSchema` after `workspaces`. The equivalent SQL:

```sql
CREATE TABLE "public".workspace_api (
    workspace_id       varchar      PRIMARY KEY
                       REFERENCES "public".workspaces(id) ON DELETE CASCADE,
    public_id          varchar(16)  NOT NULL UNIQUE,
    enabled            boolean      NOT NULL DEFAULT false,
    enabled_at         timestamp,
    schema_version     integer      NOT NULL DEFAULT 1,
    created_time       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_modified_time timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Returned by WorkspaceAPI.CheckConstraints() and run with raw SQL after creation, because
-- go-postgres-rest only accepts single-column CHECKs of the form <column> <op> <literal>.
-- Each runs on its own, and "already exists" errors on later startups are ignored.
ALTER TABLE "public".workspace_api
    ADD CONSTRAINT workspace_api_public_id_format CHECK (public_id ~ '^ws_[0-9a-z]{10}$');
ALTER TABLE "public".workspace_api
    ADD CONSTRAINT workspace_api_enabled_at CHECK (enabled = false OR enabled_at IS NOT NULL);
```

`workspaces.id` is a `varchar` holding a UUID, so `workspace_id` is `varchar` too. Times are stored in UTC, as in every other table.

`schema_version` is `integer` because go-postgres-rest's `CreateTable` rejects `bigint`. That still allows over 2 billion schema changes per workspace.

### Rules

- **Created:** the first time the API is enabled for a workspace. Workspaces that never enable it have no row.
- **Generation:** `crypto/rand`, using rejection sampling over the 36-character alphabet so every character is equally likely. On a unique violation, retry with a new ID (up to 5 times).
- **Lookup:** `WHERE public_id = $1 AND enabled = true`, cached. A disabled workspace and a missing one both return `404 not_found`.
- **Cache:** each entry holds `workspace_id` and `schema_version` and expires after a short TTL (10 s). Unknown IDs are cached as misses for the same TTL, so random IDs can't hammer the database. A disable or schema change reaches every instance within the TTL, with no invalidation calls added to the existing services.
- **Disabling:** set `enabled = false` and keep `public_id`, so the address stays the same if the API is enabled again.
- **`last_modified_time`:** set by the app on every update, as the other tables do.
- **Test workspaces:** a new seed script creates the row with `enabled = true` and `enabled_at = now()`.
- **Not a secret:** `public_id` only identifies the workspace. API keys come in the security stage.

### `schema_version`

Goes up by 1 whenever a base, table or field in the workspace is created, renamed, deleted, reordered or changes type or settings. Postgres `AFTER ... FOR EACH ROW` triggers do the bump, so it always happens in the same transaction as the change:

| Table | Fires on | Workspace found via |
|---|---|---|
| `bases` | `INSERT`, `DELETE`, `UPDATE OF title, status` | `bases.workspace_id` |
| `models` (tables) | `INSERT`, `DELETE`, `UPDATE OF title, deleted, order_index` | `models.workspace_id` |
| `columns` (fields) | `INSERT`, `DELETE`, `UPDATE OF title, uidt, dt, meta, deleted, order_index` | `columns.base_id` → `bases.workspace_id` |

One trigger function, `workspace_api_bump_schema_version`, runs `UPDATE "public".workspace_api SET schema_version = schema_version + 1 WHERE workspace_id = ...`. If the workspace has never enabled the API there is no row, so nothing happens. The function is installed with the other `DefinedFunctions`, and the triggers by `InstallSchemaVersionTriggers` ([schema_version.go](../internal/scripts/schema_version.go)) on every start. Each table has an `INSERT OR DELETE` trigger and an `UPDATE` trigger whose `WHEN` clause skips updates that change none of the listed columns, so saving a field without changing it doesn't invalidate caches. `columns.meta` is included because it holds select options and the auto/manual UUID setting, which the API validates against.

The app deletes bases, tables and fields with `DELETE` today (no soft delete), so the `INSERT OR DELETE` triggers do the work; `status` and `deleted` stay in the lists in case soft delete is added.

**Used now for:** the name-to-ID cache, which is keyed by `schema_version`. After a rename the old entries are simply never read again, so nothing has to clear them.

**Later uses:** the cache key and ETag for per-workspace docs, a trigger for regenerating typed code, and a way for SDKs to notice schema changes.

---

## 2. Endpoints

Base path: `/api/v1/{ws}`. All paths below are relative to it.

| Method | Path | Purpose |
|---|---|---|
| GET | `/bases` | List bases |
| GET | `/bases/{base}/tables` | List tables in a base |
| GET | `/bases/{base}/tables/{table}` | One table with its fields |
| GET | `/bases/{base}/tables/{table}/records` | List records |
| GET | `/bases/{base}/tables/{table}/records/{id}` | Get one record |
| POST | `/bases/{base}/tables/{table}/records` | Create one or many |
| PATCH | `/bases/{base}/tables/{table}/records/{id}` | Update one |
| PATCH | `/bases/{base}/tables/{table}/records` | Bulk update |
| DELETE | `/bases/{base}/tables/{table}/records/{id}` | Delete one |
| POST | `/bases/{base}/tables/{table}/records/delete` | Bulk delete |

Example: `GET /api/v1/ws_8f3k2m9qx7/bases/crm/tables/orders/records?limit=20`

**Routing notes:**

- The routes are registered as a Gin group `/api/v1/:ws` next to the existing groups.
- No internal route group may ever start with `ws_`, because it would shadow workspace addresses.
- The engine has no `NoRoute` handler today, so unknown paths get Gin's plain-text 404. Add one that returns the new envelope for paths starting with `/api/v1/ws_`, and keeps today's behaviour for everything else.

---

## 3. Names or IDs

`{base}`, `{table}` and field names (in request bodies and query parameters) accept either a name or an ID.

| Object | ID format | Source |
|---|---|---|
| Base | `bse_` + UUID without dashes (32 hex characters) | `bases.id` |
| Table | `tbl_` + UUID without dashes | `models.id` |
| Field | `fld_` + UUID without dashes | `columns.id` |
| Record | `rec_` + integer | the user table's `id` (`BIGSERIAL`) |

- **ID or name:** a value that starts with a known prefix is treated as an ID; anything else is a name. A base literally named `bse_...` can only be reached by its ID.
- **Matching:** exact, case-sensitive match on the title. Base names are looked up within the workspace, table names within the base, and field names within the table.
- **Deleted items:** soft-deleted tables and fields (`deleted = true`) and bases that are not `active` don't exist as far as the API is concerned. They return `404 not_found`. (The app currently hard-deletes, so these filters are a safeguard.)
- **Records:** fields are keyed by name. When two fields of a table share a name, both are keyed by their `fld_` ID in record responses, so no value is hidden.
- **Duplicate names:** a name that matches more than one item returns `409 ambiguous_name`, with the matching IDs listed in `details`. IDs always work.
- **Characters that clash with query syntax:** `select` is comma-separated and `filter[...]` uses brackets, so a field whose name contains `,`, `[` or `]` must be referenced by its ID in query parameters.
- **Reserved names:** in `sort`, `select` and `filter`, the names `id`, `created_at` and `updated_at` mean the system values. A user field with one of those names must be referenced by its ID in query parameters. Request bodies are not affected, because user fields always sit inside `fields`.
- **URL encoding:** names in the path are URL-encoded (`Sales%20Team`, `In%2FOut`). This needs `UseRawPath = true` on the Gin engine; `UnescapePathValues` stays at its default of `true`.
- **Cache:** name-to-ID maps are cached per `(workspace_id, schema_version)` and built per base or table the first time they're used.

Apps that must survive renames should use IDs.

---

## 4. Request and response format

### Record

User fields sit inside `fields`, so a user field named `id` never clashes with ours. Fields are keyed by name by default; `?keys=id` keys them by field ID (`fld_...`) instead.

```json
{
  "id": "rec_1042",
  "fields": { "Name": "Asha", "Status": "Active", "Price": 499 },
  "created_at": "2026-11-03T10:15:00Z",
  "updated_at": "2026-11-03T10:15:00Z"
}
```

Every user table also has the system columns Id, Created Time, Last Modified Time, Created By and Last Modified By. They never appear inside `fields`. Id and the two times are returned as the top-level `id`, `created_at` and `updated_at`.

### Envelope

Every response has the same three keys. Lists add `next_cursor`, `has_more` and `limit` to `meta`.

```json
{ "data": ..., "error": null, "meta": { "request_id": "req_..." } }
```

This is deliberately different from the internal API's envelope (`success`, `message`, `data`, `meta.code`, `meta.http_status` in [response.go](../internal/models/response.go)). The internal API keeps its format. The new helpers live in `internal/platform` and are used only by `/api/v1/{ws}` routes.

### Errors

```json
{
  "data": null,
  "error": {
    "code": "validation_failed",
    "message": "Some fields have invalid values.",
    "details": [{ "index": 0, "field": "Price", "issue": "must be a number" }]
  },
  "meta": { "request_id": "req_..." }
}
```

In bulk requests, each entry in `details` carries `index`, its position in the request array.

| Code | HTTP | When |
|---|---|---|
| `bad_request` | 400 | Malformed JSON, a bad query parameter, a cursor reused with a different filter or sort, or the same ID twice in one bulk request |
| `not_found` | 404 | Workspace, base, table, field or record doesn't exist |
| `ambiguous_name` | 409 | A name matches more than one base, table or field |
| `conflict` | 409 | Same idempotency key with a different body, or while the first request is still running |
| `payload_too_large` | 413 | Bulk request over the limit |
| `validation_failed` | 422 | Wrong type, unknown field, write to a read-only field, unknown select option, link field in a bulk write |
| `rate_limited` | 429 | Too many requests (with a `Retry-After` header) |
| `internal_error` | 500 | Server error |

`unauthorized` and `forbidden` arrive with the security stage.

Unknown fields in a write are rejected, not ignored, so typos surface immediately.

---

## 5. List options and write rules

### List options (on `GET /records`)

| Option | Example | Rule |
|---|---|---|
| `limit` | `?limit=50` | Default 50, max 200 |
| `cursor` | `?cursor=eyJ...` | Opaque; taken from `next_cursor` |
| `select` | `?select=Name,Status` | Only these fields are returned |
| `sort` | `?sort=-created_at,Name` | `-` means descending. Default is `id` ascending, and `id` is always added as the last tie-breaker. |
| `filter` | `?filter[Status]=Active&filter[Price][gte]=100` | Operators: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `contains`, `is_null`. All filters are combined with AND. |
| `keys` | `?keys=id` | Key fields by ID instead of name |

- **`in`:** takes a comma-separated list.
- **`is_null`:** takes `true` or `false`.
- **`contains`:** on text fields, a case-insensitive substring match. On multi-select and link fields, it means the field includes the value.
- **OR and nested filters:** these come with advanced queries (roadmap item 1.4).

**Cursor:** base64url-encoded, containing a version number, the position, and a hash of the filter and sort. Using it with a different filter or sort returns `400 bad_request`. Stage 1 may store a plain offset as the position. Because the cursor is opaque, it can later switch to keyset paging (last sort values + `id`, using nested `ComplexFilter` groups) without any client change.

### Write rules

| Action | Body | Success |
|---|---|---|
| Create one | `{ "fields": {...} }` | `201`, the record |
| Create many | `{ "records": [{ "fields": {...} }] }` | `201`, array of records |
| Update one | `{ "fields": {...} }` | `200`, the record |
| Bulk update | `{ "records": [{ "id": "rec_...", "fields": {...} }] }` | `200`, array of records |
| Delete one | — | `200`, `{ "id": "rec_...", "deleted": true }` |
| Bulk delete | `{ "ids": ["rec_..."] }` | `200`, array of `{ "id", "deleted": true }` |

- **PATCH:** changes only the fields sent. There is no PUT for now.
- **Bulk limit:** 100 records per request, all-or-nothing. Every record is validated before anything is written, so a `422` means nothing changed. The write itself is one `BulkInsert`, `BulkUpdate` or `BulkDelete` call, each of which is atomic.
- **Bulk delete with a missing ID:** returns `404 not_found` listing the missing IDs, and nothing is deleted.
- **Link fields:** can be set on single-record writes only. In bulk writes they return `422 validation_failed`, because link values are saved in a second step, outside the bulk write's transaction (see [Decisions](#10-decisions)).
- **Existing write guards still apply:** primary-key checks, auto-generated UUIDs and table constraints run on API writes exactly as they do for the app.
- **`X-Request-Id`:** returned on every response, and it matches `meta.request_id`.

### Idempotency

The `Idempotency-Key` header is accepted on POST (create and bulk delete), up to 255 characters, and kept for 24 hours.

Stored in a new table, `api_idempotency_keys`:

| Column | Notes |
|---|---|
| `workspace_id`, `key` | Primary key. Scoped per workspace for now, and per API key once keys exist. |
| `request_hash` | SHA-256 of method, path and body |
| `status` | `in_progress` or `done` |
| `status_code`, `response_body` | The stored response to replay |
| `created_time` | Rows older than 24 hours are deleted by a periodic cleanup |

- **Same key, same body:** the stored response is replayed.
- **Same key, different body:** `409 conflict`.
- **Same key while the first request is still running:** `409 conflict`. The row is inserted as `in_progress` before the handler runs.

---

## 6. Field type mapping

Based on the field types in base-ui ([Fields/index.ts](../services/base-ui/src/components/common/Fields/index.ts)) and the backend's `uidt` values. The backend spells some `uidt` values inconsistently (`longText` and `longtext` both appear), so the converter matches on the lowercased value.

The `type` column is what `GET /bases/{base}/tables/{table}` reports. Storage was checked against the code (step 21); where the app stores a value differently from its JSON form, the API converts both ways.

| Field type (`type`) | JSON | Writable | Stored as |
|---|---|---|---|
| Text, long text (`text`, `long_text`) | string | Yes | TEXT |
| Email, URL, phone number (`email`, `url`, `phone_number`) | string, checked against the type (`""` clears) | Yes | TEXT |
| Number (`number`) | whole number in the 32-bit range | Yes | INTEGER |
| Decimal, currency, percent (`decimal`, `currency`, `percent`) | number; digits kept exactly | Yes | NUMERIC |
| Rating (`rating`) | whole number from 0 to the field's maximum (default 5) | Yes | INT |
| Year (`year`) | whole number from 1 to 9999, e.g. `2026` | Yes | INTEGER |
| Date (`date`) | `"2026-11-03"` | Yes | DATE |
| Date and time (`date_time`) | RFC 3339; any offset on write, returned in UTC: `"2026-11-03T10:15:00Z"` | Yes | TIMESTAMP (UTC) |
| Time (`time`) | `"10:15"` (24-hour `HH:mm`, as the app stores it) | Yes | TEXT |
| Duration (`duration`) | whole number of **minutes**, 0 or more | Yes | TEXT |
| Checkbox (`checkbox`) | boolean | Yes | BOOLEAN |
| Single select (`single_select`) | option name (string) | Yes | TEXT |
| Multi select (`multi_select`) | array of option names | Yes | TEXT[] |
| JSON (`json`) | any JSON value | Yes | TEXT (the JSON text) |
| UUID (`uuid`) | string | No if auto-generated, otherwise yes | UUID |
| Links (`links`) | array of objects, one per linked record: `{"id": "rec_12", "<looked-up field>": value, ...}`, carrying every lookup that goes through the link, keyed by the looked-up field's name in the linked table (its `fld_` ID with `?keys=id`, or when the name is `id` or shared) | Yes, on single-record writes: an array of record IDs, or of objects with an `id` (other keys ignored), so a value read can be sent back | JSONB array of the linked records' primary-key objects |
| Lookup (`lookup`) | not a key of its own: its value is in the objects of its link field. The table schema lists lookups under the link field (`lookups: [{key, id, name, type}]`); `select=<lookup>` returns the link field with only that lookup. | No | computed |
| Attachment (`attachment`) | array of `{ id, url, name, size, type }` | Later | JSONB[] of asset objects |
| User (`user`) | user ID string, or an array of IDs when the field allows several | No (for now) | TEXT (IDs joined with commas) |
| Formula (`formula`) | the stored result (string) | No | TEXT |
| Created time, last modified time (`created_time`, `last_modified_time`) | date and time string | No | TIMESTAMP |
| Created by, last modified by (`created_by`, `last_modified_by`) | string | No | TEXT |

- **Empty values:** returned as `null`, or `[]` for array types (multi select, links, lookup, attachment, multi-user).
- **Select options:** writing an option that doesn't exist on a select field returns `422 validation_failed`.
- **Links:** stored as the linked records' primary-key objects (`{"id": 12}`, or the key columns on a table with a custom primary key). The API resolves them to `rec_` IDs; `filter[Field][contains]=rec_12` matches by the linked record. A field that holds one record (one-to-one, or the "one" side of has-many) accepts at most one ID.
- **Buttons** are not data and never appear in `fields`. **Fields of unknown types** are returned read-only, as stored.
- **System columns** (`id`, `created_time`, `last_modified_time`, `created_by`, `last_modified_by`) are never fields. API writes set the two times; `created_by` / `last_modified_by` are left as they are until API keys identify a caller.
- **Removed from the draft:** rollup and auto number. They are not SereniBase field types; the system Id is an auto-number and is exposed as the record `id`.

---

## 7. Request pipeline

Every `/api/v1/{ws}` request passes through the same chain, in this order.

**Already global, for every route:** CORS, `RequestID`, `RequestLogger`, `DatabaseQueryLogger`, `Recovery`.

1. **Request ID:** the existing global `RequestID` middleware, changed to generate `req_` + a UUID without dashes. A client-supplied `X-Request-Id` is kept only if it is at most 64 characters of `[A-Za-z0-9_-]`; otherwise a new ID replaces it, so request IDs can't be used to inject text into logs.
2. **Resolve workspace:** `public_id` → `workspace_id`, from the cache or else `workspace_api`. Returns `404 not_found` if the workspace is missing or disabled.
3. **Auth placeholder:** a no-op for now. Real keys plug in here during the security stage.
4. **Rate limit:** per workspace, configurable via `PUBLIC_API_RATE_LIMIT_PER_MINUTE` (default 600). Returns `429 rate_limited` with `Retry-After`.
   - The existing [RateLimiter](../internal/middleware/validation.go) is keyed by client IP. Add a variant that takes a key function.
   - It counts in memory, which is fine for one instance. Running several instances needs a shared store; that's planned for the security stage.
5. **Handler:** resolves base, table and field names to IDs, then runs the operation.

### Dev-only switch

`PUBLIC_API_OPEN_ACCESS` (default `false`). The `/api/v1/{ws}` routes are registered only when it is `true`, and the server refuses to start with it on while `SERVER_ENV=prod`. This keeps an API without authentication out of production, whatever the deployment's config says.

### Request logging

[RequestLogger](../internal/middleware/logging.go) already logs method, path, status and duration. Add the request ID and the workspace public ID, both read from the Gin context.

### CORS

**The public API is open to every origin** (decided 2026-11-04), so browser apps on any site, including the mini-frontends, can call it with no CORS configuration. The global CORS middleware handles every path starting with `/api/v1/ws_` itself and does not use the `CORS_*` settings for them:

- `Access-Control-Allow-Origin: *`, with no `Allow-Credentials`. The API uses no cookies or sessions, so the wildcard is safe; access will come from API keys sent in a header, which a wildcard origin allows.
- **Methods:** `GET, POST, PATCH, DELETE, OPTIONS`.
- **Headers:** a preflight gets back exactly the headers it asks for (`Access-Control-Request-Headers`), so client libraries' own headers pass. Otherwise `Content-Type, Idempotency-Key, X-Request-Id, Authorization`.
- **Exposed headers:** `X-Request-Id, Retry-After, Idempotent-Replayed`, so browser apps can read them.
- **Preflights** are answered with `204` by the middleware, before the workspace lookup, and cached by browsers for 10 minutes (`Access-Control-Max-Age: 600`).

The app's own API keeps its origin list from config (`CORS_ALLOWED_ORIGINS`), and also exposes `X-Request-Id` and `Retry-After`.

While there is no authentication, an open origin also means any web page a developer visits could call a dev server the developer's browser can reach (e.g. `localhost:8080`). This is accepted for development; production never has the API open.

---

## 8. Where the code goes

```
internal/platform/
  api.go           // API type, route registration, pipeline: resolve workspace, auth placeholder, rate limit
  envelope.go      // {data, error, meta} and error codes
  resolve.go       // workspace and name-to-ID lookup, with caches
  workspace.go     // public ID generation and enable
  ids.go           // bse_/tbl_/fld_/rec_ encode and decode
  schema.go        // base and table handlers
  fields.go        // field value conversion, per field type
  query.go         // list options (select, filter, sort, cursor) and their SQL
  records.go       // record handlers, link and lookup resolution
  idempotency.go
internal/handlers/workspace_api.go             // app-facing status and enable endpoints (internal API)
internal/scripts/seed_api_workspaces.go        // enables the API for test workspaces
internal/scripts/schema_version.go             // installs the schema_version triggers
internal/services/table/api_records.go         // exported write methods for the API
internal/services/interfaces/api_records.go
internal/models/tenant/workspace_api.go
internal/models/tenant/api_idempotency_key.go
```

The write methods go in `internal/services/table` rather than `internal/platform` because the guards they need are unexported methods of `tableManagementService`: `checkKeyValues`, `autoUUIDColumns`, `keyWriteError` and the table-constraint guard. `internal/platform` only handles HTTP, name resolution and the envelope. The write methods are a separate interface (`interfaces.APIRecordService`) rather than additions to `TableManagementService`, so the existing mocks and fakes of that interface don't change; `app.go` type-asserts the table service to `platform.RecordWriter`.

Tests: `tests/platform/` (IDs, the public ID service, the API over sqlmock, and `api_db_test.go`, which runs every endpoint against a real Postgres in a throwaway schema when `PLATFORM_TEST_PG_PASSWORD` is set; run it with `make test-public-api`), `tests/router/public_api_routes_test.go`, `tests/middleware/rate_limiter_by_test.go`, `tests/app/public_api_test.go`.

---

## 9. Implementation guide

Work through the steps in order. Each step builds on the ones before it, and its "Done when" check should pass before moving on. Steps 11–17 are the foundation the draft estimated at 2–3 h.

| # | Phase | Step | What to do | Files | Done when |
|---|---|---|---|---|---|
| 1 | Setup | Branch and package | Branch from `master` (e.g. `feature/api-platform-foundation`). Create `internal/platform/` with the empty files from section 8. | `internal/platform/*` | `go build ./...` passes |
| 2 | Setup | Config | Add a `PublicAPIConfig` (`OpenAccess`, `RateLimitPerMinute`, `SeedWorkspaces`) to `Config`. Bind `PUBLIC_API_OPEN_ACCESS` (default `false`), `PUBLIC_API_RATE_LIMIT_PER_MINUTE` (default `600`) and `PUBLIC_API_SEED_WORKSPACES` (comma-separated workspace slugs). | `internal/config/config.go` | The values load from `.env` |
| 3 | Setup | Production guard | At startup, before `router.Setup`, return an error if open access is on and `SERVER_ENV=prod`. | `internal/app/app.go` | The server refuses to start with both set |
| 4 | Setup | Env files | Add the three variables to `.env.example` (open access `false`) and to the env docs. Add `Idempotency-Key,X-Request-Id` to `CORS_ALLOWED_HEADERS`. (The mini-frontends need no origins in config: the public API is open to every origin, section 7.) Turn open access on in your local `.env`. | `.env.example`, `docs/ENVIRONMENT_VARIABLES.md` | A fresh `.env` copied from the example starts cleanly |
| 5 | Database | `workspace_api` table | Add `tenant.WorkspaceAPI` with `TableName()` and `TableSchema()` (section 1): primary key `workspace_id`, foreign key to `workspaces` with cascade delete, unique `public_id`. Register it in `CreateMasterSchema` right after `Workspace`. | `internal/models/tenant/workspace_api.go`, `internal/scripts/create_db.go` | Restarting against an existing database creates the table |
| 6 | Database | `CHECK` constraints | Return the two `ALTER TABLE ... ADD CONSTRAINT` statements (section 1) from a `CheckConstraints()` method on `tenant.WorkspaceAPI`, so the whole table is defined in one file. Run each with `repo.DB.Exec` (`addCheckConstraints`) right after the table is created, ignoring "already exists". | `internal/models/tenant/workspace_api.go`, `internal/scripts/create_db.go` | Inserting `public_id = 'abc'` fails |
| 7 | Database | `api_idempotency_keys` table | Add `tenant.APIIdempotencyKey` (section 5): primary key `(workspace_id, key)`, foreign key to `workspaces` with cascade delete, index on `created_time`. Register it in `CreateMasterSchema`. | `internal/models/tenant/api_idempotency_key.go`, `internal/scripts/create_db.go` | The table exists after a restart |
| 8 | Database | `schema_version` triggers | Add one trigger function and the three triggers from section 1 as SQL constants. Install them at the end of `CreateMasterSchema` with `CREATE OR REPLACE FUNCTION` and `DROP TRIGGER IF EXISTS` + `CREATE TRIGGER`, so every restart is safe. | `internal/constant/databaseConstant.go`, `internal/scripts/create_db.go` | Renaming a table in the app bumps `schema_version` for its workspace |
| 9 | Database | Public ID and enable | `NewPublicID()`: `crypto/rand` with rejection sampling. `WorkspaceAPIService.Enable`: insert the row already enabled if it's missing (with a new ID on a `public_id` clash, up to 5 times), or set `enabled = true` and `enabled_at = now()` on an existing row. The app calls it through two internal endpoints used by the Developer popup on the home page: `GET /api/v1/workspace/{id}/api` (any member) and `POST /api/v1/workspace/{id}/api/enable` (owner and co-owner). | `internal/platform/workspace.go`, `internal/handlers/workspace_api.go`, `internal/router/router.go` | Unit test: 10,000 generated IDs all match `^ws_[0-9a-z]{10}$` |
| 10 | Database | Seed script | Enable the API for each slug in `PUBLIC_API_SEED_WORKSPACES` and log its public ID. Call it from `runBeforeServer` after `CreateMasterSchema`, only when open access is on. | `internal/scripts/seed_api_workspaces.go`, `internal/app/app.go` | The startup log prints a `ws_...` ID for each slug |
| 11 | Pipeline | Engine settings | In `router.Setup`, set `r.UseRawPath = true`. Add a `NoRoute` handler: paths starting with `/api/v1/ws_` get the new 404 envelope, and other paths keep today's plain 404. | `internal/router/router.go` | The existing router tests still pass |
| 12 | Pipeline | Request IDs | Generate `req_` + a UUID without dashes. Keep a client's `X-Request-Id` only if it is at most 64 characters of `[A-Za-z0-9_-]`. | `internal/middleware/request_id.go` | Tests cover generated, kept and replaced IDs |
| 13 | Pipeline | Envelope and errors | Success and error helpers that always fill `meta.request_id`. Error code constants mapped to HTTP status (section 4). | `internal/platform/envelope.go` | Every error code renders in the section 4 shape |
| 14 | Pipeline | Workspace resolver | Middleware: `public_id` → workspace, through a 10 s cache that also caches misses. Puts `workspace_id`, `public_id` and `schema_version` in the Gin context. Missing or disabled → `404 not_found`. | `internal/platform/resolve.go`, `internal/platform/api.go` | A disabled workspace returns 404 within 10 s |
| 15 | Pipeline | Auth placeholder and rate limit | Add a no-op auth middleware. Add `RateLimiterBy(keyFn, perMinute, onLimit)` next to the existing `RateLimiter`, keyed by `workspace_id`. `onLimit` returns `429 rate_limited` with `Retry-After`, in the new envelope. | `internal/platform/api.go`, `internal/middleware/validation.go` | Request 601 within a minute gets 429 |
| 16 | Pipeline | Logging and CORS headers | Add the request ID and workspace public ID to `RequestLogger`'s line (from `param.Keys`). Set `Access-Control-Expose-Headers: X-Request-Id, Retry-After` in the CORS middleware, and open `/api/v1/ws_...` to every origin (section 7). | `internal/middleware/logging.go`, `internal/middleware/cors.go` | Log lines show `req_...`, and a browser app on any origin can call the API and read `X-Request-Id` |
| 17 | Pipeline | Register routes | `platform.Register(api, deps)` builds the `/:ws` group with the section 7 chain and the section 2 routes. Call it from `router.Setup` only when open access is on, and build `deps` in `app.go`. | `internal/platform/api.go`, `internal/router/router.go`, `internal/app/app.go` | Routing test: existing `/api/v1/*` routes are unchanged, and `/api/v1/ws_x/bases` reaches the resolver |
| 18 | Data | ID prefixes | Encode and decode `bse_`, `tbl_` and `fld_` ↔ UUID, and `rec_` ↔ integer. Reject malformed values. | `internal/platform/ids.go` | Round-trip unit tests pass |
| 19 | Data | Name lookup | Resolve the base (in the workspace, `status = 'active'`), the table (in the base, `deleted = false`) and fields (in the table, `deleted = false`) by exact, case-sensitive title. No match → `404 not_found`; more than one → `409 ambiguous_name` listing the matching IDs. Cache per `(workspace_id, schema_version)`. | `internal/platform/resolve.go` | Two tables with the same name return 409, and their IDs still work |
| 20 | Data | Schema endpoints | `GET /bases`, `/bases/{base}/tables` and `/bases/{base}/tables/{table}`. The table response lists its fields with ID, name, type, a read-only flag and select options. | `internal/platform/schema.go` | All three return data for a seeded workspace |
| 21 | Data | Field conversion | First check how Duration, Time and User are stored and update their rows in section 6. Then write converters for each lowercased `uidt`, in both directions. System columns never go into `fields`. | `internal/platform/fields.go`, this document | A unit test passes for every field type in section 6 |
| 22 | Data | List and get records | `GET /records` via `GetTableData`: `select`, `filter`, `sort` with `id` as the tie-breaker, and `limit + 1` rows fetched to set `has_more`. Cursor = offset + a hash of the filter and sort. Support `?keys=id`. Add `GET /records/{id}`. | `internal/platform/records.go` | Paging through 3 pages of 50 returns every record exactly once |
| 23 | Data | Write methods | Exported methods to create, update and delete many records. They run the existing guards (`checkKeyValues`, `autoUUIDColumns`, `keyWriteError`, table constraints), validate every record first, then make one `BulkInsert`, `BulkUpdate` or `BulkDelete` call. Add them to the table-management interface. | `internal/services/table/api_records.go`, `internal/services/interfaces/table_management.go` | One bad record in a batch of 100 means nothing is written |
| 24 | Data | Write handlers | Create one/many (`201`), update one/bulk, delete one/bulk, following section 5. Over 100 records → `413`; unknown field → `422`; link field in a bulk write → `422`; missing ID in a bulk delete → `404` and nothing deleted. | `internal/platform/records.go` | Every rule in section 5 has a passing test |
| 25 | Data | Idempotency | Middleware on the two POST routes: insert the key as `in_progress`, run the handler, store the response. Same body → replay; different body or still running → `409 conflict`. An hourly ticker deletes rows older than 24 hours. | `internal/platform/idempotency.go`, `internal/app/app.go` | Sending the same key twice creates one record |
| 26 | Tests | Unit tests | IDs, envelope, resolver and cache, field conversion, cursor, request IDs, and routing (including names with `%2F`). | `tests/platform/`, `tests/middleware/`, `tests/router/` | `make test-unit` passes |
| 27 | Tests | Integration tests | Every endpoint against Postgres, plus bulk rollback, idempotency replay and conflict, `ambiguous_name`, the `schema_version` bump on rename, and the production guard. Add them to `tests/integration_test.go`, because `make test-integration` runs only that file. | `tests/integration_test.go` | `make test-integration` passes |
| 28 | Docs | OpenAPI 3 spec | Write the spec by hand from this document (swag only generates Swagger 2.0). | `docs/api-platform/openapi.yaml` | The spec validates, and every endpoint and error code is in it |
| 29 | Docs | Basic API docs | A quick start with `curl` examples: enable a test workspace, list, create, update and delete records, page through results, and read errors. | `docs/api-platform/README.md` | A new developer can make a first call using only this page |

---

## 10. Decisions

Agreed on 2026-11-03.

| # | Topic | Decision |
|---|---|---|
| 1 | Record ID format | `rec_<id>`, e.g. `rec_1042`. No schema change and no new dependency. Row numbers become visible, which is accepted; who can read which rows is decided by access rules in the security stage. The format is fixed from the first external user onward. |
| 2 | Name matching | Exact, case-sensitive match on the title. The app does not start enforcing unique names in Stage 1; duplicate names return `409 ambiguous_name`, and IDs always work. |
| 3 | Link fields in writes | Allowed on single-record writes. In bulk writes they return `422 validation_failed`. Moving link writes into the same transaction as the record comes later ([relation_links.go](../internal/services/table/relation_links.go)). |
| 4 | Duration, Time and User formats | Checked against storage (step 21): Duration is whole minutes and Time is `HH:mm`, matching what the app stores; User stays read-only (section 6). |
| 5 | Rate limit | 600 requests per minute per workspace, set by `PUBLIC_API_RATE_LIMIT_PER_MINUTE`. |

Decided during the build (2026-11-04):

| # | Topic | Decision |
|---|---|---|
| 6 | Link writes | A single-record write first saves the record, then links and unlinks one pair at a time through the app's own `link_rows`, so both sides and cardinality stay correct. Link targets are checked before anything is saved; if linking still fails, the response is `422` saying the record was saved but its links were not. |
| 7 | SQL for records | Record reads and writes use their own parameterized SQL (identifiers come only from metadata and are quoted) instead of go-postgres-rest. The library ANDs every filter, builds broken SQL for an unknown operator, drops the whole `ORDER BY` for one bad sort entry, has no `jsonb` operators, and `BulkInsert` takes its columns from the first record (so other records lose column defaults). Bulk creates and updates run one statement per record in a single transaction. |
| 8 | Errors from the app's write guards | Primary key, unique key and check constraint violations return `422 validation_failed` with the same message the app shows, and the record's `index` in bulk requests. Database errors are logged and returned as `500 internal_error` without details. |
| 9 | Idempotency on server errors | A request that ends in a 5xx releases its key, so a retry with the same key runs again. Replays carry the new request's ID in `meta.request_id` and the header `Idempotent-Replayed: true`. |
| 10 | Seed workspaces | `PUBLIC_API_SEED_WORKSPACES` lists workspace **slugs**; each is enabled at startup and its address printed. |
| 11 | Link fields as objects (changed after the contract was first locked, while the API was still dev-only) | A link field is a list of `{id, <lookup>...}` objects instead of record IDs, and lookups are no longer fields of their own; see section 6. Writes still accept plain record IDs. From the first external user onward, such changes need a new version. |
| 12 | Rate limit store | The limit is counted in memory per server process, so N instances allow N × the limit. Accepted while dev-only; it moves to a shared store before production. |

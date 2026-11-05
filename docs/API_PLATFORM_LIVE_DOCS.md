# API Platform: per-workspace OpenAPI docs

Status: built under the dev switch (2026-11-04): `/openapi.json`, `/docs` (Swagger UI) and `/redoc` (ReDoc). API keys on the spec (section 5) and SDK type generation (section 6) are still to come.

This page reviews the proposal to generate each workspace's OpenAPI spec while the server runs, with Swagger UI and ReDoc pages. It checks the proposal against what Stage 1 already built and recommends how to do it.

---

## 1. Summary

- **The diagnosis is right.** Swaggo builds `/swagger/index.html` from code comments when the server is compiled, so it can't describe a workspace's tables. Without per-workspace docs, a developer can't see that `Orders` has a `Price` decimal and a `Status` with the options `Open` and `Done`.
- **Most of the groundwork already exists**, so a spec builder is a small step:
  - **Metadata:** the public API already reads the metadata it needs, while the server runs:
    - bases, tables and fields with their types;
    - the read-only flag;
    - select options;
    - link targets and their lookups.
  - **Caching:** it caches that metadata per `(workspace, schema_version)`.
  - **Type mapping:** the type mapping is already written in [fields.go](../internal/platform/fields.go). `apiTypeNames` gives the type names and `readOnlyTypes` the read-only types. Section 6 of [API_PLATFORM_FOUNDATION.md](API_PLATFORM_FOUNDATION.md) has the formats.
- **Where I'd change the proposal:**
  - Use plain structs instead of kin-openapi.
  - Ship Swagger UI and ReDoc, as FastAPI does. ReDoc loads from the jsDelivr CDN at a pinned version.
  - Use names in the spec's paths and table IDs in its type names.
  - Keep the generic filter parameter instead of listing every filter.
  - SDKs: follow the Supabase model, a hand-written client plus generated types, rather than a full client generated per workspace.

---

## 2. What the repo has today

| Piece | Where | What it gives the docs |
|---|---|---|
| Swaggo spec of the internal API | `router.go` (`/swagger/index.html`), `swaggo/swag`, `gin-swagger`, `swaggo/files` | Fixed routes only. `swaggo/files` already embeds the Swagger UI files, so no CDN is needed. |
| Workspace public API | [internal/platform](../internal/platform) at `/api/v1/{ws}` | Endpoints with a fixed shape for every table: `/bases/{base}/tables/{table}/records`, and so on |
| Table schema while the server runs | `GET /api/v1/{ws}/bases/{base}/tables/{table}` ([schema.go](../internal/platform/schema.go)) | Every field's ID, name, type, `read_only`, `options`, `linked_table_id`, `multiple` and `lookups`. This is already the input a spec builder needs. |
| Metadata cache | [resolve.go](../internal/platform/resolve.go) (`schemaCache`, keyed by `schema_version`) | Rebuilds whenever the schema changes, because triggers on bases, models and columns bump `schema_version` |
| Fixed OpenAPI file | [openapi.yaml](api-platform/openapi.yaml) | Describes `fields` generically, as `additionalProperties`. It's the template for the per-workspace spec. |
| In-app docs page | base-ui `ApiDocsPage` | Readable guide with sample data, not the user's own tables |

**The gap:** nothing turns the table schema into OpenAPI. The fixed spec says "`fields`: an object", and the per-workspace spec would say "`fields`: `{Price: number, Status: "Open" | "Done", ...}`".

---

## 3. Proposed endpoints

Add these to the existing `/:ws` group in [api.go](../internal/platform/api.go), so they get the same workspace check, dev-only switch, CORS and rate limit as the rest of the API:

```go
ws.GET("/openapi.json", a.openAPISpec) // spec for this workspace; ?base=<name or ID> limits it to one base
ws.GET("/docs", a.swaggerUI)           // Swagger UI page pointed at ./openapi.json
ws.GET("/redoc", a.redoc)              // ReDoc page pointed at ./openapi.json
```

- **Paths must not clash with base names.** `/openapi.json`, `/docs` and `/redoc` sit next to `/bases` and can't collide with a base or table name, because those always come after `/bases/`.
- **`?base=` keeps big workspaces readable.** A workspace with 40 tables makes about 200 operations, and Swagger UI gets slow and hard to scan at that size. Both pages pass their `?base=` on to the spec.
- **Swagger UI** uses the files `swaggo/files` already embeds, served by the router at `/swagger/`. Its "Try it out" sends real requests.
- **ReDoc** only reads the docs, but is easier to scan. Its script comes from `cdn.jsdelivr.net/npm/redoc@2.1.5`, as FastAPI does, so that page needs internet in the browser.
- **The spec is the raw OpenAPI document,** not wrapped in `{data, error, meta}`, so tools can read it directly. Errors (an unknown `?base=`) still use the envelope.

---

## 4. How the spec is built

One new file, `internal/platform/openapi.go`, builds the spec from the cached metadata. It reuses `a.bases`, `a.tables` and `a.fields` from [resolve.go](../internal/platform/resolve.go), and the type maps in [fields.go](../internal/platform/fields.go).

### Paths, per table

For each table:
- `GET` and `POST` on `/bases/{Base}/tables/{Table}/records`
- `PATCH` on `/bases/{Base}/tables/{Table}/records`
- `GET`, `PATCH` and `DELETE` on `/bases/{Base}/tables/{Table}/records/{record}`
- `POST` on `/bases/{Base}/tables/{Table}/records/delete`

Rules:
- **Names in paths**, URL-encoded (`In%2FOut`), because that's what developers type and what the docs show.
- **IDs for duplicates:** if a base or table name isn't unique, the path uses its `bse_`/`tbl_` ID, since the name would get `409 ambiguous_name`.
- **`operationId` and type names use the table ID** (`listRecords_tbl_1a2b...`, `Record_tbl_1a2b...`), with the human name in `title` and `summary`. Generated code then keeps working when someone renames a table, and names with spaces or other scripts don't break code generators.
- **Shared parts come from the fixed spec:** the envelope, the error schema, `Idempotency-Key`, `limit`, `cursor` and `keys` are copied from [openapi.yaml](api-platform/openapi.yaml) components, not written twice. The simplest way is to embed `openapi.yaml` with `go:embed`, parse it, and add the per-table paths and types to it, so the generic contract and the per-workspace spec can't drift apart.

### Record types, per table

Each table gets three types:
- **`Record_<tbl>`:** `{id, fields: Fields_<tbl>, created_at, updated_at}`
- **`Fields_<tbl>`:** one property per field, keyed by field name. Read-only fields are marked `readOnly: true`, and nothing is `required`, because PATCH sends only changed fields and POST fills in defaults.
- **`FieldsWrite_<tbl>`:** the same without read-only fields, used for POST and PATCH bodies. Keep it a separate type, because many generators ignore `readOnly`.

### Field type → OpenAPI schema

Drafted from section 6 of the foundation doc and `apiTypeNames`. Every schema is `nullable: true` unless it's a list.

| `type` | Schema |
|---|---|
| `text`, `long_text` | `string` |
| `email` | `string`, `format: email` |
| `url` | `string`, `format: uri` |
| `phone_number` | `string` |
| `number` | `integer`, `format: int32` |
| `decimal`, `currency`, `percent` | `number` (digits kept exactly; note that clients parsing into float64 lose this) |
| `rating` | `integer`, `minimum: 0`, `maximum: <field max, default 5>` |
| `year` | `integer`, `minimum: 1`, `maximum: 9999` |
| `date` | `string`, `format: date` |
| `date_time` | `string`, `format: date-time` |
| `time` | `string`, `pattern: ^([01]\d\|2[0-3]):[0-5]\d$`, example `10:15` |
| `duration` | `integer`, `minimum: 0`, description "minutes" |
| `checkbox` | `boolean` |
| `single_select` | `string`, `enum: [<options>]` |
| `multi_select` | `array` of `string` with `enum: [<options>]` |
| `json` | no type (`{}`: any JSON value) |
| `uuid` | `string`, `format: uuid` (`readOnly` when it's generated automatically) |
| `links` (read) | `array` of `LinkRef_<target tbl>_<field>`: `{id: string (rec_), <lookup key>: <lookup's field schema>...}` |
| `links` (write) | `array` of `oneOf: [string (rec_), {id: string}]`, with `maxItems: 1` when the field holds one record. Single-record writes only, so bulk write types leave link fields out. |
| `attachment` | `array` of `{id, url, name, size, type}`, `readOnly` |
| `user` | `string` or `array` of `string`, `readOnly` |
| `formula`, `created_by`, `last_modified_by` | `string`, `readOnly` |
| `created_time`, `last_modified_time` | `string`, `format: date-time`, `readOnly` |
| unknown types | no type, `readOnly`, which matches how the API returns them as stored |

Lookups get no property of their own, which matches the API: they appear inside `LinkRef`. A lookup's schema is the schema of the field it looks up in the linked table, through `lookupTargets` in [records.go](../internal/platform/records.go).

### Query parameters

- **`select`:** an `array` of `string`, `enum` = the table's field names, `style: form`, `explode: false`.
- **`sort`:** a `string`. The description lists the sortable fields, because the API rejects sorting by some types.
- **`filter`:** keep the single parameter described in the fixed spec. It's `filter[<field>][<op>]=value`, two levels deep, and OpenAPI 3.0's `deepObject` style only defines one level reliably, so tools render it inconsistently. Listing one parameter per field and operator would add hundreds of inputs for a wide table. Instead, the description says which operators each field allows. These come from `buildFilter` in [query.go](../internal/platform/query.go), so they can't drift from what the API accepts.
- **`keys=id`:** document it, but describe the record type keyed by name only. Describing both key styles would double every type.

### Caching

- **Cache key:** the built JSON is cached in the schema snapshot, keyed by `(workspace_id, schema_version, base filter)`. It's rebuilt only when the schema changes.
- **ETag:** a hash of the spec itself, so it also changes when a deploy changes how specs are built. `If-None-Match` gets `304`, so Swagger UI and code generators don't download it again.
- **Delay after a schema change:** the 10 s workspace cache means a new table can take up to 10 s to appear, the same as for the data endpoints.

---

## 5. The docs page and auth

**Today (Stage 1, dev-only):** the whole API is open and exists only under `PUBLIC_API_OPEN_ACCESS`. The `/docs` page follows the same rules. Requests from Swagger UI's "Try it out" already work, because CORS is open for `/api/v1/ws_*` and `X-Request-Id` is readable.

**Once API keys exist,** compare the three options in the proposal:

| Option | Verdict |
|---|---|
| Cookie session | Doesn't fit. Public API callers are other apps, not people logged in to SereniBase. A cookie also brings CSRF concerns to an API that's otherwise only reached with a header. |
| Short-lived docs token in the URL | Avoid. URLs end up in logs, browser history and `Referer` headers. |
| **Public page; the spec is fetched only after the user enters a key** | **Recommended.** The `/docs` HTML holds no data. The user enters a key; Swagger UI's `requestInterceptor` adds it to the spec request and to "Try it out" requests, and the key is kept in `sessionStorage` only. `/openapi.json` needs a key like every other endpoint. |

**Logged-in app users** don't need the key form. The in-app docs page can get the spec through the internal API with their session and show the real tables there, either in its own layout or by embedding Swagger UI. This is a natural follow-up to the current `ApiDocsPage`.

**Role-aware specs.** The spec must show only what the caller's key can reach: its tables, its operations, and its fields if field-level rules come. Otherwise the docs give away the schema. Build the spec from the same access check the data endpoints use, and add the key's role to the cache key: `(workspace_id, schema_version, role, base)`. Until access rules exist, every key sees everything, just as with the data endpoints.

---

## 6. SDKs: generated clients or generated types

The proposal suggests generating a typed client per workspace with openapi-generator. That doesn't fit the roadmap, which plans hand-written toolkits like Supabase's. In [API_PLATFORM_ROADMAP_PLAN.md](API_PLATFORM_ROADMAP_PLAN.md), developers "connect once with their workspace address and key, then add, view, edit and delete records in any table".

| | Full client generated per workspace | Generic SDK + generated types (Supabase model) |
|---|---|---|
| What ships | A new package per workspace, rebuilt on every schema change | One package per language; a workspace generates a types file only |
| Schema changes | Regenerate and republish the client | Regenerate one types file; apps still compile against the generic API |
| Auth, retries, paging, idempotency | Rewritten per generator, quality varies by language | Written once per SDK |
| Typing | Full | Full in TypeScript and Python (TypedDict or pydantic); Go through generated structs |

**Recommendation:**
- Build the generic SDKs as planned.
- Add a `types` command that reads `/openapi.json` and writes the table types: `sereni gen types --ws ws_... --lang ts`.
- Use openapi-generator only for languages the roadmap doesn't cover, such as Dart.

That way one spec serves the docs page, typed SDK use, and any other language, without a separate client per workspace.

---

## 7. Where it fits in the plan

- **Ready now:**
  - **The data API is settled:** Stage 1 is built and tested, and the link-object change is in.
  - **The spec can't fall behind on filters:** it reads its filter operators from the same code as the API, so new operators added for advanced queries (Phase 2) show up in it on their own.
- **Build order:**
  1. The `openapi.json` builder and its tests, under the dev switch. This is the most valuable part, because SDK type generation depends on it.
  2. The `/docs` page with the embedded Swagger UI.
  3. With API keys (Phase 1 security): the key on the spec and on "Try it out", and the role in the cache key.
  4. Show the real tables on the in-app docs page.
  5. The `gen types` command, as part of the SDK work.

### Files

| File | Change |
|---|---|
| [internal/platform/openapi.go](../internal/platform/openapi.go) | Spec builder: embedded base spec; its generic records paths are the templates of each table's paths |
| [internal/platform/openapi_types.go](../internal/platform/openapi_types.go) | Field type → schema mapping (the table in section 4), and the filter operators per type |
| [internal/platform/docs_page.go](../internal/platform/docs_page.go) | Swagger UI and ReDoc pages |
| [internal/platform/openapi_base.yaml](../internal/platform/openapi_base.yaml) | Copy of `docs/api-platform/openapi.yaml` for `go:embed`; a test keeps the two equal |
| `internal/platform/api.go` | Three routes |
| `internal/platform/openapi_internal_test.go` | The filter operators listed match `buildFilter`; the base copy matches the docs file |
| `tests/platform/openapi_test.go` | A sqlmock workspace → the spec passes kin-openapi `Validate` (test dependency only), with typed fields, link objects, `?base=`, `304`, and both pages |
| `tests/platform/api_db_test.go` | Fetch the spec from the Postgres workspace, validate it, and check a real record's keys against the table's type |

### Checks

- **The spec is valid OpenAPI 3.0.3** (kin-openapi `Validate`, or Redocly in CI).
- **Responses match the spec:** in the Postgres e2e, every record returned by list and get validates against `Record_<tbl>`. This is the most important test, because it shows the docs describe what the API really returns.
- **Schema changes update the spec:** renaming a field changes the spec within 10 s, and `ETag` changes with `schema_version`.
- **Generated types compile:** run openapi-generator for TypeScript on the e2e spec and compile the result once in CI.

---

## 8. Risks

- **Schema leaks:** until there are keys and roles, `/openapi.json` shows any workspace's full schema to anyone with its address. That's the same exposure as the data endpoints today, and it's acceptable only because of the dev-only switch.
- **Spec size:** large workspaces are handled by `?base=` and by building only the bases asked for.
- **Renames break code built against names:** type names use IDs, so generated code survives a rename, but paths and field keys use names. SDK users who need stability should use `keys=id` and IDs in paths, and the docs should say so.
- **Decimals:** `number` types make most generators use float64, which loses the exact digits the API keeps. Document it, and in SDKs offer a decimal type for `decimal` and `currency`.

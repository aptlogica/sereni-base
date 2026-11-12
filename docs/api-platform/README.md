# Workspace API: quick start

The workspace API lets your own apps, sites and scripts read and write the records in a SereniBase workspace over HTTP. Workspaces, bases, tables and fields are still set up in the SereniBase app; the API works with the data inside them.

> **Development only.** There is no authentication yet: anyone who knows a workspace's address can read and change its data. The API exists only when the server runs with `PUBLIC_API_OPEN_ACCESS=true`, and the server refuses to start that way when `SERVER_ENV=prod`. API keys come with the security stage.

- Full reference: [openapi.yaml](openapi.yaml) (OpenAPI 3)
- Design and decisions: [API_PLATFORM_FOUNDATION.md](../API_PLATFORM_FOUNDATION.md)

## 1. Turn it on

1. In `.env`, set `PUBLIC_API_OPEN_ACCESS=true` and rebuild the backend (`docker compose up -d --build`).
2. Enable the API for a workspace, either:
   - in the app: **Home → Workspace API → Enable API & app login** (owners and co-owners), or
   - at startup: `PUBLIC_API_SEED_WORKSPACES=<workspace-slug>` (comma-separated); the log prints each address.
3. Copy the **Base URL** shown in the Developer popup, e.g. `http://localhost:8080/api/v1/ws_8f3k2m9qx7`. The `ws_...` part never changes, even if the workspace is renamed or the API is turned off and on again.

```bash
export BASE=http://localhost:8080/api/v1/ws_8f3k2m9qx7
```

Browser apps can call it from any site: CORS is open for the workspace API (`Access-Control-Allow-Origin: *`), with no cookies involved. For example, from any page's console:

```js
const res = await fetch("http://localhost:8080/api/v1/ws_8f3k2m9qx7/bases");
console.log(await res.json(), res.headers.get("X-Request-Id"));
```

## 2. Find your tables and fields

```bash
curl "$BASE/bases"
curl "$BASE/bases/CRM/tables"
curl "$BASE/bases/CRM/tables/Orders"          # the table with its fields, types and select options
```

Bases, tables and fields can be named or given by ID (`bse_...`, `tbl_...`, `fld_...`). Names are exact and case-sensitive, and URL-encoded in paths (`Sales%20Team`, `In%2FOut`). If two items share a name you get `409 ambiguous_name` with their IDs; IDs always work, and apps that must survive renames should use them.

## 3. Read records

```bash
curl "$BASE/bases/CRM/tables/Orders/records?limit=20"
curl "$BASE/bases/CRM/tables/Orders/records/rec_1042"
curl -g "$BASE/bases/CRM/tables/Products/records/by-key?key[SKU]=A1&key[Region]=EU"   # primary key SKU + Region
```

With the default primary key, a record is found by its record ID (`rec_1042`). When the table's primary key was changed to other fields, the record ID is refused with a `400` that names the key fields, and each record in responses carries a `key` object (`"key": {"SKU": "A1", "Region": "EU"}`). Then:

- One record: `records/A1` for a one-field key, or name each key field: `records/by-key?key[SKU]=A1&key[Region]=EU` (by name or `fld_` ID, in any order; values only need URL encoding, so `_`, `/` and spaces are fine). When the record ID is part of the key, name it `id`: `by-key?key[Uuid]=01a1…&key[id]=1` (`1` or `rec_1`).
- Bulk update: `{"records": [{"key": {"SKU": "A1", "Region": "EU"}, "fields": {...}}]}`.
- Bulk delete: `{"keys": [{"SKU": "A1", "Region": "EU"}]}`.

A missing or extra key field, or a field that isn't part of the key, is a `400` saying which; a key that matches nothing is a `404` (*No record with SKU "A1" and Region "FR".*)

```json
{
  "data": [
    {
      "id": "rec_1042",
      "fields": {
        "Name": "Asha", "Status": "Active", "Price": 499,
        "Customer": [{ "id": "rec_12", "Email": "asha@example.com" }]
      },
      "created_at": "2026-11-03T10:15:00Z",
      "updated_at": "2026-11-03T10:15:00Z"
    }
  ],
  "error": null,
  "meta": { "request_id": "req_6f1c...", "next_cursor": "eyJ2Ijox...", "has_more": true, "limit": 20 }
}
```

Every response has `data`, `error` and `meta`. Empty fields are `null` (or `[]` for lists). `?keys=id` keys `fields` by field ID instead of name.

### Paging

Pass `meta.next_cursor` back as `cursor`, with the same filter and sort, until `has_more` is `false`:

```bash
curl "$BASE/bases/CRM/tables/Orders/records?limit=50&cursor=eyJ2Ijox..."
```

`limit` is 50 by default and at most 200.

The cursor holds a position in the list, so records added or deleted while you page can make a later page skip or repeat a record. Page over data that isn't changing, or sort by a field that only grows (such as `created_at`) and treat repeats as already seen.

### Filter, sort and select

```bash
# Status is Active and Price >= 100, newest first, only two fields
curl -g "$BASE/bases/CRM/tables/Orders/records?filter[Status]=Active&filter[Price][gte]=100&sort=-created_at&select=Name,Price"

# Text contains (case-insensitive), a list of values, empty fields, linked records
curl -g "$BASE/bases/CRM/tables/Orders/records?filter[Name][contains]=ash"
curl -g "$BASE/bases/CRM/tables/Orders/records?filter[Status][in]=Active,Paused"
curl -g "$BASE/bases/CRM/tables/Orders/records?filter[Notes][is_null]=true"
curl -g "$BASE/bases/CRM/tables/Orders/records?filter[Customer][contains]=rec_12"
```

Operators: `eq` (default), `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `contains`, `is_null`. Filters are combined with AND. `sort` takes a comma-separated list; `-` means descending, and `id` is always the last tie-breaker. In `select`, `sort` and `filter`, the names `id`, `created_at` and `updated_at` mean the record's own values (`filter[id]=rec_12`); a field with one of those names, or with `,`, `[` or `]` in its name, must be given by its `fld_` ID.

Formula and lookup fields are calculated when a record is read, so they can't be used to filter or sort.

### "Or" and "not" filters

Send a `where` tree to `POST /records/query`. A group matches **all**, **any** or **none** of its conditions, and a condition is a `field`, an `op` (default `eq`) and a `value`:

```bash
# Price at least 100, Status Active or Paused, and not paid; highest price first
curl -X POST "$BASE/bases/CRM/tables/Orders/records/query" -H "Content-Type: application/json" -d '{
  "where": {
    "match": "all",
    "conditions": [
      { "field": "Price", "op": "gte", "value": 100 },
      { "match": "any", "conditions": [
          { "field": "Status", "value": "Active" },
          { "field": "Status", "value": "Paused" }
      ]},
      { "match": "none", "conditions": [ { "field": "Paid", "value": true } ] }
    ]
  },
  "sort": ["-Price"],
  "limit": 20
}'
```

- `match` is `all` when left out. Groups can hold groups, up to 5 deep, with at most 50 conditions in all.
- `in` takes a JSON list: `{"field": "Status", "op": "in", "value": ["Active", "Paused"]}`.
- `{"field": "Notes", "op": "is_null"}` finds empty fields.
- `none` counts an empty field as not matching: `none` of `Price < 100` keeps orders without a price.
- The body also takes `select`, `limit`, `cursor` and `keys`. An empty `select` (`[]`) returns all fields, as when it is left out. Everything is optional: an empty body, a `where` that is `null` or `{}`, or a group without conditions lists every record. The same goes for `?where=`, `?where={}` and a bare `?filter=` on GET. Page with `next_cursor`, sending the same `where` and `sort`.
- The same tree works on `GET` as the URL-encoded `where` parameter, combined with any `filter[...]` using AND.
- The HTTP `QUERY` method works too: `curl -X QUERY "$BASE/bases/CRM/tables/Orders/records/query" ...` takes the same body and gives the same answer. Use whichever your client supports; cursors work across both.
- Mistakes say where they are: `where.conditions[1].conditions[0]: No field named "Staus". Did you mean "Status"?`

### Search

`search` finds records where **any** text field (text, long text, email, URL, phone, single select) contains the text, ignoring case:

```bash
curl "$BASE/bases/CRM/tables/Orders/records?search=acme"
curl "$BASE/bases/CRM/tables/Orders/records?search=acme&filter[Status]=Open"   # both must match
```

- Combined with `filter[...]` and `where` using AND. In `POST /records/query` send `"search": "acme"`.
- At most 200 characters; an empty search lists every record. A cursor belongs to its search: send the same `search` with `cursor`.

### Linked records inline (`expand`)

By default a link field holds only each linked record's `id` (and its lookups). `expand` adds the linked record's fields:

```bash
curl "$BASE/bases/CRM/tables/Orders/records?expand=Customer"
curl "$BASE/bases/CRM/tables/Orders/records?expand=Customer,Customer.Company"   # two levels
curl "$BASE/bases/CRM/tables/Orders/records/rec_1042?expand=Customer"
```

```json
"Customer": [{ "id": "rec_12", "Email": "asha@example.com",
               "fields": { "Name": "Asha", "Company": [{ "id": "rec_3", "fields": { "Name": "Acme Group" } }] } }]
```

- Up to 2 levels (`Customer.Company`; a third level is a 400) and 5 link fields per request. The first 100 linked records of each link are expanded; the rest keep only their `id`.
- An expanded field is returned even when `select` leaves it out. In `POST /records/query` send `"expand": ["Customer", "Customer.Company"]`.
- Expanding changes what each record shows, not which records are returned. Filtering by a linked record's fields (`Customer.Country`) isn't supported yet.
- Each expanded field is one extra database query per page, however many records the page has.

### Time limit

A read (list, query or get one record) that runs longer than 10 seconds is stopped and returns `503 query_timeout`. Narrow the filter or search, expand fewer fields, or ask for a smaller page.

## 4. Write records

```bash
# Create one (201)
curl -X POST "$BASE/bases/CRM/tables/Orders/records" -H "Content-Type: application/json" \
  -d '{"fields": {"Name": "Asha", "Status": "Active", "Price": 499, "Customer": ["rec_12"]}}'

# Create many (201), up to 100
curl -X POST "$BASE/bases/CRM/tables/Orders/records" -H "Content-Type: application/json" \
  -d '{"records": [{"fields": {"Name": "Ravi"}}, {"fields": {"Name": "Meera"}}]}'

# Update one: only the fields sent change
curl -X PATCH "$BASE/bases/CRM/tables/Orders/records/rec_1042" -H "Content-Type: application/json" \
  -d '{"fields": {"Status": "Paused"}}'

# Update many, up to 100
curl -X PATCH "$BASE/bases/CRM/tables/Orders/records" -H "Content-Type: application/json" \
  -d '{"records": [{"id": "rec_1042", "fields": {"Price": 450}}, {"id": "rec_1043", "fields": {"Price": 120}}]}'

# Delete one, or many (up to 100)
curl -X DELETE "$BASE/bases/CRM/tables/Orders/records/rec_1042"
curl -X POST "$BASE/bases/CRM/tables/Orders/records/delete" -H "Content-Type: application/json" \
  -d '{"ids": ["rec_1043", "rec_1044"]}'
```

- **All or nothing.** In a bulk request every record is checked first; if any fails, nothing is written and the error lists each problem with the record's `index`. A bulk update or delete with an ID that doesn't exist returns `404` and changes nothing.
- **Unknown fields are rejected**, not ignored, so typos surface. Read-only fields (lookups, formulas, attachments, users, auto UUIDs, created/modified) can't be written.
- **Link fields** come back as one object per linked record: its `id` plus every lookup through the link, keyed by the looked-up field's name (`"Email"` above); lookups have no key of their own. To write them, send an array of record IDs or of those objects (only `id` is read). They can only be set when writing one record, and the value replaces the field's links.
- **Value formats** per field type are in [section 6 of the design doc](../API_PLATFORM_FOUNDATION.md#6-field-type-mapping). Dates are `"2026-11-03"`, date-times RFC 3339 (returned in UTC), times `"14:15"` (12-hour input like `"2:15 PM"` is also accepted), durations whole minutes.
- **Field settings apply, as in the app.** Select and multi-select values must be one of the field's options. Email, phone and URL fields are checked only when their check is turned on in the field's settings, with the same rules the app uses; with it off, any text is saved. Decimal values are rounded to the field's decimal places, and percent values to 1 decimal. A rating can't be above the field's highest rating.

### Safe retries

Send an `Idempotency-Key` header (up to 255 characters) on POST requests. A retry with the same key and body gets the first response again (with `Idempotent-Replayed: true`) instead of writing twice. The same key with a different body, or while the first request is still running, is `409 conflict`. Keys are kept for 24 hours.

```bash
curl -X POST "$BASE/bases/CRM/tables/Orders/records" -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-2026-11-04-001" -d '{"fields": {"Name": "Asha"}}'
```

## 5. Errors

```json
{
  "data": null,
  "error": {
    "code": "validation_failed",
    "message": "Some fields have invalid values. Nothing was saved.",
    "details": [{ "field": "Price", "issue": "must be a number" }]
  },
  "meta": { "request_id": "req_6f1c..." }
}
```

| Code | HTTP | When |
|---|---|---|
| `bad_request` | 400 | Malformed JSON, a bad query parameter or where tree, a cursor reused with another filter or sort, the same ID twice in a bulk request |
| `not_found` | 404 | Workspace, base, table, field or record doesn't exist, or the workspace's API is off |
| `ambiguous_name` | 409 | A name matches more than one base, table or field (`details` lists their IDs) |
| `conflict` | 409 | Idempotency key reused with a different body, or still running; or the table was busy with other changes (nothing was saved, try again) |
| `payload_too_large` | 413 | More than 100 records in one request |
| `validation_failed` | 422 | Wrong type, unknown field, read-only field, unknown select option, link field in a bulk write, a duplicate key, a failed check constraint, or a table automation that rejected the change (its own message is shown) |
| `rate_limited` | 429 | More than 600 requests per minute for the workspace; wait `Retry-After` seconds |
| `internal_error` | 500 | Server error; quote `meta.request_id` when reporting it. If an automation on the table failed, the message says so: check the table's automations |
| `query_timeout` | 503 | A read ran longer than 10 seconds and was stopped; narrow the filter or search, expand fewer fields, or ask for a smaller page |

Every response has an `X-Request-Id` header equal to `meta.request_id`. You can send your own `X-Request-Id` (up to 64 characters of `A-Z a-z 0-9 _ -`) to trace a request through the server logs.

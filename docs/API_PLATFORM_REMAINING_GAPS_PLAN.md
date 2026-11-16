# Advanced Queries — Plan for the Remaining Gaps

**Created:** 2026-11-16 · **Gaps:** [API_PLATFORM_ADVANCED_QUERIES_GAPS.md](API_PLATFORM_ADVANCED_QUERIES_GAPS.md) · **Plan:** [API_PLATFORM_ADVANCED_QUERIES.md](API_PLATFORM_ADVANCED_QUERIES.md) · **Code:** [internal/platform/](../internal/platform/)

How each open gap gets fixed, in batches that can ship separately. Batches 1 → 2 → 3 are built in order. Batch 4 waits for the release *after* keyset paging ships, so cursors in use break only once. G19 waits until someone asks for it.

**Not in this plan:**
- **G26**, the go-postgres-rest release, which the owner will plan later.
- **G7** (server-side formulas), a separate task.
- **G8** and **G14** (indexes), which belong to the Performance layer.
- **G24**, not planned.
- **G28**, deferred.

**Decided:** G6 needs no library change. It finds the matching linked records first, then filters by them.

**Every gap also gets:**
- **Code:** the fix in `internal/platform`.
- **Tests:** unit tests, plus a Postgres subtest when the SQL changes.
- **Docs:** README; `openapi_base.yaml`, copied to `docs/api-platform/openapi.yaml`; the in-app docs (`services/base-ui/.../apiDocsContent.ts`, copied to `../base-ui`).
- **Tracking:** the gap marked ✅ in the gaps file, the plan and this file.

---

## Batch 1: quick fixes (≈2 hrs) ✅ done 2026-11-16

| Gap | Fix | Where | Status |
|---|---|---|---|
| **G10** repeated sort field | Reject `sort=Status,Status` with `400 sort: "Status" is listed twice.` (track the keys in a set while parsing). | `parseListOptions`, query.go | ✅ Fixed (2026-11-16) |
| **G25** sort keys after `id` | Stop reading sort terms after `id`: they can't change the order, so they stay out of the cursor. Same loop as G10. | query.go | ✅ Fixed (2026-11-16) |
| **G11** only the first URL option reported | `rejectURLOptions` builds one message listing every wrong option, each with its body equivalent, instead of only `names[0]`. | records.go | ✅ Fixed (2026-11-16) |
| **G16** default metric names use field names | With `keys=id`, the default name is `<op>_<fld_id>` (`f.PublicID()`); the docs recommend `as`. | `parseMetrics`, aggregate.go | ✅ Fixed (2026-11-16) |
| **G17** unknown time zone → 500 | Map Postgres' unknown-zone error (SQLSTATE `22023`, "time zone … not recognized") to `400 timezone: … is not a time zone`. | `aggregateRecords` error path, aggregate.go | ✅ Fixed (2026-11-16) |
| **G18** same field grouped twice | Reject the same field twice in `group_by` whatever its `by`: `seen` keyed by `c.identity()` only. This fixes both the duplicate key in `group` and the sort ambiguity. | `parseGroupBy`, aggregate.go | ✅ Fixed (2026-11-16) |
| **G30** a bad date value doesn't say the format | The error names the format: `use YYYY-MM-DD` for a date, `use RFC 3339 with a zone, like 2026-11-01T00:00:00Z` for a date-time. Message only. Plain dates aren't converted, because an `lte` date would wrongly leave out that day. | `buildCondition` date and date-time cases, query.go | ✅ Fixed (2026-11-16) |

## Batch 2: wrong results (≈1.75 hrs) ✅ done 2026-11-16

| Gap | Fix | Where | Status |
|---|---|---|---|
| **G27** multi-user filters treat `"u1,u2"` as one text | For multi-user fields (`typeUser` with `Multiple`), allow only `contains` and `is_null`. `contains u1` becomes `anyOf(eq "u1", ilike "u1,%", ilike "%,u1", ilike "%,u1,%")` plus the variants with `", "`, with `likeEscaper` applied to the ID. It's built from the existing `leaf` and `anyOf`, so the library doesn't change. Multi-user fields become unsortable, because they hold a list. | `filterOperators` and `buildCondition`, query.go; `unsortableTypes` | ✅ Fixed (2026-11-16) |
| **G29** number filters go through `float64` (also found: a fraction or a number past the range on a whole-number field, like `filter[Qty][gt]=1.5`, gave a 500; now compared as numbers) | Check the value with `strconv.ParseFloat`, but pass the trimmed original text. Postgres reads it as the column's number type (`IN` placeholders work the same way), so long decimals compare exactly. | the number case in `buildCondition`, query.go | ✅ Fixed (2026-11-16) |
| **G4** no cap on total expand work | (1) A budget of 5,000 linked records per request, kept in a `*int` shared by the expand tree and set in `parseExpand`. `expandedRecords` loads only what's left; links past the budget keep only their `id`, as with the 100-per-cell rule. (2) Inside expanded records, link fields carry IDs only, without lookups, unless they're expanded too: `expandedRecords` passes a flag to `buildRecords` to skip `linkLookups`. | expand.go; `buildRecords` in records.go | ✅ Fixed (2026-11-16) |

## Batch 3: new query features (≈8.5 hrs) ✅ done 2026-11-16

| Gap | Fix | Where | Status |
|---|---|---|---|
| **G5** total count | Opt-in `count=true` (GET) or `"count": true` (query body) returns `meta.total`. It runs one `SELECT COUNT(*)` with the same `whereClause` (no keyset clause, no limit), only when asked. It isn't part of the cursor hash. The docs say to ask for it on the first page only. | `parseListOptions`, `runList`, `listMeta` (envelope.go), `queryBodyKeys` | ✅ Fixed (2026-11-16) |
| **G9** one extra query per link filter | Remember, for the length of the request, each linked table and its key (`recordKeyOf`). On a default-key table the cell is always `{"id": n}`, so no query is needed at all; only custom-key tables still read the linked record's key. | `linkResolver`, records.go | ✅ Fixed (2026-11-16) |
| **G6** filter on a linked field or a lookup | Accept `filter[Customer.Country]=India`, `where {"field": "Customer.Country", …}`, and lookup names such as `Customer country`. Steps: (1) Split the name at the link, as `splitExpandPath` does (expand.go); a lookup maps to its link and looked-up field, as `lookupColumn` does (aggregate.go). (2) Build the condition on the linked table with the same `buildCondition`, and run `SELECT id + key objects FROM linked WHERE <cond> LIMIT 1001`. (3) Filter with one `jsonb_contains` per match inside `anyOf`, the way link `contains` already works. No matches gives a condition that is always false; more than 1,000 gives `400 … matches more than 1000 linked records; narrow it`. It works inside all/any/none groups. The docs say that `none` and `neq` also match records with no linked record. Lookups stay unsortable. | where.go and query.go (new `linkedCondition`); reuses the G9 `linkResolver` | ✅ Fixed (2026-11-16) |

## Batch 4: cursor changes, in the release after keyset paging ships (≈1.75 hrs)

| Gap | Fix | Where | Status |
|---|---|---|---|
| **G22** a sort field's type change isn't caught | Add each sort field's type to `optionsHash`. A cursor from before a type change is then refused with `different filter or sort`, instead of skipping rows. | `optionsHash`, query.go | Open |
| **G23** v1 offset cursors are still read | Remove the v1 branch from `decodeCursor` and the `OFFSET` from `listSQL`; a v1 cursor then gets `cursor is not valid`. Same release as G22, so cursors in use break only once. | query.go | Open (planned) |
| **G21** cursor size isn't capped | Long-text fields become unsortable, since they're the only source of very large sort values: `400 sort: records can't be sorted by "Notes" (long_text field)`. The docs keep recommending POST for long sorts. | `unsortableTypes`, query.go | Open |
| **G20** multi-key keyset paging uses OR | When every sort key is NOT NULL (`id`, `created_at`, `updated_at`) and all go the same way, write a row comparison like `(t.created_time, t.id) < ($1, $2)`, which an index can use. Otherwise keep the OR form. This pays off once the Performance layer adds indexes. | `keysetClause`, query.go | Open |

## Later: G19 (≈1.5 hrs, low priority)

| Gap | Fix | Status |
|---|---|---|
| **G19** no `having`, median, percentiles or linked-record count | Four additions to `aggregateOps` and `aggregateSQL` in aggregate.go: (1) `having`, e.g. `{"metric": "total", "op": "gt", "value": 1000}`, becomes `HAVING` on the metric expression; (2) `median`; (3) `p90`-style percentiles, using `percentile_cont(x) WITHIN GROUP (ORDER BY col)`; (4) `count_links` on a link field, as `SUM(jsonb_array_length(col))`. | Open |

---

## Totals

Batch 1 ≈2 · Batch 2 ≈1.75 · Batch 3 ≈8.5 · Batch 4 ≈1.75 · Later ≈1.5 → **≈15.5 hrs**, plus G26 (0.5 hr), which its owner will schedule.

## Checking each batch

1. `go vet ./...` and `go test ./internal/platform/ ./tests/platform/`.
2. The real-Postgres suite, run the way CI runs it:
   - **Setup:** `postgres:15-alpine` in Docker, with `PLATFORM_TEST_PG_*` and `PLATFORM_TEST_PG_SETUP=1`.
   - **Command:** `go test -race ./tests/...`.
   - **New subtests:**
     - G27: multi-user `contains`.
     - G29: an exact match on a 17-digit decimal.
     - G4: the expand budget.
     - G5: `meta.total` with filters.
     - G6: a linked field, a lookup, `none`, and more than 1,000 matches.
     - G22/G23: old cursors refused.
3. The OpenAPI test keeps `openapi_base.yaml` and `docs/api-platform/openapi.yaml` equal, and the base-ui docs file type-checks (`npx tsc --noEmit`, no errors in the docs files).
4. Mark each gap ✅ in the gaps file, the plan and this file, and recompute the totals.

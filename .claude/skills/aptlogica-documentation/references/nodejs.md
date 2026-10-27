# Node / TypeScript — TSDoc

Read this with SKILL.md's four things. This file is only the syntax and the
TypeScript-specific traps.

## Style

TSDoc (`/** ... */`), not `//` blocks, on anything exported — the `/**` form is
what the editor, `typedoc` and the next agent all read.

```ts
/** One line, imperative mood, ends with a period. */
export function f(): T {}
```

No `@param` and no `@returns` when the types already say it. `strict`,
`noUncheckedIndexedAccess` and explicit return types on exports are enforced by
the engineering gate, so the signature is trustworthy and repeating it in prose
just gives you two things to keep in sync.

Use `@param` only to state a constraint the type cannot: a unit, a range, an
ownership rule.

```ts
/**
 * Retry `fn` with exponential backoff.
 *
 * @param baseDelayMs - first delay; doubles each attempt, jittered ±20%.
 */
```

Use `@throws` when a thrown error is part of the contract, and `@remarks` for
the paragraph that does not belong on the summary line.

## File header

Every module under `src/` opens with a header naming the layer. TypeScript has
no module docstring, so it is a leading block comment above the imports:

```ts
/**
 * Lead persistence. Repository layer — owns no transaction, returns domain
 * objects, never Prisma rows.
 *
 * Called from: services/leads, services/enrichment.
 */
import { PrismaClient } from "@prisma/client";
```

A barrel file (`index.ts`) documents what the module is for and which file to
open first, not a list of its re-exports — the re-export list is right there.

## Types and interfaces

An exported type earns one line. Document the field the name does not explain,
on the field:

```ts
/** A page of results plus the cursor that fetches the next one. */
export interface Page<T> {
  items: T[];
  /** Opaque; decode only with `decodeCursor`. Not a row offset. */
  nextCursor: string | null;
}
```

Zod schemas are documentation when the field names are honest. Add a comment
only for an untrusted field, and say so explicitly:

```ts
export const LeadCreate = z.object({
  email: z.string().email(),
  /** Free text from the import CSV — untrusted, never interpolated into a
   *  prompt. */
  sourceNote: z.string().nullish(),
});
```

## Schemas and their inferred types

A Zod schema and the type inferred from it are one thing said twice. Document
the schema — that is where the constraint lives — and let the alias inherit it:

```ts
/**
 * One captured source. `trust` is a literal rather than an enum with a safe
 * member: everything fetched is untrusted content, and there is no value that
 * says otherwise.
 */
export const evidence = z.object({ ... });
export type Evidence = z.infer<typeof evidence>;
```

`docgate.py` skips `export type X = z.infer<typeof x>` for this reason. Any
other exported type — a hand-written interface, a union, a branded string —
carries its own summary, because nothing else in the file explains it.

## Functions and classes

```ts
/**
 * Return one page of an org's leads, newest first.
 *
 * Cursor-based: offset pagination over this table skips rows while writes land.
 *
 * @throws InvalidCursor - when the cursor does not decode.
 *
 * Called from: routes/leads.list, agents/outbound/selectTargets.
 */
export async function listForOrg(
  orgId: string,
  cursor: string | null,
  limit: number,
): Promise<Page<Lead>> {}
```

Classes get the responsibility plus the lifecycle rule — which for a class
holding a connection or a client is the only part people get wrong:

```ts
/**
 * Reads and writes leads for one org.
 *
 * Holds a Prisma client bound to one transaction, so construct per request or
 * per graph node. A shared instance leaks a transaction across requests.
 */
export class LeadRepository {}
```

## React components

The header says what the component renders and where its data comes from — the
prop types cover the rest:

```tsx
/**
 * Lead table with cursor pagination. Server data via `useLeads`; holds only
 * the selection in local state.
 *
 * Called from: routes/leads/page.tsx.
 */
export function LeadTable({ orgId }: LeadTableProps) {}
```

A hook documents what it subscribes to and when it refetches, because that is
the part that causes the bug:

```ts
/**
 * Subscribe to a run's lifecycle events over SSE.
 *
 * Reconnects with backoff on drop and replays from the last seen event id, so
 * a reconnect never silently loses a `blocked` transition.
 */
export function useRunEvents(runId: string) {}
```

## LangGraph.js nodes

Same three extra lines as every other stack, mirroring the declarations:

```ts
/**
 * Draft an outbound reply from retrieved context. Agent node.
 *
 * Tools:   readCrm, searchDocs              (mirrors ALLOWED_TOOLS)
 * Budget:  40_000 tokens, ceiling 120_000   (mirrors TOKEN_BUDGET, CONTEXT_CEILING)
 * Emits:   planning → callingTool → done, or blocked when context is thin
 *
 * Called from: agents/outbound/graph.
 *
 * Implements: EPIC-05-S02 / FR-COMMS-031
 */
```

The node function documents which state keys it reads and which it returns.
A `Partial<State>` return type says nothing about which keys are in it.

## Tests

Vitest `describe`/`it` strings are the documentation. `it("returns 404 when the
lead belongs to another org")` needs no comment; `it("works")` needs a rewrite.

A regression test records what broke, above the `it`:

```ts
// Regression: offset pagination duplicated row 51 under concurrent writes.
it("keeps a cursor stable across a concurrent insert", async () => {});
```

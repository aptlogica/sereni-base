#!/usr/bin/env python3
"""Repository memory builder for Aptlogica repositories.

Builds a committed, deterministic index of what is in a repository — every
source file, its layer, its one-line header, its exported symbols with line
numbers and summaries, and who calls them — so an agent can answer "where does
X live", "what calls Y" and "what is in this file" from an index instead of
re-reading the tree, and can cite `file:line` instead of guessing.

No model is involved. Every line in the index is copied from the code or
computed from it, so the index cannot hallucinate; it can only go stale, and
staleness is detectable because each entry carries the hash of the file it came
from. `--verify` is the check.

Exit codes:
    0  built, or verified clean
    1  verify found stale or missing entries
    2  could not run (bad path, nothing to index)

Usage:
    python scripts/memory.py                     # build or refresh .claude/memory/
    python scripts/memory.py --verify            # is the index current? (CI, gate)
    python scripts/memory.py --query list_leads  # one symbol, with callers
    python scripts/memory.py --file src/api/leads.py
    python scripts/memory.py --stale             # list files the index is behind on
    python scripts/memory.py --paths src web/src --out .claude/memory

Standard library only, so it runs in CI, in a container, and in a bare repo.
"""

from __future__ import annotations

import argparse
import ast
import hashlib
import json
import re
import sys
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path

# --- configuration ----------------------------------------------------------

PY_SUFFIXES = {".py"}
TS_SUFFIXES = {".ts", ".tsx", ".js", ".jsx", ".mjs"}
GO_SUFFIXES = {".go"}
SOURCE_SUFFIXES = PY_SUFFIXES | TS_SUFFIXES | GO_SUFFIXES

SKIP_DIR_PARTS = {
    "node_modules",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".git",
    "vendor",
    ".next",
    "coverage",
    "graphify-out",
}

# Directory fragment → the layer name used everywhere in the standard. First
# match wins, so the more specific fragments come first.
LAYERS = (
    ("src/agents", "agent"),
    ("src/agent", "agent"),
    ("internal/agent", "agent"),
    ("src/api", "transport"),
    ("src/routers", "transport"),
    ("src/routes", "transport"),
    ("internal/transport", "transport"),
    ("internal/handlers", "transport"),    # SereniBase
    ("internal/router", "transport"),      # SereniBase
    ("internal/middleware", "transport"),  # SereniBase
    ("src/services", "service"),
    ("internal/service", "service"),
    ("internal/services", "service"),      # SereniBase
    ("src/repositories", "repository"),
    ("internal/repository", "repository"),
    ("src/schemas", "schema"),
    ("internal/dto", "schema"),            # SereniBase
    ("src/models", "model"),
    ("internal/models", "model"),          # SereniBase
    ("src/core", "core"),
    ("internal/providers", "core"),        # SereniBase: auth, email, storage, antivirus, otp, logger adapters
    ("internal/config", "core"),           # SereniBase
    ("internal/app", "core"),              # SereniBase: wiring and startup
    ("internal/app-errors", "core"),       # SereniBase
    ("internal/constant", "core"),         # SereniBase
    ("internal/utils", "core"),            # SereniBase
    ("migrations", "migration"),
)

# Matched with exact case (see `contract_files`), so on a case-insensitive
# filesystem `docs/ARCHITECTURE.md` does not also list `docs/architecture.md`.
CONTRACT_FILES = (
    "docs/API_SPEC.md",
    "docs/DB_SCHEMA.md",
    "docs/ARCHITECTURE.md",
    "docs/architecture.md",  # SereniBase
    "docs/swagger.yaml",     # SereniBase
    "docs/adr",              # SereniBase: architecture decision records
    "DEV_PLAN.md",
    "docs/DEV_PLAN.md",
    "docs/PROGRESS.md",
    "DECISIONS.md",
)

# SereniBase: bumped whenever parsing changes. Entries are reused by content
# hash, so without this an unchanged file keeps the entry an older parser made.
INDEX_FORMAT = 2

TS_EXPORT_RE = re.compile(
    r"^export\s+(?:default\s+)?(?:async\s+)?"
    r"(?P<kind>function|class|interface|type|const|enum)\s+(?P<name>[A-Za-z_$][\w$]*)"
)
GO_FUNC_RE = re.compile(r"^func\s+(?:\([^)]*\)\s*)?(?P<name>[A-Z]\w*)\s*[\(\[]")
GO_TYPE_RE = re.compile(r"^(?:type|var|const)\s+(?P<name>[A-Z]\w*)\b")
# SereniBase: receivers, grouped declarations and the license notice, so methods
# are `Type.Method`, `const ( ... )` members are indexed, and the header is real.
GO_RECV_RE = re.compile(r"^func\s*\(\s*(?:\w+\s+)?\*?\s*(?P<recv>\w+)")
GO_BLOCK_RE = re.compile(r"^(?P<kind>const|var|type)\s*\(\s*(?://.*)?$")
GO_BLOCK_ENTRY_RE = re.compile(r"^\t(?P<name>[A-Z]\w*)\b")
LICENSE_RE = re.compile(r"copyright|licensed under|spdx-license|^websites?:|^support:", re.IGNORECASE)

# Every identifier, whether it is selected (`x.Name`), and whether a call,
# literal or type argument follows it.
TOKEN_RE = re.compile(r"(\.)?(\w+)(\s*[\(\{<])?")

# The declaration may carry a type annotation and a trailing comment
# (`ALLOWED_TOOLS: tuple[str, ...] = ("read_crm",)  # why`); capture the value
# only, or the index prints the annotation as if it were the tool list.
NODE_DECL_RE = {
    "tools": re.compile(r"(?:ALLOWED_TOOLS|AllowedTools)\s*(?::[^=\n]+)?=\s*(.+)"),
    "budget": re.compile(r"(?:TOKEN_BUDGET|TokenBudget)\s*(?::[^=\n]+)?=\s*([\d_]+)"),
    "ceiling": re.compile(r"(?:CONTEXT_CEILING|ContextCeiling)\s*(?::[^=\n]+)?=\s*([\d_]+)"),
    "model": re.compile(r"^\s*(?:MODEL|Model)\s*(?::[^=\n]+)?=\s*(.+)", re.MULTILINE),
}


@dataclass
class Symbol:
    """One exported symbol: what it is, where it is, and what it says it does."""

    name: str
    kind: str
    line: int
    summary: str = ""
    callers: list[str] = field(default_factory=list)


@dataclass
class FileEntry:
    """One source file's memory entry, stamped with the hash it was built from."""

    path: str
    layer: str
    header: str
    sha: str
    lines: int
    symbols: list[Symbol] = field(default_factory=list)
    node: dict[str, str] = field(default_factory=dict)


# --- helpers ----------------------------------------------------------------


def sha_of(text: str) -> str:
    """Return the short content hash stored with an entry."""
    return hashlib.sha256(text.encode("utf-8", "replace")).hexdigest()[:12]


def first_sentence(text: str, limit: int = 160) -> str:
    """Return the opening sentence of a docstring or comment block.

    Docstrings wrap at the source line width, so taking the first physical line
    cuts most summaries mid-phrase. Lines are joined until the sentence ends or
    the limit is reached, which is what makes the index readable.
    """
    collected: list[str] = []
    for raw in text.strip().splitlines():
        line = raw.strip().lstrip("/*# ").strip()
        # `Called from:` and `Implements:` are structured notes the index reads
        # separately; treating one as the summary hides what the symbol does.
        if line.startswith(("Called from:", "Implements:")):
            break
        if not collected and (not line or line.startswith(("@", "eslint", "type:"))):
            continue
        if not line:
            break
        collected.append(line)
        joined = " ".join(collected)
        if joined.endswith((".", "!", "?", ":")) or len(joined) >= limit:
            break
    summary = " ".join(collected)
    return summary[:limit].rstrip()


WORKSPACE_RE = re.compile(r"(?:^|/)packages/(?P<package>[^/]+)/src/")


def layer_of(path: Path) -> str:
    """Return the layer name for a file, from the directory it sits in.

    In a monorepo the layer directories sit under `packages/<name>/src`, so the
    fragments match there too. A file in a workspace package that matches no
    layer is grouped under its package name rather than dumped in `other` —
    "shared" and "runner" are the useful headings in that repo.
    """
    posix = "/" + path.as_posix()
    for fragment, name in LAYERS:
        if f"/{fragment}/" in posix:
            return name
    if is_test(path):
        return "test"
    workspace = WORKSPACE_RE.search(posix)
    if workspace:
        return workspace.group("package")
    return "other"


def is_test(path: Path) -> bool:
    """Return True for a test file."""
    parts = set(path.parts)
    return (
        "tests" in parts
        or "test" in parts
        or path.name.startswith("test_")
        or path.name.endswith(("_test.go", ".test.ts", ".spec.ts", ".test.tsx"))
    )


def iter_sources(roots: list[Path]) -> list[Path]:
    """Return every source file under `roots`, minus vendored and build dirs."""
    found: list[Path] = []
    for root in roots:
        if root.is_file():
            if root.suffix in SOURCE_SUFFIXES:
                found.append(root)
            continue
        for path in sorted(root.rglob("*")):
            if path.suffix not in SOURCE_SUFFIXES or not path.is_file():
                continue
            if SKIP_DIR_PARTS & set(path.parts):
                continue
            found.append(path)
    return found


def module_name(path: Path) -> str:
    """Return the dotted name a caller is recorded under."""
    parts = list(path.with_suffix("").parts)
    for prefix in ("src", "internal"):
        if prefix in parts:
            parts = parts[parts.index(prefix) + 1 :]
            break
    return ".".join(parts)


# --- parsing ----------------------------------------------------------------


def parse_python(path: Path, text: str) -> FileEntry:
    """Build one entry from a Python file using the AST — no regex guessing."""
    entry = FileEntry(path.as_posix(), layer_of(path), "", sha_of(text), text.count("\n") + 1)
    try:
        tree = ast.parse(text)
    except SyntaxError as exc:
        entry.header = f"(does not parse: {exc.msg})"
        return entry

    entry.header = first_sentence(ast.get_docstring(tree) or "")
    for node in tree.body:
        if isinstance(node, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)):
            if node.name.startswith("_"):
                continue
            kind = "class" if isinstance(node, ast.ClassDef) else "fn"
            entry.symbols.append(
                Symbol(node.name, kind, node.lineno, first_sentence(ast.get_docstring(node) or ""))
            )
            if isinstance(node, ast.ClassDef):
                for child in node.body:
                    if isinstance(child, (ast.FunctionDef, ast.AsyncFunctionDef)):
                        if child.name.startswith("_"):
                            continue
                        entry.symbols.append(
                            Symbol(
                                f"{node.name}.{child.name}",
                                "method",
                                child.lineno,
                                first_sentence(ast.get_docstring(child) or ""),
                            )
                        )
    return entry


def doc_block_above(lines: list[str], index: int) -> list[str]:
    """Return the comment block directly above a declaration, outermost line first.

    Only contiguous comment lines count. Walking further up picks the previous
    declaration's code as a summary — which is how an index ends up describing
    a symbol with `});`.
    """
    block: list[str] = []
    i = index - 1
    while i >= 0 and not lines[i].strip():
        i -= 1
    while i >= 0:
        line = lines[i].strip()
        if not (line.startswith(("//", "*", "/*")) or line.endswith("*/")):
            break
        cleaned = line.lstrip("/*").rstrip("*/").strip()
        if cleaned:
            block.append(cleaned)
        i -= 1
    return list(reversed(block))


def go_package_doc(lines: list[str]) -> str:
    """Return the package comment's first sentence, without license or directive lines.

    GoDoc only reads the comment touching the `package` clause. Taking every
    comment in the opening lines made the license notice every file's header.
    """
    pkg = next((n for n, line in enumerate(lines) if line.startswith("package ")), None)
    block: list[str] = []
    i = (pkg or 0) - 1
    while i >= 0 and lines[i].startswith("//"):
        text = lines[i][2:].strip()
        if not text.startswith(("go:", "+build")) and not LICENSE_RE.search(text):
            block.append(text)
        i -= 1
    return first_sentence("\n".join(reversed(block)))


def parse_regex(path: Path, text: str) -> FileEntry:
    """Build one entry from a TypeScript or Go file by line-matching declarations."""
    lines = text.splitlines()
    entry = FileEntry(path.as_posix(), layer_of(path), "", sha_of(text), len(lines))
    head = "\n".join(lines[:15])
    if path.suffix in GO_SUFFIXES:
        entry.header = go_package_doc(lines)  # SereniBase
    else:
        entry.header = first_sentence(head) if head.lstrip().startswith(("/*", "//")) else ""

    block_kind = ""  # inside a grouped `const (` / `var (` / `type (`
    for n, line in enumerate(lines):
        if path.suffix in TS_SUFFIXES:
            m = TS_EXPORT_RE.match(line.strip())
            kind = m.group("kind") if m else ""
        elif block_kind:
            if line.startswith(")"):
                block_kind = ""
                continue
            m = GO_BLOCK_ENTRY_RE.match(line)
            kind = block_kind
        else:
            opened = GO_BLOCK_RE.match(line)
            if opened:
                block_kind = opened.group("kind")
                continue
            m = GO_FUNC_RE.match(line) or GO_TYPE_RE.match(line)
            kind = line.split()[0] if m else ""
        if not m:
            continue
        name = m.group("name")
        if kind == "func":
            recv = GO_RECV_RE.match(line)
            kind, name = ("method", f"{recv.group('recv')}.{name}") if recv else ("fn", name)
        summary = first_sentence("\n".join(doc_block_above(lines, n)))
        entry.symbols.append(Symbol(name, kind, n + 1, summary))
    return entry


def parse(path: Path, text: str) -> FileEntry:
    """Build one entry for any supported language."""
    entry = parse_python(path, text) if path.suffix in PY_SUFFIXES else parse_regex(path, text)
    if entry.layer == "agent":
        for key, pattern in NODE_DECL_RE.items():
            m = pattern.search(text)
            if m:
                value = m.group(1).split("#")[0].strip().rstrip(",").strip()
                entry.node[key] = value or "()"
    return entry


# --- callers ----------------------------------------------------------------


def attach_callers(entries: list[FileEntry], bodies: dict[str, str]) -> None:
    """Fill in each symbol's caller list by scanning every other file once.

    Textual, not semantic: a name that appears in another module counts as a
    call. That over-reports on a common name and under-reports on a dynamic
    dispatch, which is why the index records callers as a starting point for a
    search and never as proof — the standard says read the file before asserting.

    Each body is tokenised once. One regex pass per symbol per file took minutes
    on a few hundred files with a few thousand symbols.
    """
    mentions: dict[str, set[str]] = {}
    calls: dict[str, set[str]] = {}
    selected: dict[str, set[str]] = {}
    for p, body in bodies.items():
        for m in TOKEN_RE.finditer(body):
            mentions.setdefault(m.group(2), set()).add(p)
            if m.group(3):
                calls.setdefault(m.group(2), set()).add(p)
            if m.group(1):
                selected.setdefault(m.group(2), set()).add(p)

    for entry in entries:
        for symbol in entry.symbols:
            leaf = symbol.name.split(".")[-1]
            if len(leaf) < 4:  # too short to attribute textually
                continue
            files = calls.get(leaf, set())
            # SereniBase: in Go a type, const or var is referenced, not called
            # (`*AuthHandler`, `constant.MaxRows`), and a handler is passed as a
            # method value (`handlers.Auth.LoginUser`) — so count those too.
            if entry.path.endswith(".go"):
                if symbol.kind in {"type", "const", "var"}:
                    files = mentions.get(leaf, set())
                else:
                    files = files | selected.get(leaf, set())
            callers = {module_name(Path(p)) for p in files if p != entry.path}
            symbol.callers = sorted(callers)[:6]


# --- rendering --------------------------------------------------------------


def render_markdown(
    entries: list[FileEntry], roots: list[Path], contracts: list[str], full: bool = False
) -> str:
    """Render the index an agent reads at the start of a task.

    Compact by default: one line per file, because an index that costs more
    tokens to read than the files it replaces has defeated itself. `--full`
    expands every symbol, for the rare case where the whole surface is wanted;
    the usual way to get symbol detail is `--query`, which reads `index.json`
    and prints only the match.
    """
    by_layer: dict[str, list[FileEntry]] = {}
    for entry in entries:
        by_layer.setdefault(entry.layer, []).append(entry)

    out: list[str] = [
        "# Repository memory",
        "",
        "Generated by `scripts/memory.py` — do not edit by hand; edit the code and",
        "rebuild. Every line here is copied from the source, so it is as true as the",
        "last build and no truer. `python scripts/memory.py --verify` says whether it",
        "is current. Treat an entry as a pointer to `file:line`, not as evidence:",
        "open the file before asserting anything about behaviour.",
        "",
        "One line per file. For a symbol's line number, summary and callers:",
        "`python scripts/memory.py --query <name>`.",
        "",
        f"Built: {datetime.now(timezone.utc).isoformat(timespec='seconds')} · "
        f"{len(entries)} files · {sum(len(e.symbols) for e in entries)} exported symbols",
        "",
    ]

    if contracts:
        out += ["## Contracts", ""]
        out += [f"- `{c}`" for c in contracts]
        out += [""]

    order = ["transport", "service", "repository", "agent", "model", "schema", "core", "other", "test"]
    for layer in sorted(by_layer, key=lambda l: order.index(l) if l in order else 99):
        files = sorted(by_layer[layer], key=lambda e: e.path)
        out += [f"## {layer} ({len(files)} files)", ""]
        for entry in files:
            header = f" — {entry.header}" if entry.header else ""
            if full:
                out.append(f"### `{entry.path}`{header}")
                if entry.node:
                    out.append(f"  node: {', '.join(f'{k}={v}' for k, v in entry.node.items())}")
                for symbol in entry.symbols:
                    summary = f" {symbol.summary}" if symbol.summary else ""
                    callers = f"  ← {', '.join(symbol.callers)}" if symbol.callers else ""
                    out.append(f"  - `{symbol.name}`:{symbol.line} ({symbol.kind}){summary}{callers}")
                out.append("")
                continue
            names = ", ".join(s.name for s in entry.symbols[:4])
            if len(entry.symbols) > 4:
                names += f", +{len(entry.symbols) - 4}"
            out.append(f"- `{entry.path}`{header}")
            if entry.node:
                out.append(f"    node: {', '.join(f'{k}={v}' for k, v in entry.node.items())}")
            if names:
                out.append(f"    exports: {names}")
        out.append("")
    return "\n".join(out) + "\n"


# --- commands ---------------------------------------------------------------


def contract_files() -> list[str]:
    """Return the contract files that exist, matching the name's case exactly."""
    found = []
    for name in CONTRACT_FILES:
        path = Path(name)
        if path.exists() and path.name in {p.name for p in path.parent.iterdir()}:
            found.append(name)
    return found


def load_index(out_dir: Path) -> dict:
    """Return the stored index, or an empty one when there is none yet."""
    path = out_dir / "index.json"
    if not path.exists():
        return {}
    return json.loads(path.read_text(encoding="utf-8"))


def build(roots: list[Path], out_dir: Path, full: bool = False) -> tuple[list[FileEntry], int]:
    """Parse every source file, attach callers, and write both index files."""
    sources = iter_sources(roots)
    if not sources:
        return [], 0

    index = load_index(out_dir)
    reusable = index.get("format") == INDEX_FORMAT
    stored = {f["path"]: f for f in index.get("files", [])} if reusable else {}
    bodies = {p.as_posix(): p.read_text(encoding="utf-8", errors="replace") for p in sources}

    entries: list[FileEntry] = []
    reparsed = 0
    for path in sources:
        text = bodies[path.as_posix()]
        prior = stored.get(path.as_posix())
        if prior and prior.get("sha") == sha_of(text):
            prior = dict(prior)
            prior["symbols"] = [Symbol(**s) for s in prior.get("symbols", [])]
            entries.append(FileEntry(**prior))
            continue
        entries.append(parse(path, text))
        reparsed += 1

    attach_callers(entries, bodies)

    contracts = contract_files()
    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "index.json").write_text(
        json.dumps(
            {
                "format": INDEX_FORMAT,
                "generated": datetime.now(timezone.utc).isoformat(timespec="seconds"),
                "roots": [str(r) for r in roots],
                "contracts": contracts,
                "files": [asdict(e) for e in entries],
            },
            indent=2,
        ),
        encoding="utf-8",
    )
    (out_dir / "INDEX.md").write_text(render_markdown(entries, roots, contracts, full), encoding="utf-8")
    return entries, reparsed


def verify(roots: list[Path], out_dir: Path) -> int:
    """Report files the index is behind on. Exit 1 when any are found."""
    stored = load_index(out_dir)
    if not stored:
        print("memory: no index yet — run `python scripts/memory.py`", file=sys.stderr)
        return 1

    known = {f["path"]: f["sha"] for f in stored["files"]}
    on_disk = {p.as_posix(): sha_of(p.read_text(encoding="utf-8", errors="replace")) for p in iter_sources(roots)}

    changed = sorted(p for p, sha in on_disk.items() if known.get(p) != sha)
    removed = sorted(set(known) - set(on_disk))

    if not changed and not removed:
        print(f"memory: index current ({len(known)} files, built {stored['generated']})")
        return 0
    for path in changed:
        print(f"stale: {path}" if path in known else f"missing: {path}")
    for path in removed:
        print(f"deleted: {path} (still in index)")
    print(f"\nmemory: {len(changed) + len(removed)} files out of date — run `python scripts/memory.py`")
    return 1


def show_file(target: str, out_dir: Path) -> int:
    """Print one file's whole entry: header, every symbol, line and caller list."""
    stored = load_index(out_dir)
    if not stored:
        print("memory: no index yet — run `python scripts/memory.py`", file=sys.stderr)
        return 2

    wanted = Path(target).as_posix()
    entry = next(
        (f for f in stored["files"] if f["path"] == wanted or f["path"].endswith("/" + wanted)),
        None,
    )
    if entry is None:
        print(
            f"memory: {target} is not in the index — it may be new, excluded, or the index may be "
            "stale; run --verify before concluding it does not exist"
        )
        return 1

    print(f"{entry['path']} [{entry['layer']}] {entry['lines']} lines")
    if entry["header"]:
        print(f"  {entry['header']}")
    if entry.get("node"):
        print("  node: " + ", ".join(f"{k}={v}" for k, v in entry["node"].items()))
    for symbol in entry["symbols"]:
        print(f"  {entry['path']}:{symbol['line']}  {symbol['name']} ({symbol['kind']})")
        if symbol["summary"]:
            print(f"      {symbol['summary']}")
        if symbol["callers"]:
            print(f"      callers (indexed, textual): {', '.join(symbol['callers'])}")
    return 0


def query(term: str, out_dir: Path) -> int:
    """Print every symbol or file matching `term`, with its callers."""
    stored = load_index(out_dir)
    if not stored:
        print("memory: no index yet — run `python scripts/memory.py`", file=sys.stderr)
        return 2

    needle = term.lower()
    hits = 0
    for entry in stored["files"]:
        if needle in entry["path"].lower():
            print(f"{entry['path']} [{entry['layer']}] {entry['header']}")
            for symbol in entry["symbols"]:
                print(f"  {entry['path']}:{symbol['line']}  {symbol['name']} ({symbol['kind']}) {symbol['summary']}")
            hits += 1
            continue
        for symbol in entry["symbols"]:
            if needle in symbol["name"].lower():
                callers = ", ".join(symbol["callers"]) or "no textual match found"
                print(f"{entry['path']}:{symbol['line']}  {symbol['name']} ({symbol['kind']}) [{entry['layer']}]")
                if symbol["summary"]:
                    print(f"    {symbol['summary']}")
                print(f"    callers (indexed, textual): {callers}")
                hits += 1
    if not hits:
        print(f"memory: nothing matching '{term}' — the index may be stale; verify before concluding it does not exist")
    return 0


def main(argv: list[str] | None = None) -> int:
    """Run the memory builder and return its exit code."""
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument(
        "--paths", nargs="*", default=None, help="roots to index (default: src, internal, cmd, web/src, app, lib)"
    )
    parser.add_argument("--out", default=".claude/memory", help="where the index lives")
    parser.add_argument("--verify", action="store_true", help="check the index against the working tree")
    parser.add_argument("--stale", action="store_true", help="alias for --verify")
    parser.add_argument("--query", metavar="TERM", help="look up a symbol or file")
    parser.add_argument("--file", metavar="PATH", help="print one file's whole entry")
    parser.add_argument("--full", action="store_true", help="expand every symbol in INDEX.md")
    args = parser.parse_args(argv)

    out_dir = Path(args.out)
    if args.file:
        return show_file(args.file, out_dir)
    if args.query:
        return query(args.query, out_dir)

    # SereniBase: cmd added to the default roots. tests/ is left out: it doubled
    # the index and its files took most caller slots; use `--paths tests` for it.
    defaults = [p for p in ("src", "internal", "cmd", "web/src", "app", "lib") if Path(p).is_dir()]
    defaults += sorted(str(p) for p in Path("packages").glob("*/src") if p.is_dir())
    roots = [Path(p) for p in (args.paths or defaults or ["."])]
    for root in roots:
        if not root.exists():
            print(f"memory: {root} does not exist", file=sys.stderr)
            return 2

    if args.verify or args.stale:
        return verify(roots, out_dir)

    entries, reparsed = build(roots, out_dir, full=args.full)
    if not entries:
        print("memory: no source files found", file=sys.stderr)
        return 2
    symbols = sum(len(e.symbols) for e in entries)
    size = (out_dir / "INDEX.md").stat().st_size
    print(
        f"memory: {len(entries)} files, {symbols} symbols "
        f"({reparsed} reparsed, {len(entries) - reparsed} from cache) → {out_dir}/INDEX.md "
        f"({size // 1024} KB, ~{size // 4000}k tokens)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

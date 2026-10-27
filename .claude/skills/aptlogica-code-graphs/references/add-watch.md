# graphify reference: add a URL and watch a folder

Load this when the user ran `/aptlogica-code-graphs add <url>` or passed `--watch`. Neither is part of the default build.

## For /aptlogica-code-graphs add

Fetch a URL and add it to the corpus, then update the graph.

Write `URL` (required), and `AUTHOR`/`CONTRIBUTOR` (if the user gave a name) verbatim to `graphify-out/.graphify_url.txt`, `graphify-out/.graphify_author.txt`, `graphify-out/.graphify_contributor.txt` with the Write tool — never hand-substitute any of these into the Python string below, they are external/user-supplied text that can contain `'`, `"`, or `` $(...) ``. Skip writing the author/contributor files (leave them empty or absent) when not provided.

```bash
$(cat graphify-out/.graphify_python) -c "
import sys
from graphify.ingest import ingest
from pathlib import Path

def _read(path):
    p = Path(path)
    if not p.exists():
        return None
    return p.read_text(encoding='utf-8').strip() or None

url = Path('graphify-out/.graphify_url.txt').read_text(encoding='utf-8').strip()
author = _read('graphify-out/.graphify_author.txt')
contributor = _read('graphify-out/.graphify_contributor.txt')

try:
    out = ingest(url, Path('./raw'), author=author, contributor=contributor)
    print(f'Saved to {out}')
except ValueError as e:
    print(f'error: {e}', file=sys.stderr)
    sys.exit(1)
except RuntimeError as e:
    print(f'error: {e}', file=sys.stderr)
    sys.exit(1)
"
```

If the command exits with an error, tell the user what went wrong - do not silently continue. After a successful save, automatically run the `--update` pipeline on `./raw` to merge the new file into the existing graph.

Supported URL types (auto-detected):
- YouTube / any video URL → audio downloaded via yt-dlp, transcribed to `.txt` on next run (requires `pip install 'graphifyy[video]'`)
- Twitter/X → fetched via oEmbed, saved as `.md` with tweet text and author
- arXiv → abstract + metadata saved as `.md`
- PDF → downloaded as `.pdf`
- Images (.png/.jpg/.webp) → downloaded, Claude vision extracts on next run
- Any webpage → converted to markdown via html2text

---

## For --watch

Start a background watcher that monitors a folder and auto-updates the graph when files change.

```bash
$(cat graphify-out/.graphify_python) -m graphify.watch "INPUT_PATH" --debounce 3
```

Replace INPUT_PATH with the folder to watch. Behavior depends on what changed:

- **Code files only (.py, .ts, .go, etc.):** re-runs AST extraction + rebuild + cluster immediately, no LLM needed. `graph.json` and `GRAPH_REPORT.md` are updated automatically.
- **Docs, papers, or images:** writes a `graphify-out/needs_update` flag and prints a notification to run `/aptlogica-code-graphs --update` (LLM semantic re-extraction required).

Debounce (default 3s): waits until file activity stops before triggering, so a wave of parallel agent writes doesn't trigger a rebuild per file.

Press Ctrl+C to stop.

For agentic workflows: run `--watch` in a background terminal. Code changes from agent waves are picked up automatically between waves. If agents are also writing docs or notes, you'll need a manual `/aptlogica-code-graphs --update` after those waves.

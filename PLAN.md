# Open-Obsidian — Implementation Plan

**Repo:** `github.com/sagnikc395/open-obsidian` | **Stack:** Go 1.27 + Fyne v2.8.0 | **Date:** 2026-08-26

## 1. Goal

Build an Obsidian clone in Go + Fyne for storing, writing and reading markdown with support for links, images, etc., with:
- Custom logic renderer for CRDT text editor
- Custom graphical logic for graph views between linked notes and notes by tag

## 2. Four Pillars

| Pillar | Requirement |
|---|---|
| A. Storage / IO | Vault = folder of `.md`; open/create/rename/delete; FS watch; plain files |
| B. Markdown | `[[wikilink]]` / `[md](url)`, images `![alt](...)` / `![[embed]]`, tags `#tag`, GFM |
| C. CRDT Editor | Custom `WidgetRenderer` for collaborative text editing |
| D. Graph View | Custom `canvas` graph: notes as nodes, links as edges, tag clusters |

## 3. Stack Decisions (verified 2026-08-26)

| Subsystem | Choice | Why |
|---|---|---|
| Fyne | `fyne.io/fyne/v2 v2.8.0` (go 1.22 floor, forward-compat 1.27) | Latest: RichText perf rewrite, tables, nested lists/quotes, GPU shapes |
| Markdown parse | `yuin/goldmark` + `yuin/goldmark-meta` + `alecthomas/chroma` | Hugo-proven, pure Go, extensible. Custom `InlineParser` for `[[`/`![[`/`#tag`. Blackfriday rejected (37% CommonMark). |
| Markdown preview | Preprocessor → `widget.RichTextFromMarkdown()` (MVP) → custom `WidgetRenderer` per AST node (callouts/embeds) | RichText is read-only, cannot host widgets inline. |
| Editor | `widget.Entry` (MultiLine) wrapped as `CRDTEditor` + custom renderer for per-rune styling + CRDT cursors | Only editable primitive in Fyne. |
| CRDT | `reearth/ygo` (pure Go Yjs, YText per note, y-protocols, WAL SQLite) | Pure Go, no CGO, wire-compat yjs@13.6, Hocuspocus `yserve`. Alternative `develerltd/go-automerge` for JSON vault. `automerge-go` (CGO) and `gotomerge` (no sync) rejected. |
| File watch | `fsnotify/fsnotify` debounce 150ms | Ignore `.obsidian/`, `.git/` |
| Search | `sahilm/fuzzy` + inverted `map[token][]file` | Bleve deferred |
| Graph layout | Custom Fruchterman-Reingold/Eades + `gonum/graph` + vendored `fynehax/Viewport` | `fyne-x/diagramwidget` is manual, `fyne-charts` is Cartesian, no built-in network graph. |
| Config | `~/.config/open-obsidian/config.json` + per-vault `.obsidian/app.json` | Minimal compat |

## 4. Architecture

```
+--------------------------------------------------------------+
| Fyne App (app.New, fyne.Do threading)                        |
|  BorderLayout:                                               |
|   Top: Toolbar (new note, toggle preview, graph, search)     |
|   Left: FileTree (widget.Tree) + Tag list + Outline          |
|   Center: HSplit[ CRDTEditor | Preview(RichText/custom)]     |
|   Right: Backlinks + Outgoing Links + Unlinked mentions      |
|   Bottom: StatusBar                                          |
|   Overlay: GraphView (custom widget, modal)                  |
+--------------------------------------------------------------+
         |                |                |             |
         v                v                v             v
   vault.Store      parser.Index     editor.CrdtDoc  graph.Layout
   - WalkDir         - goldmark AST   - ygo.Doc       - Force sim
   - fsnotify        - forward/inverse- YText          - canvas.*
   - CRUD            - tags/aliases   - UndoManager   - Viewport
   - .obsidian      - frontmatter    - awareness     - hit test
```

Data flow: `os.WriteFile` ↔ `CRDTEditor` ↔ `ygo.Doc` (YText) ↔ `parser.Index` (incremental diff) ↔ `GraphView` + `Backlinks`. Preview is derived.

## 5. Directory Layout

```
open-obsidian/
├── go.mod / go.sum
├── FyneApp.toml
├── cmd/open-obsidian/main.go
├── internal/
│   ├── vault/
│   │   ├── vault.go       # Vault{Root, Files, Watch}, Open/Scan/CRUD
│   │   ├── watcher.go     # fsnotify + debounce
│   │   └── vault_test.go
│   ├── parser/
│   │   ├── markdown.go    # goldmark + GFM/meta/highlighting
│   │   ├── wikilink.go    # [[target#heading|alias]] + ![[embed]] + resolver
│   │   ├── tags.go        # #tag regex, nested/tag, code-fence exclusion
│   │   ├── frontmatter.go # yaml aliases/tags/cssclass
│   │   └── index.go       # Forward/Inverse/Tags/Aliases, incremental update
│   ├── editor/
│   │   ├── crdt.go        # Doc{ ygo.Doc, YText, awareness }
│   │   ├── editor.go      # CRDTEditor widget (BaseWidget wrapping Entry)
│   │   └── renderer.go    # Custom logic renderer: canvas.Text per line + cursors
│   ├── preview/
│   │   ├── preview.go     # Preview widget (RichText + preprocessor)
│   │   └── renderer.go    # Custom AST→canvas renderer (future)
│   ├── graph/
│   │   ├── graph.go       # Graph{Nodes, Edges}
│   │   ├── layout.go      # Force-directed (k=√(area/N), repel=k²/d, attract=d²/k)
│   │   ├── widget.go      # GraphView widget (BaseWidget, Tappable/Draggable/Scrollable)
│   │   ├── renderer.go    # Objects() → []Circle/Line/Text + viewport matrix
│   │   └── viewport.go    # Pan/zoom
│   └── ui/
│       ├── app.go         # App wiring
│       ├── filetree.go    # widget.Tree backed by vault.Store
│       ├── backlinks.go   # Inverse links + snippet
│       ├── search.go      # QuickSwitcher + vault search
│       └── theme.go
├── assets/
└── vault_testdata/
```

## 6. Custom Renderers

### 6.1 CRDT Text Editor Renderer `internal/editor/renderer.go`

```go
type CRDTEditor struct { widget.BaseWidget; doc *crdt.Doc; content string }
func (e *CRDTEditor) CreateRenderer() fyne.WidgetRenderer { return &editorRenderer{...} }
type editorRenderer struct { editor *CRDTEditor; bg *canvas.Rectangle; lines []canvas.Text; cursors []canvas.Rectangle; objects []fyne.CanvasObject }
```

- State in widget, not renderer (CreateRenderer may be recalled on hide/show).
- Keystrokes → `YText.Insert/Delete` in `Transact`, `ApplyUpdate` → Refresh.
- Syntax spans tint `canvas.Text.Color`; remote cursors from `ygo.Awareness` as 2px `canvas.Rectangle`.
- Implements `Focusable`, `Tappable`, `Keyable`.

### 6.2 Preview Renderer `internal/preview/preview.go` + `renderer.go`

MVP: `widget.NewRichTextFromMarkdown(preprocess(md))` where `preprocess`:
- `[[target#heading|alias]]` → `[alias](note://resolved)` (ghost style if unresolved)
- `#tag` → `[#tag](tag://tag)` pills
- `![[file|100]]` → `![alt](resolvedPath)`

P1: Walk goldmark AST, create `canvas.Text`/`Image`/`Rectangle` per node; clickable via Tapped wrapper.

### 6.3 Graph Renderer `internal/graph/`

```go
type GraphView struct { widget.BaseWidget; nodes []Node; edges []Edge; layout *ForceLayout; scale, offsetX, offsetY float64 }
type graphRenderer struct { g *GraphView; circles []*canvas.Circle; lines []*canvas.Line; labels []*canvas.Text }
```

- 60fps ticker: `time.NewTicker(16ms)` → `fyne.Do(Refresh)`
- Forces: `repel=k²/d`, `attract=d²/k`, `center=(cx-x)*0.01`, damping 0.85
- Node color by first tag, size ∝ in-degree; cull labels at scale<0.6; cap anim at 2k nodes
- Interactions: Tapped→OnSelect, Dragged→pin, Scrolled→zoom [0.2,4.0]

## 7. Index & Resolver

- **Wikilink resolver** (`wikilink.go`): `map[lowerBaseName][]path`; prioritize same-folder > shortest path > first; case-insensitive; merge `aliases:` from frontmatter; `#heading` via heading index.
- **Tag index** (`tags.go`): `(?<![\w/])#([A-Za-z][\w\/\-]*)` after stripping fenced/inline code; `map[tag][]file` + `map[file][]tag`; split `/` for hierarchy.
- **Forward/Inverse** (`index.go`): `Forward map[source]map[target]int`, `Inverse` by invert; incremental on save (diff old vs new targets).

## 8. Phased Plan

| Phase | Scope | Duration |
|---|---|---|
| 0 Scaffolding | Deps, dirs, `go 1.27`, `fyne build` verify on darwin/arm64 | 1d |
| 1 Vault+Markdown | vault.Store WalkDir/Tree, Entry split, goldmark+meta, save, watcher | 1w |
| 2 Obsidian dialect | Wikilink/Embed/Tag parsers, preprocessor, images, tag pills | 1w |
| 3 Knowledge graph core | Parallel index, backlinks+snippets, QuickSwitcher, vault search, unlinked mentions | 1w |
| 4 CRDT editor | ygo.Doc per note, CRDTEditor custom renderer, syntax, undo, convergence test | 1–2w |
| 5 Graph view | GraphView + force layout + viewport; local depth 1–2 first, then global; tag coloring | 2w |
| 6 Polish | Outline, daily notes+templates, embed inline with cycle guard, callouts, auto-update links on rename, settings, drag-move, theme | ongoing |

## 9. Risks

| Risk | Mitigation |
|---|---|
| RichText wrapping at long docs | v2.8 rewrite + Scroll + chunk; P1 custom canvas removes dep |
| CRDT+CGO cross-compile | Pure-Go ygo |
| Graph O(n²) at >500 nodes | Barnes-Hut quadtree, LOD culling, static layout >2k |
| Regex false positives in code | Strip fenced/inline code before scan; AST is ground truth |
| Symlink loops / vaults-in-vaults | visited set, ignore symlink dirs |

## 10. Verification

- `go test ./...` — vault CRUD, parser edge cases, index incremental, CRDT convergence, layout stability
- `go run cmd/open-obsidian` — vault open, edit→preview, ghost create, backlinks live, graph drag/zoom 500 nodes
- `fyne-cross` smoke for darwin/linux/windows

## 11. Open Questions

1. Strict `.obsidian/` compat vs green-field config?
2. CRDT local-only vs p2p via yserve in MVP?
3. Local graph first or global flagship?
4. Images: copy-on-drop to `Attachments/` vs keep relative?
5. Source mode only vs Live Preview WYSIWYG?

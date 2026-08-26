# Open-Obsidian

Go + Fyne desktop Obsidian clone — local folder vault of markdown with wikilinks, tags, backlinks, and graph views.

## Features

- **Vault** = folder of `.md` (plain files, `.obsidian`/`.git` ignored). `internal/vault/vault.go:1` + `watcher.go:1` with `fsnotify` debounce.
- **Markdown** via `goldmark` GFM + YAML frontmatter (`tags`, `aliases`). `internal/parser/markdown.go:1`
- **Wikilinks** `[[Note]]` `[[Note#Heading|Alias]]` `![[embed]]`, **images** `![alt](path)`, **tags** `#tag` `#nested/tag`. `internal/parser/wikilink.go:1` `tags.go:1` `frontmatter.go:1`
- **Index** forward/inverse graph + tag/alias maps, incremental on save. `internal/parser/index.go:1`
- **CRDT editor** `Doc` sequence CRDT (`siteID:clock`) + `CRDTEditor` widget wrapping `widget.Entry` with custom logic renderer (`canvas.Text` per line + cursor rects + awareness). `internal/editor/crdt.go:1` `editor.go:1` `renderer.go:1`
- **Preview** `widget.RichText` with `[[`→`[alias](note://...)` + `#tag`→`[tag](tag://...)` preprocessor; custom AST→canvas renderer extension point. `internal/preview/preview.go:1` `renderer.go:1`
- **Graph view** custom widget: force-directed (Fruchterman-Reingold, `width*height` → `k`), `canvas.Circle`/`Line`/`Text`, `Viewport` pan/zoom, tag cluster colors (palette by hash), ghost dimming, LOD label culling. Global + local (BFS depth 1–6). `internal/graph/graph.go:1` `layout.go:1` `viewport.go:1` `widget.go:1` `renderer.go:1`
- **UI**: FileTree `widget.Tree`, Backlinks panel with snippets, Quick Switcher `Ctrl+O` (fuzzy), Vault Search `tag:#x` / full-text, toolbar (new note, toggle preview, graph, local graph, search). `internal/ui/app.go:1` `filetree.go:1` `backlinks.go:1` `search.go:1`

## Quickstart (Taskfile)

Requires [Task](https://taskfile.dev) (`brew install go-task`):

```bash
task --list                 # show all tasks
task build                  # -> dist/open-obsidian
task run                    # go run ./cmd/open-obsidian (VAULT=/path/to/vault)
task run -- VAULT=./vault
task test                   # go test ./... -v
task vet                    # go vet
task check                  # fmt + vet + test (CI gate)
task vault:init             # create sample vault/Welcome.md
task package                # fyne package (needs fyne CLI)
```

Raw Go (without Task):

```bash
go run ./cmd/open-obsidian          # picker → ./vault or path arg
go run ./cmd/open-obsidian /path/to/vault
go build -o /tmp/open-obsidian ./cmd/open-obsidian && /tmp/open-obsidian /path/to/vault
```

Vault picker creates `vault/Welcome.md` + `Second Note.md` samples if empty.

## Stack

`Go 1.27` · `fyne.io/fyne/v2 v2.8.1` · `yuin/goldmark` + `goldmark-highlighting` + `goldmark-meta` · `fsnotify` · `sahilm/fuzzy` · `gopkg.in/yaml.v3`

## Custom Renderers

1. **CRDT editor renderer** `internal/editor/editor.go:CreateRenderer()` — state in widget (Fyne may recreate renderer on hide/show), `editorRenderer.Layout/Refresh/Objects` owns `canvas.Text` + `canvas.Rectangle` cursors. Remote cursors from `Awareness`.
2. **Graph renderer** `internal/graph/widget.go:graphRenderer` — `Objects()` returns `[]Circle/Line/Text` transformed via `Viewport.WorldToScreen`, `Layout` resizes bg + recomputes `k`, `Refresh` via 60fps ticker `fyne.Do`.

## Tests

```bash
task test
task test:cover
task check
# raw:
go vet ./...
go test ./... -v
```

## Project Layout

See `PLAN.md:1` for phased plan and architecture. `FyneApp.toml:1` for app metadata.

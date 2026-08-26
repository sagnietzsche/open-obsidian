package ui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/sagnikc395/open-obsidian/internal/editor"
	"github.com/sagnikc395/open-obsidian/internal/graph"
	"github.com/sagnikc395/open-obsidian/internal/parser"
	"github.com/sagnikc395/open-obsidian/internal/preview"
	"github.com/sagnikc395/open-obsidian/internal/vault"
)

// App wires the whole Obsidian clone.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	store   *vault.Store
	index   *parser.Index
	watcher *vault.Watcher

	fileTree  *FileTree
	editor    *editor.CRDTEditor
	preview   *preview.Preview
	backlinks *BacklinksPanel
	graphView *graph.GraphView

	activeFile  string
	doc         *editor.Doc
	showPreview bool
}

func NewApp(fyneApp fyne.App, win fyne.Window, store *vault.Store) *App {
	idx := parser.NewIndex()
	_ = idx.Build(store.Files(), store.ReadFile)
	fyneApp.Settings().SetTheme(ObsidianTheme{})
	a := &App{fyneApp: fyneApp, window: win, store: store, index: idx, showPreview: true}
	a.doc = editor.NewDoc("local", "")
	a.editor = editor.NewCRDTEditor(a.doc)
	a.preview = preview.NewPreview()
	a.fileTree = NewFileTree(store, a.OpenFile)
	a.backlinks = NewBacklinksPanel(store, idx, a.OpenFile)
	a.graphView = graph.NewGraphView()
	a.graphView.OnSelect = a.OpenFile
	a.editor.OnChanged = a.onEditorChanged
	a.preview.OnNavigate = a.onNavigate
	// initial graph data
	nodes, edges := graph.BuildGraph(idx.Forward, idx.FileTags, false, true)
	a.graphView.SetData(nodes, edges, idx.FileTags, true)
	// tick graph layout
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if a.graphView.Tick() {
				continue
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	a.startWatcher()
	return a
}

func (a *App) startWatcher() {
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
	w, err := vault.NewWatcher(a.store, func() {
		fyne.Do(func() {
			a.refreshFromVault()
		})
	})
	if err != nil {
		// watcher optional; ignore error (e.g. too many watches)
		return
	}
	a.watcher = w
}

func (a *App) refreshFromVault() {
	// Rebuild index after external FS change; keep active file content in sync
	prevFiles := a.store.Files()
	_ = a.index.Build(prevFiles, a.store.ReadFile)
	a.fileTree.RefreshTree()
	if a.activeFile != "" {
		if content, err := a.store.ReadFile(a.activeFile); err == nil {
			// if editor dirty differs from disk, prefer disk if editor not changed recently; for now just update preview/index
			// Don't clobber unsaved editor content: only refresh if disk differs and editor matches old doc?
			// Simple: if editor text != disk, keep editor; else sync.
			if a.editor.Text() == content {
				// already in sync
			} else {
				// external edit: reload if file still exists
				a.doc.SetText(content)
				a.editor.SetText(content)
			}
			a.preview.SetContent(content, a.activeFile, a.store.Files(), a.index.Aliases)
			a.backlinks.SetActive(a.activeFile)
		} else {
			// file was deleted externally
			if len(prevFiles) > 0 {
				a.OpenFile(prevFiles[0])
			} else {
				a.activeFile = ""
			}
		}
	}
	nodes, edges := graph.BuildGraph(a.index.Forward, a.index.FileTags, false, true)
	a.graphView.SetData(nodes, edges, a.index.FileTags, true)
}

// SetVault switches the vault at runtime (folder as vault).
func (a *App) SetVault(newStore *vault.Store) {
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
	a.store = newStore
	_ = a.index.Build(newStore.Files(), newStore.ReadFile)
	a.fileTree.SetStore(newStore)
	a.backlinks.SetStore(newStore, a.index)
	a.activeFile = ""
	a.startWatcher()
	nodes, edges := graph.BuildGraph(a.index.Forward, a.index.FileTags, false, true)
	a.graphView.SetData(nodes, edges, a.index.FileTags, true)
	a.fileTree.RefreshTree()
	a.backlinks.SetActive("")
	// reset editor/preview
	a.doc.SetText("")
	a.editor.SetText("")
	a.preview.SetContent("", "", a.store.Files(), a.index.Aliases)
	a.window.SetTitle("Open-Obsidian — " + newStore.Root)
	files := newStore.Files()
	if len(files) > 0 {
		a.OpenFile(files[0])
	}
}

func (a *App) showOpenVaultDialog() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		if uri == nil {
			return
		}
		path := uri.Path()
		if path == "" {
			return
		}
		newStore, err := vault.New(path)
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		// seed if empty
		if len(newStore.Files()) == 0 {
			_ = os.MkdirAll(path, 0755)
		}
		a.SetVault(newStore)
	}, a.window)
}

// createNewNote creates a note with given display name (may include subfolder like "folder/Note").
// It handles dedup, ensures .md suffix, writes initial header, rescans, and opens it.
func (a *App) createNewNote(displayName string) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "Untitled"
	}
	// sanitize: remove leading slashes
	name = strings.TrimPrefix(name, "/")
	// ensure .md
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}
	// dedup if exists
	orig := name
	base := strings.TrimSuffix(orig, ".md")
	ext := ".md"
	i := 1
	for {
		if _, err := a.store.ReadFile(name); err != nil {
			break
		}
		name = base + " " + itoa(i) + ext
		i++
		if i > 1000 {
			break
		}
	}
	title := strings.TrimSuffix(filepath.Base(name), ".md")
	initial := "# " + title + "\n\n"
	if err := a.store.WriteFile(name, initial); err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	_ = a.store.Scan()
	_ = a.index.Build(a.store.Files(), a.store.ReadFile)
	a.fileTree.RefreshTree()
	nodes, edges := graph.BuildGraph(a.index.Forward, a.index.FileTags, false, true)
	a.graphView.SetData(nodes, edges, a.index.FileTags, true)
	a.OpenFile(name)
}

func (a *App) showNewNoteDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Note name, e.g. My Note or folder/My Note")
	entry.SetText("Untitled")
	formDialog := dialog.NewForm("New Note", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", entry),
		}, func(ok bool) {
			if !ok {
				return
			}
			a.createNewNote(entry.Text)
		}, a.window)
	formDialog.Resize(fyne.NewSize(420, 140))
	formDialog.Show()
	// focus entry
	a.window.Canvas().Focus(entry)
}

func (a *App) onEditorChanged(text string) {
	if a.activeFile == "" {
		return
	}
	if err := a.store.WriteFile(a.activeFile, text); err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	a.index.UpdateFile(a.activeFile, text, a.store.Files())
	a.preview.SetContent(text, a.activeFile, a.store.Files(), a.index.Aliases)
	a.backlinks.SetActive(a.activeFile)
	// refresh graph lazily
	nodes, edges := graph.BuildGraph(a.index.Forward, a.index.FileTags, false, true)
	a.graphView.SetData(nodes, edges, a.index.FileTags, true)
	// file tree if new file
	a.fileTree.RefreshTree()
}

func (a *App) onNavigate(target string) {
	if strings.HasPrefix(target, "note://") {
		path := strings.TrimPrefix(target, "note://")
		// handle ghost and heading anchor
		if idx := strings.Index(path, "#"); idx >= 0 {
			path = path[:idx]
		}
		if strings.HasPrefix(path, "ghost/") {
			real := strings.TrimPrefix(path, "ghost/")
			// create file
			rel := real
			if !strings.HasSuffix(strings.ToLower(rel), ".md") {
				rel += ".md"
			}
			_ = a.store.WriteFile(rel, "# "+real+"\n\n")
			_ = a.store.Scan()
			_ = a.index.Build(a.store.Files(), a.store.ReadFile)
			a.OpenFile(rel)
			a.fileTree.RefreshTree()
			return
		}
		a.OpenFile(path)
	} else if strings.HasPrefix(target, "tag://") {
		tag := strings.TrimPrefix(target, "tag://")
		files := a.index.Tags[tag]
		if len(files) > 0 {
			a.OpenFile(files[0])
		}
		dialog.ShowInformation("Tag", "#"+tag+" → "+strings.Join(files, ", "), a.window)
	}
}

func (a *App) OpenFile(rel string) {
	content, err := a.store.ReadFile(rel)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	a.activeFile = rel
	a.doc.SetText(content)
	a.editor.SetText(content)
	a.preview.SetContent(content, rel, a.store.Files(), a.index.Aliases)
	a.backlinks.SetActive(rel)
	a.window.SetTitle("Open-Obsidian — " + rel)
}

func (a *App) BuildUI() fyne.CanvasObject {
	// Toolbar
	newNoteBtn := widget.NewButton("New Note", func() { a.showNewNoteDialog() })
	openVaultBtn := widget.NewButton("Open Vault", func() { a.showOpenVaultDialog() })
	togglePreviewBtn := widget.NewButton("Toggle Preview", func() {
		a.showPreview = !a.showPreview
		if a.showPreview {
			a.preview.Show()
		} else {
			a.preview.Hide()
		}
	})
	graphBtn := widget.NewButton("Graph View", func() {
		nodes, edges := graph.BuildGraph(a.index.Forward, a.index.FileTags, false, true)
		a.graphView.SetData(nodes, edges, a.index.FileTags, true)
		d := dialog.NewCustom("Graph View", "Close", a.graphView, a.window)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})
	localGraphBtn := widget.NewButton("Local Graph", func() {
		if a.activeFile == "" {
			return
		}
		a.graphView.SetLocal(a.activeFile, 2, a.index.Forward, a.index.Inverse, a.index.FileTags)
		d := dialog.NewCustom("Local Graph — "+a.activeFile, "Close", a.graphView, a.window)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})
	searchBtn := widget.NewButton("Search", func() { ShowVaultSearch(a.window, a.store, a.index, a.OpenFile) })
	quickBtn := widget.NewButton("Quick Open", func() { ShowQuickSwitcher(a.window, a.store, a.OpenFile) })

	toolbar := container.NewHBox(openVaultBtn, newNoteBtn, togglePreviewBtn, graphBtn, localGraphBtn, searchBtn, quickBtn)
	toolbarWrap := container.NewBorder(nil, nil, widget.NewLabelWithStyle("  OPEN-OBSIDIAN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, toolbar)

	// Center split editor | preview
	center := container.NewHSplit(a.editor, a.preview)
	center.SetOffset(0.5)

	// Leftmost panel (Explorer) with header + New Note button
	newNoteIconBtn := widget.NewButton("+", func() { a.showNewNoteDialog() })
	newNoteIconBtn.Importance = widget.LowImportance
	openVaultIconBtn := widget.NewButton("Open", func() { a.showOpenVaultDialog() })
	openVaultIconBtn.Importance = widget.LowImportance
	leftHeader := container.NewHBox(
		widget.NewLabelWithStyle("EXPLORER", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		openVaultIconBtn,
		newNoteIconBtn,
	)
	// Show current vault path as subtle label below header
	vaultLabel := widget.NewLabel(a.store.Root)
	vaultLabel.Wrapping = fyne.TextWrapOff
	vaultLabel.Truncation = fyne.TextTruncateEllipsis
	leftHeaderWithVault := container.NewVBox(leftHeader, vaultLabel, widget.NewSeparator())
	// Bottom action: prominent New Note button in leftmost panel (click to add note)
	newNoteBottomBtn := widget.NewButton("+ New Note", func() { a.showNewNoteDialog() })
	newNoteBottomBtn.Importance = widget.HighImportance
	leftFooter := container.NewVBox(widget.NewSeparator(), newNoteBottomBtn)
	left := container.NewBorder(leftHeaderWithVault, leftFooter, nil, nil, a.fileTree)

	rightHeader := widget.NewLabelWithStyle("BACKLINKS & TAGS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	right := container.NewBorder(rightHeader, nil, nil, nil, a.backlinks)

	mainSplit := container.NewHSplit(left, center)
	mainSplit.SetOffset(0.22)
	outer := container.NewHSplit(mainSplit, right)
	outer.SetOffset(0.75)

	content := container.NewBorder(toolbarWrap, nil, nil, nil, outer)
	return content
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

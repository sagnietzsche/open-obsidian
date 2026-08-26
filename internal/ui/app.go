package ui

import (
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
	return a
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
	// local graph focus
	// keep global for now; local can be toggled via button
	a.window.SetTitle("Open-Obsidian — " + rel)
}

func (a *App) BuildUI() fyne.CanvasObject {
	// Toolbar
	newNoteBtn := widget.NewButton("New Note", func() {
		rel := "Untitled.md"
		// avoid collision
		base := "Untitled"
		i := 1
		for {
			if _, err := a.store.ReadFile(rel); err != nil {
				break
			}
			rel = base + " " + itoa(i) + ".md"
			i++
		}
		if err := a.store.CreateFile(rel); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		if err := a.store.WriteFile(rel, "# "+strings.TrimSuffix(filepath.Base(rel), ".md")+"\n\n"); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		_ = a.store.Scan()
		_ = a.index.Build(a.store.Files(), a.store.ReadFile)
		a.fileTree.RefreshTree()
		a.OpenFile(rel)
	})
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

	toolbar := container.NewHBox(newNoteBtn, togglePreviewBtn, graphBtn, localGraphBtn, searchBtn, quickBtn)
	toolbarWrap := container.NewBorder(nil, nil, widget.NewLabelWithStyle("  OPEN-OBSIDIAN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, toolbar)

	// Center split editor | preview
	center := container.NewHSplit(a.editor, a.preview)
	center.SetOffset(0.5)

	leftHeader := container.NewHBox(widget.NewLabelWithStyle("EXPLORER", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), layout.NewSpacer())
	rightHeader := widget.NewLabelWithStyle("BACKLINKS & TAGS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	left := container.NewBorder(leftHeader, nil, nil, nil, a.fileTree)
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

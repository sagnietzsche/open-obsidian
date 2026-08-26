package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/sahilm/fuzzy"
	"github.com/sagnikc395/open-obsidian/internal/parser"
	"github.com/sagnikc395/open-obsidian/internal/vault"
)

// QuickSwitcher shows fuzzy file finder (Ctrl+O).
func ShowQuickSwitcher(win fyne.Window, store *vault.Store, onSelect func(rel string)) {
	files := store.Files()
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Type to search notes…")
	list := widget.NewList(
		func() int { return len(files) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(files[i]) },
	)
	filtered := files
	entry.OnChanged = func(s string) {
		if strings.TrimSpace(s) == "" {
			filtered = files
		} else {
			matches := fuzzy.Find(s, files)
			filtered = make([]string, len(matches))
			for i, m := range matches {
				filtered[i] = m.Str
			}
		}
		list.Length = func() int { return len(filtered) }
		list.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(filtered[i]) }
		list.Refresh()
	}
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(filtered) && onSelect != nil {
			onSelect(filtered[id])
		}
	}
	content := container.NewBorder(entry, nil, nil, nil, list)
	d := dialog.NewCustom("Quick Switcher", "Close", content, win)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

// VaultSearch does full-text search across files.
func ShowVaultSearch(win fyne.Window, store *vault.Store, index *parser.Index, onSelect func(rel string)) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Search content, tag:#foo, path:foo …")
	results := []string{}
	list := widget.NewList(
		func() int { return len(results) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(results[i]) },
	)
	entry.OnChanged = func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			results = nil
			list.Refresh()
			return
		}
		// tag: query
		if strings.HasPrefix(s, "tag:") {
			tag := strings.TrimPrefix(s, "tag:")
			tag = strings.TrimPrefix(tag, "#")
			results = index.Tags[tag]
			if results == nil {
				results = []string{}
			}
			list.Refresh()
			return
		}
		// general full-text: scan files
		var out []string
		qLower := strings.ToLower(s)
		for _, f := range store.Files() {
			content, err := store.ReadFile(f)
			if err != nil {
				continue
			}
			if strings.Contains(strings.ToLower(content), qLower) || strings.Contains(strings.ToLower(f), qLower) {
				out = append(out, f)
			}
		}
		results = out
		list.Refresh()
	}
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(results) && onSelect != nil {
			onSelect(results[id])
		}
	}
	content := container.NewBorder(entry, nil, nil, nil, list)
	d := dialog.NewCustom("Vault Search", "Close", content, win)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}

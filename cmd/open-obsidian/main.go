package main

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/sagnikc395/open-obsidian/internal/ui"
	"github.com/sagnikc395/open-obsidian/internal/vault"
)

func main() {
	fyneApp := app.NewWithID("com.openobsidian.app")
	win := fyneApp.NewWindow("Open-Obsidian")
	win.Resize(fyne.NewSize(1200, 800))

	// Determine vault path: arg or picker
	vaultPath := ""
	if len(os.Args) > 1 {
		vaultPath = os.Args[1]
	}
	if vaultPath == "" {
		// VAULT env also supported (for `task run -- VAULT=...`)
		if v := os.Getenv("VAULT"); v != "" {
			vaultPath = v
		}
	}
	if vaultPath == "" {
		// Try default ./vault
		cwd, _ := os.Getwd()
		candidate := filepath.Join(cwd, "vault")
		if _, err := os.Stat(candidate); err == nil {
			vaultPath = candidate
		}
	}
	if vaultPath == "" {
		// Show picker screen
		showPicker(fyneApp, win)
		win.ShowAndRun()
		return
	}
	store, err := vault.New(vaultPath)
	if err != nil {
		dialog.ShowError(err, win)
		showPicker(fyneApp, win)
		win.ShowAndRun()
		return
	}
	loadVault(fyneApp, win, store)
	win.ShowAndRun()
}

func showPicker(fyneApp fyne.App, win fyne.Window) {
	label := widget.NewLabel("Select vault folder (a folder containing .md files)")
	desc := widget.NewLabel("A vault is a plain folder of Markdown files. Pick an existing folder or create a new one.")
	desc.Wrapping = fyne.TextWrapWord

	entry := widget.NewEntry()
	entry.SetPlaceHolder("/path/to/vault  or  ./vault")

	openExisting := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			dialog.ShowInformation("Vault", "Please select or enter a folder path", win)
			return
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			dialog.ShowError(err, win)
			return
		}
		store, err := vault.New(path)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if len(store.Files()) == 0 {
			// seed with welcome notes so first launch is useful
			_ = store.WriteFile("Welcome.md", "# Welcome to Open-Obsidian\n\nThis is your vault. Create notes with [[Wikilinks]] and #tags.\n\n- Try [[Second Note]]\n- Try #todo\n\n![[Welcome]] embed example\n")
			_ = store.WriteFile("Second Note.md", "# Second Note\n\nBacklink to [[Welcome]] and a #project tag.\n\n```go\nfmt.Println(\"hello\")\n```\n")
			_ = store.Scan()
		}
		loadVault(fyneApp, win, store)
	}

	createBtn := widget.NewButton("Create/Open Vault", func() {
		openExisting(entry.Text)
	})
	browseNativeBtn := widget.NewButton("Browse…", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uri == nil {
				return
			}
			entry.SetText(uri.Path())
			openExisting(uri.Path())
		}, win)
	})
	useDefaultBtn := widget.NewButton("Use ./vault", func() {
		path := "./vault"
		_ = os.MkdirAll(path, 0755)
		store, err := vault.New(path)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if len(store.Files()) == 0 {
			_ = store.WriteFile("Welcome.md", "# Welcome\n\nStart writing. Link with [[Second Note]] #welcome\n")
			_ = store.WriteFile("Second Note.md", "# Second Note\n\nBack to [[Welcome]]\n")
			_ = store.Scan()
		}
		loadVault(fyneApp, win, store)
	})

	content := container.NewVBox(
		label,
		desc,
		entry,
		container.NewHBox(createBtn, browseNativeBtn, useDefaultBtn),
		widget.NewLabel("Tip: you can also run:  go run ./cmd/open-obsidian /path/to/vault"),
	)
	win.SetContent(container.NewCenter(container.NewVBox(content)))
}

func loadVault(fyneApp fyne.App, win fyne.Window, store *vault.Store) {
	a := ui.NewApp(fyneApp, win, store)
	win.SetContent(a.BuildUI())
	// Auto-open first file if any, otherwise keep editor empty (user can create note)
	if files := store.Files(); len(files) > 0 {
		a.OpenFile(files[0])
	}
	// Shortcuts: handled via menu/buttons; Fyne shortcuts use win.Canvas().AddShortcut
	win.Canvas().AddShortcut(&fyne.ShortcutCopy{}, func(s fyne.Shortcut) {
		// placeholder for future Ctrl+O / Ctrl+F
		_ = s
	})
}

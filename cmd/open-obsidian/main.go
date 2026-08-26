package main

import (
	"os"
	"path/filepath"

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
		// Try default ./vault or ask
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
	entry := widget.NewEntry()
	entry.SetPlaceHolder("/path/to/vault  or  ./vault")
	createBtn := widget.NewButton("Create/Open Vault", func() {
		path := entry.Text
		if path == "" {
			dialog.ShowInformation("Vault", "Please enter a path", win)
			return
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			dialog.ShowError(err, win)
			return
		}
		// create sample note if empty
		store, err := vault.New(path)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if len(store.Files()) == 0 {
			_ = store.WriteFile("Welcome.md", "# Welcome to Open-Obsidian\n\nThis is your vault. Create notes with [[Wikilinks]] and #tags.\n\n- Try [[Welcome]]\n- Try #todo\n\n![[Welcome]] embed example\n")
			_ = store.WriteFile("Second Note.md", "# Second Note\n\nBacklink to [[Welcome]] and a #project tag.\n\n```go\nfmt.Println(\"hello\")\n```\n")
			_ = store.Scan()
		}
		loadVault(fyneApp, win, store)
	})
	browseBtn := widget.NewButton("Use ./vault", func() {
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
	content := container.NewVBox(label, entry, container.NewHBox(createBtn, browseBtn))
	win.SetContent(content)
}

func loadVault(fyneApp fyne.App, win fyne.Window, store *vault.Store) {
	a := ui.NewApp(fyneApp, win, store)
	win.SetContent(a.BuildUI())
	// Auto-open first file
	if files := store.Files(); len(files) > 0 {
		a.OpenFile(files[0])
	}
	// Shortcuts: handled via menu/buttons; Fyne shortcuts use win.Canvas().AddShortcut
	win.Canvas().AddShortcut(&fyne.ShortcutCopy{}, func(s fyne.Shortcut) {
		// placeholder for future Ctrl+O / Ctrl+F
		_ = s
	})
}

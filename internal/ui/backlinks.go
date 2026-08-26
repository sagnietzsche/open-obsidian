package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sagnikc395/open-obsidian/internal/parser"
	"github.com/sagnikc395/open-obsidian/internal/vault"
)

// BacklinksPanel shows backlinks + outgoing + tags for active note.
type BacklinksPanel struct {
	widget.BaseWidget
	store   *vault.Store
	index   *parser.Index
	active  string
	onOpen  func(rel string)
	content *fyne.Container
}

func NewBacklinksPanel(store *vault.Store, index *parser.Index, onOpen func(rel string)) *BacklinksPanel {
	b := &BacklinksPanel{store: store, index: index, onOpen: onOpen}
	b.content = container.NewVBox()
	b.ExtendBaseWidget(b)
	return b
}

func (b *BacklinksPanel) SetActive(rel string) {
	b.active = rel
	b.rebuild()
}

func (b *BacklinksPanel) rebuild() {
	b.content.Objects = nil
	if b.active == "" {
		b.content.Add(widget.NewLabel("No note selected"))
		b.Refresh()
		return
	}
	// Backlinks
	backs := b.index.Backlinks(b.active)
	b.content.Add(widget.NewLabelWithStyle("Backlinks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if len(backs) == 0 {
		b.content.Add(widget.NewLabel("No backlinks"))
	} else {
		for _, src := range backs {
			srcCopy := src
			btn := widget.NewButton(src, func() {
				if b.onOpen != nil {
					b.onOpen(srcCopy)
				}
			})
			btn.Alignment = widget.ButtonAlignLeading
			// snippet
			snip := snippetFor(b.store, srcCopy, b.active)
			row := container.NewVBox(btn, widget.NewLabel(snip))
			b.content.Add(row)
		}
	}
	// Outgoing
	outs := b.index.ForwardLinks(b.active)
	b.content.Add(widget.NewSeparator())
	b.content.Add(widget.NewLabelWithStyle("Outgoing", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if len(outs) == 0 {
		b.content.Add(widget.NewLabel("No outgoing links"))
	} else {
		for _, tgt := range outs {
			tgtCopy := tgt
			lbl := tgt
			if strings.HasPrefix(tgt, "ghost/") {
				lbl = "[ghost] " + strings.TrimPrefix(tgt, "ghost/")
			}
			btn := widget.NewButton(lbl, func() {
				if b.onOpen != nil && !strings.HasPrefix(tgtCopy, "ghost/") {
					b.onOpen(tgtCopy)
				}
			})
			btn.Alignment = widget.ButtonAlignLeading
			b.content.Add(btn)
		}
	}
	// Tags
	tags := b.index.FileTags[b.active]
	// also from index.FileTags already includes fm tags
	if len(tags) == 0 {
		// fallback: parse file for tags
	}
	b.content.Add(widget.NewSeparator())
	b.content.Add(widget.NewLabelWithStyle(fmt.Sprintf("Tags (%d)", len(tags)), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, t := range tags {
		tCopy := t
		btn := widget.NewButton("#"+t, func() {
			if b.onOpen != nil {
				// tag search handled via OnNavigate? For now open first file with tag
				files := b.index.Tags[tCopy]
				if len(files) > 0 {
					b.onOpen(files[0])
				}
			}
		})
		btn.Alignment = widget.ButtonAlignLeading
		b.content.Add(btn)
	}
	b.Refresh()
}

func snippetFor(store *vault.Store, src, target string) string {
	content, err := store.ReadFile(src)
	if err != nil {
		return ""
	}
	// find line containing [[target]] or target base
	base := strings.TrimSuffix(target, ".md")
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), strings.ToLower(base)) {
			if len(l) > 80 {
				l = l[:80] + "…"
			}
			return l
		}
	}
	return ""
}

func (b *BacklinksPanel) CreateRenderer() fyne.WidgetRenderer {
	scroll := container.NewScroll(b.content)
	return widget.NewSimpleRenderer(scroll)
}

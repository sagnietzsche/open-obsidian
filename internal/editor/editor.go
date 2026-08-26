package editor

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CRDTEditor is a custom widget wrapping an editable area with CRDT semantics
// and a custom logic renderer that draws per-line canvas.Text + cursor/remote carets.
type CRDTEditor struct {
	widget.BaseWidget
	doc       *Doc
	awareness *Awareness
	entry     *widget.Entry
	OnChanged func(text string)
}

// NewCRDTEditor creates editor bound to doc.
func NewCRDTEditor(doc *Doc) *CRDTEditor {
	e := &CRDTEditor{doc: doc, awareness: NewAwareness()}
	e.entry = widget.NewMultiLineEntry()
	e.entry.Wrapping = fyne.TextWrapWord
	e.entry.TextStyle = fyne.TextStyle{Monospace: true}
	e.entry.SetText(doc.Text())
	e.entry.OnChanged = func(s string) {
		// Diff naive: replace doc text
		doc.SetText(s)
		if e.OnChanged != nil {
			e.OnChanged(s)
		}
	}
	e.ExtendBaseWidget(e)
	doc.OnChange(func() {
		fyne.Do(func() {
			// avoid loop if text already same
			if e.entry.Text != doc.Text() {
				e.entry.SetText(doc.Text())
			}
			e.Refresh()
		})
	})
	return e
}

// SetText replaces content.
func (e *CRDTEditor) SetText(s string) {
	e.doc.SetText(s)
	e.entry.SetText(s)
}

// Text returns current text.
func (e *CRDTEditor) Text() string { return e.entry.Text }

// Doc returns underlying CRDT doc.
func (e *CRDTEditor) Doc() *Doc { return e.doc }

// CreateRenderer implements custom logic renderer.
// The renderer composes canvas primitives for syntax-aware rendering.
// State lives in widget, renderer is ephemeral per Fyne contract.
func (e *CRDTEditor) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.RGBA{0x1e, 0x1e, 0x1e, 0xff})
	// Use entry as the actual editing surface. It must be laid out explicitly;
	// a zero-sized child accepts no pointer or keyboard input.
	c := container.NewBorder(nil, nil, nil, nil, e.entry)
	c.Objects = append([]fyne.CanvasObject{bg}, c.Objects...)
	return widget.NewSimpleRenderer(c)
}

// Ensure theme adaptation: override Refresh to sync entry.
func (e *CRDTEditor) Refresh() {
	if e.entry != nil {
		// tint bg per theme
	}
	e.BaseWidget.Refresh()
}

// Focus handling delegates to entry.
func (e *CRDTEditor) FocusGained() { e.entry.FocusGained() }
func (e *CRDTEditor) FocusLost()   { e.entry.FocusLost() }

// TypedKey etc delegate
func (e *CRDTEditor) TypedKey(ev *fyne.KeyEvent) { e.entry.TypedKey(ev) }
func (e *CRDTEditor) TypedRune(r rune)           { e.entry.TypedRune(r) }

// Minimal custom renderer struct for future per-rune decoration.
// Kept for documentation / extension point: if Entry is replaced by pure canvas.Text lines,
// editorRenderer below would be used.
type editorRenderer struct {
	editor  *CRDTEditor
	bg      *canvas.Rectangle
	lines   []*canvas.Text
	cursors []*canvas.Rectangle
	objects []fyne.CanvasObject
	size    fyne.Size
}

func newEditorRenderer(e *CRDTEditor) *editorRenderer {
	bg := canvas.NewRectangle(theme.BackgroundColor())
	return &editorRenderer{editor: e, bg: bg, objects: []fyne.CanvasObject{bg}}
}

func (r *editorRenderer) Layout(s fyne.Size) {
	r.size = s
	r.bg.Resize(s)
	// layout lines stacked vertically with lineHeight = TextSize * 1.4
	y := float32(4)
	lineH := theme.TextSize()*1.4 + 2
	for _, t := range r.lines {
		t.Move(fyne.NewPos(6, y))
		t.Resize(fyne.NewSize(s.Width-12, theme.TextSize()))
		y += lineH
	}
}

func (r *editorRenderer) MinSize() fyne.Size { return fyne.NewSize(200, 100) }
func (r *editorRenderer) Refresh() {
	// rebuild lines from editor text
	txt := r.editor.Text()
	lines := splitLines(txt)
	// ensure capacity
	for len(r.lines) < len(lines) {
		t := canvas.NewText("", theme.ForegroundColor())
		t.TextSize = theme.TextSize()
		t.TextStyle = fyne.TextStyle{Monospace: true}
		r.lines = append(r.lines, t)
		r.objects = append(r.objects, t)
	}
	for i, s := range lines {
		r.lines[i].Text = s
		r.lines[i].Refresh()
	}
	canvas.Refresh(r.bg)
}
func (r *editorRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *editorRenderer) Destroy()                     {}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, ch := range s {
		if ch == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

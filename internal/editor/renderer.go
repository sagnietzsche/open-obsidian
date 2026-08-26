package editor

// This file documents the custom logic renderer contract for the CRDT editor.
// The active renderer is implemented in editor.go:CreateRenderer() which currently
// wraps widget.Entry + canvas.Rectangle background via widget.NewSimpleRenderer.
//
// For full per-rune control (syntax highlighting, inline wikilink pills, remote
// cursors at character precision), the editorRenderer in editor.go can be swapped
// in by returning newEditorRenderer(e) from CreateRenderer(). Its Layout/Refresh/
// Objects methods satisfy fyne.WidgetRenderer and own the Fyne canvas lifecycle.
//
// Extension points:
//   - editorRenderer.Refresh() does rune-level diff and assigns canvas.Text.Color per token type (heading=theme.Primary, code=monospace dim).
//   - Remote cursors: Awareness.All() -> []canvas.Rectangle (width 2, height lineH) positioned at (x,y) via rune width measurement.
//   - Selection: canvas.Rectangle with alpha.
// State MUST live in CRDTEditor widget, not renderer, because Fyne may recreate
// renderer on hide/show (Destroy/Create cycle).

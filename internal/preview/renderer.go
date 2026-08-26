package preview

// Custom graphical renderer plan for preview (P1):
// Walk goldmark AST via parser.NewConverter().md.Parser().Parse() and for each node
// create canvas objects:
//   heading -> canvas.Text with Bold + larger TextSize + PrimaryColor
//   paragraph -> canvas.Text with Wrapping
//   code block -> canvas.Rectangle bg + canvas.Text monospace
//   image -> canvas.Image (load via fyne.LoadResourceFromPath, scaled by container)
//   wikilink ghost -> canvas.Text with color.RGBA{0xaa,0x55,0x55,0xff} underline
// The renderer satisfies fyne.WidgetRenderer: Layout does word-wrap + stacking,
// MinSize computes via text.MinSize, Refresh diffs AST.
// This file is the extension point; current Preview uses RichText for MVP to avoid
// reimplementing markdown layout, which is substantial.
// See parser/markdown.go:PreprocessForPreview for the wikilink→md link transform
// that makes RichText sufficient for 90% of Obsidian dialect.

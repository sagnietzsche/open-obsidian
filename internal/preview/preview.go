package preview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sagnikc395/open-obsidian/internal/parser"
)

// Preview is a read-only markdown preview with wikilink/tag preprocessing.
// It wraps widget.RichText (which is fyne's markdown renderer) and provides
// OnLinkTapped for note:// and tag:// navigation.
type Preview struct {
	widget.BaseWidget
	rich       *widget.RichText
	scroll     *container.Scroll
	sourcePath string
	files      []string
	aliases    map[string]string

	OnNavigate func(target string) // target is note path or tag
}

func NewPreview() *Preview {
	p := &Preview{}
	p.rich = widget.NewRichText()
	p.rich.Wrapping = fyne.TextWrapWord
	p.scroll = container.NewScroll(p.rich)
	p.ExtendBaseWidget(p)
	return p
}

func (p *Preview) SetContent(md, sourcePath string, files []string, aliases map[string]string) {
	p.sourcePath = sourcePath
	p.files = files
	p.aliases = aliases
	pre := parser.PreprocessForPreview(md, sourcePath, files, aliases)
	p.rich.ParseMarkdown(pre)
	// Hook hyperlinks: RichText segments include HyperlinkSegment with URL.
	// Fyne's RichText handles tap via widget.HyperlinkSegment.OnTapped? We intercept by
	// scanning segments after parse and wrapping their callback.
	for _, seg := range p.rich.Segments {
		if h, ok := seg.(*widget.HyperlinkSegment); ok {
			url := h.URL.String()
			captured := url
			h.OnTapped = func() {
				if p.OnNavigate != nil {
					p.OnNavigate(captured)
				}
			}
		}
	}
	p.Refresh()
}

func (p *Preview) SetMarkdownRaw(md string) {
	p.rich.ParseMarkdown(md)
	p.Refresh()
}

func (p *Preview) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.scroll)
}

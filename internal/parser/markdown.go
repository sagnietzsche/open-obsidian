package parser

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// Converter wraps goldmark with Obsidian-like extensions.
type Converter struct {
	md goldmark.Markdown
}

func NewConverter() *Converter {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			highlighting.NewHighlighting(highlighting.WithStyle("dracula")),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)
	return &Converter{md: md}
}

// ToHTML converts markdown to HTML.
func (c *Converter) ToHTML(src string) (string, error) {
	var buf bytes.Buffer
	if err := c.md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// PreprocessForPreview applies wikilink/tag transforms before feeding to goldmark/RichText.
func PreprocessForPreview(content, sourcePath string, files []string, aliases map[string]string) string {
	// Strip frontmatter for preview (show body only)
	_, body, _ := ParseFrontmatter(content)
	// Convert wikilinks
	body = PreprocessWikilinks(body, sourcePath, files, aliases)
	// Convert tags #tag to link for RichText clickability
	body = PreprocessTags(body)
	return body
}

// tag preprocessing: #tag -> [#tag](tag://tag)
func PreprocessTags(body string) string {
	// Walk manually to respect preceding char check (RE2 no lookbehind)
	return tagRe.ReplaceAllStringFunc(body, func(m string) string {
		// m is #tag
		name := m[1:]
		return "[" + m + "](tag://" + name + ")"
	})
}

package parser

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Link represents a parsed wikilink or embed.
type Link struct {
	Raw    string // full [[...]] text
	Target string // note name without extension
	Heading string
	Alias  string
	Embed  bool // ![[...]]
	Line   int
}

// Regex for [[target#heading|alias]] and ![[...]]
var wikiRe = regexp.MustCompile(`(!?)\[\[([^\[\]|#]+)(?:#([^\]|]+))?(?:\|([^\]]+))?\]\]`)

// ExtractWikilinks parses all wikilinks in content.
func ExtractWikilinks(content string) []Link {
	clean := stripCode(content) // reuse strip to avoid code false positives? but links in code should be ignored
	// We want line numbers: split lines and match per line
	var out []Link
	lines := strings.Split(clean, "\n")
	for i, line := range lines {
		matches := wikiRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			embed := m[1] == "!"
			target := strings.TrimSpace(m[2])
			heading := strings.TrimSpace(m[3])
			alias := strings.TrimSpace(m[4])
			if target == "" {
				continue
			}
			out = append(out, Link{
				Raw:     m[0],
				Target:  target,
				Heading: heading,
				Alias:   alias,
				Embed:   embed,
				Line:    i + 1,
			})
		}
	}
	return out
}

// ResolveTarget maps a wikilink target to a vault-relative path.
// Implements Obsidian-style resolution: case-insensitive, shortest path, same-folder preference.
func ResolveTarget(target, sourcePath string, files []string, aliases map[string]string) string {
	// Alias resolution: if target is an alias, map to real file base name
	if real, ok := aliases[strings.ToLower(target)]; ok {
		target = real
	}
	targetLower := strings.ToLower(target)
	// If target already contains / or .md, treat as path-like
	hasSlash := strings.Contains(target, "/")
	candidates := []string{}
	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), ".md")
		baseLower := strings.ToLower(base)
		fLower := strings.ToLower(strings.TrimSuffix(f, ".md"))
		if hasSlash {
			if fLower == targetLower || strings.HasSuffix(fLower, "/"+targetLower) {
				candidates = append(candidates, f)
			}
		} else {
			if baseLower == targetLower {
				candidates = append(candidates, f)
			}
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	// Prefer same folder as source
	srcDir := filepath.Dir(sourcePath)
	best := candidates[0]
	bestScore := 1000
	for _, c := range candidates {
		score := len(c) // shorter preferred
		if filepath.Dir(c) == srcDir {
			score -= 100 // boost
		}
		if score < bestScore {
			bestScore = score
			best = c
		}
	}
	return best
}

// PreprocessWikilinks converts wikilinks to standard markdown links for goldmark/RichText.
// Embeds are left as image-like or handled separately.
func PreprocessWikilinks(content string, sourcePath string, files []string, aliases map[string]string) string {
	return wikiRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := wikiRe.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		embed := sub[1] == "!"
		target := strings.TrimSpace(sub[2])
		heading := strings.TrimSpace(sub[3])
		alias := strings.TrimSpace(sub[4])
		display := alias
		if display == "" {
			display = target
			if heading != "" {
				display = target + "#" + heading
			}
		}
		if embed {
			// Try to resolve as file; if image extension, keep as markdown image
			ext := strings.ToLower(filepath.Ext(target))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".svg" || ext == ".webp" {
				// image embed -> ![alias](target)
				return "![" + display + "](" + target + ")"
			}
			// note embed -> blockquote placeholder for preview
			resolved := ResolveTarget(target, sourcePath, files, aliases)
			if resolved != "" {
				return "> **Embed:** [[" + target + "]] → " + resolved
			}
			return "> **Embed (unresolved):** " + target
		}
		resolved := ResolveTarget(target, sourcePath, files, aliases)
		anchor := ""
		if heading != "" {
			anchor = "#" + heading
		}
		if resolved != "" {
			return "[" + display + "](note://" + resolved + anchor + ")"
		}
		// ghost link: still create markdown link but styled as ghost in post-process
		return "[" + display + "](note://ghost/" + target + anchor + ")"
	})
}

// ExtractMarkdownLinks via regex for [text](path) that point to .md or note://
var mdLinkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
var mdImageRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

func ExtractMarkdownLinks(content string) []string {
	var out []string
	for _, m := range mdLinkRe.FindAllStringSubmatch(content, -1) {
		out = append(out, m[1])
	}
	return out
}

func ExtractMarkdownImages(content string) []string {
	var out []string
	for _, m := range mdImageRe.FindAllStringSubmatch(content, -1) {
		out = append(out, m[1])
	}
	return out
}

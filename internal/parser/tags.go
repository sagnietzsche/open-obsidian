package parser

import (
	"regexp"
	"strings"
)

// Tag regex: #tag, #nested/tag, exclude numeric-only, exclude code fences.
// We strip code fences before applying. RE2 compatible (no lookbehind).
var tagRe = regexp.MustCompile(`#([A-Za-z][\w\-/]*[A-Za-z0-9]|[A-Za-z])`)

// ExtractTags returns tags found in body (without frontmatter).
// Handles stripping of fenced code blocks and inline code.
func ExtractTags(body string) []string {
	clean := stripCode(body)
	matches := tagRe.FindAllStringSubmatchIndex(clean, -1)
	var out []string
	seen := map[string]bool{}
	for _, loc := range matches {
		// loc[0],loc[1] = full match indices; loc[2],loc[3] = capture group
		start := loc[0]
		if start > 0 {
			prev := clean[start-1]
			if isWordChar(prev) || prev == '/' {
				continue
			}
		}
		tag := clean[loc[2]:loc[3]]
		// Obsidian rule: #1984 invalid (numeric only) — our regex already requires leading letter
		// Trim trailing punctuation that may be captured via word boundary edge
		tag = strings.TrimSuffix(tag, "/")
		if tag == "" {
			continue
		}
		// reject pure digits after first char? regex ensures first is letter, ok
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

// stripCode removes fenced ``` blocks and inline `code` before tag scan.
func stripCode(s string) string {
	// Remove fenced blocks ```...```
	var out strings.Builder
	inFence := false
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Remove inline `...` fragments
		cleaned := stripInlineCode(line)
		out.WriteString(cleaned)
		out.WriteString("\n")
	}
	return out.String()
}

func stripInlineCode(line string) string {
	var out strings.Builder
	in := false
	for _, r := range line {
		if r == '`' {
			in = !in
			continue
		}
		if !in {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func isWordChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

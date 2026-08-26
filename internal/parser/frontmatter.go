package parser

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter holds parsed YAML frontmatter.
type Frontmatter struct {
	Tags     []string `yaml:"tags"`
	Aliases  []string `yaml:"aliases"`
	CSSClass string   `yaml:"cssclass"`
	Raw      map[string]any
}

// ParseFrontmatter extracts YAML between leading --- delimiters.
// Returns frontmatter, body without frontmatter, and error.
func ParseFrontmatter(content string) (Frontmatter, string, error) {
	var fm Frontmatter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return fm, content, nil
	}
	// find closing ---
	rest := content[4:]
	// handle \r\n variant: content starts with ---\r\n => rest starts after 5? Our prefix check handles both.
	// Normalize: if content starts with ---\r\n, strip \r
	if strings.HasPrefix(content, "---\r\n") {
		rest = content[5:]
	}
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return fm, content, nil
	}
	// Ensure closing is at line start; search for \n---\n or \n---\r\n or end
	yamlPart := rest[:idx]
	remaining := rest[idx+4:] // skip \n---
	// trim leading newline
	if strings.HasPrefix(remaining, "\r\n") {
		remaining = remaining[2:]
	} else if strings.HasPrefix(remaining, "\n") {
		remaining = remaining[1:]
	}
	// Also handle --- on its own line with possible trailing \n
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(yamlPart), &raw); err != nil {
		return fm, content, err
	}
	fm.Raw = raw
	// Decode strongly typed fields
	var tmp struct {
		Tags     any `yaml:"tags"`
		Aliases  any `yaml:"aliases"`
		CSSClass string `yaml:"cssclass"`
	}
	if err := yaml.Unmarshal([]byte(yamlPart), &tmp); err == nil {
		fm.Tags = toStringSlice(tmp.Tags)
		fm.Aliases = toStringSlice(tmp.Aliases)
		fm.CSSClass = tmp.CSSClass
	}
	return fm, remaining, nil
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case string:
		// comma separated? but YAML single string tag
		return []string{x}
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return x
	default:
		return nil
	}
}

package parser

import (
	"sort"
	"strings"
	"sync"
)

// Index holds forward/inverse link graph and tag/alias maps.
type Index struct {
	mu sync.RWMutex

	Forward map[string]map[string]int // source -> target -> count
	Inverse map[string]map[string]int // target -> source -> count
	Tags    map[string][]string       // tag -> files
	FileTags map[string][]string      // file -> tags
	Aliases map[string]string         // lower alias -> canonical target base (without .md)
	Headings map[string][]string      // file -> headings
}

// NewIndex creates empty index.
func NewIndex() *Index {
	return &Index{
		Forward:  make(map[string]map[string]int),
		Inverse:  make(map[string]map[string]int),
		Tags:     make(map[string][]string),
		FileTags: make(map[string][]string),
		Aliases:  make(map[string]string),
		Headings: make(map[string][]string),
	}
}

// Build rebuilds index from vault files. files are vault-relative paths, readFn reads content.
func (idx *Index) Build(files []string, readFn func(rel string) (string, error)) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.Forward = make(map[string]map[string]int)
	idx.Inverse = make(map[string]map[string]int)
	idx.Tags = make(map[string][]string)
	idx.FileTags = make(map[string][]string)
	idx.Aliases = make(map[string]string)
	idx.Headings = make(map[string][]string)

	// First pass: collect aliases and headings so resolver has them
	for _, f := range files {
		content, err := readFn(f)
		if err != nil {
			continue
		}
		fm, body, _ := ParseFrontmatter(content)
		for _, a := range fm.Aliases {
			idx.Aliases[strings.ToLower(a)] = strings.TrimSuffix(f, ".md")
			// also map lower base of file itself?
		}
		// tags from frontmatter
		for _, t := range fm.Tags {
			t = strings.TrimPrefix(t, "#")
			idx.Tags[t] = append(idx.Tags[t], f)
			idx.FileTags[f] = append(idx.FileTags[f], t)
		}
		// headings
		idx.Headings[f] = extractHeadings(body)
		_ = body
	}
	// Ensure files themselves are resolvable by base name (implicit)
	// Second pass: links and inline tags
	for _, f := range files {
		content, err := readFn(f)
		if err != nil {
			continue
		}
		_, body, _ := ParseFrontmatter(content)
		links := ExtractWikilinks(body)
		// also count markdown links that point to .md
		for _, target := range links {
			resolved := ResolveTarget(target.Target, f, files, idx.Aliases)
			if resolved == "" {
				// ghost: keep target as ghost key
				resolved = "ghost/" + target.Target
			}
			if idx.Forward[f] == nil {
				idx.Forward[f] = make(map[string]int)
			}
			idx.Forward[f][resolved]++
			if idx.Inverse[resolved] == nil {
				idx.Inverse[resolved] = make(map[string]int)
			}
			idx.Inverse[resolved][f]++
		}
		// markdown links
		for _, dest := range ExtractMarkdownLinks(body) {
			if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") || strings.HasPrefix(dest, "note://") || strings.HasPrefix(dest, "tag://") {
				continue
			}
			if strings.HasSuffix(strings.ToLower(dest), ".md") {
				clean := strings.TrimPrefix(dest, "./")
				clean = strings.TrimSuffix(clean, ".md")
				resolved := ResolveTarget(clean, f, files, idx.Aliases)
				if resolved == "" {
					resolved = "ghost/" + clean
				}
				if idx.Forward[f] == nil {
					idx.Forward[f] = make(map[string]int)
				}
				idx.Forward[f][resolved]++
				if idx.Inverse[resolved] == nil {
					idx.Inverse[resolved] = make(map[string]int)
				}
				idx.Inverse[resolved][f]++
			}
		}
		// tags
		tags := ExtractTags(body)
		for _, t := range tags {
			// dedup per file already done in ExtractTags
			idx.Tags[t] = append(idx.Tags[t], f)
			idx.FileTags[f] = append(idx.FileTags[f], t)
		}
		// dedup filetags
		if len(idx.FileTags[f]) > 0 {
			seen := map[string]bool{}
			var uniq []string
			for _, t := range idx.FileTags[f] {
				if !seen[t] {
					seen[t] = true
					uniq = append(uniq, t)
				}
			}
			idx.FileTags[f] = uniq
		}
	}
	// sort tag file lists
	for k := range idx.Tags {
		sort.Strings(idx.Tags[k])
		// dedup
		seen := map[string]bool{}
		var uniq []string
		for _, f := range idx.Tags[k] {
			if !seen[f] {
				seen[f] = true
				uniq = append(uniq, f)
			}
		}
		idx.Tags[k] = uniq
	}
	return nil
}

// UpdateFile incrementally updates index for single file.
func (idx *Index) UpdateFile(rel string, content string, files []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	// remove old forward edges for this file
	if oldTargets, ok := idx.Forward[rel]; ok {
		for tgt := range oldTargets {
			if m, ok := idx.Inverse[tgt]; ok {
				delete(m, rel)
				if len(m) == 0 {
					delete(idx.Inverse, tgt)
				}
			}
		}
		delete(idx.Forward, rel)
	}
	// remove old tags
	if oldTags, ok := idx.FileTags[rel]; ok {
		for _, t := range oldTags {
			list := idx.Tags[t]
			n := list[:0]
			for _, f := range list {
				if f != rel {
					n = append(n, f)
				}
			}
			if len(n) == 0 {
				delete(idx.Tags, t)
			} else {
				idx.Tags[t] = n
			}
		}
		delete(idx.FileTags, rel)
	}
	// re-parse
	fm, body, _ := ParseFrontmatter(content)
	// aliases: remove old aliases for this file? Simplistic: rebuild aliases if file changed tags
	for k, v := range idx.Aliases {
		if v == strings.TrimSuffix(rel, ".md") {
			delete(idx.Aliases, k)
		}
	}
	for _, a := range fm.Aliases {
		idx.Aliases[strings.ToLower(a)] = strings.TrimSuffix(rel, ".md")
	}
	for _, t := range fm.Tags {
		t = strings.TrimPrefix(t, "#")
		idx.Tags[t] = append(idx.Tags[t], rel)
		idx.FileTags[rel] = append(idx.FileTags[rel], t)
	}
	idx.Headings[rel] = extractHeadings(body)
	links := ExtractWikilinks(body)
	for _, target := range links {
		resolved := ResolveTarget(target.Target, rel, files, idx.Aliases)
		if resolved == "" {
			resolved = "ghost/" + target.Target
		}
		if idx.Forward[rel] == nil {
			idx.Forward[rel] = make(map[string]int)
		}
		idx.Forward[rel][resolved]++
		if idx.Inverse[resolved] == nil {
			idx.Inverse[resolved] = make(map[string]int)
		}
		idx.Inverse[resolved][rel]++
	}
	for _, dest := range ExtractMarkdownLinks(body) {
		if strings.HasPrefix(dest, "http") || strings.HasPrefix(dest, "note://") || strings.HasPrefix(dest, "tag://") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(dest), ".md") {
			clean := strings.TrimPrefix(dest, "./")
			clean = strings.TrimSuffix(clean, ".md")
			resolved := ResolveTarget(clean, rel, files, idx.Aliases)
			if resolved == "" {
				resolved = "ghost/" + clean
			}
			if idx.Forward[rel] == nil {
				idx.Forward[rel] = make(map[string]int)
			}
			idx.Forward[rel][resolved]++
			if idx.Inverse[resolved] == nil {
				idx.Inverse[resolved] = make(map[string]int)
			}
			idx.Inverse[resolved][rel]++
		}
	}
	tags := ExtractTags(body)
	for _, t := range tags {
		idx.Tags[t] = append(idx.Tags[t], rel)
		idx.FileTags[rel] = append(idx.FileTags[rel], t)
	}
	// dedup
	if len(idx.FileTags[rel]) > 0 {
		seen := map[string]bool{}
		var uniq []string
		for _, t := range idx.FileTags[rel] {
			if !seen[t] {
				seen[t] = true
				uniq = append(uniq, t)
			}
		}
		idx.FileTags[rel] = uniq
		for _, t := range uniq {
			// ensure Tags dedup
			seen2 := map[string]bool{}
			var uniqFiles []string
			for _, f := range idx.Tags[t] {
				if !seen2[f] {
					seen2[f] = true
					uniqFiles = append(uniqFiles, f)
				}
			}
			sort.Strings(uniqFiles)
			idx.Tags[t] = uniqFiles
		}
	}
}

func extractHeadings(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			// heading
			h := strings.TrimLeft(trim, "#")
			h = strings.TrimSpace(h)
			if h != "" {
				out = append(out, h)
			}
		}
	}
	return out
}

// Backlinks returns sources linking to target.
func (idx *Index) Backlinks(target string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	m := idx.Inverse[target]
	if m == nil {
		return nil
	}
	var out []string
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ForwardLinks returns targets for source.
func (idx *Index) ForwardLinks(source string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	m := idx.Forward[source]
	if m == nil {
		return nil
	}
	var out []string
	for t := range m {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// AllTags returns sorted tag names.
func (idx *Index) AllTags() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	var out []string
	for k := range idx.Tags {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

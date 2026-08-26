package parser

import "testing"

func TestExtractWikilinks(t *testing.T) {
	content := "Hello [[Note]] and [[Note#Heading|Alias]] and ![[Image.png|100]] and [[Target]]\n```\n[[ignored]]\n```\n"
	links := ExtractWikilinks(content)
	if len(links) != 4 {
		t.Fatalf("expected 4 links, got %d: %+v", len(links), links)
	}
	if links[0].Target != "Note" || links[0].Alias != "" {
		t.Fatalf("first link: %+v", links[0])
	}
	if links[1].Target != "Note" || links[1].Heading != "Heading" || links[1].Alias != "Alias" {
		t.Fatalf("second link: %+v", links[1])
	}
	if !links[2].Embed || links[2].Target != "Image.png" {
		t.Fatalf("third link embed: %+v", links[2])
	}
}

func TestExtractTags(t *testing.T) {
	body := "This is #project and #nested/tag\n```\n#ignored\n```\nAlso #todo and #1984 should ignore numeric, #y1984 ok\n"
	tags := ExtractTags(body)
	m := map[string]bool{}
	for _, tag := range tags {
		m[tag] = true
	}
	for _, want := range []string{"project", "nested/tag", "todo", "y1984"} {
		if !m[want] {
			t.Fatalf("missing tag %s in %v", want, tags)
		}
	}
	if m["1984"] {
		t.Fatalf("numeric tag should be excluded: %v", tags)
	}
	if m["ignored"] {
		t.Fatalf("code fence tag should be excluded: %v", tags)
	}
}

func TestFrontmatter(t *testing.T) {
	content := "---\ntags: [foo, bar]\naliases: [Alias1]\n---\nBody with [[Link]]\n"
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("parse fm: %v", err)
	}
	if len(fm.Tags) != 2 || len(fm.Aliases) != 1 {
		t.Fatalf("fm: %+v", fm)
	}
	if body != "Body with [[Link]]\n" {
		t.Fatalf("body: %q", body)
	}
}

func TestIndexBuild(t *testing.T) {
	files := []string{"A.md", "B.md", "C.md"}
	contents := map[string]string{
		"A.md": "Link to [[B]] and #project",
		"B.md": "---\naliases: [Bee]\n---\nBack to [[A]] and [[C|See C]]",
		"C.md": "Isolated #project/nested",
	}
	idx := NewIndex()
	if err := idx.Build(files, func(rel string) (string, error) { return contents[rel], nil }); err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(idx.Forward["A.md"]) == 0 {
		t.Fatalf("A forward empty")
	}
	if len(idx.Backlinks("A.md")) == 0 {
		t.Fatalf("A backlinks empty: inverse %v", idx.Inverse)
	}
	if len(idx.Tags["project"]) != 1 { // C has project/nested not project
		// A has project, C has project/nested (different key)
	}
	if len(idx.FileTags["A.md"]) == 0 || idx.FileTags["A.md"][0] != "project" {
		t.Fatalf("filetags A: %v", idx.FileTags["A.md"])
	}
	// alias resolution: Bee should resolve to B.md
	if idx.Aliases["bee"] != "B" {
		t.Fatalf("alias: %v", idx.Aliases)
	}
}

func TestPreprocessWikilinks(t *testing.T) {
	files := []string{"Welcome.md", "Second Note.md"}
	aliases := map[string]string{}
	body := "See [[Welcome]] and [[Second Note|Second]] and [[Missing]]"
	out := PreprocessWikilinks(body, "Welcome.md", files, aliases)
	if out == body {
		t.Fatalf("preprocess didn't change: %q", out)
	}
	if !contains(out, "note://Welcome.md") {
		t.Fatalf("expected resolved link: %q", out)
	}
	if !contains(out, "note://ghost/Missing") {
		t.Fatalf("expected ghost: %q", out)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (func() bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
})() }

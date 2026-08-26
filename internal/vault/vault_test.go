package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultScan(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "sub", "c.md"), []byte("# C"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, ".obsidian"), 0755)
	_ = os.WriteFile(filepath.Join(dir, ".obsidian", "app.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "ignore.png"), []byte("x"), 0644)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	files := s.Files()
	if len(files) != 3 {
		t.Fatalf("expected 3 md, got %v", files)
	}
	// ensure .obsidian ignored
	for _, f := range files {
		if f == ".obsidian/app.json" {
			t.Fatalf("should ignore .obsidian")
		}
	}
	// write new file
	if err := s.WriteFile("new.md", "hello"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(s.Files()) != 4 {
		t.Fatalf("expected 4 after write, got %v", s.Files())
	}
	content, _ := s.ReadFile("new.md")
	if content != "hello" {
		t.Fatalf("read: %q", content)
	}
	if err := s.RenameFile("new.md", "renamed.md"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if len(s.Files()) != 4 {
		t.Fatalf("rename count: %v", s.Files())
	}
	if err := s.DeleteFile("renamed.md"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(s.Files()) != 3 {
		t.Fatalf("delete count: %v", s.Files())
	}
}

package vault

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store represents a vault root folder. Plain filesystem, no DB.
type Store struct {
	Root  string
	mu    sync.RWMutex
	files []string // relative paths using forward slashes, sorted
}

// New creates a Store for root. Root must exist and be a directory.
func New(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrInvalid
	}
	s := &Store{Root: abs}
	if err := s.Scan(); err != nil {
		return nil, err
	}
	return s, nil
}

// shouldIgnore returns true for hidden/config dirs.
func shouldIgnore(rel string) bool {
	parts := strings.FieldsFunc(filepath.ToSlash(rel), func(r rune) bool { return r == '/' })
	for _, p := range parts {
		if p == ".obsidian" || p == ".git" || p == ".trash" || p == "node_modules" {
			return true
		}
	}
	return false
}

// Scan walks Root and rebuilds file list (only .md files).
func (s *Store) Scan() error {
	var out []string
	err := filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(s.Root, path)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if shouldIgnore(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnore(rel) {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			rel = filepath.ToSlash(rel)
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(out)
	s.mu.Lock()
	s.files = out
	s.mu.Unlock()
	return nil
}

// Files returns sorted relative .md paths.
func (s *Store) Files() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]string, len(s.files))
	copy(cp, s.files)
	return cp
}

// AbsPath returns absolute path for rel.
func (s *Store) AbsPath(rel string) string {
	return filepath.Join(s.Root, filepath.FromSlash(rel))
}

// ReadFile reads content of rel.
func (s *Store) ReadFile(rel string) (string, error) {
	b, err := os.ReadFile(s.AbsPath(rel))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteFile writes data to rel, creating parent dirs.
func (s *Store) WriteFile(rel string, data string) error {
	abs := s.AbsPath(rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, []byte(data), 0644); err != nil {
		return err
	}
	// update file list if new
	s.mu.Lock()
	found := false
	for _, f := range s.files {
		if f == rel {
			found = true
			break
		}
	}
	if !found && strings.HasSuffix(strings.ToLower(rel), ".md") {
		s.files = append(s.files, rel)
		sort.Strings(s.files)
	}
	s.mu.Unlock()
	return nil
}

// CreateFile creates an empty .md if not exists.
func (s *Store) CreateFile(rel string) error {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if rel == "" {
		return fs.ErrInvalid
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".md") {
		rel += ".md"
	}
	abs := s.AbsPath(rel)
	if _, err := os.Stat(abs); err == nil {
		return fs.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	return s.WriteFile(rel, "")
}

// DeleteFile removes rel.
func (s *Store) DeleteFile(rel string) error {
	if err := os.Remove(s.AbsPath(rel)); err != nil {
		return err
	}
	s.mu.Lock()
	n := s.files[:0]
	for _, f := range s.files {
		if f != rel {
			n = append(n, f)
		}
	}
	s.files = n
	s.mu.Unlock()
	return nil
}

// RenameFile moves oldRel to newRel and returns error. Caller should update wikilinks separately.
func (s *Store) RenameFile(oldRel, newRel string) error {
	if err := os.MkdirAll(filepath.Dir(s.AbsPath(newRel)), 0755); err != nil {
		return err
	}
	if err := os.Rename(s.AbsPath(oldRel), s.AbsPath(newRel)); err != nil {
		return err
	}
	s.mu.Lock()
	for i, f := range s.files {
		if f == oldRel {
			s.files[i] = newRel
			break
		}
	}
	sort.Strings(s.files)
	s.mu.Unlock()
	return nil
}

// EnsureDir creates dir rel.
func (s *Store) EnsureDir(rel string) error {
	return os.MkdirAll(s.AbsPath(rel), 0755)
}

// AllFilesRecursive returns all files (including non-md) for attachment handling.
func (s *Store) AllFilesRecursive() []string {
	var out []string
	_ = filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(s.Root, path)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if shouldIgnore(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnore(rel) {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

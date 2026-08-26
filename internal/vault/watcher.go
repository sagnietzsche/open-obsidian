package vault

import (
	"io/fs"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches vault root recursively and debounces scan.
type Watcher struct {
	watcher  *fsnotify.Watcher
	store    *Store
	OnChange func()
	done     chan struct{}
}

// NewWatcher creates watcher on store.Root.
func NewWatcher(s *Store, onChange func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{watcher: fw, store: s, OnChange: onChange, done: make(chan struct{})}
	if err := w.addRecursive(s.Root); err != nil {
		_ = fw.Close()
		return nil, err
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root && shouldIgnorePath(path, root) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func shouldIgnorePath(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && shouldIgnore(rel)
}

// Close stops watcher.
func (w *Watcher) Close() error {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
	return w.watcher.Close()
}

func (w *Watcher) loop() {
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	var pending bool
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, err := filepath.Abs(ev.Name); err == nil {
					if stat, err := filepath.Glob(info); err == nil && len(stat) > 0 {
						if addErr := w.addRecursive(info); addErr != nil {
							_ = addErr
						}
					}
				}
			}
			if !pending {
				pending = true
				debounce.Reset(150 * time.Millisecond)
			}
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		case <-debounce.C:
			pending = false
			_ = w.store.Scan()
			if w.OnChange != nil {
				w.OnChange()
			}
		}
	}
}

// SimpleWatcher is a polling fallback used if fsnotify unavailable.
type SimpleWatcher struct {
	store    *Store
	onChange func()
	ticker   *time.Ticker
	done     chan struct{}
}

func NewPollWatcher(s *Store, onChange func(), interval time.Duration) *SimpleWatcher {
	sw := &SimpleWatcher{store: s, onChange: onChange, ticker: time.NewTicker(interval), done: make(chan struct{})}
	go sw.loop()
	return sw
}

func (sw *SimpleWatcher) loop() {
	for {
		select {
		case <-sw.done:
			return
		case <-sw.ticker.C:
			before := len(sw.store.Files())
			_ = sw.store.Scan()
			if len(sw.store.Files()) != before && sw.onChange != nil {
				sw.onChange()
			}
		}
	}
}

func (sw *SimpleWatcher) Close() {
	close(sw.done)
	sw.ticker.Stop()
}

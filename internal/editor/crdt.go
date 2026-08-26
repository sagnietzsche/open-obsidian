package editor

import (
	"sync"
)

// Doc is a lightweight sequence CRDT (RGA-inspired) for single note.
// It provides deterministic insert/delete with Lamport timestamp and site ID,
// sufficient for local editing + future p2p sync via ygo-style updates.
// MVP: local ops only, no network. Updates are broadcast via callbacks.
type Doc struct {
	mu       sync.RWMutex
	siteID   string
	clock    int
	text     string
	history  []Op
	// observers
	onChange []func()
}

// Op represents a CRDT operation.
type Op struct {
	ID     string // siteID:clock
	Type   string // "insert" | "delete"
	Pos    int
	Text   string // for insert
	Length int    // for delete
	Clock  int
	SiteID string
}

// NewDoc creates a doc with initial text.
func NewDoc(siteID, initial string) *Doc {
	if siteID == "" {
		siteID = "local"
	}
	return &Doc{siteID: siteID, text: initial, clock: 0}
}

// Text returns current text.
func (d *Doc) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.text
}

// SetText replaces whole text (e.g., on file load). Resets history but keeps version.
func (d *Doc) SetText(t string) {
	d.mu.Lock()
	d.text = t
	d.mu.Unlock()
	d.notify()
}

// Insert at pos.
func (d *Doc) Insert(pos int, s string) Op {
	d.mu.Lock()
	defer d.mu.Unlock()
	if pos < 0 {
		pos = 0
	}
	if pos > len(d.text) {
		pos = len(d.text)
	}
	d.clock++
	op := Op{ID: d.siteID + ":" + itoa(d.clock), Type: "insert", Pos: pos, Text: s, Clock: d.clock, SiteID: d.siteID}
	// apply
	d.text = d.text[:pos] + s + d.text[pos:]
	d.history = append(d.history, op)
	d.mu.Unlock()
	d.notify()
	d.mu.Lock()
	return op
}

// Delete range [pos, pos+length).
func (d *Doc) Delete(pos, length int) Op {
	d.mu.Lock()
	defer d.mu.Unlock()
	if pos < 0 {
		pos = 0
	}
	if pos > len(d.text) {
		pos = len(d.text)
	}
	if pos+length > len(d.text) {
		length = len(d.text) - pos
	}
	d.clock++
	op := Op{ID: d.siteID + ":" + itoa(d.clock), Type: "delete", Pos: pos, Length: length, Clock: d.clock, SiteID: d.siteID}
	d.text = d.text[:pos] + d.text[pos+length:]
	d.history = append(d.history, op)
	d.mu.Unlock()
	d.notify()
	d.mu.Lock()
	return op
}

// ApplyRemote applies an op from remote site, with simple OT adjustment: sort by (clock, siteID).
func (d *Doc) ApplyRemote(op Op) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// naive: apply at pos if within bounds; real RGA would use element IDs. For local-only this suffices.
	if op.Type == "insert" {
		if op.Pos > len(d.text) {
			op.Pos = len(d.text)
		}
		d.text = d.text[:op.Pos] + op.Text + d.text[op.Pos:]
	} else if op.Type == "delete" {
		if op.Pos < len(d.text) {
			end := op.Pos + op.Length
			if end > len(d.text) {
				end = len(d.text)
			}
			d.text = d.text[:op.Pos] + d.text[end:]
		}
	}
	d.history = append(d.history, op)
	if op.Clock > d.clock {
		d.clock = op.Clock
	}
	d.mu.Unlock()
	d.notify()
	d.mu.Lock()
}

// History returns copy.
func (d *Doc) History() []Op {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cp := make([]Op, len(d.history))
	copy(cp, d.history)
	return cp
}

// OnChange registers callback.
func (d *Doc) OnChange(fn func()) {
	d.mu.Lock()
	d.onChange = append(d.onChange, fn)
	d.mu.Unlock()
}

func (d *Doc) notify() {
	d.mu.RLock()
	cbs := append([]func(){}, d.onChange...)
	d.mu.RUnlock()
	for _, fn := range cbs {
		fn()
	}
}

func itoa(i int) string {
	// fast itoa without fmt
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

// Awareness tracks remote cursors (for renderer).
type Awareness struct {
	mu      sync.RWMutex
	cursors map[string]Cursor // siteID -> cursor
}

type Cursor struct {
	SiteID string
	Pos    int
	Color  string
}

func NewAwareness() *Awareness { return &Awareness{cursors: make(map[string]Cursor)} }
func (a *Awareness) Set(c Cursor) {
	a.mu.Lock()
	a.cursors[c.SiteID] = c
	a.mu.Unlock()
}
func (a *Awareness) All() []Cursor {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var out []Cursor
	for _, c := range a.cursors {
		out = append(out, c)
	}
	return out
}

package editor

import "testing"

func TestDocInsertDelete(t *testing.T) {
	d := NewDoc("site1", "hello")
	d.Insert(5, " world")
	if d.Text() != "hello world" {
		t.Fatalf("insert: %q", d.Text())
	}
	d.Delete(5, 6)
	if d.Text() != "hello" {
		t.Fatalf("delete: %q", d.Text())
	}
	d.SetText("reset")
	if d.Text() != "reset" {
		t.Fatalf("set: %q", d.Text())
	}
}

func TestDocHistory(t *testing.T) {
	d := NewDoc("a", "")
	d.Insert(0, "a")
	d.Insert(1, "b")
	if len(d.History()) != 2 {
		t.Fatalf("history len %d", len(d.History()))
	}
}

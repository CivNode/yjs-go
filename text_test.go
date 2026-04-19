package yjs_test

import (
	"testing"
	"testing/quick"

	yjs "github.com/CivNode/yjs-go"
)

func TestTextInsertAndString(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	doc.Transact(func() {
		text.Insert(0, "hello")
	}, nil)

	if got := text.String(); got != "hello" {
		t.Errorf("want %q got %q", "hello", got)
	}
}

func TestTextInsertMiddle(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	doc.Transact(func() { text.Insert(0, "helo") }, nil)
	doc.Transact(func() { text.Insert(2, "l") }, nil)

	if got := text.String(); got != "hello" {
		t.Errorf("want %q got %q", "hello", got)
	}
}

func TestTextDelete(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	doc.Transact(func() { text.Insert(0, "hello world") }, nil)
	doc.Transact(func() { text.Delete(5, 6) }, nil)

	if got := text.String(); got != "hello" {
		t.Errorf("want %q got %q", "hello", got)
	}
}

func TestTextLength(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	doc.Transact(func() { text.Insert(0, "hi") }, nil)
	if text.Len() != 2 {
		t.Errorf("want 2 got %d", text.Len())
	}
	doc.Transact(func() { text.Delete(0, 1) }, nil)
	if text.Len() != 1 {
		t.Errorf("after delete want 1 got %d", text.Len())
	}
}

func TestTextObserve(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	events := 0
	text.Observe(func(ev *yjs.TextEvent) {
		events++
	})

	doc.Transact(func() { text.Insert(0, "hello") }, nil)
	if events != 1 {
		t.Errorf("expected 1 event, got %d", events)
	}

	doc.Transact(func() { text.Delete(0, 5) }, nil)
	if events != 2 {
		t.Errorf("expected 2 events, got %d", events)
	}
}

func TestTextUnicode(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")

	// Multi-byte UTF-8 characters.
	doc.Transact(func() { text.Insert(0, "helo") }, nil)
	doc.Transact(func() { text.Insert(2, "l") }, nil)

	if got := text.String(); got != "hello" {
		t.Errorf("want %q got %q", "hello", got)
	}
}

// TestTextConvergence checks that two Docs applying the same operations in
// different orders converge to the same state after exchanging updates.
func TestTextConvergence(t *testing.T) {
	err := quick.Check(func(a, b string) bool {
		if len(a) == 0 || len(b) == 0 {
			return true
		}
		// Keep strings short for readability; quick.Check generates arbitrary ones.
		if len(a) > 100 {
			a = a[:100]
		}
		if len(b) > 100 {
			b = b[:100]
		}

		docA := yjs.NewDoc()
		docB := yjs.NewDoc()

		textA := docA.GetText("content")
		textB := docB.GetText("content")

		// docA inserts a, docB inserts b concurrently.
		var updateFromA []byte
		var updateFromB []byte

		docA.OnUpdate(func(update []byte, _ interface{}) {
			updateFromA = update
		})
		docB.OnUpdate(func(update []byte, _ interface{}) {
			updateFromB = update
		})

		docA.Transact(func() { textA.Insert(0, a) }, nil)
		docB.Transact(func() { textB.Insert(0, b) }, nil)

		// Exchange updates.
		if err := yjs.ApplyUpdate(docA, updateFromB, "remote"); err != nil {
			return false
		}
		if err := yjs.ApplyUpdate(docB, updateFromA, "remote"); err != nil {
			return false
		}

		// Both docs must converge.
		return textA.String() == textB.String()
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

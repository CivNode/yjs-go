package yjs_test

import (
	"testing"

	yjs "github.com/CivNode/yjs-go"
)

func TestNewDoc(t *testing.T) {
	doc := yjs.NewDoc()
	if doc == nil {
		t.Fatal("NewDoc returned nil")
	}
	if doc.ClientID() == 0 {
		t.Error("ClientID should be non-zero")
	}
}

func TestDocGetText(t *testing.T) {
	doc := yjs.NewDoc()
	text := doc.GetText("content")
	if text == nil {
		t.Fatal("GetText returned nil")
	}
	// Calling GetText again with same name returns the same type.
	text2 := doc.GetText("content")
	if text != text2 {
		t.Error("GetText should return the same instance for the same name")
	}
}

func TestDocGetMap(t *testing.T) {
	doc := yjs.NewDoc()
	m := doc.GetMap("data")
	if m == nil {
		t.Fatal("GetMap returned nil")
	}
}

func TestDocGetArray(t *testing.T) {
	doc := yjs.NewDoc()
	a := doc.GetArray("items")
	if a == nil {
		t.Fatal("GetArray returned nil")
	}
}

func TestDocTransact(t *testing.T) {
	doc := yjs.NewDoc()
	called := false
	doc.Transact(func() {
		called = true
	}, nil)
	if !called {
		t.Error("Transact did not call the function")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	src := yjs.NewDoc()

	// Populate Text, Map, and Array shared types.
	text := src.GetText("body")
	src.Transact(func() { text.Insert(0, "hello world") }, nil)

	m := src.GetMap("meta")
	src.Transact(func() {
		m.Set("author", "Alice")
		m.Set("version", int64(7))
	}, nil)

	arr := src.GetArray("tags")
	src.Transact(func() { arr.Push("go", "crdt", "yjs") }, nil)

	snap, err := src.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap) == 0 {
		t.Fatal("Snapshot returned empty bytes")
	}

	dst, err := yjs.RestoreDoc(snap)
	if err != nil {
		t.Fatalf("RestoreDoc: %v", err)
	}

	// Text equality.
	if got, want := dst.GetText("body").String(), "hello world"; got != want {
		t.Errorf("Text after restore: got %q want %q", got, want)
	}

	// Map equality.
	dstMap := dst.GetMap("meta")
	if v, ok := dstMap.Get("author"); !ok || v != "Alice" {
		t.Errorf("Map[author] after restore: got %v,%v", v, ok)
	}
	if v, ok := dstMap.Get("version"); !ok || v != int64(7) {
		t.Errorf("Map[version] after restore: got %v,%v", v, ok)
	}

	// Array equality.
	dstArr := dst.GetArray("tags")
	got := dstArr.ToSlice()
	if len(got) != 3 || got[0] != "go" || got[1] != "crdt" || got[2] != "yjs" {
		t.Errorf("Array after restore: got %v", got)
	}
}

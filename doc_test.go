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

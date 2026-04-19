// Package yjs implements the Yjs CRDT protocol in Go.
// It provides Doc, Text, Map, Array, and XmlFragment types with full binary
// protocol support (update-v1, update-v2, sync, awareness) compatible with
// the JavaScript yjs library and y-websocket relay.
package yjs

import "fmt"

// ID uniquely identifies a struct in the document.
// Client is the peer's numeric ID; Clock is monotonically increasing per client.
type ID struct {
	Client uint64
	Clock  uint64
}

func (id ID) String() string {
	return fmt.Sprintf("(%d,%d)", id.Client, id.Clock)
}

func compareIDs(a, b ID) int {
	if a.Client < b.Client {
		return -1
	}
	if a.Client > b.Client {
		return 1
	}
	if a.Clock < b.Clock {
		return -1
	}
	if a.Clock > b.Clock {
		return 1
	}
	return 0
}

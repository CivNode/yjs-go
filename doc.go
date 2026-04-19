package yjs

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
)

// UpdateHandler is called whenever the local document produces a new update.
type UpdateHandler func(update []byte, origin interface{})

// Doc is the root Yjs document. It owns a StructStore, manages transactions,
// and holds named shared types (Text, Map, Array, XmlFragment).
type Doc struct {
	mu sync.Mutex

	clientID uint64
	store    *StructStore
	share    map[string]SharedType // named types

	// observers
	updateHandlers []UpdateHandler

	// current transaction (nil when idle)
	currentTx *transaction
}

// NewDoc creates a new document with a random clientID.
//
// ClientIDs are 32-bit unsigned integers (matching Yjs/lib0 behavior) so they
// remain within JavaScript's safe-integer range for cross-runtime interop.
func NewDoc() *Doc {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback: non-zero constant if crypto/rand fails.
		b = [4]byte{0xca, 0xfe, 0x01, 0x02}
	}
	id := uint64(binary.LittleEndian.Uint32(b[:]))
	if id == 0 {
		id = 1
	}
	return &Doc{
		clientID: id,
		store:    newStructStore(),
		share:    make(map[string]SharedType),
	}
}

// NewDocWithClientID creates a document with a specific clientID. Useful for tests.
func NewDocWithClientID(clientID uint64) *Doc {
	if clientID == 0 {
		panic("yjs: clientID must be non-zero")
	}
	return &Doc{
		clientID: clientID,
		store:    newStructStore(),
		share:    make(map[string]SharedType),
	}
}

// ClientID returns this document's unique peer identifier.
func (d *Doc) ClientID() uint64 { return d.clientID }

// GetText returns the named Text type, creating it if it does not exist.
func (d *Doc) GetText(name string) *Text {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.share[name]; ok {
		return t.(*Text)
	}
	t := newText(d, name)
	d.share[name] = t
	return t
}

// GetMap returns the named Map type, creating it if it does not exist.
func (d *Doc) GetMap(name string) *Map {
	d.mu.Lock()
	defer d.mu.Unlock()
	if m, ok := d.share[name]; ok {
		return m.(*Map)
	}
	m := newMap(d, name)
	d.share[name] = m
	return m
}

// GetArray returns the named Array type, creating it if it does not exist.
func (d *Doc) GetArray(name string) *Array {
	d.mu.Lock()
	defer d.mu.Unlock()
	if a, ok := d.share[name]; ok {
		return a.(*Array)
	}
	a := newArray(d, name)
	d.share[name] = a
	return a
}

// GetXmlFragment returns the named XmlFragment type, creating it if it does not exist.
func (d *Doc) GetXmlFragment(name string) *XmlFragment {
	d.mu.Lock()
	defer d.mu.Unlock()
	if x, ok := d.share[name]; ok {
		return x.(*XmlFragment)
	}
	x := newXmlFragment(d, name)
	d.share[name] = x
	return x
}

// OnUpdate registers a handler that is called after each transaction with the
// binary v1 update bytes that should be broadcast to peers.
func (d *Doc) OnUpdate(h UpdateHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updateHandlers = append(d.updateHandlers, h)
}

// Transact runs fn inside a transaction. All structural changes must happen
// inside a transaction. Transactions batch items, then on commit produce the
// binary update and fire OnUpdate handlers.
func (d *Doc) Transact(fn func(), origin interface{}) {
	d.mu.Lock()
	if d.currentTx != nil {
		// Nested transact: just run fn.
		d.mu.Unlock()
		fn()
		return
	}
	tx := &transaction{doc: d, origin: origin}
	d.currentTx = tx
	d.mu.Unlock()

	fn()

	d.mu.Lock()
	d.currentTx = nil
	pending := tx.pendingItems
	pendingDs := tx.pendingDeletes
	d.mu.Unlock()

	if len(pending) == 0 && len(pendingDs) == 0 {
		return
	}

	// Encode the transaction as a v1 update and fire handlers.
	update := encodeUpdateV1(pending, pendingDs)
	d.mu.Lock()
	handlers := make([]UpdateHandler, len(d.updateHandlers))
	copy(handlers, d.updateHandlers)
	d.mu.Unlock()
	for _, h := range handlers {
		h(update, origin)
	}

	// Fire type-level observers.
	tx.fireObservers()
}

// ApplyUpdate applies a v1 binary update received from a remote peer.
func ApplyUpdate(d *Doc, update []byte, origin interface{}) error {
	items, ds, err := decodeUpdateV1(update)
	if err != nil {
		return err
	}

	d.mu.Lock()
	tx := &transaction{doc: d, origin: origin}
	d.currentTx = tx
	d.mu.Unlock()

	// Integrate items.
	for _, item := range items {
		d.mu.Lock()
		integrateItem(d, item)
		d.mu.Unlock()
	}

	// Apply delete set.
	// For each delete range, split items at the range boundaries if necessary,
	// then mark all items fully covered by the range as deleted.
	for clientID, ranges := range ds.clients {
		for _, r := range ranges {
			d.mu.Lock()
			applyDeleteRange(d, tx, clientID, r.clock, r.len)
			d.mu.Unlock()
		}
	}

	d.mu.Lock()
	d.currentTx = nil
	d.mu.Unlock()

	tx.fireObservers()
	return nil
}

// SharedType is implemented by Text, Map, Array, XmlFragment.
type SharedType interface {
	sharedTypeName() string
}

// transaction accumulates changes within a Transact call.
type transaction struct {
	doc           *Doc
	origin        interface{}
	pendingItems  []*Item
	pendingDeletes []*Item
	changedTypes  map[SharedType]struct{}
}

func (tx *transaction) addItem(item *Item) {
	tx.pendingItems = append(tx.pendingItems, item)
	if tx.changedTypes == nil {
		tx.changedTypes = make(map[SharedType]struct{})
	}
	if st, ok := item.Parent.(SharedType); ok {
		tx.changedTypes[st] = struct{}{}
	}
}

func (tx *transaction) addDeletedItem(item *Item) {
	tx.pendingDeletes = append(tx.pendingDeletes, item)
	if tx.changedTypes == nil {
		tx.changedTypes = make(map[SharedType]struct{})
	}
	if st, ok := item.Parent.(SharedType); ok {
		tx.changedTypes[st] = struct{}{}
	}
}

func (tx *transaction) fireObservers() {
	if tx.changedTypes == nil {
		return
	}
	for t := range tx.changedTypes {
		switch v := t.(type) {
		case *Text:
			v.fireObservers(tx)
		case *Map:
			v.fireObservers(tx)
		case *Array:
			v.fireObservers(tx)
		}
	}
}

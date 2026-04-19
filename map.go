package yjs

// MapEvent carries information about a change to a Map type.
type MapEvent struct {
	// Keys is the set of keys that changed.
	Keys   map[string]MapChange
	Origin interface{}
}

// MapChange describes a single key change.
type MapChange struct {
	Action string // "add", "update", "delete"
	OldVal interface{}
	NewVal interface{}
}

// Map is a CRDT key-value store. Values are arbitrary Go values encoded as
// Yjs ContentAny. Only the most-recent item per key (by clock) is live.
type Map struct {
	doc       *Doc
	name      string
	start     map[string]*Item // current live item per key
	observers []func(*MapEvent)
}

func newMap(doc *Doc, name string) *Map {
	return &Map{doc: doc, name: name, start: make(map[string]*Item)}
}

func (m *Map) sharedTypeName() string { return m.name }

// Set sets key to value in the map.
func (m *Map) Set(key string, value interface{}) {
	doc := m.doc
	doc.mu.Lock()
	tx := doc.currentTx
	doc.mu.Unlock()
	if tx == nil {
		panic("yjs: Map.Set must be called inside Transact")
	}

	doc.mu.Lock()
	defer doc.mu.Unlock()

	clock := doc.store.getState(doc.clientID)
	// Delete the old item if present.
	if old, ok := m.start[key]; ok {
		old.Deleted = true
	}
	sub := key
	item := &Item{
		ID:        ID{doc.clientID, clock},
		Parent:    m,
		ParentSub: &sub,
		Kind:      contentAny,
		Content:   value,
		Length:    1,
	}
	doc.store.addItem(item)
	m.start[key] = item
	tx.addItem(item)
}

// Get returns the value for key, or (nil, false) if absent.
func (m *Map) Get(key string) (interface{}, bool) {
	doc := m.doc
	doc.mu.Lock()
	defer doc.mu.Unlock()

	item, ok := m.start[key]
	if !ok || item.Deleted {
		return nil, false
	}
	return item.Content, true
}

// Delete removes key from the map.
func (m *Map) Delete(key string) {
	doc := m.doc
	doc.mu.Lock()
	tx := doc.currentTx
	doc.mu.Unlock()
	if tx == nil {
		panic("yjs: Map.Delete must be called inside Transact")
	}

	doc.mu.Lock()
	defer doc.mu.Unlock()

	item, ok := m.start[key]
	if !ok || item.Deleted {
		return
	}
	item.Deleted = true
	tx.addDeletedItem(item)
}

// Keys returns all non-deleted keys.
func (m *Map) Keys() []string {
	doc := m.doc
	doc.mu.Lock()
	defer doc.mu.Unlock()

	var keys []string
	for k, item := range m.start {
		if !item.Deleted {
			keys = append(keys, k)
		}
	}
	return keys
}

// Observe registers a handler for changes to this Map.
func (m *Map) Observe(fn func(*MapEvent)) {
	doc := m.doc
	doc.mu.Lock()
	defer doc.mu.Unlock()
	m.observers = append(m.observers, fn)
}

func (m *Map) fireObservers(tx *transaction) {
	if len(m.observers) == 0 {
		return
	}
	ev := &MapEvent{Origin: tx.origin}
	for _, fn := range m.observers {
		fn(ev)
	}
}

package yjs

// XmlFragment is a basic skeleton for XML node storage.
// It stores child items in a doubly-linked list, same as Array.
// Full XML manipulation (attributes, element nesting) is deferred to a future version.
type XmlFragment struct {
	doc   *Doc
	name  string
	start *Item
}

func newXmlFragment(doc *Doc, name string) *XmlFragment {
	return &XmlFragment{doc: doc, name: name}
}

func (x *XmlFragment) sharedTypeName() string { return x.name }

// Len returns the number of non-deleted child nodes.
func (x *XmlFragment) Len() uint64 {
	x.doc.mu.Lock()
	defer x.doc.mu.Unlock()

	var n uint64
	for item := x.start; item != nil; item = item.Right {
		if !item.Deleted {
			n++
		}
	}
	return n
}

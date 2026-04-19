package yjs

// contentKind distinguishes the payload of an Item.
type contentKind uint8

const (
	contentString  contentKind = 4 // ContentString ref in Yjs
	contentAny     contentKind = 8 // ContentAny
	contentType    contentKind = 7 // ContentType (nested type)
	contentDeleted contentKind = 1 // ContentDeleted (GC'd)
	contentBinary  contentKind = 3 // ContentBinary
	contentEmbed   contentKind = 5 // ContentEmbed
	contentFormat  contentKind = 6 // ContentFormat
)

// Item is a node in the doubly-linked list that forms a Yjs type's content.
// Each item corresponds to one Insert operation with its causal context
// (originLeft, originRight) for LSST-based conflict resolution.
type Item struct {
	ID ID

	// Left and Right are the previous/next live items in insertion order.
	Left  *Item
	Right *Item

	// OriginLeft and OriginRight are the IDs of the items that were to the
	// left and right of this item when it was created (causal context for
	// the LSST conflict resolution algorithm).
	OriginLeft  *ID
	OriginRight *ID

	// Parent is the YType that owns this item.
	Parent interface{} // *Text | *Map | *Array | etc.

	// ParentSub is the map key when this item lives inside a Map.
	ParentSub *string

	// Content: one of string, []interface{}, etc.
	Kind    contentKind
	Content interface{} // string for contentString, interface{} for contentAny, etc.
	Length  uint64

	Deleted bool

	// parentName holds the shared type name from a decoded update, used during
	// integration to look up the actual SharedType from the Doc.
	parentName string
}

// content returns the string value for contentString items.
func (it *Item) strContent() string {
	if it.Kind == contentString {
		return it.Content.(string)
	}
	return ""
}

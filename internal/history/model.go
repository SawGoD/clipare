// Package history keeps a local, encrypted clipboard history per device.
//
// History is never synchronized between devices: every peer records the
// clipboard events it produced or received into its own encrypted store.
package history

// ContentType classifies a clipboard item. Wire messages remain text/plain;
// classification happens on top of the text payload. File and image are
// reserved for the upcoming attachment sync and must not be produced yet.
type ContentType string

const (
	ContentText  ContentType = "text"
	ContentURL   ContentType = "url"
	ContentFile  ContentType = "file"
	ContentImage ContentType = "image"
)

// Attachment describes one file of a future multi-file clipboard item.
// Only metadata will ever travel through the history store; binaries are
// transferred separately by the upcoming file sync.
type Attachment struct {
	ID     string
	Name   string
	Size   int64
	SHA256 string
	MIME   string
}

// Item is one clipboard event as shown in the History section. ID is the
// original sync message ID and never changes while the item propagates.
type Item struct {
	ID         string
	OriginID   string
	OriginName string
	Timestamp  int64
	Type       ContentType
	Text       string
	Sensitive  bool
	Files      []Attachment
}

// record is the on-disk payload shape. Everything sensitive — including the
// origin metadata — lives inside the authenticated plaintext.
type record struct {
	OriginID   string       `json:"o"`
	OriginName string       `json:"n"`
	Type       ContentType  `json:"t"`
	Hash       string       `json:"h"`
	Created    int64        `json:"c"`
	Expires    int64        `json:"e"`
	Text       string       `json:"x"`
	Sensitive  bool         `json:"s"`
	Files      []Attachment `json:"f,omitempty"`
}

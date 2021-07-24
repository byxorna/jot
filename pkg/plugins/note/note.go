// Package note is a riff on the fs plugin. It aims to simplify the storage interface
// and more gracefully support structured TODO items
package note

import (
	"time"
)

// Note: A single note.
type Note struct {
	// Attachments: Output only. The attachments attached to this note.
	Attachments []*Attachment `json:"attachments,omitempty"`

	// Body: The body of the note.
	Body *Section `json:"body,omitempty"`

	// CreateTime: Output only. When this note was created.
	CreateTime string `json:"createTime,omitempty"`

	// Name: Output only. The resource name of this note. See general note
	// on identifiers in KeepService.
	Name string `json:"name,omitempty"`

	// Title: The title of the note. Length must be less than 1,000
	// characters.
	Title string `json:"title,omitempty"`

	// TrashTime: Output only. When this note was trashed. If `trashed`, the
	// note is eventually deleted. If the note is not trashed, this field is
	// not set (and the trashed field is `false`).
	TrashTime time.Time `json:"trashTime,omitempty"`

	// Trashed: Output only. `true` if this note has been trashed. If
	// trashed, the note is eventually deleted.
	Trashed bool `json:"trashed,omitempty"`

	// UpdateTime: Output only. When this note was last modified.
	UpdateTime time.Time `json:"updateTime,omitempty"`
}

// Attachment: An attachment to a note.
type Attachment struct {
	// MimeType: The MIME types (IANA media types) in which the attachment
	// is available.
	MimeType []string `json:"mimeType,omitempty"`

	// Name: The resource name;
	Name string `json:"name,omitempty"`
}

// Section: The content of the note.
type Section struct {
	// List: Used if this section's content is a list.
	List *ListContent `json:"list,omitempty"`

	// Text: Used if this section's content is a block of text. The length
	// of the text content must be less than 20,000 characters.
	Text *TextContent `json:"text,omitempty"`
}

// TextContent: The block of text for a single text section or list item.
type TextContent struct {
	// Text: The text of the note. The limits on this vary with the specific
	// field using this type.
	Text string `json:"text,omitempty"`
}

// ListContent: The list of items for a single list note.
type ListContent struct {
	// ListItems: The items in the list. The number of items must be less
	// than 1,000.
	ListItems []*ListItem `json:"listItems,omitempty"`
}

// ListItem: A single list item in a note's list.
type ListItem struct {
	// Checked: Whether this item has been checked off or not.
	Checked bool `json:"checked,omitempty"`

	// ChildListItems: If set, list of list items nested under this list
	// item. Only one level of nesting is allowed.
	ChildListItems []*ListItem `json:"childListItems,omitempty"`

	// Text: The text of this item. Length must be less than 1,000
	// characters.
	Text *TextContent `json:"text,omitempty"`
}

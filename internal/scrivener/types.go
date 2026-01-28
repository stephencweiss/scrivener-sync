// Package scrivener provides types and utilities for working with Scrivener projects.
package scrivener

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"time"
)

// Document represents a single document in a Scrivener project.
type Document struct {
	UUID     string
	Title    string
	Content  string
	DocType  string // "folder" or "document"
	Modified time.Time
	Children []*Document
}

// ContentHash returns an MD5 hash of the document's content for change detection.
func (d *Document) ContentHash() string {
	hash := md5.Sum([]byte(d.Content))
	return hex.EncodeToString(hash[:])
}

// IsFolder returns true if this document is a folder.
func (d *Document) IsFolder() bool {
	return d.DocType == "folder"
}

// XML structures for parsing .scrivx files
// These structures preserve ALL Scrivener XML attributes to avoid data loss

// XMLProject represents the root element of a Scrivener project file.
type XMLProject struct {
	XMLName    xml.Name  `xml:"ScrivenerProject"`
	Identifier string    `xml:"Identifier,attr,omitempty"`
	Version    string    `xml:"Version,attr,omitempty"`
	Creator    string    `xml:"Creator,attr,omitempty"`
	Device     string    `xml:"Device,attr,omitempty"`
	Author     string    `xml:"Author,attr,omitempty"`
	Modified   string    `xml:"Modified,attr,omitempty"`
	ModID      string    `xml:"ModID,attr,omitempty"`
	Binder     XMLBinder `xml:"Binder"`
	// Preserve other sections we don't modify
	Collections            *XMLRawSection `xml:"Collections,omitempty"`
	SectionTypes           *XMLRawSection `xml:"SectionTypes,omitempty"`
	LabelSettings          *XMLRawSection `xml:"LabelSettings,omitempty"`
	StatusSettings         *XMLRawSection `xml:"StatusSettings,omitempty"`
	CustomMetaDataSettings *XMLRawSection `xml:"CustomMetaDataSettings,omitempty"`
	ProjectTargets         *XMLProjectTargets `xml:"ProjectTargets,omitempty"`
	RecentWritingHistory   *XMLRecentWritingHistory `xml:"RecentWritingHistory,omitempty"`
	RecentSearches         *XMLRawSection `xml:"RecentSearches,omitempty"`
	Favorites              *XMLRawSection `xml:"Favorites,omitempty"`
	PrintSettings          *XMLPrintSettings `xml:"PrintSettings,omitempty"`
}

// XMLRawSection preserves XML elements with just inner content.
type XMLRawSection struct {
	InnerXML []byte `xml:",innerxml"`
}

// XMLProjectTargets preserves the ProjectTargets section with its attributes.
type XMLProjectTargets struct {
	Notify   string `xml:"Notify,attr,omitempty"`
	InnerXML []byte `xml:",innerxml"`
}

// XMLRecentWritingHistory preserves the RecentWritingHistory section.
type XMLRecentWritingHistory struct {
	Date     string `xml:"Date,attr,omitempty"`
	InnerXML []byte `xml:",innerxml"`
}

// XMLPrintSettings preserves print settings with all attributes.
type XMLPrintSettings struct {
	PaperSize            string `xml:"PaperSize,attr,omitempty"`
	LeftMargin           string `xml:"LeftMargin,attr,omitempty"`
	RightMargin          string `xml:"RightMargin,attr,omitempty"`
	TopMargin            string `xml:"TopMargin,attr,omitempty"`
	BottomMargin         string `xml:"BottomMargin,attr,omitempty"`
	PaperType            string `xml:"PaperType,attr,omitempty"`
	Orientation          string `xml:"Orientation,attr,omitempty"`
	HorizontalPagination string `xml:"HorizontalPagination,attr,omitempty"`
	VerticalPagination   string `xml:"VerticalPagination,attr,omitempty"`
	ScaleFactor          string `xml:"ScaleFactor,attr,omitempty"`
	HorizontallyCentered string `xml:"HorizontallyCentered,attr,omitempty"`
	VerticallyCentered   string `xml:"VerticallyCentered,attr,omitempty"`
	Collates             string `xml:"Collates,attr,omitempty"`
	PagesAcross          string `xml:"PagesAcross,attr,omitempty"`
	PagesDown            string `xml:"PagesDown,attr,omitempty"`
}

// XMLBinder represents the binder (document tree) in a Scrivener project.
type XMLBinder struct {
	Items []XMLBinderItem `xml:"BinderItem"`
}

// XMLBinderItem represents a single item (document or folder) in the binder.
type XMLBinderItem struct {
	UUID         string           `xml:"UUID,attr"`
	Type         string           `xml:"Type,attr"`
	Created      string           `xml:"Created,attr"`
	Modified     string           `xml:"Modified,attr"`
	Title        string           `xml:"Title,omitempty"`
	MetaData     *XMLMetaData     `xml:"MetaData,omitempty"`
	TextSettings *XMLTextSettings `xml:"TextSettings,omitempty"`
	Children     []XMLBinderItem  `xml:"Children>BinderItem,omitempty"`
}

// XMLMetaData contains metadata for a binder item.
type XMLMetaData struct {
	IncludeInCompile string `xml:"IncludeInCompile,omitempty"`
}

// XMLTextSettings contains text settings for a binder item.
type XMLTextSettings struct {
	TextSelection string `xml:"TextSelection,omitempty"`
}

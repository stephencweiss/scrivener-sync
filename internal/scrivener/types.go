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

// XMLProject represents the root element of a Scrivener project file.
type XMLProject struct {
	XMLName xml.Name  `xml:"ScrivenerProject"`
	Binder  XMLBinder `xml:"Binder"`
}

// XMLBinder represents the binder (document tree) in a Scrivener project.
type XMLBinder struct {
	Items []XMLBinderItem `xml:"BinderItem"`
}

// XMLBinderItem represents a single item (document or folder) in the binder.
type XMLBinderItem struct {
	UUID     string          `xml:"UUID,attr"`
	Type     string          `xml:"Type,attr"`
	Created  string          `xml:"Created,attr"`
	Modified string          `xml:"Modified,attr"`
	Title    string          `xml:"Title"`
	MetaData XMLMetaData     `xml:"MetaData"`
	Children []XMLBinderItem `xml:"Children>BinderItem"`
}

// XMLMetaData contains metadata for a binder item.
type XMLMetaData struct {
	IncludeInCompile string `xml:"IncludeInCompile"`
}

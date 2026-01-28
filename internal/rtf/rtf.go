// Package rtf provides utilities for converting between RTF and plain text.
package rtf

import (
	"regexp"
	"strings"
)

var (
	// controlWordRe matches RTF control words like \par, \b0, etc.
	controlWordRe = regexp.MustCompile(`\\[a-z]+\d*\s?`)
	// whitespaceRe matches multiple whitespace characters
	whitespaceRe = regexp.MustCompile(`\s+`)
)

// StripRTF converts RTF content to plain text by removing RTF formatting.
func StripRTF(rtfContent string) string {
	// Remove RTF control words
	text := controlWordRe.ReplaceAllString(rtfContent, " ")

	// Remove braces
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")

	// Normalize whitespace
	text = whitespaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// ToRTF converts plain text to basic RTF format compatible with Scrivener.
func ToRTF(text string) string {
	// Basic RTF wrapper - Scrivener can handle this
	rtf := `{\rtf1\ansi\ansicpg1252\cocoartf2709`
	rtf += `\cocoatextscaling0\cocoaplatform0`
	rtf += `{\fonttbl\f0\fnil\fcharset0 Helvetica;}`
	rtf += `{\colortbl;\red255\green255\blue255;}`
	rtf += `\pard\tx560\tx1120\tx1680\tx2240\tx2800\tx3360\tx3920\tx4480\tx5040\tx5600\tx6160\tx6720\pardirnatural\partightenfactor0`
	rtf += `\f0\fs24 \cf0 `

	// Escape special RTF characters
	escaped := text
	escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")

	// Convert newlines to RTF line breaks
	escaped = strings.ReplaceAll(escaped, "\n", "\\\n")

	rtf += escaped + "}"
	return rtf
}

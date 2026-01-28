// Package rtf provides utilities for converting between RTF and plain text.
package rtf

import (
	"regexp"
	"strings"
)

var (
	// headerRe matches RTF header sections like {\fonttbl...} and {\colortbl...}
	headerRe = regexp.MustCompile(`\{\\(fonttbl|colortbl|stylesheet|info)[^}]*\}`)
	// controlWordRe matches RTF control words like \par, \b0, etc.
	controlWordRe = regexp.MustCompile(`\\[a-z]+\d*\s?`)
	// multiSpaceRe matches multiple spaces (but not newlines)
	multiSpaceRe = regexp.MustCompile(`[ \t]+`)
	// multiNewlineRe matches 3+ consecutive newlines
	multiNewlineRe = regexp.MustCompile(`\n{3,}`)
)

// StripRTF converts RTF content to plain text by removing RTF formatting.
func StripRTF(rtfContent string) string {
	text := rtfContent

	// Remove RTF header sections (font tables, color tables, etc.)
	text = headerRe.ReplaceAllString(text, "")

	// Convert RTF line breaks to newlines BEFORE removing control words
	text = strings.ReplaceAll(text, "\\par\n", "\n")
	text = strings.ReplaceAll(text, "\\par\r\n", "\n")
	text = strings.ReplaceAll(text, "\\par ", "\n")
	text = strings.ReplaceAll(text, "\\par", "\n")
	text = strings.ReplaceAll(text, "\\\n", "\n")
	text = strings.ReplaceAll(text, "\\\r\n", "\n")

	// Remove remaining RTF control words
	text = controlWordRe.ReplaceAllString(text, "")

	// Remove braces
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")

	// Normalize horizontal whitespace (but preserve newlines)
	text = multiSpaceRe.ReplaceAllString(text, " ")

	// Collapse excessive newlines (3+ becomes 2)
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")

	// Trim leading/trailing whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// Trim overall
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

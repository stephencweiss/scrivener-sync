// Package rtf provides utilities for converting between RTF and markdown.
package rtf

import (
	"fmt"
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

	// Markdown patterns
	headingRe    = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)
	boldRe       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRe     = regexp.MustCompile(`\*([^*]+)\*`)
	bulletRe     = regexp.MustCompile(`(?m)^-\s+(.+)$`)

	// RTF formatting patterns for extraction
	rtfBoldRe   = regexp.MustCompile(`\{\\b\s*([^}]*)\}`)
	rtfItalicRe = regexp.MustCompile(`\{\\i\s*([^}]*)\}`)
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

// MarkdownToRTF converts markdown content to RTF format for Scrivener.
// Handles: headings, bold, italic, and bullet lists.
func MarkdownToRTF(md string) string {
	// RTF header
	rtf := `{\rtf1\ansi\ansicpg1252\cocoartf2709`
	rtf += `\cocoatextscaling0\cocoaplatform0`
	rtf += `{\fonttbl\f0\fnil\fcharset0 Helvetica;}`
	rtf += `{\colortbl;\red255\green255\blue255;}`
	rtf += "\n"

	// Process line by line to handle block-level elements
	lines := strings.Split(md, "\n")
	var result []string

	for _, line := range lines {
		converted := convertMarkdownLine(line)
		result = append(result, converted)
	}

	// Join with RTF paragraph breaks
	content := strings.Join(result, `\par` + "\n")

	rtf += content + "}"
	return rtf
}

// convertMarkdownLine converts a single markdown line to RTF.
func convertMarkdownLine(line string) string {
	// Check for headings
	if matches := headingRe.FindStringSubmatch(line); matches != nil {
		level := len(matches[1]) // Number of # characters
		text := matches[2]
		text = convertInlineFormatting(escapeRTF(text))

		// Font sizes: H1=36pt, H2=30pt, H3=26pt (RTF uses half-points)
		sizes := map[int]int{1: 72, 2: 60, 3: 52}
		fontSize := sizes[level]
		if fontSize == 0 {
			fontSize = 52
		}

		return fmt.Sprintf(`\pard\f0\fs%d\b %s\b0\fs24`, fontSize, text)
	}

	// Check for bullet points
	if matches := bulletRe.FindStringSubmatch(line); matches != nil {
		text := convertInlineFormatting(escapeRTF(matches[1]))
		return `\pard\li360\f0\fs24 \bullet  ` + text
	}

	// Regular paragraph
	text := convertInlineFormatting(escapeRTF(line))
	return `\pard\f0\fs24 ` + text
}

// convertInlineFormatting converts bold and italic markdown to RTF.
func convertInlineFormatting(text string) string {
	// Convert **bold** to {\b bold}
	text = boldRe.ReplaceAllString(text, `{\b $1}`)

	// Convert *italic* to {\i italic}
	// Be careful not to match already-converted bold markers
	text = italicRe.ReplaceAllString(text, `{\i $1}`)

	return text
}

// escapeRTF escapes special RTF characters.
func escapeRTF(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "{", "\\{")
	text = strings.ReplaceAll(text, "}", "\\}")
	return text
}

// RTFToMarkdown converts RTF content to markdown, preserving formatting.
// Handles: bold, italic, and basic structure.
func RTFToMarkdown(rtfContent string) string {
	text := rtfContent

	// Remove RTF header sections (font tables, color tables, etc.)
	text = headerRe.ReplaceAllString(text, "")

	// Convert bold: {\b text} or \b text\b0 to **text**
	// Handle nested braces format
	text = rtfBoldRe.ReplaceAllString(text, "**$1**")
	// Handle inline format: \b text\b0
	text = regexp.MustCompile(`\\b\s+([^\\]+)\\b0`).ReplaceAllString(text, "**$1**")

	// Convert italic: {\i text} or \i text\i0 to *text*
	text = rtfItalicRe.ReplaceAllString(text, "*$1*")
	text = regexp.MustCompile(`\\i\s+([^\\]+)\\i0`).ReplaceAllString(text, "*$1*")

	// Convert RTF line breaks to newlines
	text = strings.ReplaceAll(text, "\\par\n", "\n")
	text = strings.ReplaceAll(text, "\\par\r\n", "\n")
	text = strings.ReplaceAll(text, "\\par ", "\n")
	text = strings.ReplaceAll(text, "\\par", "\n")
	text = strings.ReplaceAll(text, "\\\n", "\n")
	text = strings.ReplaceAll(text, "\\\r\n", "\n")

	// Handle font size changes for headings
	// \fs72 = 36pt = H1, \fs60 = 30pt = H2, \fs52 = 26pt = H3
	text = convertFontSizesToHeadings(text)

	// Remove remaining RTF control words
	text = controlWordRe.ReplaceAllString(text, "")

	// Remove braces
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")

	// Unescape RTF special characters
	text = strings.ReplaceAll(text, "\\\\", "\\")
	text = strings.ReplaceAll(text, "\\{", "{")
	text = strings.ReplaceAll(text, "\\}", "}")

	// Normalize whitespace
	text = multiSpaceRe.ReplaceAllString(text, " ")
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")

	// Trim each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	return strings.TrimSpace(text)
}

// convertFontSizesToHeadings converts RTF font size markers to markdown headings.
func convertFontSizesToHeadings(text string) string {
	// Pattern: \fsNN followed by text until next \fs or end
	// This is a heuristic - large fonts at start of line become headings
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Check for large font size at start of line
		if strings.Contains(line, "\\fs72") || strings.Contains(line, "\\fs68") {
			// H1 - remove the font size marker and prefix with #
			line = regexp.MustCompile(`\\fs\d+\s*`).ReplaceAllString(line, "")
			line = "# " + strings.TrimSpace(line)
		} else if strings.Contains(line, "\\fs60") || strings.Contains(line, "\\fs56") {
			// H2
			line = regexp.MustCompile(`\\fs\d+\s*`).ReplaceAllString(line, "")
			line = "## " + strings.TrimSpace(line)
		} else if strings.Contains(line, "\\fs52") || strings.Contains(line, "\\fs48") {
			// H3
			line = regexp.MustCompile(`\\fs\d+\s*`).ReplaceAllString(line, "")
			line = "### " + strings.TrimSpace(line)
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

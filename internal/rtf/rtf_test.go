package rtf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test fixtures directory
var testdataDir = filepath.Join("..", "..", "testdata", "rtf")

func TestStripRTF_BasicContent(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}{\colortbl;\red0\green0\blue0;}
\pard\f0 Hello World}`

	result := StripRTF(rtf)

	if !strings.Contains(result, "Hello World") {
		t.Errorf("Expected 'Hello World' in result, got: %s", result)
	}
}

func TestStripRTF_HeaderRemoval(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}{\colortbl;\red255\green255\blue255;}
\pard\f0\fs24 Content here}`

	result := StripRTF(rtf)

	if strings.Contains(result, "fonttbl") {
		t.Error("fonttbl should be removed")
	}
	if strings.Contains(result, "colortbl") {
		t.Error("colortbl should be removed")
	}
	if !strings.Contains(result, "Content here") {
		t.Errorf("Content should be preserved, got: %s", result)
	}
}

func TestStripRTF_ParToNewline(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}
\pard Line one\par
Line two\par
Line three}`

	result := StripRTF(rtf)

	if !strings.Contains(result, "Line one") {
		t.Error("Line one should be present")
	}
	if !strings.Contains(result, "Line two") {
		t.Error("Line two should be present")
	}

	// Should have newlines between lines
	lines := strings.Split(result, "\n")
	foundLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Line") {
			foundLines++
		}
	}
	if foundLines < 3 {
		t.Errorf("Expected 3 lines starting with 'Line', got %d in: %s", foundLines, result)
	}
}

func TestRTFToMarkdown_BasicText(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}{\colortbl;\red255\green255\blue255;}
\pard\f0\fs24 This is plain text.\par
Second paragraph here.}`

	result := RTFToMarkdown(rtf)

	if !strings.Contains(result, "This is plain text.") {
		t.Errorf("Plain text not found in result: %s", result)
	}
	if !strings.Contains(result, "Second paragraph here.") {
		t.Errorf("Second paragraph not found in result: %s", result)
	}
}

func TestRTFToMarkdown_NoArtifacts(t *testing.T) {
	// This test ensures the \par vs \pard bug is fixed
	rtf := `{\rtf1\ansi\ansicpg1252\cocoartf2709\cocoatextscaling0\cocoaplatform0{\fonttbl\f0\fnil\fcharset0 Helvetica;}{\colortbl;\red255\green255\blue255;}
\pard\tx560\tx1120\pardirnatural\partightenfactor0
\f0\fs24 \cf0 Test content here.}`

	result := RTFToMarkdown(rtf)

	// Should NOT contain these artifacts
	artifacts := []string{"ddirnatural", "tightenfactor", "dirnatural", "artightenfactor"}
	for _, artifact := range artifacts {
		if strings.Contains(result, artifact) {
			t.Errorf("Found artifact '%s' in result: %s", artifact, result)
		}
	}

	if !strings.Contains(result, "Test content here.") {
		t.Errorf("Content should be preserved, got: %s", result)
	}
}

func TestRTFToMarkdown_HexCharacters(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}
\pard Don\'92t forget the \'93quotes\'94.}`

	result := RTFToMarkdown(rtf)

	if !strings.Contains(result, "Don't") {
		t.Errorf("Expected apostrophe conversion, got: %s", result)
	}
	if !strings.Contains(result, `"quotes"`) {
		t.Errorf("Expected quote conversion, got: %s", result)
	}
}

func TestRTFToMarkdown_PreservesBold(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}
\pard This is {\b bold text} here.}`

	result := RTFToMarkdown(rtf)

	if !strings.Contains(result, "**bold text**") {
		t.Errorf("Expected **bold text**, got: %s", result)
	}
}

func TestRTFToMarkdown_PreservesItalic(t *testing.T) {
	rtf := `{\rtf1\ansi{\fonttbl\f0\fnil Helvetica;}
\pard This is {\i italic text} here.}`

	result := RTFToMarkdown(rtf)

	if !strings.Contains(result, "*italic text*") {
		t.Errorf("Expected *italic text*, got: %s", result)
	}
}

func TestMarkdownToRTF_BasicText(t *testing.T) {
	md := "Hello World"

	result := MarkdownToRTF(md)

	if !strings.HasPrefix(result, `{\rtf1\ansi`) {
		t.Error("Result should start with RTF header")
	}
	if !strings.Contains(result, "Hello World") {
		t.Error("Content should be preserved")
	}
	if !strings.HasSuffix(result, "}") {
		t.Error("Result should end with closing brace")
	}
}

func TestMarkdownToRTF_Headings(t *testing.T) {
	tests := []struct {
		md       string
		expected string
	}{
		{"# Heading 1", "\\fs72"},
		{"## Heading 2", "\\fs60"},
		{"### Heading 3", "\\fs52"},
	}

	for _, tc := range tests {
		result := MarkdownToRTF(tc.md)
		if !strings.Contains(result, tc.expected) {
			t.Errorf("For '%s', expected font size %s, got: %s", tc.md, tc.expected, result)
		}
	}
}

func TestMarkdownToRTF_Bold(t *testing.T) {
	md := "This is **bold** text"

	result := MarkdownToRTF(md)

	if !strings.Contains(result, `{\b bold}`) {
		t.Errorf("Expected {\\b bold}, got: %s", result)
	}
}

func TestMarkdownToRTF_Italic(t *testing.T) {
	md := "This is *italic* text"

	result := MarkdownToRTF(md)

	if !strings.Contains(result, `{\i italic}`) {
		t.Errorf("Expected {\\i italic}, got: %s", result)
	}
}

func TestMarkdownToRTF_Bullets(t *testing.T) {
	md := "- First item\n- Second item"

	result := MarkdownToRTF(md)

	if !strings.Contains(result, `\bullet`) {
		t.Errorf("Expected bullet character, got: %s", result)
	}
}

func TestMarkdownToRTF_Roundtrip(t *testing.T) {
	// Simple text should survive a roundtrip
	original := "Hello World"

	rtf := MarkdownToRTF(original)
	result := RTFToMarkdown(rtf)

	if !strings.Contains(result, "Hello World") {
		t.Errorf("Content lost in roundtrip: original=%s, result=%s", original, result)
	}
}

func TestRTFToMarkdown_SimpleFile(t *testing.T) {
	rtfPath := filepath.Join(testdataDir, "simple.rtf")
	expectedPath := filepath.Join(testdataDir, "expected", "simple.md")

	rtfContent, err := os.ReadFile(rtfPath)
	if err != nil {
		t.Skipf("Test file not found: %v", err)
	}

	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Skipf("Expected file not found: %v", err)
	}

	result := RTFToMarkdown(string(rtfContent))

	// Normalize whitespace for comparison
	resultNorm := strings.TrimSpace(result)
	expectedNorm := strings.TrimSpace(string(expected))

	if resultNorm != expectedNorm {
		t.Errorf("RTF conversion mismatch.\nExpected:\n%s\n\nGot:\n%s", expectedNorm, resultNorm)
	}
}

func TestRTFToMarkdown_ComplexFile(t *testing.T) {
	rtfPath := filepath.Join(testdataDir, "complex.rtf")

	rtfContent, err := os.ReadFile(rtfPath)
	if err != nil {
		t.Skipf("Test file not found: %v", err)
	}

	result := RTFToMarkdown(string(rtfContent))

	// The complex file is the Coach character - verify key content is present
	if !strings.Contains(result, "Coach") {
		t.Error("Expected 'Coach' in result")
	}
	if !strings.Contains(result, "hockey") {
		t.Error("Expected 'hockey' in result")
	}

	// Verify no RTF artifacts
	artifacts := []string{"\\pard", "\\f0", "\\fs24", "fonttbl", "colortbl"}
	for _, artifact := range artifacts {
		if strings.Contains(result, artifact) {
			t.Errorf("Found RTF artifact '%s' in result", artifact)
		}
	}
}

func TestToRTF_EscapesSpecialCharacters(t *testing.T) {
	text := `Test with {braces} and \backslash`

	result := ToRTF(text)

	if !strings.Contains(result, `\{braces\}`) {
		t.Errorf("Braces should be escaped, got: %s", result)
	}
	if !strings.Contains(result, `\\backslash`) {
		t.Errorf("Backslash should be escaped, got: %s", result)
	}
}

# Scriv-Sync Test Plan

## Overview

This plan covers unit and integration tests for the scriv-sync tool. Tests will use mock data derived from the real Harcroft.scriv project structure.

## Test Categories

### 1. Unit Tests

#### RTF Package (`internal/rtf/rtf_test.go`)

| Test | Description |
|------|-------------|
| `TestStripRTF_BasicContent` | Strip RTF from simple text |
| `TestStripRTF_HeaderRemoval` | Remove fonttbl, colortbl sections |
| `TestStripRTF_ParToNewline` | Convert \par to newlines without matching \pard |
| `TestStripRTF_HexCharacters` | Convert \'92 -> ', \'93/94 -> " |
| `TestRTFToMarkdown_BasicText` | Convert RTF to clean markdown |
| `TestRTFToMarkdown_PreservesBold` | Handle {\b text} -> **text** |
| `TestRTFToMarkdown_PreservesItalic` | Handle {\i text} -> *text* |
| `TestRTFToMarkdown_NoArtifacts` | Verify no ddirnatural/tightenfactor artifacts |
| `TestMarkdownToRTF_BasicText` | Convert plain text to RTF |
| `TestMarkdownToRTF_Headings` | Convert # H1, ## H2, ### H3 |
| `TestMarkdownToRTF_Bold` | Convert **bold** to {\b bold} |
| `TestMarkdownToRTF_Italic` | Convert *italic* to {\i italic} |
| `TestMarkdownToRTF_Bullets` | Convert - bullet to RTF bullets |
| `TestMarkdownToRTF_Roundtrip` | Markdown -> RTF -> Markdown preserves content |

#### Scrivener Reader (`internal/scrivener/reader_test.go`)

| Test | Description |
|------|-------------|
| `TestReadProject_ParsesXML` | Parse .scrivx file correctly |
| `TestReadProject_PreservesAttributes` | Root element attributes preserved |
| `TestReadProject_ParsesBinder` | All binder items parsed |
| `TestReadProject_FolderTypes` | Recognize DraftFolder, ResearchFolder, TrashFolder |
| `TestReadProject_NestedChildren` | Handle nested folder structures |
| `TestReadProject_ReadsContent` | Read content.rtf files |
| `TestReadProject_ConvertToMarkdown` | Content converted via RTFToMarkdown |

#### Scrivener Writer (`internal/scrivener/writer_test.go`)

| Test | Description |
|------|-------------|
| `TestWriter_CreateDocument` | Create new document in binder |
| `TestWriter_CreateFolder` | Create new folder in binder |
| `TestWriter_UpdateContent` | Update existing document content |
| `TestWriter_PreservesProjectAttrs` | Root element attributes preserved on save |
| `TestWriter_PreservesSections` | Collections, LabelSettings, etc. preserved |
| `TestWriter_GeneratesUUID` | UUIDs are valid and unique |
| `TestWriter_TimestampFormat` | Uses Scrivener timestamp format |

#### Sync State (`internal/sync/state_test.go`)

| Test | Description |
|------|-------------|
| `TestState_NewState` | Create fresh state |
| `TestState_SaveLoad` | State persists to JSON |
| `TestState_TrackDocument` | Add document to state |
| `TestState_ContentHash` | Hash comparison works |
| `TestState_DetectChanges` | Detect modified documents |

### 2. Integration Tests

#### Full Sync Flow (`internal/sync/syncer_test.go`)

| Test | Description |
|------|-------------|
| `TestSync_InitProject` | Initialize new sync config |
| `TestSync_ScrivenerToMarkdown` | Pull changes from Scrivener |
| `TestSync_MarkdownToScrivener` | Push changes to Scrivener |
| `TestSync_ConflictDetection` | Detect bi-directional changes |
| `TestSync_OrphanHandling` | Handle deleted files |
| `TestSync_PreservesXMLStructure` | Project XML roundtrips correctly |

### 3. Test Fixtures

Create `testdata/` directory with:

```
testdata/
├── sample.scriv/              # Minimal Scrivener project
│   ├── sample.scrivx          # Project XML
│   └── Files/
│       └── Data/
│           ├── {UUID1}/content.rtf
│           └── {UUID2}/content.rtf
├── rtf/
│   ├── simple.rtf             # Basic RTF content
│   ├── formatted.rtf          # Bold/italic content
│   ├── complex.rtf            # Real Harcroft content sample
│   └── expected/              # Expected markdown output
└── markdown/
    ├── simple.md
    └── formatted.md
```

## Implementation Order

1. Create testdata fixtures from Harcroft.scriv samples
2. RTF package unit tests (most isolated)
3. Scrivener reader tests
4. Scrivener writer tests
5. State management tests
6. Integration tests (full sync flow)

## Test Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/rtf/...

# Verbose output
go test -v ./...
```

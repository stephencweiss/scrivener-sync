# Scrivener Sync Tool

Bi-directional sync between markdown files and a Scrivener project.

## Installation

```bash
# Build the sync tool
make build

# Or install globally
make install
```

## Quick Start

```bash
# Initialize a new sync project
./scriv-sync init \
  --local /Users/sweiss/code/harcroft \
  --scriv /Users/sweiss/Library/CloudStorage/Dropbox/Apps/Scrivener/Harcroft.scriv \
  --alias harcroft

# Run bi-directional sync
./scriv-sync sync harcroft

# Check status
./scriv-sync status harcroft

# List all configured projects
./scriv-sync list
```

## Commands

| Command | Description |
|---------|-------------|
| `scriv-sync init` | Initialize a new sync project |
| `scriv-sync sync <alias>` | Bi-directional sync |
| `scriv-sync pull <alias>` | Scrivener -> markdown |
| `scriv-sync push <alias>` | markdown -> Scrivener |
| `scriv-sync status <alias>` | Show pending changes |
| `scriv-sync list` | List all configured projects |

### Init Flags

| Flag | Description |
|------|-------------|
| `--local <path>` | Path to local markdown directory (required) |
| `--scriv <path>` | Path to Scrivener .scriv project (required) |
| `--alias <name>` | Alias name for this project (required) |

### Global Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview changes without applying |
| `--non-interactive` | Use config defaults, skip prompts |

## Configuration

Configuration is stored in `~/.scriv-sync/config.yaml`:

```yaml
version: "1.0"
projects:
  harcroft:
    local_path: /Users/sweiss/code/harcroft
    scriv_path: /Users/sweiss/Library/CloudStorage/Dropbox/Apps/Scrivener/Harcroft.scriv
    folder_mappings:
      - markdown_dir: characters
        scrivener_folder: Characters
        sync_enabled: true
      - markdown_dir: plot
        scrivener_folder: Plot
        sync_enabled: true
    options:
      create_missing_folders: true
      default_conflict_resolution: prompt  # prompt | markdown | scrivener | skip
      default_deletion_action: prompt      # prompt | delete | recreate | skip
```

Sync state is stored separately in `~/.scriv-sync/state/<alias>.json`.

## Sync Behavior

- **Bi-directional**: Changes on either side are detected and synced
- **Conflict detection**: When both sides change, you're prompted to choose
- **Orphan handling**: Deleted files are detected with options to delete or recreate
- **State tracking**: Tracks what's been synced per project

### File Mapping

Files are mapped by title:
- `characters/wilder-young.md` <-> Scrivener "Characters" folder -> "Wilder Young" document
- Titles are converted: `wilder-young` -> `Wilder Young`

## Building from Source

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Format code
make fmt
```

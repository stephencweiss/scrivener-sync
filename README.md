# Harcroft

A mystery novel writing project with bi-directional Scrivener sync.

## Scrivener Sync

Bi-directional sync between this markdown project and a Scrivener project.

### Installation

```bash
# Build the sync tool
make build

# Or install globally
make install
```

### Quick Start

```bash
# Initialize with your Scrivener project
./scriv-sync init ./YourProject.scriv

# Run bi-directional sync
./scriv-sync
```

### Commands

| Command | Description |
|---------|-------------|
| `scriv-sync` | Bi-directional sync (default) |
| `scriv-sync init <path>` | Initialize with folder discovery |
| `scriv-sync pull` | Scrivener → markdown |
| `scriv-sync push` | markdown → Scrivener |
| `scriv-sync status` | Show pending changes |

### Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Config file (default: `.scrivener-sync.yaml`) |
| `--dry-run` | Preview changes without applying |
| `--non-interactive` | Use config defaults, skip prompts |

### Configuration

After running `init`, edit `.scrivener-sync.yaml` to customize:

```yaml
version: "1.0"
scrivener_project: ./YourProject.scriv
markdown_root: .

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

### Sync Behavior

- **Bi-directional**: Changes on either side are detected and synced
- **Conflict detection**: When both sides change, you're prompted to choose
- **Orphan handling**: Deleted files are detected with options to delete or recreate
- **State tracking**: `.sync_state.json` tracks what's been synced

### File Mapping

Files are mapped by title:
- `characters/wilder-young.md` ↔ Scrivener "Characters" folder → "Wilder Young" document
- Titles are converted: `wilder-young` → `Wilder Young`

### Building from Source

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

> [!NOTE]
> This project is not actively maintained. You're welcome to fork and modify it. Licensed under MIT.

# Schemata

A visual PostgreSQL database schema designer controlled entirely via MCP (Model Context Protocol). The UI is read-only — all schema mutations come through MCP tools, letting your AI agent design database schemas visually in real-time.

Built with [Wails v2](https://wails.io/) (Go backend + Svelte frontend).

## Features

- **Visual Schema Design** — SVG canvas with orthogonal FK connection routing, pan/zoom, and drag-to-position tables
- **MCP-Driven** — 40+ MCP tools for creating schemas, tables, columns, foreign keys, constraints, indexes, views, and more
- **Full PostgreSQL Support** — Enums, check constraints, partial indexes (GIN/GIST/BRIN), generated columns, extensions, ON DELETE/UPDATE actions
- **Project Files** — Save/load `.schemata` YAML files for persistent designs
- **SQL Export** — Generate complete PostgreSQL DDL with proper constraint naming
- **Undo/Redo** — Full 50-step history with Ctrl+Z / Ctrl+Y
- **Table Search** — Ctrl+F to find tables and columns with autocomplete
- **Hover Details** — Popover showing column properties, FK targets, enum values, check constraints
- **Views** — Structured view columns with source table references, separate canvas tab
- **Multi-Schema** — Color-coded schemas with legend
- **Typed Tool Schemas** — All MCP tools use JSON Schema with required/optional field enforcement

## Installation

### Download

Download the latest release from [Releases](../../releases).

**Option A — Installer:**
Run `schemata-setup.exe` to install both binaries to `%LOCALAPPDATA%\Schemata\`.

**Option B — Manual:**
Extract the zip and place both files somewhere on your system:
- `schemata.exe` — The desktop app
- `schemata-mcp.exe` — The MCP server

### Setup

1. Start the Schemata desktop app
2. Click the bot icon (top-left) to see MCP setup instructions for your agent
3. Or manually add the MCP server:

**Claude Code:**
```bash
claude mcp add schemata -- "C:\path\to\schemata-mcp.exe"
```

**Cursor / Windsurf / Other agents:**
Add to your MCP config:
```json
{
  "schemata": {
    "command": "C:\\path\\to\\schemata-mcp.exe",
    "args": [],
    "type": "stdio"
  }
}
```

## MCP Tools

### Schema & Tables
`create_schema` `update_schema` `delete_schema` `create_table` `update_table` `rename_table` `delete_table`

### Columns
`add_column` `update_column` `delete_column` `reorder_columns`

### Relationships
`add_foreign_key` `update_foreign_key` `delete_foreign_key`

### Constraints & Indexes
`add_constraint` `update_constraint` `delete_constraint` `add_check` `delete_check` `add_index` `update_index` `delete_index`

### Types & Extensions
`create_enum` `update_enum` `delete_enum` `add_extension` `delete_extension`

### Views
`create_view` `update_view` `delete_view`

### Positioning
`set_table_position` `clear_table_position` `set_view_position` `clear_view_position`

### Utility
`get_schema` `export_sql` `save_project` `load_project` `clear` `undo` `redo`

## Architecture

```
┌─────────────────┐     stdio      ┌──────────────────┐
│   AI Agent      │◄──────────────►│  schemata-mcp    │
│ (Claude, etc.)  │    (MCP)       │  (Go binary)     │
└─────────────────┘                └────────┬─────────┘
                                            │ HTTP
                                   ┌────────▼─────────┐
                                   │  schemata         │
                                   │  (Wails desktop)  │
                                   │                   │
                                   │  Go: state mgmt   │
                                   │  Svelte: SVG UI   │
                                   └───────────────────┘
```

The desktop app holds the in-memory schema state and exposes an HTTP API on `localhost:9800`. The MCP server is a separate binary that translates MCP tool calls into HTTP requests. This means the desktop app runs independently — you start it yourself, and your AI agent connects via the MCP server.

## Building from Source

**Requirements:** Go 1.23+, Node.js 18+, [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone and build
git clone https://github.com/dev-idkwhoami/schemata.git
cd schemata

# Development
wails dev

# Production build (Windows)
build.bat
```

## License

MIT — see [LICENSE](LICENSE)

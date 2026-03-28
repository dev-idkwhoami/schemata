package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var apiBase string

func main() {
	log.SetOutput(os.Stderr)

	port := discoverPort()
	apiBase = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Verify the desktop app is running
	if !pingApp() {
		log.Fatalf("[MCP] Schemata desktop app is not running on port %d. Please start it first.", port)
	}
	log.Printf("[MCP] Connected to Schemata on %s", apiBase)

	s := server.NewMCPServer(
		"schemata",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("[MCP] Server error: %v", err)
	}
}

func discoverPort() int {
	// Check CLI arg first
	for i, arg := range os.Args[1:] {
		if arg == "--port" && i+1 < len(os.Args[1:])-1 {
			if p, err := strconv.Atoi(os.Args[i+2]); err == nil {
				return p
			}
		}
	}
	// Check port file
	dir, err := os.UserConfigDir()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(dir, "schemata", "port"))
		if err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				return p
			}
		}
	}
	return 9800
}

func pingApp() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(apiBase + "/api/state")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func apiPost(endpoint string, body interface{}) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(apiBase+endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("cannot reach Schemata app: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}
	if result.Status == "error" {
		return "", fmt.Errorf("%s", result.Message)
	}
	return result.Message, nil
}

func apiGet(endpoint string) (string, error) {
	resp, err := http.Get(apiBase + endpoint)
	if err != nil {
		return "", fmt.Errorf("cannot reach Schemata app: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func mcpErr(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: msg},
		},
	}
}

func defaultSchema(val string) string {
	if val == "" {
		return "public"
	}
	return val
}

// --- Input structs for typed tool handlers ---

type CreateSchemaInput struct {
	Name  string `json:"name" jsonschema:"Schema name"`
	Color string `json:"color" jsonschema:"Hex color for display (e.g. #3b82f6)"`
}

type CreateTableInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name   string `json:"name" jsonschema:"Table name"`
}

type AddColumnInput struct {
	Schema     string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table      string `json:"table" jsonschema:"Table name"`
	Name       string `json:"name" jsonschema:"Column name"`
	Type       string `json:"type" jsonschema:"PostgreSQL column type"`
	Nullable   *bool  `json:"nullable,omitempty" jsonschema:"Whether the column is nullable"`
	PrimaryKey *bool  `json:"primary_key,omitempty" jsonschema:"Whether the column is a primary key"`
	Unique     *bool  `json:"unique,omitempty" jsonschema:"Whether the column has a unique constraint"`
	Default    string `json:"default,omitempty" jsonschema:"Default value expression"`
	Comment    string `json:"comment,omitempty" jsonschema:"Column comment"`
	Position   *int   `json:"position,omitempty" jsonschema:"Position index (0-based). Omit or -1 to append at end."`
}

type UpdateColumnInput struct {
	Schema     string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table      string `json:"table" jsonschema:"Table name"`
	Column     string `json:"column" jsonschema:"Current column name"`
	Name       string `json:"name,omitempty" jsonschema:"New column name"`
	Type       string `json:"type,omitempty" jsonschema:"New column type"`
	Nullable   *bool  `json:"nullable,omitempty" jsonschema:"Set nullable (must be boolean true/false)"`
	PrimaryKey *bool  `json:"primary_key,omitempty" jsonschema:"Set primary key (must be boolean true/false)"`
	Unique     *bool  `json:"unique,omitempty" jsonschema:"Set unique (must be boolean true/false)"`
	Default    string `json:"default,omitempty" jsonschema:"Set default value"`
	Comment    string `json:"comment,omitempty" jsonschema:"Set column comment"`
}

type ReorderColumnsInput struct {
	Schema      string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table       string   `json:"table" jsonschema:"Table name"`
	ColumnOrder []string `json:"column_order" jsonschema:"Column names in desired order"`
}

type AddForeignKeyInput struct {
	FromSchema string `json:"from_schema,omitempty" jsonschema:"Source schema (defaults to 'public' if omitted)"`
	FromTable  string `json:"from_table" jsonschema:"Source table"`
	FromColumn string `json:"from_column" jsonschema:"Source column"`
	ToSchema   string `json:"to_schema,omitempty" jsonschema:"Target schema (defaults to 'public' if omitted)"`
	ToTable    string `json:"to_table" jsonschema:"Target table"`
	ToColumn   string `json:"to_column" jsonschema:"Target column"`
	OnDelete   string `json:"on_delete,omitempty" jsonschema:"ON DELETE action: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION"`
	OnUpdate   string `json:"on_update,omitempty" jsonschema:"ON UPDATE action: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION"`
}

type UpdateForeignKeyInput struct {
	FromSchema string `json:"from_schema,omitempty" jsonschema:"Source schema (defaults to 'public' if omitted)"`
	FromTable  string `json:"from_table" jsonschema:"Source table"`
	FromColumn string `json:"from_column" jsonschema:"Source column"`
	OnDelete   string `json:"on_delete,omitempty" jsonschema:"ON DELETE action: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION"`
	OnUpdate   string `json:"on_update,omitempty" jsonschema:"ON UPDATE action: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION"`
}

type DeleteForeignKeyInput struct {
	FromSchema string `json:"from_schema,omitempty" jsonschema:"Source schema (defaults to 'public' if omitted)"`
	FromTable  string `json:"from_table" jsonschema:"Source table"`
	FromColumn string `json:"from_column" jsonschema:"Source column"`
}

type RenameTableInput struct {
	Schema  string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	OldName string `json:"old_name" jsonschema:"Current table name"`
	NewName string `json:"new_name" jsonschema:"New table name"`
}

type NameOnlyInput struct {
	Name string `json:"name" jsonschema:"Name"`
}

type SchemaNameInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name   string `json:"name" jsonschema:"Name"`
}

type SchemaTableInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table  string `json:"table" jsonschema:"Table name"`
}

type SchemaTableColumnInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table  string `json:"table" jsonschema:"Table name"`
	Column string `json:"column" jsonschema:"Column name"`
}

type ConstraintInput struct {
	Schema  string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table   string   `json:"table" jsonschema:"Table name"`
	Type    string   `json:"type" jsonschema:"Constraint type: 'primary_key' or 'unique'"`
	Columns []string `json:"columns" jsonschema:"Column names that form the constraint"`
}

type UpdateConstraintInput struct {
	Schema     string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table      string   `json:"table" jsonschema:"Table name"`
	Type       string   `json:"type" jsonschema:"Constraint type: 'primary_key' or 'unique'"`
	OldColumns []string `json:"old_columns" jsonschema:"Current column names in the constraint"`
	NewColumns []string `json:"new_columns" jsonschema:"New column names for the constraint"`
}

type AddIndexInput struct {
	Schema  string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table   string   `json:"table" jsonschema:"Table name"`
	Columns []string `json:"columns" jsonschema:"Column names for the index"`
	Unique  *bool    `json:"unique,omitempty" jsonschema:"Whether this is a unique index"`
	Name    string   `json:"name,omitempty" jsonschema:"Index name (auto-generated if omitted)"`
}

type UpdateIndexInput struct {
	Schema  string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table   string   `json:"table" jsonschema:"Table name"`
	Name    string   `json:"name" jsonschema:"Current index name"`
	Columns []string `json:"columns,omitempty" jsonschema:"New column names for the index"`
	Unique  *bool    `json:"unique,omitempty" jsonschema:"Whether this is a unique index"`
	NewName string   `json:"new_name,omitempty" jsonschema:"New index name"`
}

type SchemaTableNameInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table  string `json:"table" jsonschema:"Table name"`
	Name   string `json:"name" jsonschema:"Index name"`
}

type EnumInput struct {
	Schema string   `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name   string   `json:"name" jsonschema:"Enum type name"`
	Values []string `json:"values" jsonschema:"Enum values"`
}

type DeleteEnumInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name   string `json:"name" jsonschema:"Enum type name"`
}

type AddCheckInput struct {
	Schema     string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table      string `json:"table" jsonschema:"Table name"`
	Name       string `json:"name" jsonschema:"Constraint name"`
	Expression string `json:"expression" jsonschema:"Check expression (SQL)"`
}

type DeleteCheckInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table  string `json:"table" jsonschema:"Table name"`
	Name   string `json:"name" jsonschema:"Constraint name"`
}

type SetTablePositionInput struct {
	Schema string  `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Table  string  `json:"table" jsonschema:"Table name"`
	X      float64 `json:"x" jsonschema:"X coordinate"`
	Y      float64 `json:"y" jsonschema:"Y coordinate"`
}

type UpdateTableInput struct {
	Schema  string `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name    string `json:"name" jsonschema:"Current table name"`
	Comment string `json:"comment,omitempty" jsonschema:"New table comment"`
	NewName string `json:"new_name,omitempty" jsonschema:"Rename the table to this name"`
}

type UpdateSchemaInput struct {
	Name    string `json:"name" jsonschema:"Current schema name"`
	NewName string `json:"new_name,omitempty" jsonschema:"Rename the schema to this name"`
	Color   string `json:"color,omitempty" jsonschema:"New hex color for display"`
}

type ViewColumnInput struct {
	Name         string `json:"name" jsonschema:"Column alias in the view"`
	Type         string `json:"type" jsonschema:"Column data type"`
	SourceSchema string `json:"source_schema" jsonschema:"Schema of the source table"`
	SourceTable  string `json:"source_table" jsonschema:"Source table name"`
	SourceColumn string `json:"source_column" jsonschema:"Source column name"`
}

type CreateViewInput struct {
	Schema  string            `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name    string            `json:"name" jsonschema:"View name"`
	Columns []ViewColumnInput `json:"columns" jsonschema:"View columns with source references"`
}

type UpdateViewInput struct {
	Schema  string            `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name    string            `json:"name" jsonschema:"Current view name"`
	Comment string            `json:"comment,omitempty" jsonschema:"New view comment"`
	NewName string            `json:"new_name,omitempty" jsonschema:"Rename the view to this name"`
	Columns []ViewColumnInput `json:"columns,omitempty" jsonschema:"View columns with source references"`
}

type SetViewPositionInput struct {
	Schema string  `json:"schema,omitempty" jsonschema:"Schema name (defaults to 'public' if omitted)"`
	Name   string  `json:"name" jsonschema:"View name"`
	X      float64 `json:"x" jsonschema:"X coordinate"`
	Y      float64 `json:"y" jsonschema:"Y coordinate"`
}

type FilePathInput struct {
	Path string `json:"path" jsonschema:"File path"`
}

// toAPIColumns converts MCP ViewColumnInput (snake_case) to API ViewColumn (camelCase)
func toAPIColumns(cols []ViewColumnInput) []map[string]string {
	out := make([]map[string]string, len(cols))
	for i, c := range cols {
		out[i] = map[string]string{
			"name":         c.Name,
			"type":         c.Type,
			"sourceSchema": c.SourceSchema,
			"sourceTable":  c.SourceTable,
			"sourceColumn": c.SourceColumn,
		}
	}
	return out
}

func registerTools(s *server.MCPServer) {
	// create_schema
	s.AddTool(
		mcp.NewTool("create_schema",
			mcp.WithDescription("Create a new database schema with a display color"),
			mcp.WithInputSchema[CreateSchemaInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args CreateSchemaInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/create-schema", map[string]string{
				"name":  args.Name,
				"color": args.Color,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// create_table
	s.AddTool(
		mcp.NewTool("create_table",
			mcp.WithDescription("Create a new table in a schema. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[CreateTableInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args CreateTableInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/create-table", map[string]string{
				"schema": schema,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_column
	s.AddTool(
		mcp.NewTool("add_column",
			mcp.WithDescription("Add a column to a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[AddColumnInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args AddColumnInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			nullable := false
			if args.Nullable != nil {
				nullable = *args.Nullable
			}
			primaryKey := false
			if args.PrimaryKey != nil {
				primaryKey = *args.PrimaryKey
			}
			unique := false
			if args.Unique != nil {
				unique = *args.Unique
			}
			position := -1
			if args.Position != nil {
				position = *args.Position
			}
			msg, err := apiPost("/api/add-column", map[string]interface{}{
				"schema":      schema,
				"table":       args.Table,
				"name":        args.Name,
				"type":        args.Type,
				"nullable":    nullable,
				"primary_key": primaryKey,
				"unique":      unique,
				"default":     args.Default,
				"comment":     args.Comment,
				"position":    position,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_column
	s.AddTool(
		mcp.NewTool("update_column",
			mcp.WithDescription("Update properties of an existing column. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateColumnInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateColumnInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			updates := make(map[string]interface{})
			if args.Name != "" {
				updates["name"] = args.Name
			}
			if args.Type != "" {
				updates["type"] = args.Type
			}
			if args.Nullable != nil {
				updates["nullable"] = *args.Nullable
			}
			if args.PrimaryKey != nil {
				updates["primary_key"] = *args.PrimaryKey
			}
			if args.Unique != nil {
				updates["unique"] = *args.Unique
			}
			if args.Default != "" {
				updates["default"] = args.Default
			}
			if args.Comment != "" {
				updates["comment"] = args.Comment
			}
			msg, err := apiPost("/api/update-column", map[string]interface{}{
				"schema":  schema,
				"table":   args.Table,
				"column":  args.Column,
				"updates": updates,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// reorder_columns
	s.AddTool(
		mcp.NewTool("reorder_columns",
			mcp.WithDescription("Reorder columns in a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[ReorderColumnsInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args ReorderColumnsInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/reorder-columns", map[string]interface{}{
				"schema":       schema,
				"table":        args.Table,
				"column_order": args.ColumnOrder,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_foreign_key
	s.AddTool(
		mcp.NewTool("add_foreign_key",
			mcp.WithDescription("Add a foreign key relationship between tables. Schema fields default to 'public' if omitted."),
			mcp.WithInputSchema[AddForeignKeyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args AddForeignKeyInput) (*mcp.CallToolResult, error) {
			fromSchema := defaultSchema(args.FromSchema)
			toSchema := defaultSchema(args.ToSchema)
			msg, err := apiPost("/api/add-foreign-key", map[string]string{
				"from_schema": fromSchema,
				"from_table":  args.FromTable,
				"from_column": args.FromColumn,
				"to_schema":   toSchema,
				"to_table":    args.ToTable,
				"to_column":   args.ToColumn,
				"on_delete":   args.OnDelete,
				"on_update":   args.OnUpdate,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// rename_table
	s.AddTool(
		mcp.NewTool("rename_table",
			mcp.WithDescription("Rename a table and update all foreign key references. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[RenameTableInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args RenameTableInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/rename-table", map[string]string{
				"schema":   schema,
				"old_name": args.OldName,
				"new_name": args.NewName,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_schema
	s.AddTool(
		mcp.NewTool("delete_schema",
			mcp.WithDescription("Delete a schema and all its tables, foreign keys, and enum types"),
			mcp.WithInputSchema[NameOnlyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args NameOnlyInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/delete-schema", map[string]string{
				"name": args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_table
	s.AddTool(
		mcp.NewTool("delete_table",
			mcp.WithDescription("Delete a table and its foreign keys. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaNameInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaNameInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-table", map[string]string{
				"schema": schema,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_column
	s.AddTool(
		mcp.NewTool("delete_column",
			mcp.WithDescription("Delete a column from a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaTableColumnInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaTableColumnInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-column", map[string]string{
				"schema": schema,
				"table":  args.Table,
				"column": args.Column,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_foreign_key
	s.AddTool(
		mcp.NewTool("delete_foreign_key",
			mcp.WithDescription("Delete a foreign key by source column. from_schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[DeleteForeignKeyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args DeleteForeignKeyInput) (*mcp.CallToolResult, error) {
			fromSchema := defaultSchema(args.FromSchema)
			msg, err := apiPost("/api/delete-foreign-key", map[string]string{
				"from_schema": fromSchema,
				"from_table":  args.FromTable,
				"from_column": args.FromColumn,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_constraint
	s.AddTool(
		mcp.NewTool("add_constraint",
			mcp.WithDescription("Add a composite table-level constraint (primary key or unique). Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[ConstraintInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args ConstraintInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/add-constraint", map[string]interface{}{
				"schema":  schema,
				"table":   args.Table,
				"type":    args.Type,
				"columns": args.Columns,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_constraint
	s.AddTool(
		mcp.NewTool("update_constraint",
			mcp.WithDescription("Update columns of an existing table-level constraint. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateConstraintInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateConstraintInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/update-constraint", map[string]interface{}{
				"schema":      schema,
				"table":       args.Table,
				"type":        args.Type,
				"old_columns": args.OldColumns,
				"new_columns": args.NewColumns,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_constraint
	s.AddTool(
		mcp.NewTool("delete_constraint",
			mcp.WithDescription("Delete a table-level constraint. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[ConstraintInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args ConstraintInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-constraint", map[string]interface{}{
				"schema":  schema,
				"table":   args.Table,
				"type":    args.Type,
				"columns": args.Columns,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_index
	s.AddTool(
		mcp.NewTool("add_index",
			mcp.WithDescription("Add an index to a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[AddIndexInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args AddIndexInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			unique := false
			if args.Unique != nil {
				unique = *args.Unique
			}
			msg, err := apiPost("/api/add-index", map[string]interface{}{
				"schema":  schema,
				"table":   args.Table,
				"name":    args.Name,
				"columns": args.Columns,
				"unique":  unique,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_index
	s.AddTool(
		mcp.NewTool("update_index",
			mcp.WithDescription("Update an existing index. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateIndexInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateIndexInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			updates := make(map[string]interface{})
			if len(args.Columns) > 0 {
				updates["columns"] = args.Columns
			}
			if args.Unique != nil {
				updates["unique"] = *args.Unique
			}
			if args.NewName != "" {
				updates["name"] = args.NewName
			}
			msg, err := apiPost("/api/update-index", map[string]interface{}{
				"schema":  schema,
				"table":   args.Table,
				"name":    args.Name,
				"updates": updates,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_index
	s.AddTool(
		mcp.NewTool("delete_index",
			mcp.WithDescription("Delete an index from a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaTableNameInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaTableNameInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-index", map[string]string{
				"schema": schema,
				"table":  args.Table,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_foreign_key
	s.AddTool(
		mcp.NewTool("update_foreign_key",
			mcp.WithDescription("Update FK actions on an existing foreign key. from_schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateForeignKeyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateForeignKeyInput) (*mcp.CallToolResult, error) {
			fromSchema := defaultSchema(args.FromSchema)
			msg, err := apiPost("/api/update-foreign-key", map[string]string{
				"from_schema": fromSchema,
				"from_table":  args.FromTable,
				"from_column": args.FromColumn,
				"on_delete":   args.OnDelete,
				"on_update":   args.OnUpdate,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// create_enum
	s.AddTool(
		mcp.NewTool("create_enum",
			mcp.WithDescription("Create a new enum type. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[EnumInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args EnumInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/create-enum", map[string]interface{}{
				"schema": schema,
				"name":   args.Name,
				"values": args.Values,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_enum
	s.AddTool(
		mcp.NewTool("update_enum",
			mcp.WithDescription("Update the values of an existing enum type. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[EnumInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args EnumInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/update-enum", map[string]interface{}{
				"schema": schema,
				"name":   args.Name,
				"values": args.Values,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_enum
	s.AddTool(
		mcp.NewTool("delete_enum",
			mcp.WithDescription("Delete an enum type. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[DeleteEnumInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args DeleteEnumInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-enum", map[string]string{
				"schema": schema,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_check
	s.AddTool(
		mcp.NewTool("add_check",
			mcp.WithDescription("Add a check constraint to a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[AddCheckInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args AddCheckInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/add-check", map[string]string{
				"schema":     schema,
				"table":      args.Table,
				"name":       args.Name,
				"expression": args.Expression,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_check
	s.AddTool(
		mcp.NewTool("delete_check",
			mcp.WithDescription("Delete a check constraint from a table. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[DeleteCheckInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args DeleteCheckInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-check", map[string]string{
				"schema": schema,
				"table":  args.Table,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// set_table_position
	s.AddTool(
		mcp.NewTool("set_table_position",
			mcp.WithDescription("Set the manual x,y position of a table on the canvas. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SetTablePositionInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SetTablePositionInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/set-table-position", map[string]interface{}{
				"schema": schema,
				"table":  args.Table,
				"x":      args.X,
				"y":      args.Y,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// clear_table_position
	s.AddTool(
		mcp.NewTool("clear_table_position",
			mcp.WithDescription("Clear the manual position of a table, reverting to auto-layout. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaTableInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaTableInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/clear-table-position", map[string]string{
				"schema": schema,
				"table":  args.Table,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// undo
	s.AddTool(
		mcp.NewTool("undo",
			mcp.WithDescription("Undo the last mutation"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/undo", map[string]string{})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		},
	)

	// redo
	s.AddTool(
		mcp.NewTool("redo",
			mcp.WithDescription("Redo the last undone mutation"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/redo", map[string]string{})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		},
	)

	// clear
	s.AddTool(
		mcp.NewTool("clear",
			mcp.WithDescription("Reset all state — removes all schemas, tables, and foreign keys"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/clear", map[string]string{})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		},
	)

	// export_sql
	s.AddTool(
		mcp.NewTool("export_sql",
			mcp.WithDescription("Generate PostgreSQL DDL SQL and write to a file. Supports both relative and absolute paths. Relative paths resolve from the agent's working directory."),
			mcp.WithInputSchema[FilePathInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args FilePathInput) (*mcp.CallToolResult, error) {
			sql, err := apiGet("/api/export-sql")
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			filePath := args.Path
			if !filepath.IsAbs(filePath) {
				cwd, err := os.Getwd()
				if err != nil {
					return mcpErr(fmt.Sprintf("cannot determine working directory: %v", err)), nil
				}
				filePath = filepath.Join(cwd, filePath)
			}
			if info, err := os.Stat(filePath); err == nil && info.IsDir() {
				return mcpErr(fmt.Sprintf("path %q is a directory, not a file — please include a filename (e.g. %s)", filePath, filepath.Join(filePath, "schema.sql"))), nil
			}
			if dir := filepath.Dir(filePath); dir != "." {
				os.MkdirAll(dir, 0755)
			}
			if err := os.WriteFile(filePath, []byte(sql), 0644); err != nil {
				return mcpErr(fmt.Sprintf("failed to write file: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("SQL exported to %s", filePath)), nil
		}),
	)

	// get_schema
	s.AddTool(
		mcp.NewTool("get_schema",
			mcp.WithDescription("Get a compact overview of the current schema: all tables, their columns (with types and constraints), and foreign keys"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			body, err := apiGet("/api/state")
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			// Parse and re-format as compact text
			var state struct {
				Schemas []struct{ Name, Color string } `json:"schemas"`
				Tables  []struct {
					Schema  string `json:"schema"`
					Name    string `json:"name"`
					Comment string `json:"comment"`
					Columns []struct {
						Name       string `json:"name"`
						Type       string `json:"type"`
						Nullable   bool   `json:"nullable"`
						PrimaryKey bool   `json:"primaryKey"`
						Unique     bool   `json:"unique"`
						Default    string `json:"default"`
						Comment    string `json:"comment"`
					} `json:"columns"`
					Constraints []struct {
						Type       string   `json:"type"`
						Columns    []string `json:"columns"`
						Name       string   `json:"name"`
						Expression string   `json:"expression"`
					} `json:"constraints"`
					Indexes []struct {
						Name    string   `json:"name"`
						Columns []string `json:"columns"`
						Unique  bool     `json:"unique"`
					} `json:"indexes"`
				} `json:"tables"`
				ForeignKeys []struct {
					FromSchema string `json:"fromSchema"`
					FromTable  string `json:"fromTable"`
					FromColumn string `json:"fromColumn"`
					ToSchema   string `json:"toSchema"`
					ToTable    string `json:"toTable"`
					ToColumn   string `json:"toColumn"`
					OnDelete   string `json:"onDelete"`
					OnUpdate   string `json:"onUpdate"`
				} `json:"foreignKeys"`
				EnumTypes []struct {
					Schema string   `json:"schema"`
					Name   string   `json:"name"`
					Values []string `json:"values"`
				} `json:"enumTypes"`
				Extensions []struct {
					Name string `json:"name"`
				} `json:"extensions"`
				Views []struct {
					Schema  string `json:"schema"`
					Name    string `json:"name"`
					Comment string `json:"comment"`
					Columns []struct {
						Name         string `json:"name"`
						Type         string `json:"type"`
						SourceSchema string `json:"sourceSchema"`
						SourceTable  string `json:"sourceTable"`
						SourceColumn string `json:"sourceColumn"`
					} `json:"columns"`
				} `json:"views"`
			}
			if err := json.Unmarshal([]byte(body), &state); err != nil {
				return mcp.NewToolResultText(body), nil
			}

			var sb strings.Builder

			// Show extensions at top
			if len(state.Extensions) > 0 {
				sb.WriteString("[extensions]\n")
				for _, ext := range state.Extensions {
					sb.WriteString(fmt.Sprintf("  %s\n", ext.Name))
				}
				sb.WriteString("\n")
			}

			// Show enums
			if len(state.EnumTypes) > 0 {
				sb.WriteString("[enum_types]\n")
				for _, e := range state.EnumTypes {
					sb.WriteString(fmt.Sprintf("  %s.%s: %s\n", e.Schema, e.Name, strings.Join(e.Values, ", ")))
				}
				sb.WriteString("\n")
			}

			for _, t := range state.Tables {
				if t.Comment != "" {
					sb.WriteString(fmt.Sprintf("[%s.%s] -- %s\n", t.Schema, t.Name, t.Comment))
				} else {
					sb.WriteString(fmt.Sprintf("[%s.%s]\n", t.Schema, t.Name))
				}
				for _, c := range t.Columns {
					flags := ""
					if c.PrimaryKey {
						flags += " PK"
					}
					if c.Unique && !c.PrimaryKey {
						flags += " UQ"
					}
					if c.Nullable {
						flags += " NULL"
					}
					if c.Default != "" {
						flags += " DEFAULT=" + c.Default
					}
					if c.Comment != "" {
						flags += " COMMENT=" + c.Comment
					}
					sb.WriteString(fmt.Sprintf("  %-20s %s%s\n", c.Name, c.Type, flags))
				}
				for _, tc := range t.Constraints {
					if tc.Type == "check" {
						sb.WriteString(fmt.Sprintf("  CHECK %s: %s\n", tc.Name, tc.Expression))
					} else {
						sb.WriteString(fmt.Sprintf("  CONSTRAINT %s (%s)\n", tc.Type, strings.Join(tc.Columns, ", ")))
					}
				}
				for _, idx := range t.Indexes {
					idxType := "INDEX"
					if idx.Unique {
						idxType = "UNIQUE INDEX"
					}
					sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", idxType, idx.Name, strings.Join(idx.Columns, ", ")))
				}
				sb.WriteString("\n")
			}
			if len(state.ForeignKeys) > 0 {
				sb.WriteString("[foreign_keys]\n")
				for _, fk := range state.ForeignKeys {
					line := fmt.Sprintf("  %s.%s.%s -> %s.%s.%s",
						fk.FromSchema, fk.FromTable, fk.FromColumn,
						fk.ToSchema, fk.ToTable, fk.ToColumn)
					if fk.OnDelete != "" {
						line += " ON DELETE " + fk.OnDelete
					}
					if fk.OnUpdate != "" {
						line += " ON UPDATE " + fk.OnUpdate
					}
					sb.WriteString(line + "\n")
				}
				sb.WriteString("\n")
			}

			// Show views at bottom
			if len(state.Views) > 0 {
				sb.WriteString("[views]\n")
				for _, v := range state.Views {
					header := fmt.Sprintf("  %s.%s", v.Schema, v.Name)
					if v.Comment != "" {
						header += " -- " + v.Comment
					}
					sb.WriteString(header + "\n")
					for _, c := range v.Columns {
						line := fmt.Sprintf("    %-20s %s", c.Name, c.Type)
						if c.SourceSchema != "" && c.SourceTable != "" && c.SourceColumn != "" {
							line += fmt.Sprintf("  <- %s.%s.%s", c.SourceSchema, c.SourceTable, c.SourceColumn)
						}
						sb.WriteString(line + "\n")
					}
				}
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// update_table
	s.AddTool(
		mcp.NewTool("update_table",
			mcp.WithDescription("Update properties of an existing table (comment, rename). Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateTableInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateTableInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			updates := make(map[string]interface{})
			if args.Comment != "" {
				updates["comment"] = args.Comment
			}
			if args.NewName != "" {
				updates["new_name"] = args.NewName
			}
			msg, err := apiPost("/api/update-table", map[string]interface{}{
				"schema":  schema,
				"name":    args.Name,
				"updates": updates,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_schema
	s.AddTool(
		mcp.NewTool("update_schema",
			mcp.WithDescription("Update properties of an existing schema (rename, change color)."),
			mcp.WithInputSchema[UpdateSchemaInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateSchemaInput) (*mcp.CallToolResult, error) {
			updates := make(map[string]interface{})
			if args.NewName != "" {
				updates["new_name"] = args.NewName
			}
			if args.Color != "" {
				updates["color"] = args.Color
			}
			msg, err := apiPost("/api/update-schema", map[string]interface{}{
				"name":    args.Name,
				"updates": updates,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// add_extension
	s.AddTool(
		mcp.NewTool("add_extension",
			mcp.WithDescription("Add a PostgreSQL extension to the schema"),
			mcp.WithInputSchema[NameOnlyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args NameOnlyInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/add-extension", map[string]string{
				"name": args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_extension
	s.AddTool(
		mcp.NewTool("delete_extension",
			mcp.WithDescription("Remove a PostgreSQL extension from the schema"),
			mcp.WithInputSchema[NameOnlyInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args NameOnlyInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/delete-extension", map[string]string{
				"name": args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// create_view
	s.AddTool(
		mcp.NewTool("create_view",
			mcp.WithDescription("Create a new database view. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[CreateViewInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args CreateViewInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/create-view", map[string]interface{}{
				"schema":  schema,
				"name":    args.Name,
				"columns": toAPIColumns(args.Columns),
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// update_view
	s.AddTool(
		mcp.NewTool("update_view",
			mcp.WithDescription("Update properties of an existing view. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[UpdateViewInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args UpdateViewInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			updates := make(map[string]interface{})
			if args.Comment != "" {
				updates["comment"] = args.Comment
			}
			if args.NewName != "" {
				updates["new_name"] = args.NewName
			}
			if len(args.Columns) > 0 {
				updates["columns"] = toAPIColumns(args.Columns)
			}
			msg, err := apiPost("/api/update-view", map[string]interface{}{
				"schema":  schema,
				"name":    args.Name,
				"updates": updates,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// delete_view
	s.AddTool(
		mcp.NewTool("delete_view",
			mcp.WithDescription("Delete a database view. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaNameInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaNameInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/delete-view", map[string]string{
				"schema": schema,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// set_view_position
	s.AddTool(
		mcp.NewTool("set_view_position",
			mcp.WithDescription("Set the manual x,y position of a view on the canvas. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SetViewPositionInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SetViewPositionInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/set-view-position", map[string]interface{}{
				"schema": schema,
				"name":   args.Name,
				"x":      args.X,
				"y":      args.Y,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// clear_view_position
	s.AddTool(
		mcp.NewTool("clear_view_position",
			mcp.WithDescription("Clear the manual position of a view, reverting to auto-layout. Schema defaults to 'public' if omitted."),
			mcp.WithInputSchema[SchemaNameInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args SchemaNameInput) (*mcp.CallToolResult, error) {
			schema := defaultSchema(args.Schema)
			msg, err := apiPost("/api/clear-view-position", map[string]string{
				"schema": schema,
				"name":   args.Name,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// save_project
	s.AddTool(
		mcp.NewTool("save_project",
			mcp.WithDescription("Save the current state to a .schemata YAML file"),
			mcp.WithInputSchema[FilePathInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args FilePathInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/save-project", map[string]string{
				"path": args.Path,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)

	// load_project
	s.AddTool(
		mcp.NewTool("load_project",
			mcp.WithDescription("Load state from a .schemata YAML file"),
			mcp.WithInputSchema[FilePathInput](),
		),
		mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args FilePathInput) (*mcp.CallToolResult, error) {
			msg, err := apiPost("/api/load-project", map[string]string{
				"path": args.Path,
			})
			if err != nil {
				return mcpErr(err.Error()), nil
			}
			return mcp.NewToolResultText(msg), nil
		}),
	)
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Column represents a table column.
type Column struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"`
	Nullable   bool   `json:"nullable" yaml:"nullable,omitempty"`
	PrimaryKey bool   `json:"primaryKey" yaml:"primary_key,omitempty"`
	Unique     bool   `json:"unique" yaml:"unique,omitempty"`
	Default    string `json:"default" yaml:"default,omitempty"`
	Comment    string `json:"comment,omitempty" yaml:"comment,omitempty"`
	Generated  string `json:"generated,omitempty" yaml:"generated,omitempty"`
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	FromSchema string `json:"fromSchema" yaml:"from_schema"`
	FromTable  string `json:"fromTable" yaml:"from_table"`
	FromColumn string `json:"fromColumn" yaml:"from_column"`
	ToSchema   string `json:"toSchema" yaml:"to_schema"`
	ToTable    string `json:"toTable" yaml:"to_table"`
	ToColumn   string `json:"toColumn" yaml:"to_column"`
	OnDelete   string `json:"onDelete,omitempty" yaml:"on_delete,omitempty"`
	OnUpdate   string `json:"onUpdate,omitempty" yaml:"on_update,omitempty"`
}

// TableConstraint represents a table-level constraint (composite primary key, unique, or check).
type TableConstraint struct {
	Type       string   `json:"type" yaml:"type"`                          // "primary_key", "unique", or "check"
	Columns    []string `json:"columns" yaml:"columns"`                    // column names involved
	Name       string   `json:"name,omitempty" yaml:"name,omitempty"`      // constraint name (used for check constraints)
	Expression string   `json:"expression,omitempty" yaml:"expression,omitempty"` // check expression
}

// Index represents a table index.
type Index struct {
	Name    string   `json:"name" yaml:"name"`
	Columns []string `json:"columns" yaml:"columns"`
	Unique  bool     `json:"unique" yaml:"unique,omitempty"`
	Type    string   `json:"type,omitempty" yaml:"type,omitempty"`
	Where   string   `json:"where,omitempty" yaml:"where,omitempty"`
}

// Position represents x,y coordinates for manual table positioning on the canvas.
type Position struct {
	X float64 `json:"x" yaml:"x"`
	Y float64 `json:"y" yaml:"y"`
}

// Table represents a database table.
type Table struct {
	Schema      string            `json:"schema" yaml:"schema"`
	Name        string            `json:"name" yaml:"name"`
	Comment     string            `json:"comment,omitempty" yaml:"comment,omitempty"`
	Columns     []Column          `json:"columns" yaml:"columns"`
	Constraints []TableConstraint `json:"constraints" yaml:"constraints,omitempty"`
	Indexes     []Index           `json:"indexes" yaml:"indexes,omitempty"`
	Position    *Position         `json:"position,omitempty" yaml:"position,omitempty"`
}

// Schema represents a database schema with a display color.
type Schema struct {
	Name  string `json:"name" yaml:"name"`
	Color string `json:"color" yaml:"color"`
}

// EnumType represents a user-defined enum type.
type EnumType struct {
	Schema string   `json:"schema" yaml:"schema"`
	Name   string   `json:"name" yaml:"name"`
	Values []string `json:"values" yaml:"values"`
}

// Extension represents a PostgreSQL extension.
type Extension struct {
	Name string `json:"name" yaml:"name"`
}

// ViewColumn represents a column in a database view.
type ViewColumn struct {
	Name         string `json:"name" yaml:"name"`
	Type         string `json:"type" yaml:"type"`
	SourceSchema string `json:"sourceSchema,omitempty" yaml:"source_schema,omitempty"`
	SourceTable  string `json:"sourceTable,omitempty" yaml:"source_table,omitempty"`
	SourceColumn string `json:"sourceColumn,omitempty" yaml:"source_column,omitempty"`
}

// View represents a database view.
type View struct {
	Schema   string       `json:"schema" yaml:"schema"`
	Name     string       `json:"name" yaml:"name"`
	Columns  []ViewColumn `json:"columns" yaml:"columns"`
	Comment  string       `json:"comment,omitempty" yaml:"comment,omitempty"`
	Position *Position    `json:"position,omitempty" yaml:"position,omitempty"`
}

// AppState holds the entire application state.
type AppState struct {
	Schemas     []Schema     `json:"schemas" yaml:"schemas"`
	Tables      []Table      `json:"tables" yaml:"tables"`
	ForeignKeys []ForeignKey `json:"foreignKeys" yaml:"foreign_keys"`
	EnumTypes   []EnumType   `json:"enumTypes" yaml:"enum_types,omitempty"`
	Extensions  []Extension  `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	Views       []View       `json:"views,omitempty" yaml:"views,omitempty"`
}

// App is the main application struct with thread-safe state management.
type App struct {
	ctx         context.Context
	mu          sync.RWMutex
	state       AppState
	currentFile string
	undoStack   []AppState
	redoStack   []AppState
	maxHistory  int
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		maxHistory: 50,
		state: AppState{
			Schemas: []Schema{
				{Name: "public", Color: "#6366f1"},
			},
			Tables:      []Table{},
			ForeignKeys: []ForeignKey{},
			EnumTypes:   []EnumType{},
			Extensions:  []Extension{},
			Views:       []View{},
		},
	}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// cloneState deep-clones the current state using JSON round-trip.
func (a *App) cloneState() AppState {
	data, _ := json.Marshal(a.state)
	var clone AppState
	json.Unmarshal(data, &clone)
	return clone
}

// pushUndo saves the current state to the undo stack (caller must hold lock).
func (a *App) pushUndo() {
	snapshot := a.cloneState()
	a.undoStack = append(a.undoStack, snapshot)
	if len(a.undoStack) > a.maxHistory {
		a.undoStack = a.undoStack[1:]
	}
	a.redoStack = nil
}

// Undo reverts the last mutation.
func (a *App) Undo() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.undoStack) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	a.redoStack = append(a.redoStack, a.cloneState())
	a.state = a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]
	go a.emitState()
	return nil
}

// Redo re-applies the last undone mutation.
func (a *App) Redo() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}
	a.undoStack = append(a.undoStack, a.cloneState())
	a.state = a.redoStack[len(a.redoStack)-1]
	a.redoStack = a.redoStack[:len(a.redoStack)-1]
	go a.emitState()
	return nil
}

// emitState sends the full state to the frontend via Wails events.
func (a *App) emitState() {
	if a.ctx == nil {
		return
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	runtime.EventsEmit(a.ctx, "state_update", a.state)
}

// GetState returns the current state (called from frontend on load).
func (a *App) GetState() AppState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// GetStateJSON returns the current state as a JSON string.
func (a *App) GetStateJSON() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	data, _ := json.Marshal(a.state)
	return string(data)
}

// CreateSchema adds a new schema.
func (a *App) CreateSchema(name, color string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	for _, s := range a.state.Schemas {
		if s.Name == name {
			return errSchemaExists(name)
		}
	}
	a.state.Schemas = append(a.state.Schemas, Schema{Name: name, Color: color})
	go a.emitState()
	return nil
}

// CreateTable adds a new table to a schema.
func (a *App) CreateTable(schema, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	// Auto-create "public" schema if defaulting and it doesn't exist
	if schema == "public" && !a.schemaExists("public") {
		a.state.Schemas = append(a.state.Schemas, Schema{Name: "public", Color: "#6366f1"})
	}
	if !a.schemaExists(schema) {
		return errSchemaNotFound(schema)
	}
	for _, t := range a.state.Tables {
		if t.Schema == schema && t.Name == name {
			return errTableExists(schema, name)
		}
	}
	a.state.Tables = append(a.state.Tables, Table{
		Schema:      schema,
		Name:        name,
		Columns:     []Column{},
		Constraints: []TableConstraint{},
		Indexes:     []Index{},
	})
	go a.emitState()
	return nil
}

// AddColumn adds a column to a table. Position is 0-based; use -1 to append at end.
func (a *App) AddColumn(schema, table, name, colType string, nullable, primaryKey, unique bool, defaultVal, comment string, position int, generated string) error {
	if schema == "" {
		schema = "public"
	}
	if generated != "" && defaultVal != "" {
		return fmt.Errorf("column %q cannot have both a default value and a generated expression", name)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	for _, c := range t.Columns {
		if c.Name == name {
			return errColumnExists(schema, table, name)
		}
	}
	col := Column{
		Name:       name,
		Type:       colType,
		Nullable:   nullable,
		PrimaryKey: primaryKey,
		Unique:     unique,
		Default:    defaultVal,
		Comment:    comment,
		Generated:  generated,
	}
	if position >= 0 && position < len(t.Columns) {
		t.Columns = append(t.Columns[:position], append([]Column{col}, t.Columns[position:]...)...)
	} else {
		t.Columns = append(t.Columns, col)
	}
	go a.emitState()
	return nil
}

// UpdateColumn updates properties of an existing column.
func (a *App) UpdateColumn(schema, table, column string, updates map[string]interface{}) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	idx := -1
	for i, c := range t.Columns {
		if c.Name == column {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errColumnNotFound(schema, table, column)
	}
	c := &t.Columns[idx]
	if v, ok := updates["type"]; ok {
		if s, ok := v.(string); ok {
			c.Type = s
		}
	}
	if v, ok := updates["nullable"]; ok {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("nullable must be a boolean, got %T", v)
		}
		c.Nullable = b
	}
	if v, ok := updates["primary_key"]; ok {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("primary_key must be a boolean, got %T", v)
		}
		c.PrimaryKey = b
	}
	if v, ok := updates["unique"]; ok {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("unique must be a boolean, got %T", v)
		}
		c.Unique = b
	}
	if v, ok := updates["default"]; ok {
		if s, ok := v.(string); ok {
			c.Default = s
			if s != "" {
				c.Generated = ""
			}
		}
	}
	if v, ok := updates["comment"]; ok {
		if s, ok := v.(string); ok {
			c.Comment = s
		}
	}
	if v, ok := updates["generated"]; ok {
		if s, ok := v.(string); ok {
			c.Generated = s
			if s != "" {
				c.Default = ""
			}
		}
	}
	// Handle name change last, including FK reference updates
	if v, ok := updates["name"]; ok {
		if newName, ok := v.(string); ok && newName != "" && newName != c.Name {
			oldName := c.Name
			c.Name = newName
			// Update any foreign key references pointing to/from this column
			for i := range a.state.ForeignKeys {
				fk := &a.state.ForeignKeys[i]
				if fk.FromSchema == schema && fk.FromTable == table && fk.FromColumn == oldName {
					fk.FromColumn = newName
				}
				if fk.ToSchema == schema && fk.ToTable == table && fk.ToColumn == oldName {
					fk.ToColumn = newName
				}
			}
			// Update column name in table constraints
			for ci := range t.Constraints {
				for cj, cn := range t.Constraints[ci].Columns {
					if cn == oldName {
						t.Constraints[ci].Columns[cj] = newName
					}
				}
			}
			// Update column name in indexes
			for ii := range t.Indexes {
				for ij, ic := range t.Indexes[ii].Columns {
					if ic == oldName {
						t.Indexes[ii].Columns[ij] = newName
					}
				}
			}
		}
	}
	go a.emitState()
	return nil
}

// ReorderColumns reorders the columns of a table to match the given order.
func (a *App) ReorderColumns(schema, table string, columnOrder []string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	if len(columnOrder) != len(t.Columns) {
		return fmt.Errorf("column_order length (%d) does not match table column count (%d)", len(columnOrder), len(t.Columns))
	}
	// Build lookup and check for duplicates
	colMap := make(map[string]Column, len(t.Columns))
	for _, c := range t.Columns {
		colMap[c.Name] = c
	}
	seen := make(map[string]bool, len(columnOrder))
	reordered := make([]Column, 0, len(columnOrder))
	for _, name := range columnOrder {
		if seen[name] {
			return fmt.Errorf("duplicate column name %q in column_order", name)
		}
		seen[name] = true
		c, ok := colMap[name]
		if !ok {
			return errColumnNotFound(schema, table, name)
		}
		reordered = append(reordered, c)
	}
	t.Columns = reordered
	// Reorder columns within table constraints to match new column order
	colIndex := make(map[string]int, len(columnOrder))
	for i, name := range columnOrder {
		colIndex[name] = i
	}
	for ci := range t.Constraints {
		sort.Slice(t.Constraints[ci].Columns, func(i, j int) bool {
			return colIndex[t.Constraints[ci].Columns[i]] < colIndex[t.Constraints[ci].Columns[j]]
		})
	}
	go a.emitState()
	return nil
}

// validFKAction checks if the given string is a valid foreign key action.
func validFKAction(action string) bool {
	switch strings.ToUpper(action) {
	case "", "CASCADE", "SET NULL", "SET DEFAULT", "RESTRICT", "NO ACTION":
		return true
	}
	return false
}

// AddForeignKey adds a foreign key relationship.
func (a *App) AddForeignKey(fromSchema, fromTable, fromColumn, toSchema, toTable, toColumn, onDelete, onUpdate string) error {
	if fromSchema == "" {
		fromSchema = "public"
	}
	if toSchema == "" {
		toSchema = "public"
	}
	if !validFKAction(onDelete) {
		return errInvalidFKAction(onDelete)
	}
	if !validFKAction(onUpdate) {
		return errInvalidFKAction(onUpdate)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	// Validate source
	ft := a.findTable(fromSchema, fromTable)
	if ft == nil {
		return errTableNotFound(fromSchema, fromTable)
	}
	if !a.columnExists(ft, fromColumn) {
		return errColumnNotFound(fromSchema, fromTable, fromColumn)
	}
	// Validate target
	tt := a.findTable(toSchema, toTable)
	if tt == nil {
		return errTableNotFound(toSchema, toTable)
	}
	if !a.columnExists(tt, toColumn) {
		return errColumnNotFound(toSchema, toTable, toColumn)
	}
	// Check duplicate
	for _, fk := range a.state.ForeignKeys {
		if fk.FromSchema == fromSchema && fk.FromTable == fromTable && fk.FromColumn == fromColumn {
			return errForeignKeyExists(fromSchema, fromTable, fromColumn)
		}
	}
	a.state.ForeignKeys = append(a.state.ForeignKeys, ForeignKey{
		FromSchema: fromSchema,
		FromTable:  fromTable,
		FromColumn: fromColumn,
		ToSchema:   toSchema,
		ToTable:    toTable,
		ToColumn:   toColumn,
		OnDelete:   strings.ToUpper(onDelete),
		OnUpdate:   strings.ToUpper(onUpdate),
	})
	go a.emitState()
	return nil
}

// RenameTable renames a table and updates all foreign key references.
func (a *App) RenameTable(schema, oldName, newName string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, oldName)
	if t == nil {
		return errTableNotFound(schema, oldName)
	}
	// Check new name doesn't conflict
	if a.findTable(schema, newName) != nil {
		return errTableExists(schema, newName)
	}
	t.Name = newName
	// Update all foreign key references
	for i := range a.state.ForeignKeys {
		fk := &a.state.ForeignKeys[i]
		if fk.FromSchema == schema && fk.FromTable == oldName {
			fk.FromTable = newName
		}
		if fk.ToSchema == schema && fk.ToTable == oldName {
			fk.ToTable = newName
		}
	}
	go a.emitState()
	return nil
}

// DeleteSchema removes a schema and all its tables, foreign keys, and enum types.
func (a *App) DeleteSchema(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	if !a.schemaExists(name) {
		return errSchemaNotFound(name)
	}
	// Remove tables belonging to this schema
	tables := make([]Table, 0, len(a.state.Tables))
	for _, t := range a.state.Tables {
		if t.Schema != name {
			tables = append(tables, t)
		}
	}
	a.state.Tables = tables
	// Remove foreign keys referencing this schema
	fks := make([]ForeignKey, 0, len(a.state.ForeignKeys))
	for _, fk := range a.state.ForeignKeys {
		if fk.FromSchema != name && fk.ToSchema != name {
			fks = append(fks, fk)
		}
	}
	a.state.ForeignKeys = fks
	// Remove enum types belonging to this schema
	enums := make([]EnumType, 0, len(a.state.EnumTypes))
	for _, e := range a.state.EnumTypes {
		if e.Schema != name {
			enums = append(enums, e)
		}
	}
	a.state.EnumTypes = enums
	// Remove views belonging to this schema
	views := make([]View, 0, len(a.state.Views))
	for _, v := range a.state.Views {
		if v.Schema != name {
			views = append(views, v)
		}
	}
	a.state.Views = views
	// Remove the schema itself
	schemas := make([]Schema, 0, len(a.state.Schemas))
	for _, s := range a.state.Schemas {
		if s.Name != name {
			schemas = append(schemas, s)
		}
	}
	a.state.Schemas = schemas
	go a.emitState()
	return nil
}

// DeleteTable removes a table and its foreign keys.
func (a *App) DeleteTable(schema, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	idx := -1
	for i, t := range a.state.Tables {
		if t.Schema == schema && t.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errTableNotFound(schema, name)
	}
	a.state.Tables = append(a.state.Tables[:idx], a.state.Tables[idx+1:]...)
	// Remove related foreign keys
	fks := make([]ForeignKey, 0)
	for _, fk := range a.state.ForeignKeys {
		if (fk.FromSchema == schema && fk.FromTable == name) ||
			(fk.ToSchema == schema && fk.ToTable == name) {
			continue
		}
		fks = append(fks, fk)
	}
	a.state.ForeignKeys = fks
	go a.emitState()
	return nil
}

// DeleteColumn removes a column from a table and its foreign keys.
func (a *App) DeleteColumn(schema, table, column string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	idx := -1
	for i, c := range t.Columns {
		if c.Name == column {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errColumnNotFound(schema, table, column)
	}
	t.Columns = append(t.Columns[:idx], t.Columns[idx+1:]...)
	// Remove column from table constraints; drop constraint if fewer than 2 columns remain
	kept := make([]TableConstraint, 0, len(t.Constraints))
	for _, tc := range t.Constraints {
		// Check constraints have no column refs to clean
		if tc.Type == "check" {
			kept = append(kept, tc)
			continue
		}
		var cols []string
		for _, c := range tc.Columns {
			if c != column {
				cols = append(cols, c)
			}
		}
		if len(cols) >= 2 {
			kept = append(kept, TableConstraint{Type: tc.Type, Columns: cols, Name: tc.Name, Expression: tc.Expression})
		}
	}
	t.Constraints = kept
	// Remove column from indexes; drop indexes that become empty
	keptIdx := make([]Index, 0, len(t.Indexes))
	for _, idx := range t.Indexes {
		var cols []string
		for _, c := range idx.Columns {
			if c != column {
				cols = append(cols, c)
			}
		}
		if len(cols) > 0 {
			keptIdx = append(keptIdx, Index{Name: idx.Name, Columns: cols, Unique: idx.Unique})
		}
	}
	t.Indexes = keptIdx
	// Remove related foreign keys
	fks := make([]ForeignKey, 0)
	for _, fk := range a.state.ForeignKeys {
		if (fk.FromSchema == schema && fk.FromTable == table && fk.FromColumn == column) ||
			(fk.ToSchema == schema && fk.ToTable == table && fk.ToColumn == column) {
			continue
		}
		fks = append(fks, fk)
	}
	a.state.ForeignKeys = fks
	go a.emitState()
	return nil
}

// DeleteForeignKey removes a foreign key by source column.
func (a *App) DeleteForeignKey(fromSchema, fromTable, fromColumn string) error {
	if fromSchema == "" {
		fromSchema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	idx := -1
	for i, fk := range a.state.ForeignKeys {
		if fk.FromSchema == fromSchema && fk.FromTable == fromTable && fk.FromColumn == fromColumn {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errForeignKeyNotFound(fromSchema, fromTable, fromColumn)
	}
	a.state.ForeignKeys = append(a.state.ForeignKeys[:idx], a.state.ForeignKeys[idx+1:]...)
	go a.emitState()
	return nil
}

// Clear resets all state.
func (a *App) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	a.state = AppState{
		Schemas:     []Schema{},
		Tables:      []Table{},
		ForeignKeys: []ForeignKey{},
		EnumTypes:   []EnumType{},
		Extensions:  []Extension{},
		Views:       []View{},
	}
	go a.emitState()
}

// SaveSQL opens a save dialog and writes the SQL export to the chosen file.
func (a *App) SaveSQL() (string, error) {
	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Export SQL",
		DefaultFilename:      "schema.sql",
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "SQL Files", Pattern: "*.sql"},
		},
	})
	if err != nil {
		return "", err
	}
	if filepath == "" {
		return "", nil
	}
	sql := a.ExportSQL()
	if err := writeFile(filepath, sql); err != nil {
		return "", err
	}
	return filepath, nil
}

// ExportSQL generates PostgreSQL DDL from the current state.
func (a *App) ExportSQL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return generateSQL(&a.state)
}

// SaveProject saves the current state as YAML to the given file path.
func (a *App) SaveProject(filePath string) error {
	a.mu.RLock()
	data, err := marshalYAML(&a.state)
	a.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}
	a.mu.Lock()
	a.currentFile = filePath
	a.mu.Unlock()
	return nil
}

// LoadProject loads state from a YAML file.
func (a *App) LoadProject(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	state, err := unmarshalYAML(data)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.pushUndo()
	a.state = *state
	a.currentFile = filePath
	a.mu.Unlock()
	go a.emitState()
	return nil
}

// SaveProjectDialog opens a save dialog for .schemata files, then saves.
func (a *App) SaveProjectDialog() (string, error) {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save Project",
		DefaultFilename:      "project.schemata",
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "Schemata Files", Pattern: "*.schemata"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil
	}
	if err := a.SaveProject(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

// OpenProjectDialog opens an open dialog for .schemata files, then loads.
func (a *App) OpenProjectDialog() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Project",
		Filters: []runtime.FileFilter{
			{DisplayName: "Schemata Files", Pattern: "*.schemata"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil
	}
	if err := a.LoadProject(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

// GetCurrentFile returns the path of the currently open project file.
func (a *App) GetCurrentFile() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentFile
}

// AddConstraint adds a table-level constraint (composite primary key or unique).
func (a *App) AddConstraint(schema, table, constraintType string, columns []string) error {
	if schema == "" {
		schema = "public"
	}
	if constraintType != "primary_key" && constraintType != "unique" && constraintType != "check" {
		return fmt.Errorf("invalid constraint type %q: must be \"primary_key\", \"unique\", or \"check\"", constraintType)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	// Validate all columns exist
	for _, col := range columns {
		if !a.columnExists(t, col) {
			return errColumnNotFound(schema, table, col)
		}
	}
	// For primary_key: check no existing table-level PK
	if constraintType == "primary_key" {
		for _, tc := range t.Constraints {
			if tc.Type == "primary_key" {
				return fmt.Errorf("table %q.%q already has a table-level primary key constraint", schema, table)
			}
		}
	}
	// Check for duplicate constraint (same type and columns)
	for _, tc := range t.Constraints {
		if tc.Type == constraintType && sameColumns(tc.Columns, columns) {
			return fmt.Errorf("constraint %s(%s) already exists on %q.%q", constraintType, strings.Join(columns, ", "), schema, table)
		}
	}
	t.Constraints = append(t.Constraints, TableConstraint{Type: constraintType, Columns: columns})
	go a.emitState()
	return nil
}

// UpdateConstraint updates the columns of an existing table-level constraint.
func (a *App) UpdateConstraint(schema, table, constraintType string, oldColumns, newColumns []string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	// Validate all new columns exist
	for _, col := range newColumns {
		if !a.columnExists(t, col) {
			return errColumnNotFound(schema, table, col)
		}
	}
	// Find the constraint
	found := false
	for i := range t.Constraints {
		if t.Constraints[i].Type == constraintType && sameColumns(t.Constraints[i].Columns, oldColumns) {
			t.Constraints[i].Columns = newColumns
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("constraint %s(%s) not found on %q.%q", constraintType, strings.Join(oldColumns, ", "), schema, table)
	}
	go a.emitState()
	return nil
}

// DeleteConstraint removes a table-level constraint matching type and columns.
func (a *App) DeleteConstraint(schema, table, constraintType string, columns []string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	idx := -1
	for i := range t.Constraints {
		if t.Constraints[i].Type == constraintType && sameColumns(t.Constraints[i].Columns, columns) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("constraint %s(%s) not found on %q.%q", constraintType, strings.Join(columns, ", "), schema, table)
	}
	t.Constraints = append(t.Constraints[:idx], t.Constraints[idx+1:]...)
	go a.emitState()
	return nil
}

// sameColumns checks if two string slices contain the same elements regardless of order.
func sameColumns(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

// validIndexType checks if the given index type is valid.
func validIndexType(t string) bool {
	switch t {
	case "", "btree", "hash", "gin", "gist", "brin":
		return true
	}
	return false
}

// AddIndex adds an index to a table.
func (a *App) AddIndex(schema, table, name string, columns []string, unique bool, indexType string, where string) error {
	if schema == "" {
		schema = "public"
	}
	if !validIndexType(indexType) {
		return fmt.Errorf("invalid index type %q: must be one of btree, hash, gin, gist, brin", indexType)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	// Validate all columns exist
	for _, col := range columns {
		if !a.columnExists(t, col) {
			return errColumnNotFound(schema, table, col)
		}
	}
	// Auto-generate name if empty
	if name == "" {
		name = "idx_" + table + "_" + strings.Join(columns, "_")
	}
	// Check name uniqueness
	for _, idx := range t.Indexes {
		if idx.Name == name {
			return errIndexExists(schema, table, name)
		}
	}
	t.Indexes = append(t.Indexes, Index{Name: name, Columns: columns, Unique: unique, Type: indexType, Where: where})
	go a.emitState()
	return nil
}

// UpdateIndex updates properties of an existing index.
func (a *App) UpdateIndex(schema, table, name string, updates map[string]interface{}) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	idx := -1
	for i, ix := range t.Indexes {
		if ix.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errIndexNotFound(schema, table, name)
	}
	ix := &t.Indexes[idx]
	if v, ok := updates["columns"]; ok {
		if arr, ok := v.([]interface{}); ok {
			cols := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					cols = append(cols, s)
				}
			}
			ix.Columns = cols
		} else if arr, ok := v.([]string); ok {
			ix.Columns = arr
		}
	}
	if v, ok := updates["unique"]; ok {
		if b, ok := v.(bool); ok {
			ix.Unique = b
		}
	}
	if v, ok := updates["name"]; ok {
		if s, ok := v.(string); ok && s != "" {
			ix.Name = s
		}
	}
	if v, ok := updates["type"]; ok {
		if s, ok := v.(string); ok {
			if !validIndexType(s) {
				return fmt.Errorf("invalid index type %q: must be one of btree, hash, gin, gist, brin", s)
			}
			ix.Type = s
		}
	}
	if v, ok := updates["where"]; ok {
		if s, ok := v.(string); ok {
			ix.Where = s
		}
	}
	go a.emitState()
	return nil
}

// DeleteIndex removes an index from a table.
func (a *App) DeleteIndex(schema, table, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	idx := -1
	for i, ix := range t.Indexes {
		if ix.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errIndexNotFound(schema, table, name)
	}
	t.Indexes = append(t.Indexes[:idx], t.Indexes[idx+1:]...)
	go a.emitState()
	return nil
}

// UpdateForeignKey updates properties of an existing foreign key.
func (a *App) UpdateForeignKey(fromSchema, fromTable, fromColumn string, updates map[string]interface{}) error {
	if fromSchema == "" {
		fromSchema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	idx := -1
	for i, fk := range a.state.ForeignKeys {
		if fk.FromSchema == fromSchema && fk.FromTable == fromTable && fk.FromColumn == fromColumn {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errForeignKeyNotFound(fromSchema, fromTable, fromColumn)
	}
	fk := &a.state.ForeignKeys[idx]
	if v, ok := updates["on_delete"]; ok {
		if s, ok := v.(string); ok {
			if !validFKAction(s) {
				return errInvalidFKAction(s)
			}
			fk.OnDelete = strings.ToUpper(s)
		}
	}
	if v, ok := updates["on_update"]; ok {
		if s, ok := v.(string); ok {
			if !validFKAction(s) {
				return errInvalidFKAction(s)
			}
			fk.OnUpdate = strings.ToUpper(s)
		}
	}
	go a.emitState()
	return nil
}

// CreateEnum adds a new enum type.
func (a *App) CreateEnum(schema, name string, values []string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	if !a.schemaExists(schema) {
		return errSchemaNotFound(schema)
	}
	for _, e := range a.state.EnumTypes {
		if e.Schema == schema && e.Name == name {
			return errEnumExists(schema, name)
		}
	}
	if values == nil {
		values = []string{}
	}
	a.state.EnumTypes = append(a.state.EnumTypes, EnumType{Schema: schema, Name: name, Values: values})
	go a.emitState()
	return nil
}

// UpdateEnum replaces the values of an existing enum type.
func (a *App) UpdateEnum(schema, name string, values []string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	if values == nil {
		values = []string{}
	}
	for i, e := range a.state.EnumTypes {
		if e.Schema == schema && e.Name == name {
			a.state.EnumTypes[i].Values = values
			go a.emitState()
			return nil
		}
	}
	return errEnumNotFound(schema, name)
}

// DeleteEnum removes an enum type.
func (a *App) DeleteEnum(schema, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	for i, e := range a.state.EnumTypes {
		if e.Schema == schema && e.Name == name {
			a.state.EnumTypes = append(a.state.EnumTypes[:i], a.state.EnumTypes[i+1:]...)
			go a.emitState()
			return nil
		}
	}
	return errEnumNotFound(schema, name)
}

// AddCheck adds a check constraint to a table.
func (a *App) AddCheck(schema, table, name, expression string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	// Check name uniqueness among constraints
	for _, tc := range t.Constraints {
		if tc.Name == name && name != "" {
			return fmt.Errorf("constraint %q already exists on %q.%q", name, schema, table)
		}
	}
	t.Constraints = append(t.Constraints, TableConstraint{
		Type:       "check",
		Name:       name,
		Expression: expression,
		Columns:    []string{},
	})
	go a.emitState()
	return nil
}

// DeleteCheck removes a check constraint from a table.
func (a *App) DeleteCheck(schema, table, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	for i, tc := range t.Constraints {
		if tc.Type == "check" && tc.Name == name {
			t.Constraints = append(t.Constraints[:i], t.Constraints[i+1:]...)
			go a.emitState()
			return nil
		}
	}
	return fmt.Errorf("check constraint %q not found on %q.%q", name, schema, table)
}

// UpdateTable updates properties of an existing table.
func (a *App) UpdateTable(schema, name string, updates map[string]interface{}) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, name)
	if t == nil {
		return errTableNotFound(schema, name)
	}
	if v, ok := updates["comment"]; ok {
		if s, ok := v.(string); ok {
			t.Comment = s
		}
	}
	// Accept both "name" and "new_name" keys for renaming
	var renameVal interface{}
	if v, ok := updates["new_name"]; ok {
		renameVal = v
	} else if v, ok := updates["name"]; ok {
		renameVal = v
	}
	if renameVal != nil {
		if newName, ok := renameVal.(string); ok && newName != "" && newName != t.Name {
			// Check new name doesn't conflict
			if a.findTable(schema, newName) != nil {
				return errTableExists(schema, newName)
			}
			oldName := t.Name
			t.Name = newName
			// Update all foreign key references
			for i := range a.state.ForeignKeys {
				fk := &a.state.ForeignKeys[i]
				if fk.FromSchema == schema && fk.FromTable == oldName {
					fk.FromTable = newName
				}
				if fk.ToSchema == schema && fk.ToTable == oldName {
					fk.ToTable = newName
				}
			}
		}
	}
	go a.emitState()
	return nil
}

// UpdateSchema updates properties of an existing schema.
func (a *App) UpdateSchema(name string, updates map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	var s *Schema
	for i := range a.state.Schemas {
		if a.state.Schemas[i].Name == name {
			s = &a.state.Schemas[i]
			break
		}
	}
	if s == nil {
		return errSchemaNotFound(name)
	}
	if v, ok := updates["color"]; ok {
		if c, ok := v.(string); ok {
			s.Color = c
		}
	}
	// Accept both "name" and "new_name" keys for renaming
	var renameVal interface{}
	if v, ok := updates["new_name"]; ok {
		renameVal = v
	} else if v, ok := updates["name"]; ok {
		renameVal = v
	}
	if renameVal != nil {
		if newName, ok := renameVal.(string); ok && newName != "" && newName != s.Name {
			// Validate new name doesn't conflict
			for _, existing := range a.state.Schemas {
				if existing.Name == newName {
					return errSchemaExists(newName)
				}
			}
			oldName := s.Name
			s.Name = newName
			// Cascade to tables
			for i := range a.state.Tables {
				if a.state.Tables[i].Schema == oldName {
					a.state.Tables[i].Schema = newName
				}
			}
			// Cascade to foreign keys
			for i := range a.state.ForeignKeys {
				fk := &a.state.ForeignKeys[i]
				if fk.FromSchema == oldName {
					fk.FromSchema = newName
				}
				if fk.ToSchema == oldName {
					fk.ToSchema = newName
				}
			}
			// Cascade to enum types
			for i := range a.state.EnumTypes {
				if a.state.EnumTypes[i].Schema == oldName {
					a.state.EnumTypes[i].Schema = newName
				}
			}
			// Cascade to views
			for i := range a.state.Views {
				if a.state.Views[i].Schema == oldName {
					a.state.Views[i].Schema = newName
				}
			}
		}
	}
	go a.emitState()
	return nil
}

// AddExtension adds a PostgreSQL extension.
func (a *App) AddExtension(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ext := range a.state.Extensions {
		if ext.Name == name {
			return fmt.Errorf("extension %q already exists", name)
		}
	}
	a.pushUndo()
	a.state.Extensions = append(a.state.Extensions, Extension{Name: name})
	go a.emitState()
	return nil
}

// DeleteExtension removes a PostgreSQL extension.
func (a *App) DeleteExtension(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	idx := -1
	for i, ext := range a.state.Extensions {
		if ext.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("extension %q not found", name)
	}
	a.pushUndo()
	a.state.Extensions = append(a.state.Extensions[:idx], a.state.Extensions[idx+1:]...)
	go a.emitState()
	return nil
}

// CreateView adds a new view.
func (a *App) CreateView(schema, name string, columns []ViewColumn) error {
	if schema == "" {
		schema = "public"
	}
	if columns == nil {
		columns = []ViewColumn{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.schemaExists(schema) {
		return errSchemaNotFound(schema)
	}
	for _, v := range a.state.Views {
		if v.Schema == schema && v.Name == name {
			return errViewExists(schema, name)
		}
	}
	a.pushUndo()
	a.state.Views = append(a.state.Views, View{Schema: schema, Name: name, Columns: columns})
	go a.emitState()
	return nil
}

// UpdateView updates properties of an existing view.
func (a *App) UpdateView(schema, name string, updates map[string]interface{}) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	var v *View
	for i := range a.state.Views {
		if a.state.Views[i].Schema == schema && a.state.Views[i].Name == name {
			v = &a.state.Views[i]
			break
		}
	}
	if v == nil {
		return errViewNotFound(schema, name)
	}
	if val, ok := updates["comment"]; ok {
		if s, ok := val.(string); ok {
			v.Comment = s
		}
	}
	if val, ok := updates["columns"]; ok {
		if arr, ok := val.([]interface{}); ok {
			cols := make([]ViewColumn, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					vc := ViewColumn{}
					if s, ok := m["name"].(string); ok {
						vc.Name = s
					}
					if s, ok := m["type"].(string); ok {
						vc.Type = s
					}
					if s, ok := m["source_schema"].(string); ok {
						vc.SourceSchema = s
					} else if s, ok := m["sourceSchema"].(string); ok {
						vc.SourceSchema = s
					}
					if s, ok := m["source_table"].(string); ok {
						vc.SourceTable = s
					} else if s, ok := m["sourceTable"].(string); ok {
						vc.SourceTable = s
					}
					if s, ok := m["source_column"].(string); ok {
						vc.SourceColumn = s
					} else if s, ok := m["sourceColumn"].(string); ok {
						vc.SourceColumn = s
					}
					cols = append(cols, vc)
				}
			}
			v.Columns = cols
		}
	}
	// Accept both "name" and "new_name" keys for renaming
	var renameVal interface{}
	if val, ok := updates["new_name"]; ok {
		renameVal = val
	} else if val, ok := updates["name"]; ok {
		renameVal = val
	}
	if renameVal != nil {
		if newName, ok := renameVal.(string); ok && newName != "" && newName != v.Name {
			// Check uniqueness
			for _, existing := range a.state.Views {
				if existing.Schema == schema && existing.Name == newName {
					return errViewExists(schema, newName)
				}
			}
			v.Name = newName
		}
	}
	go a.emitState()
	return nil
}

// DeleteView removes a view.
func (a *App) DeleteView(schema, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	idx := -1
	for i, v := range a.state.Views {
		if v.Schema == schema && v.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errViewNotFound(schema, name)
	}
	a.pushUndo()
	a.state.Views = append(a.state.Views[:idx], a.state.Views[idx+1:]...)
	go a.emitState()
	return nil
}

// --- internal helpers (caller must hold lock) ---

func (a *App) schemaExists(name string) bool {
	for _, s := range a.state.Schemas {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (a *App) findTable(schema, name string) *Table {
	for i := range a.state.Tables {
		if a.state.Tables[i].Schema == schema && a.state.Tables[i].Name == name {
			return &a.state.Tables[i]
		}
	}
	return nil
}

func (a *App) findView(schema, name string) *View {
	for i := range a.state.Views {
		if a.state.Views[i].Schema == schema && a.state.Views[i].Name == name {
			return &a.state.Views[i]
		}
	}
	return nil
}

func (a *App) columnExists(t *Table, name string) bool {
	for _, c := range t.Columns {
		if c.Name == name {
			return true
		}
	}
	return false
}

// SetTablePosition sets the manual x,y position for a table on the canvas.
func (a *App) SetTablePosition(schema, table string, x, y float64) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	t.Position = &Position{X: x, Y: y}
	go a.emitState()
	return nil
}

// ClearTablePosition removes the manual position for a table, reverting to auto-layout.
func (a *App) ClearTablePosition(schema, table string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	t := a.findTable(schema, table)
	if t == nil {
		return errTableNotFound(schema, table)
	}
	t.Position = nil
	go a.emitState()
	return nil
}

// SetViewPosition sets the manual x,y position for a view on the canvas.
func (a *App) SetViewPosition(schema, name string, x, y float64) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	v := a.findView(schema, name)
	if v == nil {
		return errViewNotFound(schema, name)
	}
	v.Position = &Position{X: x, Y: y}
	go a.emitState()
	return nil
}

// ClearViewPosition removes the manual position for a view, reverting to auto-layout.
func (a *App) ClearViewPosition(schema, name string) error {
	if schema == "" {
		schema = "public"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushUndo()
	v := a.findView(schema, name)
	if v == nil {
		return errViewNotFound(schema, name)
	}
	v.Position = nil
	go a.emitState()
	return nil
}

// GetMCPPath returns the absolute path to the schemata-mcp executable.
func (a *App) GetMCPPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "schemata-mcp.exe"
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "schemata-mcp.exe")
}

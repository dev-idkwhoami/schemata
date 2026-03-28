package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ForeignKeyYAML is a compact YAML representation of a foreign key.
type ForeignKeyYAML struct {
	From     string `yaml:"from"`                // "schema.table.column"
	To       string `yaml:"to"`                  // "schema.table.column"
	OnDelete string `yaml:"on_delete,omitempty"` // FK action
	OnUpdate string `yaml:"on_update,omitempty"` // FK action
}

// ColumnYAML mirrors Column with omitempty on optional fields.
type ColumnYAML struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Nullable   bool   `yaml:"nullable,omitempty"`
	PrimaryKey bool   `yaml:"primary_key,omitempty"`
	Unique     bool   `yaml:"unique,omitempty"`
	Default    string `yaml:"default,omitempty"`
	Comment    string `yaml:"comment,omitempty"`
}

// TableYAML mirrors Table for YAML serialization.
type TableYAML struct {
	Name        string            `yaml:"name"`
	Schema      string            `yaml:"schema"`
	Columns     []ColumnYAML      `yaml:"columns"`
	Constraints []TableConstraint `yaml:"constraints,omitempty"`
	Indexes     []Index           `yaml:"indexes,omitempty"`
	Position    *Position         `yaml:"position,omitempty"`
}

// AppStateYAML is the YAML-friendly representation of AppState.
type AppStateYAML struct {
	Schemas     []Schema         `yaml:"schemas"`
	Tables      []TableYAML      `yaml:"tables"`
	ForeignKeys []ForeignKeyYAML `yaml:"foreign_keys"`
	EnumTypes   []EnumType       `yaml:"enum_types,omitempty"`
	Extensions  []Extension      `yaml:"extensions,omitempty"`
	Views       []View           `yaml:"views,omitempty"`
}

// foreignKeyToYAML converts a ForeignKey to its compact YAML form.
func foreignKeyToYAML(fk ForeignKey) ForeignKeyYAML {
	return ForeignKeyYAML{
		From:     fmt.Sprintf("%s.%s.%s", fk.FromSchema, fk.FromTable, fk.FromColumn),
		To:       fmt.Sprintf("%s.%s.%s", fk.ToSchema, fk.ToTable, fk.ToColumn),
		OnDelete: fk.OnDelete,
		OnUpdate: fk.OnUpdate,
	}
}

// foreignKeyFromYAML converts a compact YAML foreign key back to a ForeignKey.
func foreignKeyFromYAML(fky ForeignKeyYAML) (ForeignKey, error) {
	fromParts := strings.SplitN(fky.From, ".", 3)
	if len(fromParts) != 3 {
		return ForeignKey{}, fmt.Errorf("invalid foreign key 'from' format: %q (expected schema.table.column)", fky.From)
	}
	toParts := strings.SplitN(fky.To, ".", 3)
	if len(toParts) != 3 {
		return ForeignKey{}, fmt.Errorf("invalid foreign key 'to' format: %q (expected schema.table.column)", fky.To)
	}
	return ForeignKey{
		FromSchema: fromParts[0],
		FromTable:  fromParts[1],
		FromColumn: fromParts[2],
		ToSchema:   toParts[0],
		ToTable:    toParts[1],
		ToColumn:   toParts[2],
		OnDelete:   fky.OnDelete,
		OnUpdate:   fky.OnUpdate,
	}, nil
}

// appStateToYAML converts AppState to AppStateYAML.
func appStateToYAML(state *AppState) AppStateYAML {
	tables := make([]TableYAML, len(state.Tables))
	for i, t := range state.Tables {
		cols := make([]ColumnYAML, len(t.Columns))
		for j, c := range t.Columns {
			cols[j] = ColumnYAML{
				Name:       c.Name,
				Type:       c.Type,
				Nullable:   c.Nullable,
				PrimaryKey: c.PrimaryKey,
				Unique:     c.Unique,
				Default:    c.Default,
				Comment:    c.Comment,
			}
		}
		constraints := t.Constraints
		if constraints == nil {
			constraints = []TableConstraint{}
		}
		indexes := t.Indexes
		if indexes == nil {
			indexes = []Index{}
		}
		tables[i] = TableYAML{
			Name:        t.Name,
			Schema:      t.Schema,
			Columns:     cols,
			Constraints: constraints,
			Indexes:     indexes,
			Position:    t.Position,
		}
	}

	fks := make([]ForeignKeyYAML, len(state.ForeignKeys))
	for i, fk := range state.ForeignKeys {
		fks[i] = foreignKeyToYAML(fk)
	}

	enumTypes := state.EnumTypes
	if enumTypes == nil {
		enumTypes = []EnumType{}
	}

	extensions := state.Extensions
	if extensions == nil {
		extensions = []Extension{}
	}

	views := state.Views
	if views == nil {
		views = []View{}
	}

	return AppStateYAML{
		Schemas:     state.Schemas,
		Tables:      tables,
		ForeignKeys: fks,
		EnumTypes:   enumTypes,
		Extensions:  extensions,
		Views:       views,
	}
}

// appStateFromYAML converts AppStateYAML back to AppState.
func appStateFromYAML(sy AppStateYAML) (*AppState, error) {
	tables := make([]Table, len(sy.Tables))
	for i, t := range sy.Tables {
		cols := make([]Column, len(t.Columns))
		for j, c := range t.Columns {
			cols[j] = Column{
				Name:       c.Name,
				Type:       c.Type,
				Nullable:   c.Nullable,
				PrimaryKey: c.PrimaryKey,
				Unique:     c.Unique,
				Default:    c.Default,
				Comment:    c.Comment,
			}
		}
		constraints := t.Constraints
		if constraints == nil {
			constraints = []TableConstraint{}
		}
		indexes := t.Indexes
		if indexes == nil {
			indexes = []Index{}
		}
		tables[i] = Table{
			Name:        t.Name,
			Schema:      t.Schema,
			Columns:     cols,
			Constraints: constraints,
			Indexes:     indexes,
			Position:    t.Position,
		}
	}

	fks := make([]ForeignKey, 0, len(sy.ForeignKeys))
	for _, fky := range sy.ForeignKeys {
		fk, err := foreignKeyFromYAML(fky)
		if err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}

	enumTypes := sy.EnumTypes
	if enumTypes == nil {
		enumTypes = []EnumType{}
	}

	extensions := sy.Extensions
	if extensions == nil {
		extensions = []Extension{}
	}

	views := sy.Views
	if views == nil {
		views = []View{}
	}

	return &AppState{
		Schemas:     sy.Schemas,
		Tables:      tables,
		ForeignKeys: fks,
		EnumTypes:   enumTypes,
		Extensions:  extensions,
		Views:       views,
	}, nil
}

// marshalYAML converts AppState to clean YAML bytes.
func marshalYAML(state *AppState) ([]byte, error) {
	sy := appStateToYAML(state)
	return yaml.Marshal(sy)
}

// unmarshalYAML parses YAML back into AppState.
func unmarshalYAML(data []byte) (*AppState, error) {
	var sy AppStateYAML
	if err := yaml.Unmarshal(data, &sy); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	// Ensure non-nil slices
	if sy.Schemas == nil {
		sy.Schemas = []Schema{}
	}
	if sy.Tables == nil {
		sy.Tables = []TableYAML{}
	}
	if sy.ForeignKeys == nil {
		sy.ForeignKeys = []ForeignKeyYAML{}
	}
	if sy.EnumTypes == nil {
		sy.EnumTypes = []EnumType{}
	}
	if sy.Extensions == nil {
		sy.Extensions = []Extension{}
	}
	if sy.Views == nil {
		sy.Views = []View{}
	}
	return appStateFromYAML(sy)
}

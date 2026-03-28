package main

import "fmt"

func errSchemaExists(name string) error {
	return fmt.Errorf("schema %q already exists", name)
}

func errSchemaNotFound(name string) error {
	return fmt.Errorf("schema %q not found", name)
}

func errTableExists(schema, name string) error {
	return fmt.Errorf("table %q.%q already exists", schema, name)
}

func errTableNotFound(schema, name string) error {
	return fmt.Errorf("table %q.%q not found", schema, name)
}

func errColumnExists(schema, table, column string) error {
	return fmt.Errorf("column %q in %q.%q already exists", column, schema, table)
}

func errColumnNotFound(schema, table, column string) error {
	return fmt.Errorf("column %q in %q.%q not found", column, schema, table)
}

func errForeignKeyExists(schema, table, column string) error {
	return fmt.Errorf("foreign key from %q.%q.%q already exists", schema, table, column)
}

func errForeignKeyNotFound(schema, table, column string) error {
	return fmt.Errorf("foreign key from %q.%q.%q not found", schema, table, column)
}

func errIndexExists(schema, table, name string) error {
	return fmt.Errorf("index %q on %q.%q already exists", name, schema, table)
}

func errIndexNotFound(schema, table, name string) error {
	return fmt.Errorf("index %q on %q.%q not found", name, schema, table)
}

func errEnumExists(schema, name string) error {
	return fmt.Errorf("enum type %q.%q already exists", schema, name)
}

func errEnumNotFound(schema, name string) error {
	return fmt.Errorf("enum type %q.%q not found", schema, name)
}

func errInvalidFKAction(action string) error {
	return fmt.Errorf("invalid foreign key action %q: must be CASCADE, SET NULL, SET DEFAULT, RESTRICT, or NO ACTION", action)
}

func errViewExists(schema, name string) error {
	return fmt.Errorf("view %q.%q already exists", schema, name)
}

func errViewNotFound(schema, name string) error {
	return fmt.Errorf("view %q.%q not found", schema, name)
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const defaultAPIPort = 9800

// startAPIServer starts a local HTTP API server for the MCP server to connect to.
func startAPIServer(app *App) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app.GetState())
	})

	mux.HandleFunc("/api/create-schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.CreateSchema(req.Name, req.Color); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Schema %q created", req.Name))
	})

	mux.HandleFunc("/api/create-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.CreateTable(req.Schema, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Table %q.%q created", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/add-column", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema     string `json:"schema"`
			Table      string `json:"table"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Nullable   bool   `json:"nullable"`
			PrimaryKey bool   `json:"primary_key"`
			Unique     bool   `json:"unique"`
			Default    string `json:"default"`
			Comment    string `json:"comment"`
			Position   int    `json:"position"`
			Generated  string `json:"generated"`
		}
		req.Position = -1
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.AddColumn(req.Schema, req.Table, req.Name, req.Type, req.Nullable, req.PrimaryKey, req.Unique, req.Default, req.Comment, req.Position, req.Generated); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Column %q added to %q.%q", req.Name, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/update-column", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string                 `json:"schema"`
			Table   string                 `json:"table"`
			Column  string                 `json:"column"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateColumn(req.Schema, req.Table, req.Column, req.Updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Column %q in %q.%q updated", req.Column, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/reorder-columns", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema      string   `json:"schema"`
			Table       string   `json:"table"`
			ColumnOrder []string `json:"column_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.ReorderColumns(req.Schema, req.Table, req.ColumnOrder); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Columns in %q.%q reordered", req.Schema, req.Table))
	})

	mux.HandleFunc("/api/add-foreign-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FromSchema string `json:"from_schema"`
			FromTable  string `json:"from_table"`
			FromColumn string `json:"from_column"`
			ToSchema   string `json:"to_schema"`
			ToTable    string `json:"to_table"`
			ToColumn   string `json:"to_column"`
			OnDelete   string `json:"on_delete"`
			OnUpdate   string `json:"on_update"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.FromSchema == "" {
			req.FromSchema = "public"
		}
		if req.ToSchema == "" {
			req.ToSchema = "public"
		}
		if err := app.AddForeignKey(req.FromSchema, req.FromTable, req.FromColumn, req.ToSchema, req.ToTable, req.ToColumn, req.OnDelete, req.OnUpdate); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("FK %s.%s.%s -> %s.%s.%s created",
			req.FromSchema, req.FromTable, req.FromColumn, req.ToSchema, req.ToTable, req.ToColumn))
	})

	mux.HandleFunc("/api/rename-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string `json:"schema"`
			OldName string `json:"old_name"`
			NewName string `json:"new_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.RenameTable(req.Schema, req.OldName, req.NewName); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Table %q.%q renamed to %q.%q", req.Schema, req.OldName, req.Schema, req.NewName))
	})

	mux.HandleFunc("/api/delete-schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.DeleteSchema(req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Schema %q deleted", req.Name))
	})

	mux.HandleFunc("/api/delete-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteTable(req.Schema, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Table %q.%q deleted", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/delete-column", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Table  string `json:"table"`
			Column string `json:"column"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteColumn(req.Schema, req.Table, req.Column); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Column %q deleted from %q.%q", req.Column, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/delete-foreign-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FromSchema string `json:"from_schema"`
			FromTable  string `json:"from_table"`
			FromColumn string `json:"from_column"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.FromSchema == "" {
			req.FromSchema = "public"
		}
		if err := app.DeleteForeignKey(req.FromSchema, req.FromTable, req.FromColumn); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("FK from %s.%s.%s deleted", req.FromSchema, req.FromTable, req.FromColumn))
	})

	mux.HandleFunc("/api/add-constraint", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string   `json:"schema"`
			Table   string   `json:"table"`
			Type    string   `json:"type"`
			Columns []string `json:"columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.AddConstraint(req.Schema, req.Table, req.Type, req.Columns); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Constraint %s(%s) added to %q.%q", req.Type, strings.Join(req.Columns, ", "), req.Schema, req.Table))
	})

	mux.HandleFunc("/api/update-constraint", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema     string   `json:"schema"`
			Table      string   `json:"table"`
			Type       string   `json:"type"`
			OldColumns []string `json:"old_columns"`
			NewColumns []string `json:"new_columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateConstraint(req.Schema, req.Table, req.Type, req.OldColumns, req.NewColumns); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Constraint %s updated on %q.%q", req.Type, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/delete-constraint", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string   `json:"schema"`
			Table   string   `json:"table"`
			Type    string   `json:"type"`
			Columns []string `json:"columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteConstraint(req.Schema, req.Table, req.Type, req.Columns); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Constraint %s(%s) deleted from %q.%q", req.Type, strings.Join(req.Columns, ", "), req.Schema, req.Table))
	})

	mux.HandleFunc("/api/set-table-position", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string  `json:"schema"`
			Table  string  `json:"table"`
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.SetTablePosition(req.Schema, req.Table, req.X, req.Y); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Position set for %q.%q", req.Schema, req.Table))
	})

	mux.HandleFunc("/api/clear-table-position", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Table  string `json:"table"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.ClearTablePosition(req.Schema, req.Table); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Position cleared for %q.%q", req.Schema, req.Table))
	})

	mux.HandleFunc("/api/undo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		if err := app.Undo(); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, "Undo successful")
	})

	mux.HandleFunc("/api/redo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		if err := app.Redo(); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, "Redo successful")
	})

	mux.HandleFunc("/api/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		app.Clear()
		jsonOK(w, "All state cleared")
	})

	mux.HandleFunc("/api/export-sql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(app.ExportSQL()))
	})

	mux.HandleFunc("/api/save-project", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.SaveProject(req.Path); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Project saved to %s", req.Path))
	})

	mux.HandleFunc("/api/load-project", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.LoadProject(req.Path); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Project loaded from %s", req.Path))
	})

	mux.HandleFunc("/api/add-index", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string   `json:"schema"`
			Table   string   `json:"table"`
			Name    string   `json:"name"`
			Columns []string `json:"columns"`
			Unique  bool     `json:"unique"`
			Type    string   `json:"type"`
			Where   string   `json:"where"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.AddIndex(req.Schema, req.Table, req.Name, req.Columns, req.Unique, req.Type, req.Where); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Index added to %q.%q", req.Schema, req.Table))
	})

	mux.HandleFunc("/api/update-index", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string                 `json:"schema"`
			Table   string                 `json:"table"`
			Name    string                 `json:"name"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateIndex(req.Schema, req.Table, req.Name, req.Updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Index %q on %q.%q updated", req.Name, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/delete-index", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Table  string `json:"table"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteIndex(req.Schema, req.Table, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Index %q deleted from %q.%q", req.Name, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/update-foreign-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FromSchema string `json:"from_schema"`
			FromTable  string `json:"from_table"`
			FromColumn string `json:"from_column"`
			OnDelete   string `json:"on_delete"`
			OnUpdate   string `json:"on_update"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.FromSchema == "" {
			req.FromSchema = "public"
		}
		updates := make(map[string]interface{})
		if req.OnDelete != "" {
			updates["on_delete"] = req.OnDelete
		}
		if req.OnUpdate != "" {
			updates["on_update"] = req.OnUpdate
		}
		if err := app.UpdateForeignKey(req.FromSchema, req.FromTable, req.FromColumn, updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("FK from %s.%s.%s updated", req.FromSchema, req.FromTable, req.FromColumn))
	})

	mux.HandleFunc("/api/create-enum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string   `json:"schema"`
			Name   string   `json:"name"`
			Values []string `json:"values"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.CreateEnum(req.Schema, req.Name, req.Values); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Enum type %q.%q created", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/update-enum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string   `json:"schema"`
			Name   string   `json:"name"`
			Values []string `json:"values"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateEnum(req.Schema, req.Name, req.Values); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Enum type %q.%q updated", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/delete-enum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteEnum(req.Schema, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Enum type %q.%q deleted", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/add-check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema     string `json:"schema"`
			Table      string `json:"table"`
			Name       string `json:"name"`
			Expression string `json:"expression"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.AddCheck(req.Schema, req.Table, req.Name, req.Expression); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Check constraint %q added to %q.%q", req.Name, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/delete-check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Table  string `json:"table"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteCheck(req.Schema, req.Table, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Check constraint %q deleted from %q.%q", req.Name, req.Schema, req.Table))
	})

	mux.HandleFunc("/api/update-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string                 `json:"schema"`
			Name    string                 `json:"name"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateTable(req.Schema, req.Name, req.Updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Table %q.%q updated", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/update-schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name    string                 `json:"name"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.UpdateSchema(req.Name, req.Updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Schema %q updated", req.Name))
	})

	mux.HandleFunc("/api/add-extension", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.AddExtension(req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Extension %q added", req.Name))
	})

	mux.HandleFunc("/api/delete-extension", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if err := app.DeleteExtension(req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Extension %q deleted", req.Name))
	})

	mux.HandleFunc("/api/create-view", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string       `json:"schema"`
			Name    string       `json:"name"`
			Columns []ViewColumn `json:"columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.CreateView(req.Schema, req.Name, req.Columns); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("View %q.%q created", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/update-view", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema  string                 `json:"schema"`
			Name    string                 `json:"name"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.UpdateView(req.Schema, req.Name, req.Updates); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("View %q.%q updated", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/delete-view", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.DeleteView(req.Schema, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("View %q.%q deleted", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/set-view-position", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string  `json:"schema"`
			Name   string  `json:"name"`
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.SetViewPosition(req.Schema, req.Name, req.X, req.Y); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Position set for view %q.%q", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/clear-view-position", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error())
			return
		}
		if req.Schema == "" {
			req.Schema = "public"
		}
		if err := app.ClearViewPosition(req.Schema, req.Name); err != nil {
			jsonError(w, err.Error())
			return
		}
		jsonOK(w, fmt.Sprintf("Position cleared for view %q.%q", req.Schema, req.Name))
	})

	mux.HandleFunc("/api/current-file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": app.GetCurrentFile()})
	})

	go func() {
		addr := fmt.Sprintf("127.0.0.1:%d", defaultAPIPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[API] Failed to start on %s: %v\n", addr, err)
			return
		}
		// Write port file so MCP server can discover us
		writePortFile(defaultAPIPort)
		log.Printf("[API] Listening on %s\n", addr)
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("[API] Server error: %v\n", err)
		}
	}()
}

func jsonOK(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": msg})
}

func jsonError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": msg})
}

// writePortFile writes the API port to a known location for the MCP server to find.
func writePortFile(port int) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	portDir := filepath.Join(dir, "schemata")
	os.MkdirAll(portDir, 0755)
	portFile := filepath.Join(portDir, "port")
	os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644)
}

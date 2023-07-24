package catalog

import (
	"github.com/kyleconroy/sqlc/internal/source"
	"github.com/kyleconroy/sqlc/internal/sql/ast"
	"github.com/kyleconroy/sqlc/internal/sql/sqlerr"
	"strings"
)

// View
type View struct {
	SQLString string
	ViewRel   *ast.ViewStmt
	Table
}

func (c *Catalog) MemoryView(stmt *ast.ViewStmt, sqlString string, colGen columnGenerator) error {
	cols, err := colGen.OutputColumns(stmt.Query)
	if err != nil {
		return err
	}

	catName := ""
	if stmt.View.Catalogname != nil {
		catName = *stmt.View.Catalogname
	}
	schemaName := ""
	if stmt.View.Schemaname != nil {
		schemaName = *stmt.View.Schemaname
	}

	strippedSQL, _, err := source.StripComments(sqlString)
	if err != nil {
		return err
	}

	// This is jank but removes the CREATE VIEW <> AS from the SQL string
	trimmedSpace := strings.TrimSpace(strippedSQL)
	trimmedSpace = trimmedSpace[strings.Index(trimmedSpace, "\n"):]

	tbl := View{
		ViewRel:   stmt,
		SQLString: trimmedSpace,
		Table: Table{
			Rel: &ast.TableName{
				Catalog: catName,
				Schema:  schemaName,
				Name:    *stmt.View.Relname,
			},
			Columns: cols,
		},
	}

	ns := tbl.Table.Rel.Schema
	if ns == "" {
		ns = c.DefaultSchema
	}
	schema, err := c.getSchema(ns)
	if err != nil {
		return err
	}
	// Cannot replace a table with a memory view.
	_, _, err = schema.getTable(tbl.Table.Rel)
	if err == nil {
		return sqlerr.RelationExists(tbl.Table.Rel.Name)
	}

	_, _, err = schema.getView(tbl.Table.Rel)
	if err == nil {
		return sqlerr.RelationExists(tbl.Table.Rel.Name)
	}

	schema.MemoryViews = append(schema.MemoryViews, &tbl)

	return nil
}

func (c *Catalog) createView(stmt *ast.ViewStmt, colGen columnGenerator) error {
	cols, err := colGen.OutputColumns(stmt.Query)
	if err != nil {
		return err
	}

	catName := ""
	if stmt.View.Catalogname != nil {
		catName = *stmt.View.Catalogname
	}
	schemaName := ""
	if stmt.View.Schemaname != nil {
		schemaName = *stmt.View.Schemaname
	}

	tbl := Table{
		Rel: &ast.TableName{
			Catalog: catName,
			Schema:  schemaName,
			Name:    *stmt.View.Relname,
		},
		Columns: cols,
	}

	ns := tbl.Rel.Schema
	if ns == "" {
		ns = c.DefaultSchema
	}
	schema, err := c.getSchema(ns)
	if err != nil {
		return err
	}
	_, existingIdx, err := schema.getTable(tbl.Rel)
	if err == nil && !stmt.Replace {
		return sqlerr.RelationExists(tbl.Rel.Name)
	}

	if stmt.Replace && err == nil {
		schema.Tables[existingIdx] = &tbl
	} else {
		schema.Tables = append(schema.Tables, &tbl)
	}

	return nil
}

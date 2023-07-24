package rewrite

import (
	"fmt"
	"github.com/kyleconroy/sqlc/internal/source"
	"github.com/kyleconroy/sqlc/internal/sql/catalog"

	"github.com/kyleconroy/sqlc/internal/sql/ast"
	"github.com/kyleconroy/sqlc/internal/sql/astutils"
)

func MemoryViews(raw *ast.RawStmt, c *catalog.Catalog) (*ast.RawStmt, []source.Edit, error) {
	var applyError error
	var edits []source.Edit
	node := astutils.Apply(raw, func(cr *astutils.Cursor) bool {
		node := cr.Node()

		switch {
		case isTableName(node):
			tn := node.(*ast.RangeVar)
			var cName, schemaName string
			if tn.Catalogname != nil {
				cName = *tn.Catalogname
			}
			if tn.Schemaname != nil {
				schemaName = *tn.Schemaname
			} else {
				schemaName = c.DefaultSchema
			}

			view, err := c.GetView(&ast.TableName{
				Catalog: cName,
				Schema:  schemaName,
				Name:    *tn.Relname,
			})
			if err == nil {
				// This is the correct AST, but this breaks the returned
				// models.go type. We need to figure out how to fix that to
				// make this replace work and keep the SQL AST correct.
				//cr.Replace(&ast.RangeSubselect{
				//	Lateral:  false,
				//	Subquery: view.ViewRel.Query,
				//	Alias: &ast.Alias{
				//		Aliasname: tn.Relname,
				//	},
				//})
				edits = append(edits,
					source.Edit{
						Location: tn.Location - raw.StmtLocation,
						Old:      *tn.Relname,
						New:      fmt.Sprintf("(%s) AS %s", view.SQLString, *tn.Relname),
					})
				return false
			}
		}
		return true
	}, nil)

	return node.(*ast.RawStmt), edits, applyError
}

func isTableName(node ast.Node) bool {
	_, ok := node.(*ast.RangeVar)
	return ok
}

func isQueryTemplate(node ast.Node) bool {
	call, ok := node.(*ast.FuncCall)
	if !ok {
		return false
	}

	if call.Func == nil {
		return false
	}

	isValid := call.Func.Schema == "sqlc" && call.Func.Name == "template"
	return isValid
}

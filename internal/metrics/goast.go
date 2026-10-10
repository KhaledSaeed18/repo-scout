package metrics

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// analyzeGo measures Go source with the standard library's parser instead
// of patterns, so keywords inside strings and comments never count and every
// function and method is found. ok is false when the file does not parse;
// the caller then falls back to the pattern analyzer.
//
// Complexity is McCabe's, summed over the file: 1 per function plus one per
// if, for, range, non-default case, select case, && and ||. Closures count
// toward the function that holds them, as gocyclo does. Nesting is left to
// the caller so it stays comparable across languages.
func analyzeGo(content string) (r Result, ok bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.SkipObjectResolution)
	if err != nil {
		return Result{}, false
	}
	r.Imports = len(file.Imports)

	var lengths []int
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				r.Exports++
			}
			r.Complexity += 1 + decisions(d)
			lengths = append(lengths, fset.Position(d.End()).Line-fset.Position(d.Pos()).Line+1)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						r.Exports++
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							r.Exports++
						}
					}
				}
			}
			// Package-level function literals (var handler = func() {...})
			// are functions too.
			ast.Inspect(d, func(n ast.Node) bool {
				if lit, ok := n.(*ast.FuncLit); ok {
					r.Complexity += 1 + decisions(lit)
					return false
				}
				return true
			})
		}
	}

	r.FuncCount = len(lengths)
	total := 0
	for _, l := range lengths {
		total += l
		r.MaxFuncLen = max(r.MaxFuncLen, l)
	}
	if r.FuncCount > 0 {
		r.AvgFuncLen = float64(total) / float64(r.FuncCount)
	}
	return r, true
}

// decisions counts the branch points inside n, including nested closures.
func decisions(n ast.Node) int {
	count := 0
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			count++
		case *ast.CaseClause:
			if x.List != nil { // default adds no path
				count++
			}
		case *ast.CommClause:
			if x.Comm != nil {
				count++
			}
		case *ast.BinaryExpr:
			if x.Op == token.LAND || x.Op == token.LOR {
				count++
			}
		}
		return true
	})
	return count
}

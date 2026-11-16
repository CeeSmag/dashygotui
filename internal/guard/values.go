package guard

import (
	"go/ast"
	"go/token"
)

// The rules for an addons package as a whole about its values (docs/DESIGN.md,
// "the map"): A3 and A12.

// valueFindings applies A3 and A12 to the files of the addons package in dir.
func valueFindings(files []Source) []Finding {
	var out []Finding
	at := func(rule string, s Source, n ast.Node, msg, hint string) {
		out = append(out, Finding{Rule: rule, File: s.Rel, Line: s.Line(n.Pos()), Msg: msg, Hint: hint})
	}
	tops := map[*ast.ValueSpec]bool{}
	names := map[string]bool{}
	for _, s := range files {
		for _, d := range s.File.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				// A12: no init.
				if d.Recv == nil && d.Name.Name == "init" {
					at("A12", s, d, "init() in an addons package", "declare the addon as a top-level value and place it in "+MapFile)
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					vs := spec.(*ast.ValueSpec)
					tops[vs] = true
					for _, id := range vs.Names {
						names[id.Name] = true
					}
					if msg := valueSpecProblem(s, vs); msg != "" {
						at("A3", s, vs, msg, "declare one unexported addon per var, as a composite literal")
					}
				}
			}
		}
	}

	// A3: an addon value is never assigned to.
	for _, s := range files {
		if !IsFeatureFile(s.Rel) {
			continue
		}
		report := func(e ast.Expr, n ast.Node) {
			id := rootIdent(e)
			if id == nil || !names[id.Name] {
				return
			}
			if id.Obj != nil { // declared in this file: is it the top-level one, or a local that shadows it?
				vs, ok := id.Obj.Decl.(*ast.ValueSpec)
				if !ok || !tops[vs] {
					return
				}
			}
			at("A3", s, n, "assigns to the package-level addon "+id.Name+": it is shared by every dashboard in the process",
				"keep changing state in dashygotui.Local, or in the Store if it must be saved")
		}
		ast.Inspect(s.File, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok != token.DEFINE {
					for _, l := range n.Lhs {
						report(l, n)
					}
				}
			case *ast.IncDecStmt:
				report(n.X, n)
			}
			return true
		})
	}
	return out
}

// valueSpecProblem says what is wrong with a top-level var of an addons package
// with a generated list, or "" when it is a proper addon value.
func valueSpecProblem(s Source, vs *ast.ValueSpec) string {
	switch {
	case IsMapFile(s.Rel):
		return "package-level variable " + joinNames(vs.Names) + " in the map: the map places addons, each declared in its own file"
	case !IsFeatureFile(s.Rel):
		return "package-level variable " + joinNames(vs.Names) + " outside an addon file"
	case len(vs.Names) != 1 || len(vs.Values) != 1:
		return "package-level variables " + joinNames(vs.Names) + " declared together: declare one addon per var"
	case vs.Names[0].Name == "_":
		return "a blank package-level variable is not an addon"
	case vs.Names[0].IsExported():
		return "addon " + vs.Names[0].Name + " is exported: addon names are unexported, the package exports only Map"
	}
	lit, ok := vs.Values[0].(*ast.CompositeLit)
	if !ok || !namedType(lit.Type) {
		return "package-level variable " + vs.Names[0].Name + " is not an addon literal"
	}
	return ""
}

// namedType reports whether t names a type, possibly instantiated, as an addon
// literal does (dashboard.Option, stdaddons.Toggle[dashboard.State], jump), as
// opposed to a map, slice, array or anonymous struct literal.
func namedType(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return true
	case *ast.IndexExpr:
		return namedType(t.X)
	case *ast.IndexListExpr:
		return namedType(t.X)
	}
	return false
}

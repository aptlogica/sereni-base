package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type row struct {
	Code        string
	HTTPStatus  int
	Message     string
	Description string
}

var httpStatus = map[string]int{
	"StatusOK":                  200,
	"StatusCreated":             201,
	"StatusAccepted":            202,
	"StatusNoContent":           204,
	"StatusBadRequest":          400,
	"StatusUnauthorized":        401,
	"StatusForbidden":           403,
	"StatusNotFound":            404,
	"StatusConflict":            409,
	"StatusRequestTimeout":      408,
	"StatusUnprocessableEntity": 422,
	"StatusTooManyRequests":     429,
	"StatusInternalServerError": 500,
	"StatusNotImplemented":      501,
	"StatusServiceUnavailable":  503,
	"StatusGatewayTimeout":      504,
	"StatusFailedDependency":    424,
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

func unquote(s string) string {
	u, err := strconv.Unquote(s)
	if err != nil {
		return s
	}
	return u
}

func evalString(fset *token.FileSet, e ast.Expr, strConsts map[string]string) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING { return unquote(v.Value) }
		return v.Value
	case *ast.Ident:
		if x, ok := strConsts[v.Name]; ok { return x }
		return v.Name
	case *ast.SelectorExpr:
		k := exprString(fset, v)
		if x, ok := strConsts[k]; ok { return x }
		return k
	default:
		return exprString(fset, e)
	}
}

func evalStatus(e ast.Expr) int {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.INT { n, _ := strconv.Atoi(v.Value); return n }
	case *ast.SelectorExpr:
		if x, ok := v.X.(*ast.Ident); ok && x.Name == "http" {
			if n, ok := httpStatus[v.Sel.Name]; ok { return n }
		}
	}
	return 0
}

func isResponseMetaMap(fset *token.FileSet, mt *ast.MapType) bool {
	if mt == nil { return false }
	return exprString(fset, mt.Key) == "ResponseCode" && exprString(fset, mt.Value) == "MetaResponse"
}

func keyExprToCode(fset *token.FileSet, e ast.Expr, codeMap map[string]string) string {
	keyExpr := exprString(fset, e)
	if c, ok := codeMap[keyExpr]; ok { return c }
	if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING { return unquote(bl.Value) }
	return keyExpr
}

func main() {
	root := filepath.FromSlash("internal/utils/response/constants")
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, root, func(fi os.FileInfo) bool {
		name := fi.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)
	if err != nil { panic(err) }
	pkg := pkgs["constants"]

	strConsts := map[string]string{}
	codeMap := map[string]string{}
	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files { files = append(files, f) }
	sort.Slice(files, func(i, j int) bool { return fset.Position(files[i].Pos()).Filename < fset.Position(files[j].Pos()).Filename })

	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl); if !ok { continue }
			if gd.Tok == token.CONST {
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec); if !ok { continue }
					for i, n := range vs.Names {
						if i < len(vs.Values) {
							if bl, ok := vs.Values[i].(*ast.BasicLit); ok && bl.Kind == token.STRING { strConsts[n.Name] = unquote(bl.Value) }
						}
					}
				}
			}
			if gd.Tok != token.VAR { continue }
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec); if !ok || len(vs.Names)==0 || len(vs.Values)==0 { continue }
				varName := vs.Names[0].Name
				cl, ok := vs.Values[0].(*ast.CompositeLit); if !ok { continue }
				for _, e := range cl.Elts {
					kv, ok := e.(*ast.KeyValueExpr); if !ok { continue }
					field := exprString(fset, kv.Key)
					if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
						codeMap[varName+"."+field] = unquote(bl.Value)
					}
				}
			}
		}
	}

	var rowsSuccess, rowsError []row
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl); if !ok || gd.Tok != token.VAR { continue }
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec); if !ok || len(vs.Names)==0 || len(vs.Values)==0 { continue }
				varName := vs.Names[0].Name
				var mt *ast.MapType
				if t, ok := vs.Type.(*ast.MapType); ok { mt = t }
				if cl, ok := vs.Values[0].(*ast.CompositeLit); ok {
					if t, ok := cl.Type.(*ast.MapType); ok { mt = t }
				}
				if !isResponseMetaMap(fset, mt) { continue }
				cl, ok := vs.Values[0].(*ast.CompositeLit); if !ok { continue }
				for _, e := range cl.Elts {
					kv, ok := e.(*ast.KeyValueExpr); if !ok { continue }
					r := row{Code:keyExprToCode(fset, kv.Key, codeMap)}
					switch v := kv.Value.(type) {
					case *ast.CallExpr:
						if exprString(fset, v.Fun)=="CreateMetaResponse" && len(v.Args)>=3 {
							r.HTTPStatus = evalStatus(v.Args[0]); r.Message = evalString(fset, v.Args[1], strConsts); r.Description = evalString(fset, v.Args[2], strConsts)
						}
					case *ast.CompositeLit:
						for _, e2 := range v.Elts {
							kv2, ok := e2.(*ast.KeyValueExpr); if !ok { continue }
							switch exprString(fset, kv2.Key) {
							case "HTTPStatus": r.HTTPStatus = evalStatus(kv2.Value)
							case "Message": r.Message = evalString(fset, kv2.Value, strConsts)
							case "Description": r.Description = evalString(fset, kv2.Value, strConsts)
							}
						}
					}
					if r.HTTPStatus==0 { continue }
					if r.Message=="" { r.Message = "-" }
					if r.Description=="" { r.Description = "-" }
					if strings.Contains(strings.ToLower(varName), "success") { rowsSuccess = append(rowsSuccess, r) } else if strings.Contains(strings.ToLower(varName), "error") { rowsError = append(rowsError, r) }
				}
			}
		}
	}

	uniq := func(in []row) []row {
		seen := map[string]bool{}
		out := make([]row, 0, len(in))
		for _, r := range in {
			k := r.Code+"|"+strconv.Itoa(r.HTTPStatus)+"|"+r.Message+"|"+r.Description
			if seen[k] { continue }
			seen[k] = true
			out = append(out, r)
		}
		return out
	}
	rowsSuccess = uniq(rowsSuccess); rowsError = uniq(rowsError)
	sort.Slice(rowsSuccess, func(i,j int) bool { return rowsSuccess[i].Code < rowsSuccess[j].Code })
	sort.Slice(rowsError, func(i,j int) bool { return rowsError[i].Code < rowsError[j].Code })

	fout, err := os.Create(filepath.FromSlash("internal/utils/response/RESPONSE_CODES.md")); if err != nil { panic(err) }
	defer fout.Close()
	esc := func(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
	w := func(s string, a ...any) { _, _ = fmt.Fprintf(fout, s, a...) }
	w("# API Response Codes\n\n")
	w("Source: `internal/utils/response/constants`\n\n")
	w("## Success Codes\n\n")
	w("| Custom Code | HTTP Status | Message | Description |\n")
	w("|---|---:|---|---|\n")
	for _, r := range rowsSuccess { w("| `%s` | %d | %s | %s |\n", esc(r.Code), r.HTTPStatus, esc(r.Message), esc(r.Description)) }
	w("\n## Failure Codes\n\n")
	w("| Custom Code | HTTP Status | Message | Description |\n")
	w("|---|---:|---|---|\n")
	for _, r := range rowsError { w("| `%s` | %d | %s | %s |\n", esc(r.Code), r.HTTPStatus, esc(r.Message), esc(r.Description)) }
	w("\n## Totals\n\n")
	w("- Success codes: **%d**\n", len(rowsSuccess))
	w("- Failure codes: **%d**\n", len(rowsError))
}

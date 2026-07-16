// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package harvest // import "go.opentelemetry.io/obi/pkg/internal/transform/route/harvest"

import (
	"io"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/token"
)

// readJSFileForScan returns the bounded contents of a JS/TS file using the same
// safety guards as the line scanner.
func readJSFileForScan(path string) (string, bool, error) {
	f, ok, err := openJSFileForScan(path)
	if err != nil || !ok {
		return "", ok, err
	}
	defer f.Close()

	b, err := io.ReadAll(io.LimitReader(f, MaxJSFileScanBytes))
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

var astRouteMethods = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"patch":   "PATCH",
	"delete":  "DELETE",
	"del":     "DELETE",
	"head":    "HEAD",
	"options": "OPTIONS",
	"all":     "ALL",
}

// resolveASTRoutes extracts routes whose path is supplied through a variable,
// a template literal, or string concatenation, which the line-based regex
// harvesters cannot match. It returns nil when the source cannot be parsed
// (for example TypeScript), leaving the regex-based results untouched.
func (e *RouteExtractor) resolveASTRoutes(filePath, src string) []RoutePattern {
	prog, err := parser.ParseFile(nil, filePath, src, 0)
	if err != nil {
		return nil
	}

	// collect string-valued variable bindings first, so identifiers used as
	// route paths resolve regardless of declaration order
	strVars := map[string]string{}
	walk(prog, func(n ast.Node) {
		b, ok := n.(*ast.Binding)
		if !ok {
			return
		}
		id, ok := b.Target.(*ast.Identifier)
		if !ok {
			return
		}
		if s, ok := stringValue(b.Initializer, nil); ok {
			strVars[id.Name.String()] = s
		}
	})

	var routes []RoutePattern
	walk(prog, func(n ast.Node) {
		call, ok := n.(*ast.CallExpression)
		if !ok || len(call.ArgumentList) == 0 {
			return
		}
		method, ok := routeMethod(call.Callee)
		if !ok {
			return
		}
		first := call.ArgumentList[0]
		// plain string literals are already harvested by the regex pass
		if _, isLit := first.(*ast.StringLiteral); isLit {
			return
		}
		path, ok := stringValue(first, strVars)
		if !ok || !strings.HasPrefix(path, "/") {
			return
		}
		routes = append(routes, RoutePattern{
			Method: method,
			Path:   path,
			File:   filePath,
			Line:   lineOf(prog, call.Idx0()),
		})
	})
	return routes
}

func routeMethod(callee ast.Expression) (string, bool) {
	dot, ok := callee.(*ast.DotExpression)
	if !ok {
		return "", false
	}
	method, ok := astRouteMethods[strings.ToLower(dot.Identifier.Name.String())]
	return method, ok
}

// stringValue resolves expr to a constant string when possible: string
// literals, identifiers looked up in vars (when non-nil), template literals,
// and binary `+` concatenation of resolvable operands.
func stringValue(expr ast.Expression, vars map[string]string) (string, bool) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value.String(), true
	case *ast.Identifier:
		if vars == nil {
			return "", false
		}
		s, ok := vars[v.Name.String()]
		return s, ok
	case *ast.TemplateLiteral:
		return templateValue(v, vars)
	case *ast.BinaryExpression:
		if v.Operator == token.PLUS {
			left, ok1 := stringValue(v.Left, vars)
			right, ok2 := stringValue(v.Right, vars)
			if ok1 && ok2 {
				return left + right, true
			}
		}
	}
	return "", false
}

// templateValue renders a template literal into a route path. An interpolated
// identifier that does not resolve to a string becomes a `:name` placeholder,
// preserving the route shape (`/users/${id}` -> /users/:id).
func templateValue(tmpl *ast.TemplateLiteral, vars map[string]string) (string, bool) {
	var b strings.Builder
	for i, el := range tmpl.Elements {
		b.WriteString(el.Parsed.String())
		if i < len(tmpl.Expressions) {
			expr := tmpl.Expressions[i]
			if s, ok := stringValue(expr, vars); ok {
				b.WriteString(s)
				continue
			}
			if id, ok := expr.(*ast.Identifier); ok {
				b.WriteString(":" + id.Name.String())
				continue
			}
			return "", false
		}
	}
	return b.String(), true
}

func lineOf(prog *ast.Program, idx file.Idx) int {
	if prog.File == nil {
		return 0
	}
	return prog.File.Position(int(idx) - prog.File.Base()).Line
}

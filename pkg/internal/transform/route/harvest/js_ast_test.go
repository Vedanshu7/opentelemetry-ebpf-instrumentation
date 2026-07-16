// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package harvest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scanJSContent(t *testing.T, content string) []RoutePattern {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.js")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	extractor := NewRouteExtractor()
	require.NoError(t, extractor.scanFile(path))
	return extractor.GetRoutes()
}

func assertHasRoute(t *testing.T, routes []RoutePattern, method, path string) {
	t.Helper()
	for _, r := range routes {
		if r.Method == method && r.Path == path {
			return
		}
	}
	t.Fatalf("expected route %s %s, got %v", method, path, routes)
}

func TestResolveASTRoutes_VariablePath(t *testing.T) {
	routes := scanJSContent(t, `
const usersPath = '/users';
app.get(usersPath, (req, res) => res.send('ok'));
`)
	assertHasRoute(t, routes, "GET", "/users")
}

func TestResolveASTRoutes_VariableDeclaredAfterUse(t *testing.T) {
	routes := scanJSContent(t, `
router.post(createPath, handler);
const createPath = '/api/items';
`)
	assertHasRoute(t, routes, "POST", "/api/items")
}

func TestResolveASTRoutes_TemplateLiteralWithResolvedVar(t *testing.T) {
	routes := scanJSContent(t, "const ver = 'v1';\napp.get(`/api/${ver}/users`, handler);\n")
	assertHasRoute(t, routes, "GET", "/api/v1/users")
}

func TestResolveASTRoutes_TemplateLiteralWithUnresolvedParam(t *testing.T) {
	routes := scanJSContent(t, "app.delete(`/users/${id}`, handler);\n")
	assertHasRoute(t, routes, "DELETE", "/users/:id")
}

func TestResolveASTRoutes_StringConcatenationLiterals(t *testing.T) {
	routes := scanJSContent(t, "app.post('/api' + '/orders', handler);\n")
	assertHasRoute(t, routes, "POST", "/api/orders")
}

func TestResolveASTRoutes_StringConcatenationVariableAndLiteral(t *testing.T) {
	routes := scanJSContent(t, "const base = '/api';\napp.put(base + '/settings', handler);\n")
	assertHasRoute(t, routes, "PUT", "/api/settings")
}

func TestResolveASTRoutes_NestedInsideFunction(t *testing.T) {
	routes := scanJSContent(t, `
function registerRoutes(app) {
  const p = '/health';
  app.get(p, handler);
}
`)
	assertHasRoute(t, routes, "GET", "/health")
}

func TestResolveASTRoutes_LineNumberIsSet(t *testing.T) {
	routes := scanJSContent(t, "const p = '/x';\napp.get(p, handler);\n")
	require.NotEmpty(t, routes)
	for _, r := range routes {
		if r.Path == "/x" {
			assert.Positive(t, r.Line, "line number should be set")
			assert.NotEmpty(t, r.File, "file should be set")
		}
	}
}

func TestResolveASTRoutes_SkipsUnparseableSource(t *testing.T) {
	// TypeScript annotations are not valid JS; the AST pass must return nil
	// without error rather than emitting bogus routes
	got := (&RouteExtractor{}).resolveASTRoutes("app.ts", `app.get(p: string, handler);`)
	assert.Nil(t, got)
}

func TestResolveASTRoutes_IgnoresStringLiteralArgs(t *testing.T) {
	got := (&RouteExtractor{}).resolveASTRoutes("app.js", `app.get('/literal', handler);`)
	assert.Empty(t, got)
}

func TestResolveASTRoutes_IgnoresUnknownMethod(t *testing.T) {
	got := (&RouteExtractor{}).resolveASTRoutes("app.js", `const p = '/x'; app.listen(p);`)
	assert.Empty(t, got)
}

func TestScanFile_LiteralRoutesSkipASTPass(t *testing.T) {
	// a template literal without interpolation is fully harvested by the regex
	// pass; the AST pass must not run and duplicate it
	routes := scanJSContent(t, "app.get(`/plain`, handler);\n")
	require.Len(t, routes, 1)
	assert.Equal(t, "/plain", routes[0].Path)
}

func TestRouteExtractor_VariableRoutesApp(t *testing.T) {
	extractor := NewRouteExtractor()
	exampleFile := filepath.Join("nodejs", "test_files", "variable-routes-app.js")
	require.NoError(t, extractor.scanFile(exampleFile))

	routes := extractor.GetRoutes()
	require.NotEmpty(t, routes)

	expected := []struct{ method, path string }{
		{"GET", "/users"},
		{"GET", "/api/v1/products"},
		{"DELETE", "/users/:id"},
		{"POST", "/api/orders"},
		{"PUT", "/api/settings"},
		{"GET", "/health"},
		{"GET", "/admin/stats"},
		{"GET", "/admin/disabled"},
		{"GET", "/while/v1"},
		{"GET", "/do/while"},
		{"PATCH", "/mode/:mode"},
		{"ALL", "/any/mode"},
		{"GET", "/try/v1"},
		{"GET", "/catch/route"},
		{"GET", "/finally/v1"},
		{"GET", "/returned/route"},
		{"GET", "/async/v1"},
		{"GET", "/api/from-object"},
		{"GET", "/legacy"},
		{"GET", "/flags/:key"},
		{"GET", "/labeled/route"},
		{"GET", "/seq/route"},
		{"GET", "/generator/route"},
		{"GET", "/with/route"},
	}
	for _, e := range expected {
		assertHasRoute(t, routes, e.method, e.path)
	}
}

func TestWalk_NilNode(t *testing.T) {
	walk(nil, func(ast.Node) { t.Fatal("fn must not be called for a nil node") })
}

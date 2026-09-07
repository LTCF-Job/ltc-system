package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpMethods 是 routes.go 用來註冊路由的 gin 方法名，用於從 AST 篩出路由註冊呼叫。
var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
}

// routeGroupPrefixes 對應 routes.go 內各 router 變數的路徑前綴。
var routeGroupPrefixes = map[string]string{
	"r":      "",
	"apiV1":  "/api/v1",
	"public": "",
}

type registeredRoute struct {
	method string
	path   string
	pos    string
}

// parseRegisteredRoutes 直接從 routes.go 的 AST 取出所有路由，避免測試自行維護一份會與
// 實際註冊漂移的路徑清單。
func parseRegisteredRoutes(t *testing.T) []registeredRoute {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	require.NoError(t, err)

	var routes []registeredRoute
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !httpMethods[sel.Sel.Name] || len(call.Args) == 0 {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		prefix, ok := routeGroupPrefixes[recv.Name]
		if !ok {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		path, err := strconv.Unquote(lit.Value)
		require.NoError(t, err)

		routes = append(routes, registeredRoute{
			method: sel.Sel.Name,
			path:   prefix + path,
			pos:    fset.Position(call.Pos()).String(),
		})
		return true
	})
	return routes
}

// TestRoutesRegisterWithoutConflict 把 routes.go 實際註冊的每條路徑放進真正的 gin engine。
// gin 的 radix tree 在同層 static 與 wildcard 相衝時會 panic，而 panic 發生在 server 啟動當下、
// 沒有任何測試涵蓋 newRouter，等於整個服務起不來卻要到部署才會發現。
func TestRoutesRegisterWithoutConflict(t *testing.T) {
	routes := parseRegisteredRoutes(t)
	require.Greater(t, len(routes), 50, "未掃到預期數量的路由註冊，測試可能已失效")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	noop := func(c *gin.Context) {}

	for _, route := range routes {
		require.NotPanics(t, func() {
			engine.Handle(route.method, route.path, noop)
		}, "路由 %s %s 與既有路由相衝（%s）", route.method, route.path, route.pos)
	}
}

// TestDriverReportDeleteRoutesDispatchDistinctly 鎖住 /driver-reports 底下三條 DELETE 的分派結果。
// 這三條在同一層同時有 static 段（columns、row-conflicts）與 wildcard（:id），路徑順序或寫法
// 一旦改動就可能讓「忽略此筆」誤打到刪除整份匯報表。
func TestDriverReportDeleteRoutesDispatchDistinctly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	hit := ""
	mark := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			hit = name + ":" + c.Param("id")
			c.Status(http.StatusNoContent)
		}
	}
	api := engine.Group("/api/v1")
	api.DELETE("/driver-reports/columns/:id", mark("column"))
	api.DELETE("/driver-reports/row-conflicts/:id", mark("rowConflict"))
	api.DELETE("/driver-reports/:id", mark("form"))

	const id = "6f2f5f9e-5c3f-4a1b-9a3d-2f0f1b7a9c11"
	cases := []struct {
		name    string
		path    string
		expects string
	}{
		{"欄位對應待維護", "/api/v1/driver-reports/columns/" + id, "column:" + id},
		{"同車同個案衝突", "/api/v1/driver-reports/row-conflicts/" + id, "rowConflict:" + id},
		{"匯報表本體", "/api/v1/driver-reports/" + id, "form:" + id},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hit = ""
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, tc.path, nil))

			assert.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, tc.expects, hit)
		})
	}
}

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	identityapp "ltc-system/apps/api/internal/modules/identity/app"
)

// TestRouteModuleKeysAreRegistered 掃描 routes.go 內所有 auth.RequirePermission 的 module 引數，
// 確認每個都登記在 identityapp.ModuleKeys。未登記的 key 不會有任何角色矩陣涵蓋，該路由會對所有
// 人永久 403 且沒有任何錯誤訊息，靠測試才擋得住這種無聲失效。
func TestRouteModuleKeysAreRegistered(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	require.NoError(t, err)

	found := 0
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "RequirePermission" || len(call.Args) != 4 {
			return true
		}
		lit, ok := call.Args[2].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			t.Errorf("RequirePermission 的 module 引數必須是字串字面值，位置 %s", fset.Position(call.Pos()))
			return true
		}
		key, err := strconv.Unquote(lit.Value)
		require.NoError(t, err)
		found++
		assert.True(t, identityapp.IsModuleKey(key), "module key %q 未登記於 identityapp.ModuleKeys（%s）", key, fset.Position(call.Pos()))
		return true
	})

	assert.Greater(t, found, 50, "未掃到預期數量的 RequirePermission 呼叫，測試可能已失效")
}

// TestNoRequireRolesInRoutes 鎖住「所有角色共用同一套權限判斷」的結論：路由層不得再出現
// 寫死角色字面值的 RequireRoles，否則自訂角色又會被無條件擋下。
func TestNoRequireRolesInRoutes(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	require.NoError(t, err)

	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "RequireRoles" {
			t.Errorf("routes.go 不得使用 RequireRoles，位置 %s", fset.Position(sel.Pos()))
		}
		return true
	})
}

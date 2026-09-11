package auth

import "context"

// RequestSecurityState 是同一個 HTTP request 內共享授權投影的最小容器。
// Middleware 先驗證帳號狀態後，後續 custom permission resolver 可直接重用同一份投影，
// 避免 user state 與 permission 兩條解析路徑在一次 request 內重複查資料庫。
type RequestSecurityState struct {
	Loaded            bool
	Found             bool
	Status            string
	RoleKey           string
	PermissionVersion string
	CustomPermissions map[string]ModulePermission
}

type requestSecurityStateKey struct{}

// WithRequestSecurityState 在 context 放入 request-local 的授權投影容器。
func WithRequestSecurityState(ctx context.Context) (context.Context, *RequestSecurityState) {
	state := &RequestSecurityState{}
	return context.WithValue(ctx, requestSecurityStateKey{}, state), state
}

// RequestSecurityStateFromContext 取得目前 request 的授權投影容器；未經 middleware
// 建立時回傳 nil，呼叫端應退回一般 source 解析流程。
func RequestSecurityStateFromContext(ctx context.Context) *RequestSecurityState {
	state, _ := ctx.Value(requestSecurityStateKey{}).(*RequestSecurityState)
	return state
}

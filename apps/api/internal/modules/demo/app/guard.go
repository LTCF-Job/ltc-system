// Package app 提供 Demo data-plane 的重置業務邏輯：清空並重新載入共用資料集。
package app

import "sync"

// ConcurrencyGuard 讓一般 API 請求與 Demo 重置互斥：重置需獨佔鎖，一般請求共享鎖。
type ConcurrencyGuard struct {
	mu sync.RWMutex
}

// NewConcurrencyGuard 建立 ConcurrencyGuard 實例。
func NewConcurrencyGuard() *ConcurrencyGuard {
	return &ConcurrencyGuard{}
}

// BeginRequest 標記一般請求開始，回傳的 release 必須在請求結束時呼叫。
func (g *ConcurrencyGuard) BeginRequest() (release func()) {
	g.mu.RLock()
	return g.mu.RUnlock
}

// BeginReset 等待所有進行中的請求釋放後才取得獨佔鎖，重置期間新請求會排隊等候。
func (g *ConcurrencyGuard) BeginReset() (release func()) {
	g.mu.Lock()
	return g.mu.Unlock
}

// Package pgxdb 提供跨 repository 共用單一 pgx 事務的最小機制。
package pgxdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier 為 *pgxpool.Pool 與 pgx.Tx 共同的最小介面，repository 可不分是否在事務中直接使用。
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ctxKey struct{}

// TxRunner 負責開啟／提交／回滾單一 pgx 事務，並將其掛載於 context 供 repository 取用。
type TxRunner struct {
	pool *pgxpool.Pool
}

// NewTxRunner 建立 TxRunner 實例。
func NewTxRunner(pool *pgxpool.Pool) *TxRunner {
	return &TxRunner{pool: pool}
}

// WithTx 在單一事務中執行 fn；fn 回傳 nil 時提交，否則回滾並回傳原始錯誤。
func (r *TxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	return fn(context.WithValue(ctx, ctxKey{}, tx))
}

// FromContext 取得目前 context 綁定的事務；若無則回傳 fallback（通常是 repository 自身的 pool）。
func FromContext(ctx context.Context, fallback Querier) Querier {
	if tx, ok := ctx.Value(ctxKey{}).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

// TxFromContext 取得目前 context 綁定的 pgx.Tx，第二個回傳值標示是否存在。
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(ctxKey{}).(pgx.Tx)
	return tx, ok
}

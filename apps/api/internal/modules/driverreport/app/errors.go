package app

import (
	"errors"
	"fmt"
)

// inputError marks a service-layer error whose message is a deterministic,
// pre-written Chinese description of what the caller needs to fix (a bad ID
// format, a missing required combination of fields, an unresolvable
// reference) — as opposed to an underlying system/DB failure. The transport
// layer uses IsInputError to decide between a 400 (show the reason) and a
// 500 (log only), instead of collapsing every service error to one status
// code regardless of cause.
type inputError struct{ msg string }

func (e *inputError) Error() string { return e.msg }

// newInputError builds a user-facing validation error. Only call this with a
// literal format string and values that are themselves already safe to
// display (IDs the caller supplied, field names) — never with a wrapped
// system error's raw text.
func newInputError(format string, args ...interface{}) error {
	return &inputError{msg: fmt.Sprintf(format, args...)}
}

// IsInputError reports whether err (or an error it wraps) is a user-input
// validation error safe to surface verbatim to the caller.
func IsInputError(err error) bool {
	var ie *inputError
	return errors.As(err, &ie)
}

// ErrRowConflictAlreadyResolved 代表要裁決的同車同個案衝突已被其他人處理過或不存在；
// module_adapters.go 的 driverReportRideIngestor 轉接 ride 模組的
// app.ErrRowConflictAlreadyResolved 時應改用這個本模組自有的 sentinel，讓 transport
// 層不必跨模組 import ride/app 就能用 errors.Is 判斷（見 layering-rules.md 模組邊界）。
// 目前 module_adapters.go 尚未接上這個轉譯，transport 層先以錯誤文字比對作為過渡措施，
// 詳見 driver_report_handler.go 的 respondRowConflictError。
var ErrRowConflictAlreadyResolved = errors.New("row conflict already resolved")

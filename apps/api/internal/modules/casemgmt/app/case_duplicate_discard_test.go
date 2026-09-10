package app

import (
	"context"

	"github.com/google/uuid"
)

// 疑似重複個案「忽略此筆」的測試替身與案例都定義在 discard_duplicate_test.go；
// 這裡只補上 DuplicateStagingStore 介面中該路徑用不到、但必須存在才能滿足介面的方法。

func (f *fakeDuplicateStagingStore) Insert(context.Context, DuplicateCandidate) (uuid.UUID, bool, error) {
	return uuid.Nil, false, nil
}

func (f *fakeDuplicateStagingStore) ListPending(context.Context) ([]DuplicateCandidate, error) {
	return f.listResult, nil
}

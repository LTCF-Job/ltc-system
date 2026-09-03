package pgxdb

import "github.com/google/uuid"

// UUIDStrings 將 uuid.UUID 陣列轉為字串陣列，供 SQL 查詢的 ::uuid[] 陣列參數使用。
func UUIDStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

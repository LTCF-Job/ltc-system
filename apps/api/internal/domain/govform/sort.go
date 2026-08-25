package govform

import "sort"

// SortClaimRows 依據規格書 7.5 執行多案與單案的申報行排序。
func SortClaimRows(rows []ClaimRow, isMultiCase bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		// 1. 若為單檔多案模式，先依身分證字號排序
		if isMultiCase && rows[i].NationalID != rows[j].NationalID {
			return rows[i].NationalID < rows[j].NationalID
		}

		// 2. 趟次序列 (LegSeq) 升冪：leg1 全月 -> leg2 全月 -> leg3 全月 -> leg4 全月
		if rows[i].LegSeq != rows[j].LegSeq {
			return rows[i].LegSeq < rows[j].LegSeq
		}

		// 3. 方向：去程 (outbound) 優先於回程 (inbound)
		if rows[i].Direction != rows[j].Direction {
			return rows[i].Direction == "outbound"
		}

		// 4. 服務日期升冪
		return rows[i].ServiceDate.Before(rows[j].ServiceDate)
	})
}

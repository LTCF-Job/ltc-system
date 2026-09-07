-- 匯入冪等改由逐列比對（reconcileRideSource）保證，檔案雜湊 claim 已無讀取端。
-- 保留這張表只會讓「同一份檔案不能重傳」這個已移除的語意看起來還存在。
DROP TABLE IF EXISTS driver_report_imports;

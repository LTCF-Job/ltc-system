# apps/api

Go + Gin 後端。沒用 ORM，pgx 直接寫 SQL；沒用 DI 框架，全部在 `cmd/server/main.go` 手動 wiring。

## 跑起來

```bash
go run ./cmd/server          # 啟動 API，預設吃 .env 或環境變數
go run ./cmd/migrate up      # 套用 migration
go test ./...                # 單元測試
go vet ./...                  # 靜態檢查
```

必要環境變數（沒設會直接啟動失敗，看 `internal/config/config.go`）：

```
APP_ENV=local            # 只接受 local 或 production，其他值直接拒絕啟動
DATABASE_URL=postgres://postgres:postgres@localhost:5432/ltc_system?sslmode=disable
ENCRYPTION_KEY=<32 bytes base64>
HMAC_KEY=<32 bytes base64，不可跟 ENCRYPTION_KEY 一樣>
```

`APP_ENV=production` 時另外強制要求 `SUPABASE_JWKS_URL` 跟 `ALLOWED_ORIGINS`，缺一個就直接拒絕啟動。

## 技術文件

這份 README 只放怎麼跑起來。架構、API、業務流程分開放在 [`docs/tech/`](../../docs/tech/README.md)：

- [後端框架與分層架構](../../docs/tech/backend-framework.md)
- [後端 API 路由總覽](../../docs/tech/backend-api-reference.md)
- [後端核心業務流程](../../docs/tech/backend-flows.md)

## Migration

`apps/api/migrations/` 底下 `NNNNNN_描述.up.sql` / `.down.sql` 成對出現，`cmd/migrate` 依檔名數字順序執行。**不要**同時啟用 Supabase CLI 的 migration（`supabase/migrations`），會產生兩套 migration history 互相打架。加欄位或加表就照現有檔名規則加一組新的 up/down。

## 測試

測試檔跟被測檔同層放（`ride_service_test.go` 緊鄰 `ride_service.go`），用 `testify` 斷言，table-driven 為主。細部慣例看 [`.agents/skills/golang-unit-testing/SKILL.md`](../../.agents/skills/golang-unit-testing/SKILL.md)。CI 實際跑 `go vet ./...` 跟 `go test -race ./...`，本機開發至少跑一次 `-race` 版本再推。

## 匯出 Job（cmd/exporter）

跟 `cmd/server` 共用同一份 `config`／`service`／`repository`，是獨立的批次程式，不常駐。本機測試直接 `go run ./cmd/exporter <期別> <地區>`，流程見 [後端核心業務流程](../../docs/tech/backend-flows.md)。

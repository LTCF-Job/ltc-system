---
name: excel-import-export-integrity
description: Use when adding or changing an Excel/CSV import, template-download, or export feature (apps/api spreadsheet handler or apps/web mock/demo download endpoint) — verifies the produced file actually opens instead of trusting that no error was returned.
---

# Excel/CSV import-export integrity

`excelize.NewFile().WriteToBuffer()` returning `nil` error does not prove the bytes open in Excel. The one confirmed failure mode in this repo: a `POST/GET .../template` or export endpoint returns bytes that Excel reports as corrupted, because a layer between "business logic ran without error" and "bytes reached the browser" produced or forwarded invalid binary — most often the MSW mock for that same endpoint returning `new Blob(['some string'])` instead of a real workbook.

## Verification steps (do all that apply)

1. **Backend render function**: after generating bytes (`RenderXxxTemplate`, an export renderer), round-trip them — `excelize.OpenReader(bytes.NewReader(result))` and `f.GetRows(sheet)` — in a test. A render function is only proven correct when something re-opens what it produced; `err == nil` from the write call alone is not that proof.
2. **HTTP handler that serves those bytes** (`c.Data`, `c.Writer.Write`): run one `httptest` call through the real handler (and router if middleware sits in front of it) and feed `recorder.Body` back through `excelize.OpenReader`. This catches corruption from middleware or response-writing code between the render function and the wire, not just the render function itself.
3. **MSW mock for the same endpoint** (`apps/web/src/mocks/handlers/*.ts`): if the endpoint is reachable in mock/demo mode, its response must be a real, parseable workbook — never a placeholder string wrapped in `Blob`. Use `apps/web/src/mocks/utils/mockExcel.ts`'s `createMockExcelBlob()` for a generic valid `.xlsx`, or add a `createXxxTemplateExcelBlob()` there (base64 of a real generated workbook) when the mock needs entity-specific headers. This is the same rule [[mock-and-demo-boundaries]] states generically ("mock responses aligned with the public API contract"); this skill exists because that general phrasing did not stop a hand-typed string from being used as fake `.xlsx` content.
4. **Content-Type and filename extension** must match the actual byte format served — an `.xlsx` filename with `text/csv` bytes (or vice versa) opens as corrupted even when the bytes themselves are valid for their real format.

## Self-check before calling the feature done

- [ ] The render function's own bytes were re-opened by a parser, not just written without error.
- [ ] If an HTTP handler serves those bytes, the full handler (with middleware) was exercised once and its response body re-opened by a parser.
- [ ] Every mock/demo handler for the same download endpoint returns a real parseable file, not a string literal.
- [ ] Content-Type and the filename's extension agree with the actual byte format.
---
name: excel-import-export-integrity
description: Use when adding or changing an Excel import, template-download, or export feature (apps/api spreadsheet handler or apps/web mock/demo download endpoint) — verifies the produced file actually opens, the mock/demo path mirrors production, and no CSV option gets reintroduced (this project is .xlsx-only).
---

# Excel import-export integrity

CSV support was removed project-wide (2026-08-30, both `apps/api` and `apps/web`) — every import/export feature is `.xlsx`-only now. Do not add a CSV format option (query param, dropdown, `accept=".csv"`) to a new or existing import/export feature; if a task seems to call for one, confirm with the user first rather than reintroducing it by default.

Three confirmed failure modes in this repo, all from the same root cause — mock/demo code for an import/export endpoint was made to satisfy "returns something" instead of "returns/reads what production returns/reads":

1. A `POST/GET .../template` or export endpoint returns bytes Excel reports as corrupted, because the MSW mock for that endpoint returned `new Blob(['some string'])` instead of a real workbook.
2. A downloaded template opens fine but shows the wrong columns, because the mock returned a generic unrelated workbook (or stale demo data) instead of content matching the entity's actual current fields.
3. A mock/demo import's dry-run preview ignores the file the user actually uploaded and always shows the same canned rows — correct-looking but disconnected from reality, which is just as confusing as failure mode 1 once a user notices.

`excelize.NewFile().WriteToBuffer()` returning `nil` error proves neither of the first two — the file being openable and the file being correct are two separate things to verify.

## Governing rule: mock 比照正式 (mock mirrors production)

The mock/demo path for an import/export feature is not a separate, looser implementation — it is a stand-in for the real one and must match it: same columns in the same order, same field names, same required/optional split, same warning/error categories, same sample data shape. Treat "the mock returns *a* valid file" as insufficient; the acceptance bar is "the mock returns *the* file production would return, or a byte-identical embed of it."

## Verification steps (do all that apply)

1. **Backend render function**: after generating bytes (`RenderXxxTemplate`, an export renderer), round-trip them — `excelize.OpenReader(bytes.NewReader(result))` and `f.GetRows(sheet)` — in a test. A render function is only proven correct when something re-opens what it produced; `err == nil` from the write call alone is not that proof.
2. **HTTP handler that serves those bytes** (`c.Data`, `c.Writer.Write`): run one `httptest` call through the real handler (and router if middleware sits in front of it) and feed `recorder.Body` back through `excelize.OpenReader`. This catches corruption from middleware or response-writing code between the render function and the wire, not just the render function itself.
3. **MSW mock template/export for the same endpoint** (`apps/web/src/mocks/handlers/*.ts`): generate the mock blob from the real backend renderer's actual output, base64-embedded via `apps/web/src/mocks/utils/mockExcel.ts`'s `createXxxTemplateExcelBlob()` pattern — run the real `RenderXxxTemplate` once (a throwaway in-module Go test that writes the bytes to a file, or `base64.StdEncoding.EncodeToString`), embed that exact base64, and diff-check the decoded bytes against the original generation to confirm byte-identity. The shared generic `createMockExcelBlob()` is only acceptable as a placeholder before the entity has a real template to embed, never as the final state for a shipped feature — a valid-but-wrong-columns file is corruption from the user's point of view even though Excel opens it without complaint.
4. **MSW mock dry-run/commit for the same import endpoint must genuinely parse the uploaded file**, not return canned rows regardless of input. Read the `File` from `request.formData()` and parse it with `apps/web/src/mocks/utils/parseImportFile.ts`'s `readXlsxRows`/`buildColumnIndex` (built on the `xlsx` package), then re-implement the same row-by-row validation the real backend's parse/commit functions do: same required-field skip rules, same `field` values on warnings, same message strings (copy them verbatim, don't paraphrase) — so a screenshot of the mock UI is indistinguishable from production, and uploading your own file actually reflects in the preview instead of showing someone else's example data. Install `xlsx` from SheetJS's own CDN tarball (`npm install https://cdn.sheetjs.com/xlsx-<version>/xlsx-<version>.tgz`), not `npm install xlsx` from the npm registry — the registry package has open, unpatched high-severity advisories (prototype pollution, ReDoS) that SheetJS only fixes in their own distribution channel.
5. **Never put template instructional text as a plain cell value below the sample data rows** — a row-by-row parser (real backend or a genuine mock parser) has no natural end-of-data marker, so any non-empty trailing row gets read as a phantom data row and fails required-field validation. Attach such notes as a cell comment on a header cell instead (`excelize`'s `f.AddComment(sheet, excelize.Comment{Cell: "A1", Text: "..."})`) so `GetRows`/a spreadsheet parser never returns it as a row.
6. **Demo/mock sample data for the same entity** (`apps/web/src/mocks/data/mockData.ts`): when the feature adds or changes a field, add it to that entity's mock rows too, and bump `DEMO_DATA_VERSION` in `apps/web/src/mocks/data/demoStore.ts` — otherwise a browser with a persisted `localStorage` snapshot keeps showing the pre-change shape indefinitely, which looks identical to "the mock wasn't updated" from the user's side even after the array literal in `mockData.ts` is correct.
7. **Content-Type and filename extension** must match the actual byte format served — an `.xlsx` filename with the wrong content type opens as corrupted even when the bytes themselves are valid.
8. **Template/export field coverage when the task doesn't name specific columns**: diff the import template's header row and the export's column list against the entity's current field set (DTO/model + any fields the same feature just added). A field missing from the template silently drops on import, and one missing from the export silently hides data that's otherwise visible in the API/UI — treat each as a gap to close, not an out-of-scope difference.

## Self-check before calling the feature done

- [ ] No CSV format option was added or left in place — the feature is `.xlsx`-only.
- [ ] The render function's own bytes were re-opened by a parser, not just written without error.
- [ ] If an HTTP handler serves those bytes, the full handler (with middleware) was exercised once and its response body re-opened by a parser.
- [ ] The mock/demo template or export blob is a byte-identical embed of the real backend's output (or is explicitly a placeholder, not the shipped state), not a generic or unrelated file.
- [ ] The mock dry-run/commit endpoint actually parses the uploaded file's rows (via `parseImportFile.ts`) and applies the same validation as the real backend, rather than returning fixed demo rows.
- [ ] `xlsx` (if newly added) was installed from SheetJS's own CDN tarball, not the vulnerable npm-registry package.
- [ ] Any instructional/note text in a generated template is a cell comment, not a plain trailing row.
- [ ] Any field this feature added to the entity was added to its mock sample data, and `DEMO_DATA_VERSION` was bumped.
- [ ] Content-Type and the filename's extension agree with the actual byte format.
- [ ] When no specific columns were requested, the template/export column list was diffed against the entity's current fields, including any just added by this feature.

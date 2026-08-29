# apps/api layering rules

Concrete import matrix, model ownership, and file-splitting triggers for the `apps/api` Go module. The principles behind them are in [SKILL.md](../SKILL.md).

`apps/api/internal/arch/arch_test.go` encodes these rules. A rule that is not encoded there is advice; a rule that is encoded there fails `go test ./...`.

## 1. Package roles

| Package | Role |
| --- | --- |
| `cmd/server` | Composition root: builds dependencies, wires routes. `main.go` constructs, `routes.go` registers. |
| `internal/modules/<capability>/transport` | Gin handlers and API DTOs; binds input, calls `app`, assembles the response envelope. |
| `internal/modules/<capability>/app` | Use cases, business policy, and the port interfaces this module consumes. |
| `internal/modules/<capability>/infra` | SQL queries, transaction mechanics, external-provider calls; implements `app` ports. |
| `internal/platform/*` | Shared technical kernel: `config`, `httpx`, `auth`, `logging`, `pgxdb`. |
| `internal/domain/*` | Shared business kernel: stable concepts, value objects, pure functions. |
| `internal/arch` | Architecture tests only; contains no production code. |

## 2. Capability map

Each module owns one business capability. Put new work in the module that already owns the concept; a new module is for a capability none of these covers.

| module | Owns |
| --- | --- |
| `masterdata` | Sites, vehicles, drivers, regions, driver–vehicle assignment |
| `casemgmt` | Case master data, schedules, transport preference, case profile workbook |
| `caseimport` | Batch Excel/CSV parsing, preview, and commit of cases |
| `ride` | Google-form ingestion, ride-record merge, manual correction, conflict resolution |
| `formsync` | Google form registration and column mapping |
| `reporting` | Trip summary, Hsinchu schedule, dashboard, precheck, government claim export |
| `ops` | Driver attendance, fuel logs, vehicle maintenance |
| `notification` | Notification recipients and delivery log |
| `holiday` | Public holidays and government-calendar sync |
| `audit` | Audit-log writes and queries; the only owner of `audit_log` SQL |
| `task` | Missing-report detection and month-end scheduled jobs |

## 3. Import matrix

`ok` = allowed, blank = rejected. The flat layer-first packages are gone; every
capability lives under `internal/modules/`, and `arch_test.go`'s baseline is
empty. A new violation is a defect to fix, not an entry to add.

| from \ to | transport | app | infra | platform | domain |
| --- | --- | --- | --- | --- | --- |
| `cmd/server` | ok | ok | ok | ok | ok |
| `<mod>/transport` | own module | own module | | ok | ok |
| `<mod>/app` | | own module | | ok | ok |
| `<mod>/infra` | | own module | own module | ok | ok |
| `platform/*` | | | | ok | ok |
| `domain/*` | | | | | ok |

`infra` imports its own `app` to implement that module's port types, and for nothing else. Module A reaches module B only through a port that `cmd/server` injects — see section 4.

Third-party confinement:

| Package | Allowed only in |
| --- | --- |
| `github.com/gin-gonic/gin` | `*/transport`, `platform/{auth,httpx,logging}` |
| `github.com/jackc/pgx` | `*/infra`, `platform/pgxdb` |
| `github.com/xuri/excelize` | the `infra` of a module whose deliverable is a spreadsheet: `reporting`, `caseimport`, `ops`, `casemgmt` |
| `google.golang.org/api` | `formsync/infra` |
| anything outside stdlib and `golang.org/x/text` | rejected in `internal/domain/**` |

## 4. Cross-module collaboration

A use case that needs data another module owns declares what it needs and lets the composition root supply it. Three steps:

1. Declare the port in the **consuming** module's `app/ports.go`, with the narrowest signature the use case needs, using types that module owns (`ride/app.DriverRef`, not `masterdata/app.Driver`).
2. Write the adapter in `cmd/server`: `module_adapters.go` for data lookups, `audit_adapters.go` for audit writes. The adapter holds the providing module's repository or service and converts to the consuming module's types.
3. Inject it in `main.go`.

These two files are the only place that knows which module talks to which. An `import` of another module's package from inside a module fails `TestImportMatrix`.

Audit follows the same shape rather than being special-cased: `audit` owns the SQL, every other module declares its own `AuditWriter` port over its own `AuditEntry` type.

## 5. Model ownership

Four model families; each is converted explicitly at the boundary that owns the inner side.

| Family | Home | Tags |
| --- | --- | --- |
| API DTO | `<mod>/transport/<cap>_dto.go` | `json:` and `binding:` — the only place `binding:` appears |
| Application model | `<mod>/app` | none |
| Domain model | `internal/domain/*` | none |
| Persistence row | `<mod>/infra/<cap>_rows.go` | none |

A struct carrying a `binding:` tag that also appears in a `rows.Scan()` call site is a defect: it makes the database column layout the public API contract.

## 6. Where validation lives

| Check | Layer |
| --- | --- |
| Shape, type, required field | `transport`, via `binding:` on the DTO |
| Business rule, uniqueness, state transition, cross-entity consistency | `app` |
| Invariant that is always true of the concept | `domain` |
| Database constraint | Safety net behind one of the above, never the primary check |

## 7. Error ownership

- The layer that makes the decision declares the sentinel: `app` for business outcomes, the domain package for invariant violations.
- Cross-boundary wrapping uses `%w`; the sentinel identity survives to the handler.
- Handlers compare with `errors.Is`. `region_handler.go` is the working example.
- Exactly one place maps a sentinel to an HTTP status and error code: `<mod>/transport/errors.go`.
- Error codes come from the `httpx` constants.
- Response bodies carry a stable message; the underlying `err` goes to the log.

## 8. Transaction ownership

- Atomicity within one aggregate lives inside a single `infra` method.
- Atomicity across repositories is opened by `app` through `platform/pgxdb.TxRunner`, which passes a transaction-bound context down. `infra` methods accept that context and join the existing transaction.
- A write that must not exist without its audit record participates in the caller's transaction. Discarding the audit error (`_ = auditRepo.Insert(...)`) is acceptable only on read paths.
- Partial-commit semantics are a decision to state in the use case's doc comment: all-or-nothing, or per-row savepoint with a skip list.

## 9. Port ownership

- Ports are consumer-side: declared in `<mod>/app/ports.go`, named for what the use case needs (`DriverStore`, `HolidayProvider`), not for the implementation.
- Port signatures use application or domain types. Vendor SDK types and persistence rows stay behind the port.
- Define a port once a second implementation or a test double exists.

## 10. Naming

```
<mod>/transport/  <cap>_handler.go  <cap>_dto.go  errors.go
<mod>/app/        <cap>_service.go  ports.go  errors.go
<mod>/infra/      <cap>_repo.go     <cap>_rows.go
```

Types: `<Cap>Handler`, `<Cap>Service`, `<Cap>Repository`, `<Cap>Row`. Constructors: `New<Cap>X`.

## 11. Demo and mock boundary

Classification and activation rules are in [mock-and-demo-boundaries](../../mock-and-demo-boundaries/SKILL.md). Three Go-specific additions:

- Business data reaches a handler through a port. A literal record inside a production handler is a defect.
- An operation the handler cannot perform returns an error. A success envelope over a write that never happened (`{"updated": true}`) reports data loss as success.
- Offline and nil-dependency fallback is a config-gated decision made in `cmd/server`. Inside a use case, `if repo == nil` hides it.

## 12. File-splitting triggers

Length alone is a signal, not a defect. Split when two or more hold:

- more than ~450 lines
- three or more distinct responsibilities (parsing, policy, persistence orchestration, rendering)
- two capabilities in one file
- two unrelated fixture sets needed to test it

Split along business capability, inside the owning module.

## 13. Adding an endpoint

Pick the owning module from section 2 first, then:

1. Request and response DTOs in `<mod>/transport/<cap>_dto.go`, with `binding:` for shape checks.
2. Use case method in `<mod>/app/<cap>_service.go`; new dependencies become methods on `<mod>/app/ports.go`.
3. Implementation in `<mod>/infra/<cap>_repo.go`, with row structs in `<cap>_rows.go`. Data another module owns comes in through section 4 instead.
4. Handler in `<mod>/transport/<cap>_handler.go`: bind, call, `httpx.RespondSuccess` / `httpx.RespondError`; sentinel mapping in `errors.go`.
5. Wire the constructor in `cmd/server/main.go` and the route with its `RequireRoles` set in `cmd/server/routes.go`.
6. Run `go test ./...` in `apps/api`. `internal/arch` fails on a boundary the change crossed.

## 14. Adding a module

A capability that section 2 does not cover gets its own `internal/modules/<capability>/{transport,app,infra}`. Beyond the endpoint steps:

- Give the module its own application models even where they mirror another module's; shared model types are how two modules become one.
- Add the module to section 2, and to `excelModules` in `arch_test.go` when its deliverable is a spreadsheet.
- Route every dependency on an existing module through section 4.

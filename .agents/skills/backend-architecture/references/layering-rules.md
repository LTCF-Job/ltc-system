# apps/api layering rules

Concrete import matrix, model ownership, and file-splitting triggers for the `apps/api` Go module. The principles behind them are in [SKILL.md](../SKILL.md).

`apps/api/internal/arch/arch_test.go` encodes these rules. A rule that is not encoded there is advice; a rule that is encoded there fails `go test ./...`.

This file describes both the **current** layout (`transport`/`app`/`infra`, enforced today) and the **target** layout (`domain`/`application`/`adapters`, defined in [`docs/tech/application-architecture.md`](../../../../docs/tech/application-architecture.md)). A module keeps its current names until it has actually been sliced into the target layout *and* `arch_test.go` has been extended to recognize that module's new zones — writing target-named packages before that extension fails `go test ./...`. See "Known gaps in the architecture test" below before relying on baseline being empty as proof of coverage.

## 1. Package roles

| Package (current) | Package (target) | Role |
| --- | --- | --- |
| `cmd/server` | `internal/bootstrap` | Composition root: builds dependencies, wires routes, cross-module bridges, jobs. `main.go` constructs, `routes.go` registers. |
| `internal/modules/<capability>/transport` | `internal/modules/<capability>/adapters/http` | Gin handlers and API DTOs; binds input, calls the use-case layer, assembles the response envelope. |
| `internal/modules/<capability>/app` | `internal/modules/<capability>/application` | Use cases, business policy, and the port interfaces this module consumes. |
| `internal/modules/<capability>/infra` | `internal/modules/<capability>/adapters/postgres` (or `/spreadsheet` for excel adapters) | SQL queries, transaction mechanics, external-provider calls; implements the application layer's ports. |
| — | `internal/modules/<capability>/domain` | Module-local aggregates, value objects, domain policy, domain errors. New in the target layout; a module gains this package only when it is sliced. |
| `internal/platform/*` | `internal/platform/*` (unchanged) | Shared technical kernel: `config`, `httpx`, `auth`, `logging`, `pgxdb`. |
| `internal/domain/*` | `internal/sharedkernel` | Shared business kernel: stable concepts, value objects, pure functions used across contexts. |
| `internal/arch` | `internal/arch` (unchanged) | Architecture tests only; contains no production code. |

A module is in the current layout until it has been sliced; only then does it gain a `domain` package and rename `transport`/`app`/`infra` to `adapters/http`/`application`/`adapters/postgres`. Mixed layouts across different modules are expected during the migration.

## 2. Capability map

Each module owns one business capability. Put new work in the module that already owns the concept; a new module is for a capability none of these covers.

| module | Owns |
| --- | --- |
| `masterdata` | Sites, vehicles, drivers, driver–vehicle assignment |
| `caregiver` | Caregiver master data and its batch Excel/CSV import |
| `casemgmt` | Case master data, schedules, transport preference, case profile workbook |
| `caseimport` | Batch Excel/CSV parsing, preview, and commit of cases. Target: merge into `casemgmt` (`docs/tech/application-architecture.md` §3); `casemgmt/app` is already 1128 lines and `caseimport/app` adds ~1016 more, so the merge needs a file-splitting plan under section 12 before it lands, not after |
| `ride` | Ride-record merge (consumes parsed driver-report data via a port), ride calendar, manual correction, conflict resolution |
| `driverreport` | Driver pickup-report form registration, `.xlsx` import, parsing, and column mapping — owns ingestion; `ride` consumes it through `driverreport/app.RideIngestor` |
| `reporting` | Trip summary, Hsinchu schedule, dashboard, precheck, government claim export |
| `ops` | Driver attendance, fuel logs, vehicle maintenance |
| `notification` | Notification recipients and delivery log |
| `holiday` | Public holidays and government-calendar sync |
| `audit` | Audit-log writes and queries; the only owner of `audit_log` SQL |
| `task` | Missing-report detection and month-end scheduled jobs |
| `identity` | User accounts and roles: Supabase Auth Admin API wiring, self password change, the `roles` table |

## 3. Import matrix

`ok` = allowed, blank = rejected. The flat layer-first packages are gone; every
capability lives under `internal/modules/`, and `arch_test.go`'s baseline is
empty. A new violation is a defect to fix, not an entry to add.

Current layout (in force for every module that has not been sliced yet):

| from \ to | transport | app | infra | platform | domain |
| --- | --- | --- | --- | --- | --- |
| `cmd/server` | ok | ok | ok | ok | ok |
| `<mod>/transport` | own module | own module | | ok | ok |
| `<mod>/app` | | own module | | ok | ok |
| `<mod>/infra` | | own module | own module | ok | ok |
| `platform/*` | | | | ok | ok |
| `domain/*` | | | | | ok |

Target layout (in force once a module is sliced and `arch_test.go` recognizes its new zones):

| from \ to | adapters/http | application | adapters/postgres | domain (own module) | domain (other module) | platform | sharedkernel |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `bootstrap` | ok | ok | ok | | | ok | ok |
| `<mod>/adapters/*` | own module | own module | own module | | | ok | ok |
| `<mod>/application` | | own module | | ok | | ok | ok |
| `<mod>/domain` | | | | | | | ok |
| `platform/*` | | | | | | ok | ok |
| `sharedkernel` | | | | | | | ok |

`<mod>/domain` is module-local: it must not be imported by any other module's package. `arch_test.go`'s current `allowedInternal` does not enforce this yet — see "Known gaps in the architecture test" below.

`infra`/`adapters/postgres` imports its own `app`/`application` to implement that module's port types, and for nothing else. Module A reaches module B only through a port that `cmd/server`/`bootstrap` injects — see section 4.

Third-party confinement:

| Package | Allowed only in (current) | Allowed only in (target) |
| --- | --- | --- |
| `github.com/gin-gonic/gin` | `*/transport`, `platform/{auth,httpx,logging}` | `*/adapters/http`, `platform/{auth,httpx,logging}` |
| `github.com/jackc/pgx` | `*/infra`, `platform/pgxdb` | `*/adapters/postgres`, `platform/pgxdb` |
| `github.com/xuri/excelize` | the `infra` of a module whose deliverable is a spreadsheet: `reporting`, `caseimport`, `ops`, `casemgmt`, `caregiver`, `driverreport` | the `adapters/spreadsheet` of the same modules |
| anything outside stdlib and `golang.org/x/text` | rejected in `internal/domain/**` | rejected in `internal/sharedkernel/**` and `<mod>/domain/**` |

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
- Handlers compare with `errors.Is`. `site_handler.go` is the working example.
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

Current (in force until a module is sliced):

```
<mod>/transport/  <cap>_handler.go  <cap>_dto.go  errors.go
<mod>/app/        <cap>_service.go  ports.go  errors.go
<mod>/infra/      <cap>_repo.go     <cap>_rows.go
```

Types: `<Cap>Handler`, `<Cap>Service`, `<Cap>Repository`, `<Cap>Row`. Constructors: `New<Cap>X`.

Target (once a module is sliced):

```
<mod>/domain/            <cap>.go          <cap>_policy.go   errors.go
<mod>/application/       <cap>_service.go  ports.go          errors.go
<mod>/adapters/http/     <cap>_handler.go  <cap>_dto.go      errors.go
<mod>/adapters/postgres/ <cap>_repo.go     <cap>_rows.go
```

Type and constructor names are unchanged by the rename.

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

## 15. Known gaps in the architecture test

`arch_test.go`'s baseline being empty proves today's `transport`/`app`/`infra` layout has no violations — it does not prove the target `domain`/`application`/`adapters` layout is protected. Five gaps in the current implementation stay open until `arch_test.go` is extended; do not slice a module into the target layout before its gap is closed:

| Gap | Where | Effect |
| --- | --- | --- |
| `allowedInternal` allows any zone to import `to.kind()=="domain"` unconditionally | `arch_test.go:54-57`, combined with `zone.kind()` (`:31-37`) reading `parts[2]` for `mod/<module>/domain` | Once a module gains a `domain` package, every other module can import it directly with no test failure — the module-local boundary in section 3 is unenforced |
| `zoneOf` returns `""` for an unrecognized first path segment, and the walker skips `""` zones | `arch_test.go:357`, `:389` | `internal/sharedkernel/` and `internal/bootstrap/` are invisible to every check |
| The scan root is `internal/`, not the repo root | `arch_test.go:20` (`internalRoot = ".."`) | `cmd/server/` (the pre-rename composition root) has never been scanned |
| `externalConfinement` matches third-party imports against the literal current-layout kind strings `"transport"`/`"infra"` | `arch_test.go:76-87` | `adapters/http` and `adapters/postgres` would be rejected for importing gin and pgx — the confinement rules need a target-aware update, not just relaxing |
| `isPersistencePackage` matches on `strings.HasSuffix(name, "infra")` | `arch_test.go:335` | Renaming a package to `postgres` silently disables `TestRequestBodiesUseTransportDTOs` for that package |

Extending `arch_test.go` to close these gaps is part of the migration's stage one (`docs/tech/application-architecture.md` §11), not optional cleanup.

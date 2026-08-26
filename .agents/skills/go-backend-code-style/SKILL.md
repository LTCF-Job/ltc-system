---
name: go-backend-code-style
description: Use when editing or reviewing Go backend source, especially handlers, services, repositories, domain packages, middleware, configuration, dependency wiring, SQL, errors, and API responses.
---

# Go backend code style

Apply this skill to Go backend source changes while following the repository's existing patterns first.

## Core rules

- Read the target file and nearby same-layer examples before editing.
- Keep functions focused on one responsibility and keep control flow easy to scan.
- Use standard Go naming, formatting, error wrapping, and zero-value behavior.
- Match existing receiver, package alias, logging, configuration, and response conventions.
- Avoid speculative helpers, generic wrappers, new DTOs, or abstractions without a concrete caller.
- Preserve special-case checks until their protection purpose is understood.

## Layer ownership

- Handlers or delivery adapters bind input, call the application boundary, and assemble transport responses.
- Services or use cases own business validation, policy, filtering, pagination, and execution flow.
- Repositories own persistence queries, cache access, and transaction mechanics.
- Domain packages own stable business concepts and invariants; avoid placing ordinary infrastructure helpers there.
- Composition-root code owns dependency construction and route, job, or consumer wiring.

Use the repository's actual package names. Do not move code between layers only to match a generic clean-architecture template.

## Errors and boundaries

- Add context when returning errors across a meaningful boundary; preserve sentinel or typed error identity when callers depend on it.
- Map internal errors to transport responses at the delivery boundary.
- Keep database, cache, HTTP-client, and vendor-specific errors out of domain decisions where practical.
- Treat money, time zones, pagination cursors, nil collections, and partial failures explicitly according to existing contracts.
- Keep transaction ownership in one clear repository or adapter entry when atomicity or row locks matter.

## Change workflow

1. Locate the owning layer and the nearest comparable implementation.
2. Make the smallest change that preserves the public contract.
3. Add a focused test when the changed behavior has a testable seam or existing coverage.
4. Run `gofmt` on changed Go files, then the smallest relevant test command.
5. Prefer `go build ./...` when a broader compile check is needed; do not start or restart services as part of this skill.

For architecture decisions, use the backend architecture skill. For test design, use the Go unit-testing skill. Avoid duplicating their full guidance here.

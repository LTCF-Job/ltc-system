---
name: backend-architecture
description: Use when designing, reviewing, or changing backend application structure, especially boundaries between delivery, application, domain, persistence, and external adapters.
---

# Backend architecture

Use this skill to keep backend changes aligned with the repository's existing boundaries. It is architecture guidance, not a mandate to replace the current framework or directory layout.

## Review posture

- Read the target files, nearby same-layer examples, build manifests, tests, and dependency wiring before proposing a boundary change.
- Use the repository's vocabulary. Names such as `handler`, `service`, `usecase`, `repository`, `domain`, and `adapter` are examples, not required package names.
- Prefer the smallest change that improves ownership, cohesion, or testability.
- Treat external templates as concepts. Do not introduce a new layer, framework, service, or package split only to match them.

## Dependency direction

Keep business decisions independent from HTTP, UI, database clients, cache clients, queues, file systems, and vendor SDKs.

- Delivery adapters translate transport input and output.
- Application/use-case code coordinates a business operation and applies business policy.
- Domain code owns stable business concepts and invariants.
- Persistence and integration adapters implement data and external-system details.
- Composition-root code constructs dependencies and wires routes, jobs, or consumers.

Dependencies may point inward from adapters, but inner business code must not import outer delivery or infrastructure details. Follow existing interface ownership when the repository already has a consistent convention.

## Layer decisions

- Put request binding, authentication context extraction, response assembly, and transport error mapping in delivery adapters.
- Put validation, filtering, pagination, status transitions, and execution flow in the application/use-case layer when they are business behavior.
- Put database queries, cache access, transaction mechanics, retries, and provider calls in repositories or integration adapters.
- Put entities, value objects, constants, and business-facing interfaces in domain packages when they are stable concepts.
- Put dependency construction and route/job wiring in the composition root.
- Keep DTOs at the boundary that owns their contract; do not let transport or persistence models become accidental domain models.

## Consistency and failure boundaries

Identify the owner of each transaction, lock, retry, idempotency key, and external side effect. Keep atomic work inside one repository or adapter transaction entry when the data store requires it. Make partial failure, retry behavior, and compensation explicit rather than hiding them in unrelated layers.

## Review workflow

1. Map the request or event path from entrypoint to business decision and side effects.
2. Mark imports and calls that cross delivery, application, domain, persistence, or integration boundaries.
3. Locate mixed responsibilities, duplicated policy, leaked DTOs, and oversized modules.
4. Separate confirmed facts, hypotheses, and runtime evidence.
5. Recommend an incremental migration with a concrete first seam and focused verification.

For each finding, report severity, evidence path and symbol, impact, smallest boundary change, and validation needed. Use the repository's testing and workflow skills for implementation and verification.

## Non-goals

This skill does not require framework migration, directory renaming, microservice extraction, broad refactoring, performance benchmarking, or security penetration testing unless explicitly requested.

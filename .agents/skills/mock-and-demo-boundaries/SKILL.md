---
name: mock-and-demo-boundaries
description: Use when adding, reviewing, or reorganizing mock data, demo fallbacks, offline behavior, seed data, fixtures, or HTTP interception handlers.
---

# Mock and demo boundaries

Keep development and test conveniences explicit, isolated, and faithful to the real contract. This skill applies across frontend and backend code.

## Classify the data first

- `mock`: a controlled replacement for an external dependency during a test or local development run.
- `demo`: intentionally curated content used to demonstrate a product flow.
- `fixture`: stable input and expected output for a test.
- `seed`: data loaded into a real local or test data store.
- `fallback`: behavior used after a real dependency fails; it must be an explicit product decision, not an accidental mock.

Do not mix these purposes in one file or activation path.

## Boundary rules

- Keep mock handlers, fixtures, demo content, and seed scripts in clearly named locations near their owning feature or test boundary.
- Activate them through explicit development or test configuration; production code must not silently import demo data or bypass authorization.
- Keep mock responses aligned with the public API contract, including status codes, errors, pagination, nullability, and response envelopes where applicable.
- Prefer the same adapter or port used by production code so callers do not gain a mock-only shape.
- Keep stateful mocks resettable and deterministic. Never depend on execution order or shared user data unless the test explicitly owns that state.
- Keep seed data separate from schema migrations and make repeatability or idempotency clear.

## Review checklist

Check whether the data has a single purpose, whether activation is visible, whether it can leak into production, and whether it can drift from the real contract. Check that offline mode, fallback behavior, authentication shortcuts, and fake records are labeled and documented at their boundary.

When changing a mock or fixture, update the focused contract, component, or integration tests that prove the intended shape. Use the repository's established interception and fixture tools; a named tool in an example is optional.

## Non-goals

Do not build a second application path, duplicate business rules in demo data, or introduce a generic mock framework without a concrete maintenance problem.

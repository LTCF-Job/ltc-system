---
name: golang-unit-testing
description: Use only when writing or improving Go unit tests, including table-driven tests for domain rules, parsers, formatters, service behavior, and error paths.
---

# Go unit testing

Use this skill when the requested work changes Go test code or explicitly asks for Go unit tests. Follow the target package's existing test style before introducing a library or pattern.

## Test design

- Prefer fast, deterministic unit tests that exercise a public behavior or a narrow testable seam.
- Use table-driven tests when cases share setup and expected structure; give each case a precise name.
- Cover successful behavior, validation failures, boundary values, nil or empty input, and meaningful dependency errors.
- Assert observable outcomes: return values, errors, state changes, calls, and contract fields.
- Avoid asserting incidental implementation details such as private helper order or exact SQL unless it is the contract under test.

## Dependency boundaries

Replace databases, caches, networks, queues, file systems, clocks, randomness, and vendor clients with small fakes, stubs, or mocks when the unit under test does not own those details. Keep integration and contract tests separate when real infrastructure is required.

Use the package's established assertion and mocking tools. Do not add a new testing dependency for a single assertion when the standard library or existing helper is sufficient.

## Test workflow

1. Read the implementation, nearby tests, and package contract.
2. Identify the smallest stable seam and define the expected behavior.
3. Add focused cases, including the relevant error path.
4. Run the changed package's focused test command, then its broader package tests when useful.
5. Report external services or runtime conditions that were not exercised.

Keep tests readable and local. Do not turn unit tests into hidden end-to-end tests by starting services or depending on shared mutable data.

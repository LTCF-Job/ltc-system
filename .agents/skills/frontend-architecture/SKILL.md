---
name: frontend-architecture
description: Use when designing, reviewing, or changing a Vue 3 frontend structure, including pages, feature modules, composables, API clients, state, and TypeScript contracts.
---

# Frontend architecture

Use this skill to keep Vue 3 frontend changes understandable, testable, and aligned with the repository's existing conventions.

## Review posture

- Read the current entrypoints, router, representative pages, feature modules, API clients, stores, composables, and tests before proposing a restructure.
- Prefer feature-oriented boundaries over a large global folder of unrelated components when the codebase has enough scale to justify them.
- Preserve existing framework and library choices unless the user explicitly requests a migration.
- Do not create a new abstraction merely to make folders look uniform.

## Responsibility boundaries

- Pages/views compose a screen and coordinate route-level data and layout.
- Feature modules own user-facing workflows, feature components, local models, and feature-specific behavior.
- Presentational components render props and emit events; they should not silently fetch unrelated data or mutate global state.
- Composables own reusable reactive behavior and lifecycle coordination.
- API clients own transport details, serialization, authentication headers, and error normalization.
- Stores own durable client state and cross-view coordination; keep short-lived view state local.
- Server-state or query-cache tools, when present, own request caching, invalidation, loading, and retry state.
- TypeScript contracts should distinguish API DTOs, form models, view models, and domain-like client models when their responsibilities differ.

## Data-flow rules

Keep a visible one-way path from route or user action to feature behavior, API client, state update, and rendered result. Normalize API errors at a stable boundary. Avoid duplicating the same request, mapping, pagination, or permission rule in multiple pages.

Keep mock, fixture, seed, and demo behavior behind an explicit development or test boundary. It must not silently replace production requests or alter production authorization behavior.

## Review workflow

1. Trace representative screens from route entry to network request and rendered state.
2. Identify cross-feature imports, global-state overuse, duplicated API mapping, and components with mixed responsibilities.
3. Check loading, empty, error, retry, pagination, and permission states.
4. Check that public contracts and generated or shared types match the actual API.
5. Recommend the smallest feature seam and focused unit/component/browser verification.

Use the repository's established Vue, TypeScript, testing, and mock-tool conventions. A library named in an example is optional; the existing project convention takes precedence.

## Non-goals

This skill does not require a framework migration, design-system replacement, global state rewrite, folder-wide rename, or visual redesign unless explicitly requested.

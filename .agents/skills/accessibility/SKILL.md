---
name: accessibility
description: Use when designing, reviewing, or changing Vue 3 user interfaces where keyboard access, screen-reader semantics, focus management, forms, tables, charts, responsive zoom, or WCAG-oriented verification matters.
---

# Frontend accessibility

Apply this skill to the interaction and information semantics of the LTC web app. Keep the existing Vue 3, Element Plus, Vue Router, TypeScript, ECharts, and Playwright choices.

## Review order

1. Trace the route or component from entry point to the user task and its success, loading, empty, error, permission, and disabled states.
2. Identify the semantic structure: page heading, landmarks, navigation, regions, labels, descriptions, tables, status messages, and dialogs.
3. Walk the task with keyboard only, including focus entry, focus order, visible focus, escape or cancel behavior, and focus return.
4. Check the same task at 200% zoom and narrow viewport widths without hiding essential labels, values, errors, or actions.
5. Verify with the affected Playwright flow and targeted static inspection. Treat automated checks as evidence for the tested path, not as proof of complete WCAG conformance.

## Semantic and keyboard rules

- Use native HTML semantics and visible text before adding ARIA. Give each page one meaningful `h1`, with heading levels that describe the content hierarchy.
- Keep navigation, main content, supplementary content, and repeated regions discoverable through landmarks. A route transition should expose the new page title and leave focus in a predictable place.
- Make every interactive operation available from the keyboard. Use a native button or link for actions and navigation; preserve a visible `:focus-visible` indicator with adequate contrast.
- Keep tab order aligned with reading and task order. Do not use positive `tabindex` values, pointer-only hover controls, or an icon-only control without an accessible name.
- For `el-dialog`, `el-drawer`, dropdowns, and custom overlays, verify an accessible name, modal or non-modal behavior, keyboard dismissal where appropriate, focus containment, and return to the invoking control.
- Keep collapsed navigation and permission-filtered navigation truthful: `aria-expanded`, `aria-controls`, visible labels, and available routes must agree with the actual state.

## Forms, tables, and operational data

- Associate every input with a visible label. Connect help text and validation text with descriptions or error semantics, and identify the invalid field without relying on color alone.
- Preserve entered values and explain recoverable submission errors near the affected control. Loading and completion feedback must be announced when it is not already clear from the changed UI.
- Give data tables meaningful headers and an understandable row or cell context. Keep sorting, filtering, pagination, selection, and row actions usable without a mouse.
- Treat status, counts, alerts, and permission results as information with text. Pair color with text, icon, pattern, or value, and use `role="status"` or `role="alert"` only for messages that should be announced.
- Give charts a concise accessible summary or data alternative. Decorative SVG and chart visuals should be hidden from assistive technology; the summary must not invent trends or live values.

## Visual, motion, and responsive checks

- Preserve readable contrast for text, controls, focus indicators, disabled states, and status meaning against the LTC dashboard palette.
- Keep text readable at browser zoom and avoid clipping, forced horizontal scrolling, or information that appears only on hover. Preserve a usable mobile layout for dense tables and navigation.
- Respect `prefers-reduced-motion: reduce`: remove looping or decorative motion and keep state changes understandable without animation. Do not use motion as the only feedback channel.
- Keep touch targets and adjacent actions distinguishable. Tooltips supplement labels; they do not replace names, instructions, or error text.

## Project boundaries

- Preserve route guards, role permissions, API contracts, and Demo/MSW activation boundaries while improving accessibility.
- Reuse Element Plus semantics and existing project components when they meet the task. Add a wrapper or composable only when it creates a stable, reusable accessibility boundary.
- Do not claim an accessibility audit, WCAG conformance, or assistive-technology support from type-check, build, or a single browser test. Report the exact paths and checks performed.

## Completion checklist

- [ ] The page has a meaningful title, heading, landmarks, and readable focus order.
- [ ] Every interactive control has a keyboard path and accessible name.
- [ ] Dialogs, drawers, menus, and collapsed navigation manage focus and state semantics.
- [ ] Forms expose labels, help, validation, and recoverable errors.
- [ ] Tables, charts, statuses, and alerts have text equivalents or meaningful semantics.
- [ ] Color, zoom, viewport size, and reduced motion do not remove essential meaning or operation.
- [ ] The affected Playwright flow or another explicit check records what was verified and what remains unverified.

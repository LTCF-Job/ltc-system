---
name: admin-ui-design
description: Use when designing or reviewing Vue 3 backoffice and operations interfaces such as CRUD workspaces, data tables, filters, bulk actions, approval flows, dashboards, settings, and audit views.
---

# Admin UI design

Design for staff who complete repeated operational tasks with real records. Optimize for orientation, safe decisions, efficient scanning, and recoverable actions. Coordinate with `frontend-architecture` for code boundaries, `ltc-dashboard-visual-language` for the visual system, and `accessibility` for inclusive interaction checks.

## Start from the operator task

1. Identify the user role, goal, record scope, permissions, frequency, and consequence of an error.
2. Define the primary task and the smallest useful path from page entry to completed result.
3. Separate overview, search, detail, edit, approval, and history concerns when combining them would make the next action unclear.
4. Map loading, empty, error, permission-denied, stale-data, unsaved-change, and success states before choosing layout.

## Information architecture

- Use stable navigation labels based on the business task, not internal API or database names.
- Keep one clear page purpose and one dominant primary action. Place secondary and destructive actions near the record or workflow they affect.
- Make the current scope visible: organization, date range, region, status, selected records, and last refresh or data source where relevant.
- Preserve route, role, and permission boundaries. A hidden action must not be the only explanation for why a user cannot complete a task; show the appropriate permission or read-only state.
- Use progressive disclosure for advanced filters, optional columns, and infrequent settings while keeping active filters and applied scope visible.

## Tables and record workspaces

- Choose columns by decisions the operator makes. Keep identifiers, status, owner, time, and next action easy to locate; move supporting fields into detail or expandable context.
- 時間欄位一律格式化至秒數（`YYYY-MM-DD HH:mm:ss`），避免直接顯示毫秒（`.SSS`）或原始時區字尾，並設定足夠欄寬（如 170px）避免文字折行。
- Provide useful default sorting, filtering, pagination, and empty results guidance. Preserve filter state when returning from a detail view when that supports the task.
- Keep row actions predictable and distinguish view, edit, retry, approve, export, and destructive operations by label and placement.
- Support bulk actions only when selection scope, affected count, permission, validation, and result reporting are explicit. Require confirmation for irreversible or high-impact operations.
- Use a detail view, drawer, or dialog according to task depth: keep quick context nearby, while multi-step or high-risk edits deserve a stable page or focused workspace.

## Forms and workflow feedback

- Group fields by the operator's decision order and show requiredness, defaults, constraints, and dependencies close to the field.
- Keep validation actionable and preserve user input on recoverable failures. Explain server-side conflicts, stale records, and permission changes in the context of the attempted action.
- Make save, submit, approve, retry, and cancel states observable. Prevent duplicate submissions while retaining a keyboard-accessible route to completion or recovery.
- Treat destructive actions as a separate workflow with clear scope, consequence, confirmation, and post-action result. Prefer reversible recovery when the domain allows it.
- Use consistent labels and status vocabulary across list, detail, dashboard, export, and audit views. Never fabricate operational status, trend, freshness, or completion claims.

## Density, responsive behavior, and trust

- Use density that matches the task: compact rows for scanning, more spacing around decisions, and clear grouping for forms and exceptions.
- Keep important values and actions available at normal zoom and narrow widths. Adapt columns, filters, and navigation deliberately; do not rely on horizontal scrolling for the primary task.
- Show freshness, source, and scope when data can become stale or is derived from Demo/MSW. Keep mock or demo indicators visible at the boundary where they affect interpretation.
- Use one neutral text hierarchy for ordinary content. Reserve text color for real status, risk, action, link, or disabled meaning; do not color labels or values merely to decorate a card.
- Keep labels short and plain. Use a label when it identifies a field, scope, status, or action; avoid decorative badges, excessive uppercase text, repeated captions, and labels that restate the surrounding heading.
- Keep feedback near the action that caused it, and provide a page-level summary when an operation affects multiple records.

## Review checklist

- [ ] The page purpose, user role, scope, and primary action are clear on entry.
- [ ] Navigation and permissions match the available routes and operations.
- [ ] Search, filters, sorting, pagination, selection, and bulk actions explain their scope and result.
- [ ] Detail, edit, approval, destructive, and history flows use an appropriate surface and recovery path.
- [ ] Loading, empty, error, stale-data, denied, unsaved-change, and success states are designed.
- [ ] Operational claims and Demo/MSW data are truthful and clearly bounded.
- [ ] 時間欄位一律格式化至秒數（`YYYY-MM-DD HH:mm:ss`），無毫秒或 raw 時區字尾，且寬度足夠。
- [ ] Text colors and labels carry necessary meaning rather than visual decoration.
- [ ] Keyboard, focus, labels, zoom, reduced motion, and screen-reader behavior pass the repository accessibility review.
- [ ] The affected Playwright flow or another explicit check verifies the operator's primary task.

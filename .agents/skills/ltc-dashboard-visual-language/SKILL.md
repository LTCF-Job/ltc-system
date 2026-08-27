---
name: ltc-dashboard-visual-language
description: Apply the LTC operations dashboard visual language when creating or modifying Vue 3 admin pages, dashboards, monitoring screens, data cards, sidebars, tables, alerts, charts, or internal business interfaces that should match the supplied healthcare operations dashboard reference.
---

# LTC Dashboard Visual Language

Apply a calm, precise, high-trust operations interface style. Optimize for fast scanning, clear hierarchy, consistent spacing, restrained color, and accessible states.

## Before implementation

1. Inspect the current route, page, layout, shared components, design tokens, and responsive behavior.
2. Preserve existing routing, API contracts, state ownership, permissions, and business behavior.
3. Reuse existing project components and tokens when they provide an equivalent pattern.
4. Apply this skill to presentation and interaction patterns; keep business logic in its existing boundary.

For detailed values and component rules, read [visual-tokens.md](references/visual-tokens.md) when implementing or reviewing the visual layer.

## Technology boundaries

沿用專案現有技術棧；例如使用 Vue 時就使用 Vue 實作，不引入 React 或另一套平行技術。

## Information integrity and color restraint

- Show operational claims, live status, trends, and alerts only when they have a real data source.
- Remove hardcoded trends, fake real-time indicators, and fake alerts when no source or behavior supports them.
- Preserve functional Demo controls, routes, permissions, and API-backed data.
- Keep Sidebar, Header, and main content colors within the established neutral palette.
- Use semantic colors only for real risk, status, or action meaning; use neutral colors for non-semantic KPI and shortcut elements.
- Keep ordinary text within a small neutral hierarchy. Avoid assigning a different color to every label, value, or card section.

## Visual direction

- Use a cool pale gray page background with a white rounded application shell.
- Keep the Sidebar narrow, quiet, and clearly separated from the content area with a subtle divider.
- Use a white header, compact brand mark, page title, and one visually dominant primary action.
- Use muted off-white cards with subtle borders, soft shadows, and medium rounded corners.
- Preserve generous whitespace. Let data density come from alignment and grouping, not decoration.
- Use a neutral modern sans-serif typeface with compact metadata and strong numeric emphasis.
- Use blue for primary actions; reserve red, amber, green, purple, and cyan for consistent status or metric meanings.
- Prefer thin line trends, small waveform charts, horizontal capacity bars, and compact status indicators.

## Layout rules

Use this information hierarchy:

```text
Application shell
├── Sidebar
│   ├── Organization or product label
│   ├── Navigation groups
│   └── Active navigation item
└── Main area
    ├── Header and primary action
    └── Page content
        ├── Page title
        ├── KPI cards
        └── Operational panels
```

- Keep the main content visually dominant.
- Use a two-column panel layout when related operational information can be compared side by side.
- Stack KPI cards and panels on narrow viewports.
- Convert the Sidebar to a drawer or compact navigation when it cannot remain readable.
- Prevent horizontal scrolling and preserve the visibility of values, labels, and status text on mobile.

## Component rules

### KPI cards

Use a short plain label, a large numeric value, and an optional small trend visualization. The value is the strongest element. Keep the card surface and label treatment visually quiet.

### Operational panels

Use a clear title, optional urgency summary, consistent row heights, aligned values, and a restrained footer action such as `View all`.

### Alerts

Use a small status dot plus text. Keep status semantics consistent: red for critical, amber for warning, blue for information, and green for healthy or resolved. Never communicate status through color alone.

### Progress bars

Use a pale neutral track, a solid semantic fill, rounded ends, and a right-aligned percentage or count such as `48/50`. Avoid gradients in data bars.

### Buttons

Use one blue primary button per viewport where possible. Keep secondary actions textual, outlined, or on a muted surface. Define hover, focus-visible, active, disabled, loading, and error states.

For a compact group of three or more peer actions, optionally use a restrained dock-inspired magnification effect:

- Scale the hovered action to at most `1.12` to `1.16` times its base size.
- Apply a smaller scale influence to immediately adjacent actions, then fall back to the base size by distance.
- Use spring-like easing or a short eased transition so the effect feels fluid rather than abrupt.
- Keep the action group anchored so magnification does not cause layout shift or move surrounding content.
- Show a compact tooltip only when the icon meaning is not already clear from visible text.
- Keep the effect for toolbars, quick actions, or utility controls; do not use it for dense tables, forms, or every button on a page.
- Preserve equivalent keyboard focus and active states without requiring pointer proximity.

## Motion and micro-interactions

Use motion to explain feedback, hierarchy, and state changes. Keep it quiet enough that operators can focus on their work.

- Add a short lift, color, or shadow transition to clickable cards and buttons on hover.
- Add a small press or scale response to primary buttons on activation.
- Fade or slide Sidebar and drawer transitions over a short distance.
- Animate KPI values, progress fills, and status changes only when the underlying data changes.
- Use a brief highlight or pulse for newly arrived critical alerts; stop it after the user can identify the change.
- Use skeleton shimmer only during genuine loading and keep it low contrast.
- Prefer opacity, transform, border-color, and shadow transitions over large size or layout changes.
- Keep most micro-interactions within `120ms` to `220ms`; use up to `300ms` for panels, drawers, or meaningful state transitions.
- Use one easing style consistently, such as `ease-out` for entering and `ease-in-out` for reversible transitions.
- Respect `prefers-reduced-motion: reduce` by removing looping, bouncing, parallax, and non-essential transitions.

Avoid decorative particles, excessive bounce, continuous animation, rotating elements without progress meaning, and animations that delay an operator action.

## Accessibility and states

- Keep keyboard focus clearly visible with sufficient contrast.
- Provide text labels or accessible descriptions for charts and status indicators.
- Design loading, empty, error, retry, disabled, and permission-denied states with the same visual language.
- Pair color with text, icons, patterns, or values so meaning survives color-vision differences.
- Preserve readable contrast even when using muted text and pale surfaces.

## Implementation checklist

- [ ] The page has one clear primary action.
- [ ] Sidebar, header, cards, and panels share the same spacing rhythm.
- [ ] The visual hierarchy is understandable within a few seconds.
- [ ] Blue is reserved for primary actions and key emphasis.
- [ ] Status colors have stable meanings across the product.
- [ ] Cards use subtle borders and shadows rather than heavy decoration.
- [ ] Dense data remains readable at normal zoom.
- [ ] Focus, loading, empty, error, and disabled states are present.
- [ ] Motion is attached to a user action or meaningful data/state change.
- [ ] Motion is brief, subtle, and disabled or reduced for `prefers-reduced-motion`.
- [ ] Magnification, when used, is limited to compact action groups and does not cause layout shift.
- [ ] Mobile layout has no horizontal overflow.
- [ ] Visual changes do not alter existing business behavior or API contracts.
- [ ] The implementation stays within the existing project technology stack.
- [ ] Operational claims, trends, live states, and alerts have a real source or are clearly marked as demo data.
- [ ] Non-semantic KPI and shortcut elements use the neutral palette.

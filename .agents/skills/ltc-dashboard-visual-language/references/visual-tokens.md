# LTC Dashboard Visual Tokens

Use the project's existing token system when equivalent tokens exist. These values are the reference direction for new visual work.

## Colors

```css
--page-background: #f4f7fa;
--surface: #ffffff;
--surface-muted: #f7f9fa;
--border-subtle: #e9edf0;
--text-primary: #101522;
--text-secondary: #586174;
--text-muted: #8b94a5;
--accent-blue: #2f6fed;
--accent-blue-dark: #1f55c7;
--status-critical: #ed294b;
--status-warning: #e18a00;
--status-info: #399bd3;
--status-success: #16c889;
--accent-purple: #7430c7;
--accent-cyan: #12b8c9;
```

Use saturated colors for semantic emphasis, not large background areas.

## Shape and depth

- Application shell: `18px` to `24px` radius
- Cards: `12px` to `16px` radius
- Buttons and inputs: `8px` to `10px` radius
- Prefer subtle borders before shadows.
- Keep shadows soft, diffuse, and low contrast.
- Avoid thick borders, strong drop shadows, and decorative glassmorphism.

## Spacing and sizing

Use a 4px or 8px spacing scale.

- Page padding: `20px` to `24px`
- Card gap: `12px` to `16px`
- Section gap: `20px` to `28px`
- Sidebar width: `160px` to `190px`
- Header height: `48px` to `56px`
- Card padding: `14px` to `18px`

## Typography

- Page title: `20px` to `24px`, weight `600`
- Section title: `13px` to `15px`, weight `600`
- KPI number: `26px` to `32px`, weight `500` to `600`
- Body text: `12px` to `14px`
- Metadata and labels: `10px` to `12px`
- Navigation labels: `11px` to `13px`

Use plain, concise text for labels. Use uppercase only when it materially improves scanning for compact metadata or KPI headings; avoid decorative label treatments and repeated captions.

## Motion tokens

```css
--motion-fast: 120ms;
--motion-standard: 180ms;
--motion-panel: 260ms;
--motion-ease-out: cubic-bezier(0.22, 1, 0.36, 1);
--motion-ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
--motion-dock-max-scale: 1.16;
--motion-dock-adjacent-scale: 1.06;
```

Use `transform`, `opacity`, `background-color`, `border-color`, and `box-shadow` for most interactions. Avoid animating layout-heavy properties such as `width`, `height`, `top`, and `left` when a transform can provide the same feedback.

Recommended effects:

| Component | Trigger | Effect | Duration |
| --- | --- | --- | --- |
| Primary button | hover | Slight color lift and soft shadow | 120–180ms |
| Primary button | active | Subtle press scale, no bounce | 120ms |
| Card | hover | Small upward translation and border emphasis | 180ms |
| Drawer or Sidebar | open/close | Opacity plus horizontal translation | 220–260ms |
| Progress bar | value change | Smooth fill transition | 180–260ms |
| New critical alert | inserted | One brief highlight or pulse | 220–300ms |

## Dock-inspired action group

Use this pattern only for a compact set of related utility actions. Calculate each item's influence from pointer distance or focus proximity, then interpolate between the base scale, adjacent scale, and maximum scale. Keep the group in a fixed or flex-stable container so the visual size change does not reflow the page.

在 Vue 專案中，沿用現有 Vue 技術實作互動；參考元件只借鑑效果，不引入另一套技術。

Provide a non-pointer equivalent: focused items may receive the maximum scale, while neighboring items remain at the adjacent scale. Disable magnification and keep state changes immediate when `prefers-reduced-motion` is enabled.

Use the existing framework's implementation for this interaction. A reference component's framework or dependency is not a reason to introduce another technology stack.

For reduced-motion users, keep state changes immediate or use a short opacity transition only. Never rely on animation alone to communicate urgency or completion.

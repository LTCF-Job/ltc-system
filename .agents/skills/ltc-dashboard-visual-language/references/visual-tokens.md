# LTC Dashboard Visual Tokens

Use the project's existing token system when equivalent tokens exist. These are the actual variable names defined in `apps/web/src/styles/tokens.scss` and `apps/web/src/styles/element-overrides.scss` — not a separate naming scheme. For the full component-level rules (buttons, tables, dialogs, status colors), see [component-contract.md](component-contract.md).

## Colors

```css
--app-bg: #f4f7fa;
--app-surface: #ffffff;
--app-card-bg: #f7f9fa;
--app-border-color: #e9edf0;
--app-border-light: #f0f2f4;
--app-text-primary: #101522;
--app-text-regular: #303949;
--app-text-secondary: #586174;
--app-text-muted: #8b94a5;

--app-primary: #0c70ed;
--app-primary-dark: #0043bc;
--app-primary-light: #e6f3ff;

--app-status-success-bg: #e8faf3;
--app-status-success-fg: #00792b;
--app-status-warning-bg: #fff4dc;
--app-status-warning-fg: #975a00;
--app-status-danger-bg: #ffebef;
--app-status-danger-fg: #bf1943;
--app-status-danger-border: #ffb3b8;
--app-status-info-bg: #eaf5fb;
--app-status-info-fg: #00788a;
--app-status-neutral-bg: #f1f3f4;
--app-status-neutral-fg: #586174;

--app-role-admin-fg / -dot
--app-role-dispatcher-fg / -dot
--app-role-staff-fg / -dot
--app-role-driver-fg / -dot
--app-role-viewer-fg / -dot

/* 側欄面板（淺色，與主內容以細分隔線區隔） */
--app-nav-bg: #ffffff;
--app-nav-border: var(--app-border-color);
--app-nav-fg: var(--app-text-regular);
--app-nav-fg-muted: var(--app-text-secondary);
--app-nav-hover-bg: #f2f5f9;
--app-nav-active-bg: var(--app-primary-light);
--app-nav-active-fg: var(--app-primary-dark);

/* 容量長條 */
--app-bar-track: #eef1f5;
--app-bar-height: 7px;
```

側欄配色一律走 `--app-nav-*`，`DefaultLayout.vue` 不再傳 `background-color`／`text-color` 給 `el-menu`；改由 `.el-menu-vertical` 設定 `--el-menu-bg-color`／`--el-menu-text-color`／`--el-menu-active-color`／`--el-menu-hover-bg-color` 對接。

Use `--app-status-*-bg` for tag/badge backgrounds and `--app-status-*-fg` for their text/dot color. Do not hardcode a new green/red/amber — every status color in the product must resolve to one of these five pairs.

## Shape and depth

```css
--app-radius-xs: 6px;
--app-radius-sm: 9px;
--app-radius-md: 14px;
--app-radius-lg: 20px;
--app-radius-full: 9999px;
--app-shadow-sm: 0 1px 2px rgba(16, 21, 34, 0.04);
--app-shadow-md: 0 8px 24px rgba(16, 21, 34, 0.07);
```

- Application shell: `--app-radius-lg`
- Cards: `--app-radius-md`
- Buttons and inputs: `--app-radius-sm`
- Small pills/dots: `--app-radius-xs` or `--app-radius-full`
- Prefer subtle borders before shadows. Keep shadows soft and low contrast.

## Spacing and sizing

```css
--app-space-1: 4px;
--app-space-2: 8px;
--app-space-3: 12px;
--app-space-4: 16px;
--app-space-6: 24px;
--app-space-8: 32px;

--app-header-height: 56px;
--app-aside-width: 220px;
--app-aside-width-collapsed: 68px;
```

- Page padding: `--app-space-4` to `--app-space-6`
- Card gap: `--app-space-3` to `--app-space-4`
- Section gap: `--app-space-6`
- Card padding: `--app-space-4`

Do not write a new spacing magic number. If none of these fit, that is a signal to reconsider the layout, not to add a one-off value.

## Typography

```css
--app-font-xs: 13px;
--app-font-sm: 13px;
--app-font-md: 14px;
--app-font-lg: 16px;
--app-font-xl: 20px;
--app-font-2xl: 24px;
--app-font-3xl: 34px;

/* 大寫微標籤（KPI 標籤、側欄分組標題） */
--app-label-size: 11px;
--app-label-tracking: 0.08em;
```

- Page title: `--app-font-xl`, weight `600`
- Section title: `--app-font-sm` to `--app-font-lg`, weight `600`
- Body text: `--app-font-sm` to `--app-font-md`
- Metadata and labels: `--app-font-xs`
- Navigation labels: `--app-font-sm`

Use plain, concise text for labels. Use uppercase only when it materially improves scanning for compact metadata; avoid decorative label treatments and repeated captions.

## Motion

No dedicated motion token variables exist in the codebase yet. Keep transitions short (120ms–220ms for buttons/cards, up to 300ms for panels/drawers) and consistent with the existing `.el-button`, `.el-card.is-hover-shadow`, and `.calendar-cell` transitions already defined in `element-overrides.scss`. Respect `prefers-reduced-motion: reduce`.

Do not introduce a dock-style magnification effect — the codebase does not use one, and adding it would be a new pattern outside this system.

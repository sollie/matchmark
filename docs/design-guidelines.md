# Design guidelines

See ADR 0028 (which supersedes 0027) for the reasoning.

## Stack

Tailwind v4, Bits UI primitives, svelte-sonner, NumberFlow, `@lucide/svelte` icons. Shared components are owned in `source/web/src/lib/ui/`; compose them rather than hand-building dialogs, popovers, menus, selects, tabs, tooltips, toasts or drawers. Do not mix legacy CSS classes and Tailwind utilities in one screen; `src/legacy.css` is removed as surfaces migrate.

| Task | Use |
| --- | --- |
| Button | `ui/Button.svelte` (`variant`: primary, secondary, danger, ghost; `size`: sm, md, lg, icon) |
| Text input, labelled field | `ui/Input.svelte`, `ui/Field.svelte` |
| Select | `ui/Select.svelte` (Bits UI Select; `label` is the accessible name) |
| Modal dialog | `ui/Dialog.svelte` |
| Confirm a destructive or irreversible action (replaces `window.confirm`) | `ui/ConfirmDialog.svelte` |
| Bottom or side sheet | `ui/Drawer.svelte` (Bits UI Dialog; `vaul-svelte` is unmaintained) |
| Tabs | `ui/Tabs.svelte` |
| Table | `ui/Table.svelte` (scrolling region with a required `label`) |
| Status badge | `ui/Badge.svelte` (`tone`: success, warning, danger, info, neutral) |
| Toasts | `ui/toast.ts` (`toast.success()`, `toast.error()`); one `ui/Toaster.svelte` in `App.svelte` |
| Empty, loading, error | `ui/EmptyState.svelte`, `ui/LoadingState.svelte` (`role="status"`), `ui/ErrorState.svelte` (`role="alert"`) |
| Menu or popover | Bits UI `Popover`, see `lib/MenuShell.svelte` |
| Public top bar | `lib/SiteHeader.svelte` |
| Animated numbers | `@number-flow/svelte` |
| Class merging | `cn()` from `ui/utils.ts` |

Locate controls in e2e tests by accessible name; Bits UI parts keep correct roles. Bits UI portals content to `<body>`.

## Tokens

All colours, radii, type sizes and shadows are defined once in `src/tokens.css`. The palette uses `light-dark()`, so no `dark:` variants are needed; Mode (`data-theme` on `<html>`, set from `matchmark-theme`) only switches `color-scheme`.

- Colour utilities: `bg-bg`, `bg-surface`, `bg-surface-alt`, `border-border`, `text-fg`, `text-muted`, `text-link`, `bg-brand`, `text-on-brand`, `bg-brand-tint`, `bg-danger`, `text-on-danger`, `bg-header`, status pairs `bg-success-bg text-success-fg` (also `warning`, `danger`, `neutral`), and `outline-focus`. The default Tailwind palette is removed; use only these.
- Radius: `rounded-sm` 4px, `rounded-md` 8px, `rounded-lg` 12px, `rounded-xl` 16px.
- Type: `text-sm` 0.85rem, `text-base` 1rem, `text-lg` 1.25rem, `text-xl` 1.6rem; headings use a tighter weight and tracking from the base layer. Numbers are tabular.
- Shadows: `shadow-card`, `shadow-overlay`.
- Touch targets are at least 44px (`min-h-11`) for primary controls.
- `src/lib/tokens.test.ts` enforces WCAG contrast for the foreground and background pairs in both modes; add a pair when adding a token pair.

## Visual direction

- One standout moment per page; keep the rest restrained. No gradients, glows or constant motion everywhere.
- Customize through tokens rather than per-component colour overrides.
- Motion is CSS only, gated with `motion-safe:`; a global rule also neutralises animation and transition under `prefers-reduced-motion`.
- Status is never colour only: pair a badge or state with text.

## Adding a dependency

Confirm recent releases, Svelte 5 support and licence before installing; reuse what the project already has.

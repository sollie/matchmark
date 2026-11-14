---
status: superseded by 0028
---

# Headless component libraries instead of hand-built UI

The SPA hand-rolled its dialog and had no toast or animated-number support. For the frontend redesign we assemble established libraries and restyle them with the project's own design tokens, rather than building interactive components from scratch. The picks follow Emil Kowalski's curated list, using the Svelte builds where the original is React-only.

- Dialogs, popovers, menus, selects, tabs, tooltips: Bits UI (headless, accessible). Replaces base-ui, which is React-only.
- Toasts: svelte-sonner (Svelte port of Sonner).
- Animated numbers (scores, counters): `@number-flow/svelte` (NumberFlow).
- Animation: Svelte's `transition:`/`animate:` and CSS first; add `motion` only for springs, layout or gesture animation.
- Drawers: Vaul via `vaul-svelte`, added only when a drawer is needed, after confirming it supports the current Svelte.
- Command menu: cmdk-sv or Bits UI Command, when a command menu is needed.

## Considered Options

- shadcn-svelte: rejected, it requires Tailwind and the project styles with plain CSS tokens in `src/app.css` and `src/palettes.css`.
- Keep hand-built components: rejected, they miss focus management, stacking and keyboard handling.

## Consequences

Components are styled through CSS classes using existing tokens, so default library looks never ship. New libraries need a maintenance and Svelte 5 check before adding. Guidelines live in `docs/design-guidelines.md`.

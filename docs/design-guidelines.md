# Design guidelines

See ADR 0027 for the reasoning.

## Components

- Do not hand-build dialogs, popovers, menus, selects, tooltips, toasts, drawers or command menus. Use the libraries below, wrapped once in `source/web/src/lib/` and reused.
- Style library parts with classes that use the tokens in `src/app.css` (spacing, radius, type, shadow) and the palette variables in `src/palettes.css`. Never ship a library's default look.
- Locate controls in e2e tests by accessible name; library parts keep correct roles.

| Task | Library |
| --- | --- |
| Dialog, popover, dropdown, select, tabs, tooltip | `bits-ui` (see `lib/Modal.svelte`) |
| Toasts | `svelte-sonner` (one `<Toaster>` in `App.svelte`; call `toast()`) |
| Animated numbers (scores, totals) | `@number-flow/svelte` |
| Drawers / bottom sheets (scorer on phones) | `vaul-svelte` when needed |
| Command menu | `cmdk-sv` or Bits UI Command when needed |
| Springs, layout or gesture animation | `motion` when needed |
| Simple hover, fade, enter/exit | CSS or Svelte `transition:` |
| State | Svelte 5 runes, no state library |

Not in the list and not needed yet: drag and drop, virtualization, charts. Pick a Svelte-compatible library then and record it in an ADR.

## Visual direction

- One standout moment per page; keep the rest restrained. No gradients, glows or constant motion everywhere.
- Customize typography, colour, radius and spacing through tokens rather than per-component overrides.
- Respect `prefers-reduced-motion`.

## Adding a dependency

Confirm recent releases, Svelte 5 support and licence before installing; reuse what the project already has.

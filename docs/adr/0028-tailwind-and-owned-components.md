---
status: accepted
supersedes: 0027
---

# Tailwind v4 with owned, customized components

ADR 0027 chose headless libraries styled with plain-CSS tokens and rejected shadcn-svelte because it needs Tailwind. The UI redesign (#555) adopts Tailwind v4, so that reason falls away. The library picks of 0027 (Bits UI, svelte-sonner, NumberFlow) stand; the styling approach changes.

- Styling: Tailwind v4 through `@tailwindcss/vite`, with no preflight reset. Utilities are generated from the brand tokens in `src/tokens.css` (`@theme`), whose CSS variables (`--bg`, `--surface`, `--brand`, ...) back the theme.
- Components: shadcn-svelte style, owned in `src/lib/ui/`, built on Bits UI with `tailwind-variants`, `clsx` and `tailwind-merge`. Each is customized (tokens, radius, type, touch size) so the default shadcn look never ships. The shadcn default colour palette is removed from the Tailwind theme.
- The hand-written legacy stylesheet is gone. `src/app.css` holds only a minimal base layer (box model, typography, focus ring, reduced motion) and everything else is Tailwind utilities.
- One brand palette with light and dark variants through `light-dark()`; the five other palettes and the selector are removed. Mode (Auto, Light, Dark) stays per browser.
- Drawers: `vaul-svelte` was last released in March 2025 and only declares Svelte 4 and 5-next peers, so it is not used. `ui/Drawer.svelte` is a Bits UI Dialog presented as a bottom or left sheet.
- Icons: `@lucide/svelte`, imported per icon.
- Motion: CSS only, gated by `motion-safe:` and a global `prefers-reduced-motion` rule.

## Considered Options

- Keep plain-CSS tokens (0027): rejected, it duplicated what Tailwind does and made responsive layouts verbose.
- Run the shadcn-svelte CLI: not needed, the handful of components are small and written to the brand directly.

## Consequences

The API keeps a `palette` field on the club settings; the SPA ignores it (backend changes are out of scope for the redesign). Guidelines and the token list live in `docs/design-guidelines.md`. New dependencies need a maintenance and Svelte 5 check before adding.

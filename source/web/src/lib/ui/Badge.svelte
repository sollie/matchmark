<script lang="ts" module>
  import { tv, type VariantProps } from "tailwind-variants";

  export const badgeVariants = tv({
    base: "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-sm font-semibold whitespace-nowrap",
    variants: {
      tone: {
        success: "bg-success-bg text-success-fg",
        warning: "bg-warning-bg text-warning-fg",
        danger: "bg-danger-bg text-danger-fg",
        info: "bg-brand-tint text-link",
        neutral: "bg-neutral-bg text-neutral-fg",
      },
    },
    defaultVariants: { tone: "neutral" },
  });

  export type BadgeTone = VariantProps<typeof badgeVariants>["tone"];
</script>

<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    tone,
    class: className,
    children,
  }: { tone?: BadgeTone; class?: string; children: Snippet } = $props();
</script>

<span class={cn(badgeVariants({ tone }), className)}>{@render children()}</span>

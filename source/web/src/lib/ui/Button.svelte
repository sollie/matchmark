<script lang="ts" module>
  import { tv, type VariantProps } from "tailwind-variants";

  export const buttonVariants = tv({
    base: "inline-flex cursor-pointer items-center justify-center gap-2 rounded-md border border-transparent font-semibold whitespace-nowrap no-underline outline-offset-2 transition-colors focus-visible:outline-3 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-50 motion-safe:active:scale-[0.98] motion-safe:transition-transform",
    variants: {
      variant: {
        primary: "bg-brand text-on-brand hover:bg-brand-hover",
        secondary: "border-border bg-surface text-fg hover:bg-surface-alt",
        danger: "bg-danger text-on-danger hover:bg-danger-hover",
        ghost: "bg-transparent text-fg hover:bg-surface-alt",
      },
      size: {
        sm: "min-h-9 px-3 text-sm",
        md: "min-h-11 px-4 text-base",
        lg: "min-h-14 px-6 text-lg",
        icon: "size-11 p-0",
      },
    },
    defaultVariants: { variant: "primary", size: "md" },
  });

  export type ButtonVariant = VariantProps<typeof buttonVariants>["variant"];
  export type ButtonSize = VariantProps<typeof buttonVariants>["size"];
</script>

<script lang="ts">
  import type { HTMLButtonAttributes } from "svelte/elements";
  import { cn } from "./utils";

  let {
    variant,
    size,
    class: className,
    type = "button",
    children,
    ...rest
  }: HTMLButtonAttributes & { variant?: ButtonVariant; size?: ButtonSize } = $props();
</script>

<button {type} class={cn(buttonVariants({ variant, size }), className)} {...rest}>
  {@render children?.()}
</button>

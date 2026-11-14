<script lang="ts">
  import { Select } from "bits-ui";
  import Check from "@lucide/svelte/icons/check";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { cn } from "./utils";
  import { controlClass } from "./control";

  let {
    value = $bindable(""),
    items,
    label,
    placeholder = "Select",
    disabled = false,
    class: className,
    onValueChange,
  }: {
    value?: string;
    items: { value: string; label: string; disabled?: boolean }[];
    label: string;
    placeholder?: string;
    disabled?: boolean;
    class?: string;
    onValueChange?: (value: string) => void;
  } = $props();

  const selectedLabel = $derived(items.find((i) => i.value === value)?.label);
</script>

<Select.Root type="single" bind:value {items} {disabled} {onValueChange}>
  <Select.Trigger
    aria-label={label}
    class={cn(
      controlClass,
      "inline-flex cursor-pointer items-center justify-between gap-2",
      className,
    )}
  >
    <span class={cn("truncate", !selectedLabel && "text-muted")}>
      {selectedLabel ?? placeholder}
    </span>
    <ChevronDown class="size-4 shrink-0 text-muted" aria-hidden="true" />
  </Select.Trigger>
  <Select.Portal>
    <Select.Content
      sideOffset={4}
      class="z-60 max-h-72 min-w-(--bits-select-anchor-width) overflow-hidden rounded-lg border border-border bg-surface text-fg shadow-overlay"
    >
      <Select.Viewport class="p-1">
        {#each items as item (item.value)}
          <Select.Item
            value={item.value}
            label={item.label}
            disabled={item.disabled}
            class="flex min-h-11 cursor-pointer items-center justify-between gap-2 rounded-md px-3 text-base outline-none data-disabled:cursor-not-allowed data-disabled:opacity-50 data-highlighted:bg-brand-tint"
          >
            {#snippet children({ selected })}
              <span>{item.label}</span>
              {#if selected}<Check class="size-4 text-brand" aria-hidden="true" />{/if}
            {/snippet}
          </Select.Item>
        {/each}
      </Select.Viewport>
    </Select.Content>
  </Select.Portal>
</Select.Root>

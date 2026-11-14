<script lang="ts">
  import { Tabs } from "bits-ui";
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    value = $bindable(),
    items,
    label,
    class: className,
    panel,
  }: {
    value?: string;
    items: { value: string; label: string; disabled?: boolean }[];
    label: string;
    class?: string;
    panel: Snippet<[string]>;
  } = $props();
</script>

<Tabs.Root bind:value class={className}>
  <Tabs.List aria-label={label} class="flex gap-1 overflow-x-auto border-b border-border">
    {#each items as item (item.value)}
      <Tabs.Trigger
        value={item.value}
        disabled={item.disabled}
        class={cn(
          "-mb-px min-h-11 cursor-pointer border-b-3 border-transparent px-4 font-semibold whitespace-nowrap text-muted outline-offset-[-2px] hover:text-fg focus-visible:outline-3 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-50 data-[state=active]:border-brand data-[state=active]:text-brand",
        )}
      >
        {item.label}
      </Tabs.Trigger>
    {/each}
  </Tabs.List>
  {#each items as item (item.value)}
    <Tabs.Content
      value={item.value}
      class="pt-4 outline-offset-2 focus-visible:outline-3 focus-visible:outline-focus"
    >
      {@render panel(item.value)}
    </Tabs.Content>
  {/each}
</Tabs.Root>

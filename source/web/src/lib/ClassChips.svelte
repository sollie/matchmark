<script lang="ts">
  import { badgeVariants } from "./ui/Badge.svelte";
  import { classChips, type Classifications } from "./classification";

  let { classifications }: { classifications: Classifications | undefined } = $props();
  const chips = $derived(classChips(classifications));
</script>

{#if chips.length > 0}
  <ul class="m-0 flex list-none flex-wrap gap-1 p-0" aria-label="Classifications">
    {#each chips as chip (chip.category)}
      <li class={badgeVariants({ tone: "info" })}>{chip.category} <strong>{chip.name}</strong></li>
    {/each}
  </ul>
{/if}

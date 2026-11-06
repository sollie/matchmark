<script lang="ts">
  import { classChips, type Classifications } from "./classification";

  let { classifications }: { classifications: Classifications | undefined } = $props();
  const chips = $derived(classChips(classifications));
</script>

{#if chips.length > 0}
  <ul class="chips" aria-label="Classifications">
    {#each chips as chip (chip.category)}
      <li class="badge info">{chip.category} <strong>{chip.name}</strong></li>
    {/each}
  </ul>
{/if}

<style>
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    list-style: none;
    margin: 0;
    padding: 0;
  }
</style>

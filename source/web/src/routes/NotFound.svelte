<script lang="ts">
  import EmptyState from "../lib/ui/EmptyState.svelte";
</script>

<main class="mx-auto flex max-w-md flex-col gap-4 p-4 pt-12">
  <EmptyState
    heading="h1"
    title="Page not found"
    description="The page you are looking for does not exist."
  >
    {#snippet action()}
      <a
        href="/"
        class="inline-flex min-h-11 items-center justify-center rounded-md bg-brand px-4 font-semibold text-on-brand no-underline outline-offset-2 hover:bg-brand-hover focus-visible:outline-3 focus-visible:outline-focus"
        >Back to start</a
      >
    {/snippet}
  </EmptyState>
</main>

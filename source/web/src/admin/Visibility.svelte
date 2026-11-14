<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import type { components } from "../lib/api/schema";

  type Visibility = components["schemas"]["Visibility"];

  let { matchId }: { matchId: number } = $props();

  const labels: [keyof Visibility, string][] = [
    ["show_names", "Names"],
    ["show_clubs", "Clubs"],
    ["show_countries", "Countries"],
    ["show_segment_scores", "Segment scores"],
    ["show_ring_counts", "X count"],
    ["group_by_category", "Group standings by category"],
  ];

  let visibility = $state<Visibility | null>(null);
  let message = $state("");
  let error = $state("");

  onMount(async () => {
    const result = await api.GET("/api/matches/{matchID}/visibility", {
      params: { path: { matchID: matchId } },
    });
    if (result.data) visibility = result.data;
    else error = failure(result);
  });

  async function save() {
    if (!visibility) return;
    message = "";
    error = "";
    const result = await api.PUT("/api/matches/{matchID}/visibility", {
      params: { path: { matchID: matchId } },
      body: visibility,
    });
    if (result.data) {
      visibility = result.data;
      message = "Saved";
    } else error = failure(result);
  }
</script>

<section class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <h2 class="m-0 text-lg">Spectator visibility</h2>
  {#if visibility}
    <div class="flex flex-wrap items-center gap-x-6 gap-y-3">
      {#each labels as [key, label] (key)}
        <label class="flex min-h-11 cursor-pointer items-center gap-2">
          <input type="checkbox" class="size-5 accent-brand" bind:checked={visibility[key]} />
          {label}
        </label>
      {/each}
      <Button onclick={save}>Save</Button>
    </div>
  {/if}
  {#if message}<p class="m-0 text-success-fg" role="status">{message}</p>{/if}
  {#if error}<ErrorState title={error} />{/if}
</section>

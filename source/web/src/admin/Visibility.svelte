<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";

  type Visibility = components["schemas"]["Visibility"];

  let { competitionId }: { competitionId: number } = $props();

  const labels: [keyof Visibility, string][] = [
    ["show_names", "Names"],
    ["show_clubs", "Clubs"],
    ["show_countries", "Countries"],
    ["show_stage_scores", "Stage scores"],
    ["show_ring_counts", "Ring counts"],
  ];

  let visibility = $state<Visibility | null>(null);
  let message = $state("");
  let error = $state("");

  onMount(async () => {
    const result = await api.GET("/api/competitions/{competitionID}/visibility", {
      params: { path: { competitionID: competitionId } },
    });
    if (result.data) visibility = result.data;
    else error = failure(result);
  });

  async function save() {
    if (!visibility) return;
    message = "";
    error = "";
    const result = await api.PUT("/api/competitions/{competitionID}/visibility", {
      params: { path: { competitionID: competitionId } },
      body: visibility,
    });
    if (result.data) {
      visibility = result.data;
      message = "Saved";
    } else error = failure(result);
  }
</script>

<section class="card">
  <h2>Spectator visibility</h2>
  {#if visibility}
    <ul class="flags">
      {#each labels as [key, label] (key)}<li><span class="badge {visibility[key] ? "success" : "neutral"}">{label}: {visibility[key] ? "Shown" : "Hidden"}</span></li>{/each}
    </ul>
    <div class="row">
      {#each labels as [key, label] (key)}
        <label class="check"><input type="checkbox" bind:checked={visibility[key]} /> {label}</label>
      {/each}
      <button class="primary" onclick={save}>Save</button>
    </div>
  {/if}
  {#if message}<p class="muted" role="status">{message}</p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</section>

<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { pairingUrl, qrSvg } from "../lib/pairing";
  import type { components } from "../lib/api/schema";

  type Pairing = components["schemas"]["Pairing"];
  type EntryWithStages = components["schemas"]["EntryWithStages"];

  let {
    competitionId,
    entries,
    finished,
    draft = false,
  }: { competitionId: number; entries: EntryWithStages[]; finished: boolean; draft?: boolean } = $props();

  let pairings = $state<Pairing[]>([]);
  let lane = $state<number | null>(null);
  let issued = $state<{ lane: number; url: string; svg: string } | null>(null);
  let error = $state("");

  const lanes = $derived([...new Set(entries.filter((e) => !e.withdrawn).map((e) => e.lane))].sort((a, b) => a - b));

  async function load() {
    const result = await api.GET("/api/competitions/{competitionID}/pairings", {
      params: { path: { competitionID: competitionId } },
    });
    if (result.data) pairings = result.data.pairings;
    else error = failure(result);
  }

  onMount(load);

  async function pair(e: SubmitEvent) {
    e.preventDefault();
    if (lane === null) return;
    error = "";
    const result = await api.POST("/api/competitions/{competitionID}/pairings", {
      params: { path: { competitionID: competitionId } },
      body: { lane },
    });
    if (result.data) {
      const url = pairingUrl(result.data.token);
      issued = { lane: result.data.lane, url, svg: qrSvg(url) };
      await load();
    } else error = failure(result);
  }

  async function revoke(p: Pairing) {
    error = "";
    const result = await api.DELETE("/api/competitions/{competitionID}/pairings/{pairingID}", {
      params: { path: { competitionID: competitionId, pairingID: p.id } },
    });
    if (result.response.ok) await load();
    else error = failure(result);
  }
</script>

<section class="card">
  <h2>Scorer pairing</h2>
  {#if finished}
    <p class="muted">Competition finished. Pairings are revoked.</p>
  {:else}
    <form class="row" onsubmit={pair}>
      <label>
        Lane
        <select aria-label="Pairing lane" bind:value={lane} required>
          <option value={null} disabled>Select lane</option>
          {#each lanes as l (l)}<option value={l}>Lane {l}</option>{/each}
        </select>
      </label>
      <button type="submit" class="primary">Create pairing link</button>
    </form>
    {#if draft}
      <p class="muted">You can pair devices before the start. They show a waiting screen until the competition is started. Re-check pairings if you change lanes.</p>
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#if issued}
    <div class="card qr-card">
      <h3>Lane {issued.lane}</h3>
      <ol class="muted steps">
        <li>Open the camera on the scorer's device and scan the code, or copy the link below.</li>
        <li>The link is shown once and pairs the device with lane {issued.lane}.</li>
      </ol>
      <div class="qr" role="img" aria-label={`QR code pairing lane ${issued.lane}`}>{@html issued.svg}</div>
      <label>
        Pairing link
        <input readonly value={issued.url} onfocus={(e) => e.currentTarget.select()} />
      </label>
      <button class="secondary" onclick={() => (issued = null)}>Hide</button>
    </div>
  {/if}

  <div class="scroll">
  <table>
    <thead><tr><th>Lane</th><th>Status</th><th></th></tr></thead>
    <tbody>
      {#each pairings as p (p.id)}
        <tr>
          <td>{p.lane}</td>
          <td><span class="badge {p.revoked ? "neutral" : "success"}">{p.revoked ? "revoked" : "paired"}</span></td>
          <td>{#if !p.revoked}<button class="danger" onclick={() => revoke(p)}>Revoke</button>{/if}</td>
        </tr>
      {:else}
        <tr><td colspan="3" class="muted">No devices paired.</td></tr>
      {/each}
    </tbody>
  </table>
  </div>
</section>

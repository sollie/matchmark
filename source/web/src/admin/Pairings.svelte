<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { pairingUrl, qrSvg } from "../lib/pairing";
  import type { components } from "../lib/api/schema";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import NativeSelect from "../lib/ui/NativeSelect.svelte";
  import Table from "../lib/ui/Table.svelte";

  type Pairing = components["schemas"]["Pairing"];
  type EntryWithSegments = components["schemas"]["EntryWithSegments"];

  let {
    matchId,
    entries,
    finished,
    draft = false,
  }: {
    matchId: number;
    entries: EntryWithSegments[];
    finished: boolean;
    draft?: boolean;
  } = $props();

  let pairings = $state<Pairing[]>([]);
  let lane = $state<number | null>(null);
  let issued = $state<{ lane: number; url: string; svg: string } | null>(null);
  let error = $state("");

  const active = $derived(pairings.filter((p) => !p.revoked).length);

  const lanes = $derived(
    [...new Set(entries.filter((e) => !e.withdrawn).map((e) => e.lane))].sort((a, b) => a - b),
  );

  async function load() {
    const result = await api.GET("/api/matches/{matchID}/pairings", {
      params: { path: { matchID: matchId } },
    });
    if (result.data) pairings = result.data.pairings;
    else error = failure(result);
  }

  onMount(load);

  async function pair(e: SubmitEvent) {
    e.preventDefault();
    if (lane === null) return;
    error = "";
    const result = await api.POST("/api/matches/{matchID}/pairings", {
      params: { path: { matchID: matchId } },
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
    const result = await api.DELETE("/api/matches/{matchID}/pairings/{pairingID}", {
      params: { path: { matchID: matchId, pairingID: p.id } },
    });
    if (result.response.ok) await load();
    else error = failure(result);
  }
</script>

<section class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <div class="flex flex-wrap items-baseline justify-between gap-2">
    <h2 class="m-0 text-lg">Scorer pairing</h2>
    <p class="m-0 text-sm text-muted">{active} active {active === 1 ? "device" : "devices"}</p>
  </div>
  {#if finished}
    <p class="m-0 text-muted">Match finished. Pairings are revoked.</p>
  {:else}
    <form class="flex flex-wrap items-end gap-3" onsubmit={pair}>
      <Field label="Lane" class="w-full sm:w-56">
        <NativeSelect aria-label="Pairing lane" bind:value={lane} required>
          <option value={null} disabled>Select lane</option>
          {#each lanes as l (l)}<option value={l}>Lane {l}</option>{/each}
        </NativeSelect>
      </Field>
      <Button type="submit">Create pairing link</Button>
    </form>
    {#if draft}
      <p class="m-0 text-muted">
        You can pair devices before the start. They show a waiting screen until the match is
        started. Re-check pairings if you change lanes.
      </p>
    {/if}
  {/if}
  {#if error}<ErrorState title={error} />{/if}

  {#if issued}
    <div class="flex flex-col gap-3 rounded-lg border border-border bg-surface-alt p-4">
      <h3 class="m-0 text-lg">Lane {issued.lane}</h3>
      <ol class="m-0 list-decimal pl-5 text-muted">
        <li>Open the camera on the scorer's device and scan the code, or copy the link below.</li>
        <li>The link is shown once and pairs the device with lane {issued.lane}.</li>
      </ol>
      <div
        class="size-48 self-start rounded-md bg-white p-2 [&_svg]:size-full"
        role="img"
        aria-label={`QR code pairing lane ${issued.lane}`}
      >
        {@html issued.svg}
      </div>
      <Field label="Pairing link">
        <Input readonly value={issued.url} onfocus={(e) => e.currentTarget.select()} />
      </Field>
      <div><Button variant="secondary" onclick={() => (issued = null)}>Hide</Button></div>
    </div>
  {/if}

  {#if pairings.length === 0}
    <EmptyState title="No devices paired." heading="h3" />
  {:else}
    <Table label="Pairings">
      <thead
        ><tr><th>Lane</th><th>Status</th><th><span class="sr-only">Actions</span></th></tr></thead
      >
      <tbody>
        {#each pairings as p (p.id)}
          <tr>
            <td>{p.lane}</td>
            <td
              ><Badge tone={p.revoked ? "neutral" : "success"}
                >{p.revoked ? "revoked" : "paired"}</Badge
              ></td
            >
            <td class="text-right">
              {#if !p.revoked}<Button variant="danger" size="sm" onclick={() => revoke(p)}
                  >Revoke</Button
                >{/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </Table>
  {/if}
</section>

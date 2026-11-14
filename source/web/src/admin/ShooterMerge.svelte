<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import Button from "../lib/ui/Button.svelte";
  import Dialog from "../lib/ui/Dialog.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import ShooterPicker from "./ShooterPicker.svelte";

  type Shooter = components["schemas"]["Shooter"];
  type Preview = components["schemas"]["ShooterMergePreview"];

  let {
    source,
    onmerged,
    oncancel,
  }: { source: Shooter; onmerged: (target: Shooter) => void; oncancel: () => void } = $props();

  let target = $state<Shooter | null>(null);
  let preview = $state<Preview | null>(null);
  let confirmed = $state(false);
  let busy = $state(false);
  let error = $state("");

  const name = (s: Shooter) => `${s.first_name} ${s.last_name}`;

  async function pick(s: Shooter | null) {
    target = s;
    preview = null;
    confirmed = false;
    error = "";
    if (!s) return;
    const result = await api.GET("/api/shooters/{shooterID}/merge-preview", {
      params: { path: { shooterID: source.id }, query: { target_id: s.id } },
    });
    if (target?.id !== s.id) return;
    if (result.data) preview = result.data;
    else error = failure(result);
  }

  async function merge() {
    if (!target || !preview) return;
    busy = true;
    error = "";
    const result = await api.POST("/api/shooters/{shooterID}/merge", {
      params: { path: { shooterID: source.id } },
      body: { target_id: target.id },
    });
    busy = false;
    if (result.response.ok) {
      onmerged(target);
      return;
    }
    if (result.response.status === 409 && result.error && "conflicts" in result.error) {
      preview = result.error;
      confirmed = false;
      return;
    }
    error = failure(result);
  }
</script>

<Dialog
  title={`Merge ${name(source)}`}
  class="w-[min(40rem,calc(100vw-1.5rem))]"
  onclose={oncancel}
>
  <div class="flex flex-col gap-4">
    <p class="m-0">
      All entries of {name(source)} move to the shooter you pick, and {name(source)} is deleted. The target
      keeps its own details.
    </p>
    {#if error}<ErrorState title={error} />{/if}
    <ShooterPicker label="Merge into" exclude={[source.id]} onselect={pick} />

    {#if preview && target}
      <div class="flex flex-col gap-2">
        <h3 class="m-0 text-base">Entries to move</h3>
        {#if preview.entries.length === 0}
          <p class="m-0 text-muted">No entries.</p>
        {:else}
          <ul class="m-0 list-disc pl-5">
            {#each preview.entries as e (e.entry_id)}
              <li>{e.match_name}, {e.match_date}, {e.match_status}</li>
            {/each}
          </ul>
        {/if}
      </div>

      {#if preview.waid_conflict}
        <ErrorState
          title={`The shooters have different WAIDs (${source.waid} and ${target.waid}), so they are different people.`}
        />
      {:else}
        <p class="m-0">
          {preview.resulting_waid
            ? `WAID after merge: ${preview.resulting_waid}`
            : "Neither shooter has a WAID."}
        </p>
      {/if}

      {#if preview.conflicts.length > 0}
        <ErrorState title="Both shooters are entered in these matches. Withdraw or fix one first:">
          {#snippet action()}
            <ul class="m-0 list-disc pl-5">
              {#each preview!.conflicts as c (c.match_id)}
                <li>
                  <a class="text-link" href={`/admin/matches/${c.match_id}`}>{c.match_name}</a>, {c.match_date}
                </li>
              {/each}
            </ul>
          {/snippet}
        </ErrorState>
      {/if}

      {#if preview.can_merge}
        <label class="flex items-center gap-2">
          <input type="checkbox" class="size-5 accent-brand" bind:checked={confirmed} />
          I understand this cannot be undone
        </label>
      {/if}
    {/if}

    <div class="flex flex-wrap gap-2">
      <Button variant="danger" disabled={busy || !preview?.can_merge || !confirmed} onclick={merge}
        >Merge shooters</Button
      >
      <Button variant="secondary" onclick={oncancel}>Cancel</Button>
    </div>
  </div>
</Dialog>

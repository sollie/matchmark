<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";

  let { role, waids = $bindable() }: { role: string; waids: string[] } = $props();

  let candidates = $state<{ username: string; waid: string }[]>([]);
  const pickable = $derived(candidates.filter((c) => !waids.includes(c.waid)));
  const lower = $derived(role.toLowerCase());

  onMount(async () => {
    const { data } = await api.GET("/api/officials/candidates");
    if (data) candidates = data.candidates;
  });

  function pick(e: Event & { currentTarget: HTMLSelectElement }) {
    const waid = e.currentTarget.value;
    e.currentTarget.value = "";
    if (waid) waids.push(waid);
  }
</script>

<fieldset class="officials">
  <legend>{role}s</legend>
  {#each waids as _, i (i)}
    <span class="official">
      <label>
        {role} WAID {i + 1}
        <input bind:value={waids[i]} maxlength="32" inputmode="numeric" size="10" />
      </label>
      <button type="button" onclick={() => waids.splice(i, 1)}>
        Remove {lower}
        {i + 1}
      </button>
    </span>
  {/each}
  {#if pickable.length > 0}
    <select aria-label={`Pick ${lower} from users`} value="" onchange={pick}>
      <option value="">Pick a user…</option>
      {#each pickable as c (c.waid)}
        <option value={c.waid}>{c.username} ({c.waid})</option>
      {/each}
    </select>
  {/if}
  <button type="button" onclick={() => waids.push("")}>Add {lower}</button>
</fieldset>

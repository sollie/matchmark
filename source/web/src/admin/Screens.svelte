<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { openStream } from "../lib/sse";
  import type { components } from "../lib/api/schema";

  type Screen = components["schemas"]["Screen"];

  type AssignableMatch = { id: number; name: string; date: string; status: string };

  let screens = $state<Screen[]>([]);
  let matches = $state<AssignableMatch[]>([]);
  let code = $state("");
  let name = $state("");
  let allMatch = $state("");
  let renaming = $state<number | null>(null);
  let newName = $state("");
  let error = $state("");
  let notice = $state("");

  async function load() {
    const [s, m] = await Promise.all([
      api.GET("/api/screens"),
      api.GET("/api/screens/assignable-matches"),
    ]);
    if (s.data) screens = s.data.screens;
    else error = failure(s);
    if (m.data) matches = m.data.matches;
    else error = failure(m);
  }

  onMount(() => {
    void load();
    const stream = openStream({
      url: "/api/screens/stream",
      onEvent: (event) => {
        if (event === "changed") void load();
      },
    });
    return () => stream.close();
  });

  async function act(request: Promise<{ response: Response }>, message: string) {
    error = "";
    notice = "";
    const result = await request;
    if (!result.response.ok) {
      error = failure(result as Parameters<typeof failure>[0]);
      return false;
    }
    notice = message;
    await load();
    return true;
  }

  async function approve(e: SubmitEvent) {
    e.preventDefault();
    const ok = await act(
      api.POST("/api/screens/approve", { body: { code: code.trim(), name: name.trim() } }),
      "Screen approved.",
    );
    if (ok) {
      code = "";
      name = "";
    }
  }

  const assign = (s: Screen, value: string) =>
    act(
      api.PUT("/api/screens/{screenID}", {
        params: { path: { screenID: s.id } },
        body: { match_id: value === "" ? null : Number(value) },
      }),
      `${s.name} updated.`,
    );

  async function rename(e: SubmitEvent, s: Screen) {
    e.preventDefault();
    const ok = await act(
      api.PUT("/api/screens/{screenID}", {
        params: { path: { screenID: s.id } },
        body: { name: newName.trim() },
      }),
      "Screen renamed.",
    );
    if (ok) renaming = null;
  }

  const remove = (s: Screen) =>
    act(
      api.DELETE("/api/screens/{screenID}", { params: { path: { screenID: s.id } } }),
      `${s.name} removed.`,
    );

  const applyToAll = () =>
    act(
      api.PUT("/api/screens/assignment", { body: { match_id: Number(allMatch) } }),
      "Match applied to all screens.",
    );

  const label = (m: AssignableMatch) =>
    `${m.name} (${m.date}${m.status === "draft" ? ", draft" : ""})`;
</script>

<h1>Screens</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if notice}<p role="status">{notice}</p>{/if}

<form class="row card" onsubmit={approve}>
  <label>
    Pending code
    <input
      bind:value={code}
      maxlength="6"
      minlength="6"
      autocomplete="off"
      autocapitalize="characters"
      required
    />
  </label>
  <label>
    Screen name
    <input bind:value={name} maxlength="60" placeholder="Screen N" autocomplete="off" />
  </label>
  <button type="submit" class="primary">Approve screen</button>
</form>
<p class="muted">
  A spectator display shows a 6-character code after choosing remote control. Enter it here to add
  the screen.
</p>

<form
  class="row card"
  onsubmit={(e) => {
    e.preventDefault();
    void applyToAll();
  }}
>
  <label>
    Match for all screens
    <select bind:value={allMatch} required>
      <option value="" disabled>Select a match</option>
      {#each matches as m (m.id)}
        <option value={String(m.id)}>{label(m)}</option>
      {/each}
    </select>
  </label>
  <button type="submit" disabled={screens.length === 0}>Apply to all screens</button>
</form>

<div class="card scroll">
  <table>
    <thead><tr><th>Screen</th><th>Status</th><th>Match</th><th>Actions</th></tr></thead>
    <tbody>
      {#each screens as s (s.id)}
        <tr>
          <td>
            {#if renaming === s.id}
              <form class="row" onsubmit={(e) => rename(e, s)}>
                <label>
                  New name for {s.name}
                  <input bind:value={newName} maxlength="60" required />
                </label>
                <button type="submit" class="primary">Save</button>
                <button type="button" onclick={() => (renaming = null)}>Cancel</button>
              </form>
            {:else}
              {s.name}
            {/if}
          </td>
          <td>{s.online ? "Online" : "Offline"}</td>
          <td>
            <select
              aria-label="Match for {s.name}"
              value={s.match_id === null ? "" : String(s.match_id)}
              onchange={(e) => assign(s, e.currentTarget.value)}
            >
              <option value="">Default (newest active match)</option>
              {#each matches as m (m.id)}
                <option value={String(m.id)}>{label(m)}</option>
              {/each}
              {#if s.match_id !== null && !matches.some((m) => m.id === s.match_id)}
                <option value={String(s.match_id)}>Finished match</option>
              {/if}
            </select>
          </td>
          <td>
            <button
              type="button"
              onclick={() => {
                renaming = s.id;
                newName = s.name;
              }}>Rename</button
            >
            <button type="button" onclick={() => remove(s)}>Remove</button>
          </td>
        </tr>
      {:else}
        <tr><td colspan="4" class="muted">No screens yet.</td></tr>
      {/each}
    </tbody>
  </table>
</div>

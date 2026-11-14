<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { openStream } from "../lib/sse";
  import type { components } from "../lib/api/schema";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import NativeSelect from "../lib/ui/NativeSelect.svelte";
  import Table from "../lib/ui/Table.svelte";

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

<div class="mx-auto flex max-w-5xl flex-col gap-4">
  <h1 class="m-0 text-xl">Screens</h1>
  {#if error}<ErrorState title={error} />{/if}
  {#if notice}<p class="m-0 font-medium text-success-fg" role="status">{notice}</p>{/if}

  <div class="grid gap-4 lg:grid-cols-2">
    <form
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
      onsubmit={approve}
      aria-label="Approve a screen"
    >
      <div class="grid gap-3 sm:grid-cols-2">
        <Field label="Pending code">
          <Input
            bind:value={code}
            maxlength={6}
            minlength={6}
            autocomplete="off"
            autocapitalize="characters"
            class="font-mono tracking-widest uppercase"
            required
          />
        </Field>
        <Field label="Screen name">
          <Input bind:value={name} maxlength={60} placeholder="Screen N" autocomplete="off" />
        </Field>
      </div>
      <div><Button type="submit">Approve screen</Button></div>
      <p class="m-0 text-sm text-muted">
        A spectator display shows a 6-character code after choosing remote control. Enter it here to
        add the screen.
      </p>
    </form>

    <form
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
      aria-label="Assign all screens"
      onsubmit={(e) => {
        e.preventDefault();
        void applyToAll();
      }}
    >
      <Field label="Match for all screens">
        <NativeSelect bind:value={allMatch} required>
          <option value="" disabled>Select a match</option>
          {#each matches as m (m.id)}
            <option value={String(m.id)}>{label(m)}</option>
          {/each}
        </NativeSelect>
      </Field>
      <div>
        <Button type="submit" variant="secondary" disabled={screens.length === 0}
          >Apply to all screens</Button
        >
      </div>
    </form>
  </div>

  {#if screens.length === 0}
    <EmptyState
      title="No screens yet."
      description="Approve a pending code to add the first screen."
    />
  {:else}
    <Table label="Screens">
      <thead><tr><th>Screen</th><th>Status</th><th>Match</th><th>Actions</th></tr></thead>
      <tbody>
        {#each screens as s (s.id)}
          <tr>
            <td>
              {#if renaming === s.id}
                <form class="flex flex-wrap items-end gap-2" onsubmit={(e) => rename(e, s)}>
                  <Field label="New name for {s.name}" class="min-w-40 flex-1">
                    <Input bind:value={newName} maxlength={60} required />
                  </Field>
                  <Button type="submit" size="sm">Save</Button>
                  <Button variant="secondary" size="sm" onclick={() => (renaming = null)}
                    >Cancel</Button
                  >
                </form>
              {:else}
                <span class="font-semibold">{s.name}</span>
              {/if}
            </td>
            <td
              ><Badge tone={s.online ? "success" : "neutral"}
                >{s.online ? "Online" : "Offline"}</Badge
              ></td
            >
            <td class="min-w-56">
              <NativeSelect
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
              </NativeSelect>
            </td>
            <td>
              <span class="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => {
                    renaming = s.id;
                    newName = s.name;
                  }}>Rename</Button
                >
                <Button variant="danger" size="sm" onclick={() => remove(s)}>Remove</Button>
              </span>
            </td>
          </tr>
        {/each}
      </tbody>
    </Table>
  {/if}
</div>

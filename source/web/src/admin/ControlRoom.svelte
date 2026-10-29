<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";

  type ControlRoom = components["schemas"]["ControlRoom"];
  type Lane = components["schemas"]["ControlRoomLane"];

  let {
    matchId,
    version,
    changed,
  }: { matchId: number; version: number; changed: () => Promise<void> } = $props();

  let room = $state<ControlRoom | null>(null);
  let error = $state("");

  async function load() {
    const result = await api.GET("/api/matches/{matchID}/control-room", {
      params: { path: { matchID: matchId } },
    });
    if (result.data) room = result.data;
    else error = failure(result);
  }

  $effect(() => {
    void version;
    void load();
  });

  async function run(call: Promise<{ error?: unknown; response: Response }>) {
    error = "";
    const result = await call;
    if (!result.response.ok) error = failure(result);
    await load();
    await changed();
  }

  const segmentPath = (lane: Lane, segment: number) => ({
    matchID: matchId,
    entryID: lane.entry_id,
    segment,
  });

  const startNext = () =>
    run(
      api.POST("/api/matches/{matchID}/next-segment", { params: { path: { matchID: matchId } } }),
    );
  const acceptAll = () =>
    run(api.POST("/api/matches/{matchID}/accept-all", { params: { path: { matchID: matchId } } }));
  const accept = (lane: Lane, segment: number) =>
    run(
      api.POST("/api/matches/{matchID}/entries/{entryID}/segments/{segment}/accept", {
        params: { path: segmentPath(lane, segment) },
      }),
    );
  const unlock = (lane: Lane, segment: number) =>
    run(
      api.POST("/api/matches/{matchID}/entries/{entryID}/segments/{segment}/unlock", {
        params: { path: segmentPath(lane, segment) },
      }),
    );

  const statusLabel = (lane: Lane) =>
    lane.status === "accepted"
      ? "Accepted"
      : lane.status === "awaiting_acceptance"
        ? "Awaiting acceptance"
        : lane.disputed
          ? "Disputed"
          : "Open";
  const statusTone = (lane: Lane) =>
    lane.status === "accepted"
      ? "success"
      : lane.status === "awaiting_acceptance"
        ? "info"
        : lane.disputed
          ? "danger"
          : "warning";
</script>

<section class="card">
  <h2>Control room</h2>
  {#if room}
    <p>Current segment: <strong>{room.current_segment}</strong></p>
    <div class="row">
      <button class="primary" disabled={!room.can_start_next} onclick={startNext}
        >Start next segment</button
      >
      <button class="primary" disabled={!room.can_accept_all} onclick={acceptAll}>Accept all</button
      >
    </div>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <div class="scroll">
      <table class="lanes">
        <thead>
          <tr
            ><th>Lane</th><th>Shooter</th><th>Scorer</th><th>Status</th><th>Disputes</th><th
              >Device</th
            ><th>Actions</th></tr
          >
        </thead>
        <tbody>
          {#each room.lanes as lane (lane.entry_id)}
            <tr>
              <td>{lane.lane}</td>
              <td>{lane.shooter_name}</td>
              <td>{lane.scorer_name || "Admin"}</td>
              <td><span class="badge {statusTone(lane)}">{statusLabel(lane)}</span></td>
              <td>{lane.disputes}</td>
              <td>
                <span class="badge {lane.paired ? 'success' : 'neutral'}"
                  >{lane.paired ? "Paired" : "Not paired"}</span
                >
                <span class="badge {lane.connected ? 'success' : 'neutral'}"
                  >{lane.connected ? "Connected" : "Not connected"}</span
                >
              </td>
              <td class="actions">
                {#if lane.status === "awaiting_acceptance"}
                  <button
                    class="secondary"
                    aria-label={`Control room lane ${lane.lane}: Accept`}
                    onclick={() => accept(lane, room!.current_segment)}>Accept</button
                  >
                {/if}
                {#if lane.status !== "open"}
                  <button
                    class="secondary"
                    aria-label={`Control room lane ${lane.lane}: Unlock`}
                    onclick={() => unlock(lane, room!.current_segment)}>Unlock</button
                  >
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted">
      Connected is best-effort: it shows an open connection from the lane's device, not whether the
      phone works. It is not stored.
    </p>
  {:else if error}
    <p class="error" role="alert">{error}</p>
  {:else}
    <p class="muted">Loading…</p>
  {/if}
</section>

<style>
  .lanes .actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }
  .lanes button {
    min-height: var(--touch);
  }
</style>

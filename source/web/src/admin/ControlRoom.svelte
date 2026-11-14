<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import { segmentLabel } from "../lib/rulebook.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import Table from "../lib/ui/Table.svelte";

  type ControlRoom = components["schemas"]["ControlRoom"];
  type Lane = components["schemas"]["ControlRoomLane"];

  let {
    matchId,
    course,
    version,
    changed,
  }: { matchId: number; course: string; version: number; changed: () => Promise<void> } = $props();

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

  const count = (status: Lane["status"]) =>
    room?.lanes.filter((l) => l.status === status).length ?? 0;

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

<section class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <h2 class="m-0 text-lg">Control room</h2>
  {#if room}
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="m-0 text-base">
        Current segment:
        <strong class="text-xl">{segmentLabel(course, room.current_segment)}</strong>
      </p>
      <div class="flex flex-wrap gap-2">
        <Button disabled={!room.can_start_next} onclick={startNext}>Start next segment</Button>
        <Button disabled={!room.can_accept_all} onclick={acceptAll}>Accept all</Button>
      </div>
    </div>
    <ul class="m-0 flex list-none flex-wrap gap-2 p-0">
      <li><Badge tone="warning">{count("open")} open</Badge></li>
      <li><Badge tone="info">{count("awaiting_acceptance")} awaiting</Badge></li>
      <li><Badge tone="success">{count("accepted")} accepted</Badge></li>
    </ul>
    {#if error}<ErrorState title={error} />{/if}
    <Table label="Control room lanes">
      <thead>
        <tr>
          <th>Lane</th><th>Shooter</th><th>Scorer</th><th>Status</th><th>Disputes</th><th>Device</th
          ><th><span class="sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody>
        {#each room.lanes as lane (lane.entry_id)}
          <tr>
            <td class="font-semibold">{lane.lane}</td>
            <td>{lane.shooter_name}</td>
            <td>{lane.scorer_name || "Admin"}</td>
            <td><Badge tone={statusTone(lane)}>{statusLabel(lane)}</Badge></td>
            <td>{lane.disputes}</td>
            <td>
              <span class="flex flex-wrap gap-1">
                <Badge tone={lane.paired ? "success" : "neutral"}
                  >{lane.paired ? "Paired" : "Not paired"}</Badge
                >
                <Badge tone={lane.connected ? "success" : "neutral"}
                  >{lane.connected ? "Connected" : "Not connected"}</Badge
                >
              </span>
            </td>
            <td>
              <span class="flex flex-wrap justify-end gap-2">
                {#if lane.status === "awaiting_acceptance"}
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`Control room lane ${lane.lane}: Accept`}
                    onclick={() => accept(lane, room!.current_segment)}>Accept</Button
                  >
                {/if}
                {#if lane.status !== "open"}
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`Control room lane ${lane.lane}: Unlock`}
                    onclick={() => unlock(lane, room!.current_segment)}>Unlock</Button
                  >
                {/if}
              </span>
            </td>
          </tr>
        {/each}
      </tbody>
    </Table>
    <p class="m-0 text-sm text-muted">
      Connected is best-effort: it shows an open connection from the lane's device, not whether the
      phone works. It is not stored.
    </p>
  {:else if error}
    <ErrorState title={error} />
  {:else}
    <LoadingState />
  {/if}
</section>

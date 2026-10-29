<script lang="ts">
  import { onMount } from "svelte";
  import { router, matchRoute } from "../lib/router.svelte";
  import { admin, refreshAdminSession } from "../lib/session.svelte";
  import ChangePassword from "./ChangePassword.svelte";
  import Login from "./Login.svelte";
  import Shooters from "./Shooters.svelte";
  import Matches from "./Matches.svelte";
  import Match from "./Match.svelte";
  import Screens from "./Screens.svelte";
  import Users from "./Users.svelte";
  import AuditLog from "./AuditLog.svelte";
  import ClubSettings from "./ClubSettings.svelte";

  onMount(refreshAdminSession);

  const currentMatch = $derived(matchRoute("/admin/matches/:id", router.path));

  const activeSection = $derived(
    router.path.startsWith("/admin/shooters")
      ? "shooters"
      : router.path === "/admin" || router.path.startsWith("/admin/matches")
        ? "matches"
        : "",
  );
</script>

{#if !admin.checked}
  <main><p class="muted">Loading…</p></main>
{:else if !admin.authed}
  <Login />
{:else if admin.mustChangePassword}
  <ChangePassword forced />
{:else}
  <nav>
    <a href="/admin" aria-current={activeSection === "matches" ? "page" : undefined}>Matches</a>
    <a href="/admin/shooters" aria-current={activeSection === "shooters" ? "page" : undefined}
      >Shooters</a
    >
    <a href="/">Spectator view</a>
  </nav>
  <main>
    {#if router.path === "/admin" || router.path === "/admin/matches"}
      <Matches />
    {:else if router.path === "/admin/shooters"}
      <Shooters />
    {:else if router.path === "/admin/users"}
      {#if admin.isAdmin}
        <Users />
      {:else}
        <h1>Not allowed</h1>
        <p>Only admins can manage users.</p>
      {/if}
    {:else if router.path === "/admin/screens"}
      {#if admin.isAdmin}
        <Screens />
      {:else}
        <h1>Not allowed</h1>
        <p>Only admins can manage screens.</p>
      {/if}
    {:else if router.path === "/admin/club"}
      {#if admin.isAdmin}
        <ClubSettings />
      {:else}
        <h1>Not allowed</h1>
        <p>Only admins can edit the club settings.</p>
      {/if}
    {:else if router.path === "/admin/audit"}
      {#if admin.isAdmin}
        <AuditLog />
      {:else}
        <h1>Not allowed</h1>
        <p>Only admins can read the audit log.</p>
      {/if}
    {:else if router.path === "/admin/password"}
      <ChangePassword />
    {:else if currentMatch}
      {#key currentMatch.id}
        <Match id={Number(currentMatch.id)} />
      {/key}
    {:else}
      <h1>Page not found</h1>
    {/if}
  </main>
{/if}

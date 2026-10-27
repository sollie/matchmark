<script lang="ts">
  import { onMount } from "svelte";
  import { router, matchRoute, navigate } from "../lib/router.svelte";
  import { admin, refreshAdminSession, logout } from "../lib/session.svelte";
  import ChangePassword from "./ChangePassword.svelte";
  import Login from "./Login.svelte";
  import Shooters from "./Shooters.svelte";
  import Matches from "./Matches.svelte";
  import Match from "./Match.svelte";
  import Users from "./Users.svelte";
  import AuditLog from "./AuditLog.svelte";
  import ClubSettings from "./ClubSettings.svelte";

  onMount(refreshAdminSession);

  const currentMatch = $derived(matchRoute("/admin/matches/:id", router.path));

  const activeSection = $derived(
    router.path.startsWith("/admin/shooters")
      ? "shooters"
      : router.path.startsWith("/admin/users")
        ? "users"
        : router.path.startsWith("/admin/audit")
          ? "audit"
          : router.path.startsWith("/admin/club")
            ? "club"
            : "matches",
  );

  async function signOut() {
    await logout();
    navigate("/admin");
  }
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
    <a href="/admin/shooters" aria-current={activeSection === "shooters" ? "page" : undefined}>Shooters</a>
    {#if admin.isAdmin}
      <a href="/admin/users" aria-current={activeSection === "users" ? "page" : undefined}>Users</a>
      <a href="/admin/club" aria-current={activeSection === "club" ? "page" : undefined}>Club</a>
      <a href="/admin/audit" aria-current={activeSection === "audit" ? "page" : undefined}>Audit log</a>
    {/if}
    {#if admin.isAdmin && admin.canDownloadBackup}
      <a href="/api/admin/backup" download>Backup</a>
    {/if}
    <a href="/admin/password" aria-current={router.path === "/admin/password" ? "page" : undefined}>Change password</a>
    <a href="/">Spectator view</a>
    <span class="spacer"></span>
    <button onclick={signOut}>Sign out</button>
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

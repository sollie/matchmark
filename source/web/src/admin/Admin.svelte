<script lang="ts">
  import { onMount } from "svelte";
  import { router, matchRoute, navigate } from "../lib/router.svelte";
  import { admin, refreshAdminSession, logout } from "../lib/session.svelte";
  import Login from "./Login.svelte";
  import Shooters from "./Shooters.svelte";
  import Competitions from "./Competitions.svelte";
  import Competition from "./Competition.svelte";

  onMount(refreshAdminSession);

  const competitionRoute = $derived(matchRoute("/admin/competitions/:id", router.path));

  const activeSection = $derived(
    router.path.startsWith("/admin/shooters") ? "shooters" : "competitions",
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
{:else}
  <nav>
    <a href="/admin" aria-current={activeSection === "competitions" ? "page" : undefined}>Competitions</a>
    <a href="/admin/shooters" aria-current={activeSection === "shooters" ? "page" : undefined}>Shooters</a>
    <a href="/api/admin/backup" download>Backup</a>
    <a href="/">Spectator view</a>
    <span class="spacer"></span>
    <button onclick={signOut}>Sign out</button>
  </nav>
  <main>
    {#if router.path === "/admin" || router.path === "/admin/competitions"}
      <Competitions />
    {:else if router.path === "/admin/shooters"}
      <Shooters />
    {:else if competitionRoute}
      {#key competitionRoute.id}
        <Competition id={Number(competitionRoute.id)} />
      {/key}
    {:else}
      <h1>Page not found</h1>
    {/if}
  </main>
{/if}

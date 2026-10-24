<script lang="ts">
  import { router } from "./lib/router.svelte";
  import Spectator from "./routes/Spectator.svelte";
  import Scorer from "./routes/Scorer.svelte";
  import Admin from "./admin/Admin.svelte";
  import NotFound from "./routes/NotFound.svelte";
  import Footer from "./lib/Footer.svelte";
  import { club, clubLogoUrl, refreshClub } from "./lib/club.svelte";

  $effect(() => {
    void refreshClub();
  });
</script>

<header class="site-header">
  <a class="brand-link" href="/"><img src="/brand/logo-horizontal-dark.svg" alt="MatchMark" width="158" height="32" /></a>
  {#if club.hasLogo || club.name}
    <span class="club-badge">
      {#if club.hasLogo}<img src={clubLogoUrl(club.version)} alt="" />{/if}
      {#if club.name}<span>{club.name}</span>{/if}
    </span>
  {/if}
</header>

<div class="app-content">
{#if router.path === "/"}
  <Spectator />
{:else if router.path === "/score"}
  <Scorer />
{:else if router.path === "/admin" || router.path.startsWith("/admin/")}
  <Admin />
{:else}
  <NotFound />
{/if}
</div>

<Footer />

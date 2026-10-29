<script lang="ts">
  import { router } from "./lib/router.svelte";
  import Spectator from "./routes/Spectator.svelte";
  import Results from "./routes/Results.svelte";
  import Scorecard from "./routes/Scorecard.svelte";
  import Scorer from "./routes/Scorer.svelte";
  import Admin from "./admin/Admin.svelte";
  import NotFound from "./routes/NotFound.svelte";
  import Footer from "./lib/Footer.svelte";
  import SiteHeader from "./lib/SiteHeader.svelte";
  import { refreshClub } from "./lib/club.svelte";

  $effect(() => {
    void refreshClub();
  });
</script>

<SiteHeader />

<div class="app-content">
  {#if router.path === "/"}
    <Spectator />
  {:else if router.path === "/results" || router.path.startsWith("/results/")}
    <Results />
  {:else if router.path.startsWith("/scorecard/")}
    <Scorecard />
  {:else if router.path === "/score"}
    <Scorer />
  {:else if router.path === "/admin" || router.path.startsWith("/admin/")}
    <Admin />
  {:else}
    <NotFound />
  {/if}
</div>

<Footer />

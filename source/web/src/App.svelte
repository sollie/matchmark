<script lang="ts">
  import { router } from "./lib/router.svelte";
  import Spectator from "./routes/Spectator.svelte";
  import Scorer from "./routes/Scorer.svelte";
  import Admin from "./admin/Admin.svelte";
  import NotFound from "./routes/NotFound.svelte";
  import Footer from "./lib/Footer.svelte";
</script>

<header class="site-header">
  <a class="brand-name" href="/">MatchMark</a>
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

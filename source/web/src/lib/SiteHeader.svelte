<script lang="ts">
  import { router } from "./router.svelte";
  import { club, clubLogoUrl } from "./club.svelte";
  import { admin } from "./session.svelte";
  import AccountIcon from "./AccountIcon.svelte";
  import AccountMenu from "./AccountMenu.svelte";

  const scorerRoute = $derived(router.path === "/score");
  const adminRoute = $derived(router.path === "/admin" || router.path.startsWith("/admin/"));
</script>

<header class="site-header">
  <a class="brand-link" href="/"
    ><img src="/brand/logo-horizontal-dark.svg" alt="MatchMark" width="158" height="32" /></a
  >
  {#if club.hasLogo || club.name}
    <span class="club-badge">
      {#if club.hasLogo}<img src={clubLogoUrl(club.version)} alt="" />{/if}
      {#if club.name}<span class="club-name">{club.name}</span>{/if}
    </span>
  {/if}
  {#if adminRoute}
    {#if admin.authed && !admin.mustChangePassword}<AccountMenu />{/if}
  {:else if !scorerRoute}
    <a class="account-link" href="/admin" aria-label="Sign in" title="Sign in"><AccountIcon /></a>
  {/if}
</header>

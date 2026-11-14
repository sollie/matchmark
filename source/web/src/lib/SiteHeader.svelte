<script lang="ts">
  import { router } from "./router.svelte";
  import { club, clubLogoUrl } from "./club.svelte";
  import { admin } from "./session.svelte";
  import SiteMenu from "./SiteMenu.svelte";
  import AccountMenu from "./AccountMenu.svelte";

  const scorerRoute = $derived(router.path === "/score");
  const spectatorRoute = $derived(router.path === "/");
  const adminRoute = $derived(router.path === "/admin" || router.path.startsWith("/admin/"));
</script>

<header
  class="site-header print:hidden flex min-h-12 items-center gap-2 border-b border-border bg-header px-4"
>
  <a
    class="inline-flex items-center rounded-md outline-offset-2 focus-visible:outline-3 focus-visible:outline-white"
    href="/"
    ><img
      class="block h-8 w-auto"
      src="/brand/logo-horizontal-dark.svg"
      alt="MatchMark"
      width="158"
      height="32"
    /></a
  >
  {#if club.hasLogo || club.name}
    <span
      class="club-badge ml-auto inline-flex min-w-0 items-center gap-2 font-semibold text-white"
    >
      {#if club.hasLogo}<img
          class="block h-8 max-w-24 object-contain"
          src={clubLogoUrl(club.version)}
          alt=""
        />{/if}
      {#if club.name}<span class={["truncate", club.hasLogo && "max-[480px]:hidden"]}
          >{club.name}</span
        >{/if}
    </span>
  {:else}
    <span class="ml-auto"></span>
  {/if}
  {#if adminRoute}
    {#if admin.authed && !admin.mustChangePassword}<AccountMenu />{/if}
  {:else}
    <SiteMenu adminLink={!scorerRoute} remoteControl={spectatorRoute} />
  {/if}
</header>

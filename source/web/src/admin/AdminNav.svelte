<script lang="ts">
  import Calendar from "@lucide/svelte/icons/calendar";
  import Download from "@lucide/svelte/icons/download";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import History from "@lucide/svelte/icons/scroll-text";
  import Building from "@lucide/svelte/icons/landmark";
  import KeyRound from "@lucide/svelte/icons/key-round";
  import Monitor from "@lucide/svelte/icons/monitor";
  import Target from "@lucide/svelte/icons/target";
  import User from "@lucide/svelte/icons/user";
  import Users from "@lucide/svelte/icons/users";
  import { router } from "../lib/router.svelte";
  import { admin } from "../lib/session.svelte";
  import { cn } from "../lib/ui/utils";

  let { group = "all" }: { group?: "all" | "more" } = $props();

  const matchesActive = $derived(
    router.path === "/admin" || router.path.startsWith("/admin/matches"),
  );

  function active(path: string) {
    return router.path === path || router.path.startsWith(path + "/");
  }

  const link =
    "flex min-h-11 items-center gap-3 rounded-md px-3 text-fg no-underline outline-offset-[-2px] hover:bg-surface-alt focus-visible:outline-3 focus-visible:outline-focus aria-[current=page]:bg-brand-tint aria-[current=page]:font-semibold aria-[current=page]:text-brand";
  const heading = "m-0 mt-3 px-3 text-sm font-semibold text-muted";
</script>

{#if group === "all"}
  <a href="/admin" class={link} aria-current={matchesActive ? "page" : undefined}
    ><Calendar class="size-5" aria-hidden="true" />Matches</a
  >
  <a
    href="/admin/shooters"
    class={link}
    aria-current={active("/admin/shooters") ? "page" : undefined}
    ><Target class="size-5" aria-hidden="true" />Shooters</a
  >
{/if}
{#if admin.isAdmin}
  <h2 class={heading}>Administration</h2>
  <a href="/admin/users" class={link} aria-current={active("/admin/users") ? "page" : undefined}
    ><Users class="size-5" aria-hidden="true" />Users</a
  >
  <a href="/admin/screens" class={link} aria-current={active("/admin/screens") ? "page" : undefined}
    ><Monitor class="size-5" aria-hidden="true" />Screens</a
  >
  <a href="/admin/club" class={link} aria-current={active("/admin/club") ? "page" : undefined}
    ><Building class="size-5" aria-hidden="true" />Club</a
  >
  <a href="/admin/audit" class={link} aria-current={active("/admin/audit") ? "page" : undefined}
    ><History class="size-5" aria-hidden="true" />Audit log</a
  >
  {#if admin.canDownloadBackup}
    <a href="/api/admin/backup" download class={link}
      ><Download class="size-5" aria-hidden="true" />Backup</a
    >
  {/if}
{/if}
<h2 class={heading}>Profile</h2>
<a href="/admin/account" class={link} aria-current={active("/admin/account") ? "page" : undefined}
  ><User class="size-5" aria-hidden="true" />WAID</a
>
<a href="/admin/password" class={link} aria-current={active("/admin/password") ? "page" : undefined}
  ><KeyRound class="size-5" aria-hidden="true" />Change password</a
>
<a class={cn(link, "mt-3 text-muted")} href="/" target="_blank" rel="noopener"
  ><ExternalLink class="size-5" aria-hidden="true" />Spectator view</a
>

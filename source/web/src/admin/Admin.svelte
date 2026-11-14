<script lang="ts">
  import { onMount } from "svelte";
  import { router, matchRoute } from "../lib/router.svelte";
  import { admin, refreshAdminSession } from "../lib/session.svelte";
  import Account from "./Account.svelte";
  import ChangePassword from "./ChangePassword.svelte";
  import Login from "./Login.svelte";
  import Shooters from "./Shooters.svelte";
  import Matches from "./Matches.svelte";
  import Match from "./Match.svelte";
  import Screens from "./Screens.svelte";
  import Users from "./Users.svelte";
  import AuditLog from "./AuditLog.svelte";
  import ClubSettings from "./ClubSettings.svelte";
  import AdminNav from "./AdminNav.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import Drawer from "../lib/ui/Drawer.svelte";
  import Calendar from "@lucide/svelte/icons/calendar";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import Target from "@lucide/svelte/icons/target";

  onMount(refreshAdminSession);

  let moreOpen = $state(false);

  $effect(() => {
    void router.path;
    moreOpen = false;
  });

  const tab =
    "flex min-h-14 flex-1 cursor-pointer flex-col items-center justify-center gap-0.5 border-0 bg-transparent px-2 text-sm text-muted no-underline outline-offset-[-2px] focus-visible:outline-3 focus-visible:outline-focus aria-[current=page]:font-semibold aria-[current=page]:text-brand";

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
  <main class="p-4"><LoadingState /></main>
{:else if !admin.authed}
  <Login />
{:else if admin.mustChangePassword}
  <ChangePassword forced />
{:else}
  <div class="min-[820px]:flex min-[820px]:items-start">
    <nav
      aria-label="Admin sections"
      class="sticky top-0 hidden max-h-dvh w-60 shrink-0 flex-col gap-1 self-start overflow-y-auto border-r border-border bg-surface p-3 min-[820px]:flex"
    >
      <AdminNav />
    </nav>
    <nav
      aria-label="Admin tabs"
      class="fixed inset-x-0 bottom-0 z-30 flex border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] min-[820px]:hidden"
    >
      <a href="/admin" class={tab} aria-current={activeSection === "matches" ? "page" : undefined}
        ><Calendar class="size-5" aria-hidden="true" />Matches</a
      >
      <a
        href="/admin/shooters"
        class={tab}
        aria-current={activeSection === "shooters" ? "page" : undefined}
        ><Target class="size-5" aria-hidden="true" />Shooters</a
      >
      <button type="button" class={tab} aria-haspopup="dialog" onclick={() => (moreOpen = true)}
        ><Ellipsis class="size-5" aria-hidden="true" />More</button
      >
    </nav>
    <Drawer bind:open={moreOpen} title="More">
      <div class="flex flex-col gap-1"><AdminNav group="more" /></div>
    </Drawer>
    <main class="w-full min-w-0 flex-1 pb-20 min-[820px]:pb-4">
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
      {:else if router.path === "/admin/account"}
        <Account />
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
  </div>
{/if}

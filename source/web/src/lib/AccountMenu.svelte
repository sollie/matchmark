<script lang="ts">
  import { router, navigate } from "./router.svelte";
  import { admin, logout } from "./session.svelte";
  import AccountIcon from "./AccountIcon.svelte";

  let open = $state(false);
  let root: HTMLElement | undefined = $state();
  let trigger: HTMLButtonElement | undefined = $state();

  const ownsPage = $derived(
    ["/admin/users", "/admin/club", "/admin/audit", "/admin/password"].some(
      (p) => router.path === p || router.path.startsWith(p + "/"),
    ),
  );

  $effect(() => {
    void router.path;
    open = false;
  });

  function onWindowClick(e: MouseEvent) {
    if (open && root && !root.contains(e.target as Node)) open = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && open) {
      open = false;
      trigger?.focus();
    }
  }

  async function signOut() {
    open = false;
    await logout();
    navigate("/");
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKeydown} />

<div class="account-menu" bind:this={root}>
  <button
    bind:this={trigger}
    type="button"
    class="account-link"
    aria-label="Account menu"
    title="Account menu"
    aria-haspopup="menu"
    aria-expanded={open}
    aria-current={ownsPage ? "page" : undefined}
    onclick={() => (open = !open)}
  >
    <AccountIcon />
  </button>
  {#if open}
    <div class="account-panel">
      <p class="account-who">
        <strong>{admin.username}</strong>
        <span class="muted">{admin.isAdmin ? "Admin" : "User"}</span>
      </p>
      {#if admin.isAdmin}
        <h2>Administration</h2>
        <a href="/admin/users">Users</a>
        <a href="/admin/club">Club</a>
        <a href="/admin/audit">Audit log</a>
        {#if admin.canDownloadBackup}<a href="/api/admin/backup" download>Backup</a>{/if}
      {/if}
      <h2>Account</h2>
      <a href="/admin/password">Change password</a>
      <button type="button" onclick={signOut}>Sign out</button>
    </div>
  {/if}
</div>

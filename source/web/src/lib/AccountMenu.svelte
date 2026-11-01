<script lang="ts">
  import { router, navigate } from "./router.svelte";
  import { admin, logout } from "./session.svelte";
  import MenuShell from "./MenuShell.svelte";
  import ThemeSelector from "./ThemeSelector.svelte";

  const ownsPage = $derived(
    ["/admin/users", "/admin/screens", "/admin/club", "/admin/audit", "/admin/password"].some(
      (p) => router.path === p || router.path.startsWith(p + "/"),
    ),
  );

  async function signOut(close: () => void) {
    close();
    await logout();
    navigate("/");
  }
</script>

<MenuShell label="Account menu" current={ownsPage}>
  {#snippet children(close)}
    <p class="account-who">
      <strong>{admin.username}</strong>
      <span class="muted">{admin.isAdmin ? "Admin" : "User"}</span>
    </p>
    {#if admin.isAdmin}
      <h2>Administration</h2>
      <a href="/admin/users">Users</a>
      <a href="/admin/screens">Screens</a>
      <a href="/admin/club">Club</a>
      <a href="/admin/audit">Audit log</a>
      {#if admin.canDownloadBackup}<a href="/api/admin/backup" download>Backup</a>{/if}
    {/if}
    <ThemeSelector />
    <h2>Account</h2>
    <a href="/admin/password">Change password</a>
    <button type="button" onclick={() => signOut(close)}>Sign out</button>
  {/snippet}
</MenuShell>

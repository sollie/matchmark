<script lang="ts">
  import { navigate } from "./router.svelte";
  import { admin, logout } from "./session.svelte";
  import MenuShell from "./MenuShell.svelte";
  import ThemeSelector from "./ThemeSelector.svelte";

  async function signOut(close: () => void) {
    close();
    await logout();
    navigate("/");
  }
</script>

<MenuShell label="Account menu">
  {#snippet children(close)}
    <p class="m-0 flex flex-col border-b border-border px-3 py-2">
      <strong>{admin.username}</strong>
      <span class="text-muted">{admin.isAdmin ? "Admin" : "User"}</span>
    </p>
    <ThemeSelector />
    <h2>Account</h2>
    <button type="button" onclick={() => signOut(close)}>Sign out</button>
  {/snippet}
</MenuShell>

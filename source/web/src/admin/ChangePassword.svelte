<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import { admin, logout } from "../lib/session.svelte";

  let { forced = false, ondone = () => {} }: { forced?: boolean; ondone?: () => void } = $props();

  let current = $state("");
  let next = $state("");
  let confirm = $state("");
  let error = $state("");
  let done = $state(false);
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    done = false;
    if (next !== confirm) {
      error = "The new passwords do not match";
      return;
    }
    busy = true;
    const { error: err, response } = await api.POST("/api/admin/password", {
      body: { current_password: current, new_password: next },
    });
    busy = false;
    if (response.ok) {
      current = next = confirm = "";
      done = true;
      admin.mustChangePassword = false;
      ondone();
    } else if (response.status === 401) {
      error = "The current password is incorrect";
    } else if (response.status === 429) {
      error = "Too many attempts. Try again later.";
    } else {
      error = errorMessage(err, "Could not change the password");
    }
  }

  async function signOut() {
    await logout();
  }
</script>

<main class={forced ? "centered" : undefined}>
  <div class="card">
    <h1>Change password</h1>
    {#if forced}
      <p>You must choose a new password before you can continue.</p>
    {/if}
    <form onsubmit={submit}>
      <label>
        Current password
        <input type="password" bind:value={current} autocomplete="current-password" required />
      </label>
      <label>
        New password
        <input type="password" bind:value={next} autocomplete="new-password" minlength="16" maxlength="128" required />
      </label>
      <label>
        Confirm new password
        <input type="password" bind:value={confirm} autocomplete="new-password" minlength="16" maxlength="128" required />
      </label>
      <p class="muted">16 to 128 characters. Your other sessions are signed out.</p>
      <button class="primary" type="submit" disabled={busy}>Change password</button>
      {#if forced}<button type="button" onclick={signOut}>Sign out</button>{/if}
    </form>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    {#if done}<p role="status">Password changed.</p>{/if}
  </div>
</main>

<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import PasswordInput from "../lib/PasswordInput.svelte";
  import { navigate } from "../lib/router.svelte";
  import { logout, refreshAdminSession } from "../lib/session.svelte";

  let { forced = false, ondone = () => {} }: { forced?: boolean; ondone?: () => void } = $props();

  const minLength = 16;

  let current = $state("");
  let next = $state("");
  let confirm = $state("");
  let error = $state("");
  let done = $state(false);
  let busy = $state(false);

  const lengthMet = $derived(next.length >= minLength);
  const matches = $derived(confirm !== "" && confirm === next);
  const mismatch = $derived(confirm !== "" && confirm !== next);

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
      await refreshAdminSession();
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
    {#if done && !forced}
      <div class="success" role="status">
        <span class="check" aria-hidden="true">✓</span>
        <h1>Password changed</h1>
        <p>Your other sessions were signed out.</p>
        <button class="primary" type="button" onclick={() => navigate("/admin")}>Continue</button>
      </div>
    {:else}
      <h1>Change password</h1>
      {#if forced}
        <p>You must choose a new password before you can continue.</p>
      {/if}
      <form onsubmit={submit}>
        <PasswordInput
          label="Current password"
          bind:value={current}
          autocomplete="current-password"
          required
        />
        <PasswordInput
          label="New password"
          bind:value={next}
          autocomplete="new-password"
          class={lengthMet ? "valid" : undefined}
          minlength={minLength}
          maxlength={128}
          required
        />
        <p class={lengthMet ? "hint valid" : "hint muted"} aria-live="polite">
          {#if lengthMet}✓ {minLength} characters or more{:else}{next.length} / {minLength} characters{/if}
        </p>
        <PasswordInput
          label="Confirm new password"
          bind:value={confirm}
          autocomplete="new-password"
          class={matches ? "valid" : undefined}
          aria-invalid={mismatch ? "true" : undefined}
          minlength={minLength}
          maxlength={128}
          required
        />
        <p class={matches ? "hint valid" : mismatch ? "hint error" : "hint"} aria-live="polite">
          {#if matches}✓ Passwords match{:else if mismatch}Passwords do not match{/if}
        </p>
        <p class="muted">16 to 128 characters. Your other sessions are signed out.</p>
        <button class="primary" type="submit" disabled={busy}>Change password</button>
        {#if forced}<button type="button" onclick={signOut}>Sign out</button>{/if}
      </form>
      {#if error}<p class="error" role="alert">{error}</p>{/if}
    {/if}
  </div>
</main>

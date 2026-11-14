<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import PasswordInput from "../lib/PasswordInput.svelte";
  import { navigate } from "../lib/router.svelte";
  import { logout, refreshAdminSession } from "../lib/session.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";

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

<main class="mx-auto max-w-md p-4 pt-6 sm:pt-10">
  <div class="rounded-xl border border-border bg-surface p-6 shadow-card">
    {#if done && !forced}
      <div class="flex flex-col items-center gap-3 text-center" role="status">
        <span
          class="inline-flex size-12 items-center justify-center rounded-full bg-success-bg text-xl text-success-fg"
          aria-hidden="true">✓</span
        >
        <h1 class="m-0 text-xl">Password changed</h1>
        <p class="m-0 text-muted">Your other sessions were signed out.</p>
        <Button onclick={() => navigate("/admin")}>Continue</Button>
      </div>
    {:else}
      <h1 class="m-0 mb-1 text-xl">Change password</h1>
      {#if forced}
        <p class="m-0 mb-2 text-muted">You must choose a new password before you can continue.</p>
      {/if}
      <form class="mt-4 flex flex-col gap-4" onsubmit={submit}>
        <PasswordInput
          label="Current password"
          bind:value={current}
          autocomplete="current-password"
          required
        />
        <div class="flex flex-col gap-1">
          <PasswordInput
            label="New password"
            bind:value={next}
            autocomplete="new-password"
            minlength={minLength}
            maxlength={128}
            required
          />
          <p
            class={["m-0 text-sm", lengthMet ? "text-success-fg" : "text-muted"]}
            aria-live="polite"
          >
            {#if lengthMet}✓ {minLength} characters or more{:else}{next.length} / {minLength} characters{/if}
          </p>
        </div>
        <div class="flex flex-col gap-1">
          <PasswordInput
            label="Confirm new password"
            bind:value={confirm}
            autocomplete="new-password"
            aria-invalid={mismatch ? "true" : undefined}
            minlength={minLength}
            maxlength={128}
            required
          />
          <p
            class={["m-0 min-h-5 text-sm", matches ? "text-success-fg" : "text-danger-fg"]}
            aria-live="polite"
          >
            {#if matches}✓ Passwords match{:else if mismatch}Passwords do not match{/if}
          </p>
        </div>
        <p class="m-0 text-sm text-muted">
          16 to 128 characters. Your other sessions are signed out.
        </p>
        {#if error}<ErrorState title={error} />{/if}
        <div class="flex flex-wrap gap-2">
          <Button type="submit" disabled={busy}>Change password</Button>
          {#if forced}<Button variant="secondary" onclick={signOut}>Sign out</Button>{/if}
        </div>
      </form>
    {/if}
  </div>
</main>

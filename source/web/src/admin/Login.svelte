<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import PasswordInput from "../lib/PasswordInput.svelte";
  import { navigate } from "../lib/router.svelte";
  import { refreshAdminSession } from "../lib/session.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";

  let username = $state("");
  let password = $state("");
  let error = $state("");
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    const { error: err, response } = await api.POST("/api/admin/login", {
      body: { username, password },
    });
    busy = false;
    if (response.ok) {
      password = "";
      navigate("/admin");
      await refreshAdminSession();
    } else {
      error =
        response.status === 429
          ? "Too many attempts. Try again later."
          : errorMessage(err, "Login failed");
    }
  }
</script>

<main class="mx-auto flex max-w-md flex-col gap-5 p-4 pt-8 sm:pt-12">
  <div class="rounded-xl border border-border bg-surface p-6 shadow-card">
    <div class="login-logo">
      <img
        class="logo-light"
        src="/brand/logo-stacked.svg"
        alt="MatchMark"
        width="163"
        height="88"
      />
      <img
        class="logo-dark"
        src="/brand/logo-stacked-dark.svg"
        alt="MatchMark"
        width="163"
        height="88"
      />
    </div>
    <h1 class="mb-4 text-center text-xl">Admin sign in</h1>
    <form class="flex flex-col gap-4" onsubmit={submit}>
      <Field label="Username">
        <Input type="text" bind:value={username} autocomplete="username" required />
      </Field>
      <PasswordInput
        label="Password"
        bind:value={password}
        autocomplete="current-password"
        required
      />
      <Button type="submit" disabled={busy}>Sign in</Button>
    </form>
    {#if error}<ErrorState class="mt-4" title="Sign in failed" message={error} />{/if}
  </div>
</main>

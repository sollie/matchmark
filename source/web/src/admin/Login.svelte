<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import PasswordInput from "../lib/PasswordInput.svelte";
  import { navigate } from "../lib/router.svelte";
  import { refreshAdminSession } from "../lib/session.svelte";

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

<main class="centered">
  <div class="card">
    <picture class="login-logo">
      <source srcset="/brand/logo-stacked-dark.svg" media="(prefers-color-scheme: dark)" />
      <img src="/brand/logo-stacked.svg" alt="MatchMark" width="163" height="88" />
    </picture>
    <h1>Admin sign in</h1>
    <form onsubmit={submit}>
      <label>
        Username
        <input type="text" bind:value={username} autocomplete="username" required />
      </label>
      <PasswordInput
        label="Password"
        bind:value={password}
        autocomplete="current-password"
        required
      />
      <button class="primary" type="submit" disabled={busy}>Sign in</button>
    </form>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  </div>
</main>

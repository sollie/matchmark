<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import { admin } from "../lib/session.svelte";

  let username = $state("");
  let password = $state("");
  let error = $state("");
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    const { error: err, response } = await api.POST("/api/admin/login", { body: { username, password } });
    busy = false;
    if (response.ok) {
      password = "";
      admin.authed = true;
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
    <h1>Admin sign in</h1>
    <form onsubmit={submit}>
      <label>
        Username
        <input type="text" bind:value={username} autocomplete="username" required />
      </label>
      <label>
        Password
        <input type="password" bind:value={password} autocomplete="current-password" required />
      </label>
      <button class="primary" type="submit" disabled={busy}>Sign in</button>
    </form>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  </div>
</main>

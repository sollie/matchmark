<script lang="ts">
  import { api } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import { admin } from "../lib/session.svelte";

  let password = $state("");
  let error = $state("");
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    const { error: err, response } = await api.POST("/api/admin/login", { body: { password } });
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

<main>
  <h1>MatchMark admin</h1>
  <form class="row" onsubmit={submit}>
    <label>
      Password
      <input type="password" bind:value={password} autocomplete="current-password" required />
    </label>
    <button type="submit" disabled={busy}>Sign in</button>
  </form>
  {#if error}<p class="error">{error}</p>{/if}
</main>

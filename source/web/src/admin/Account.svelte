<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { admin, refreshAdminSession } from "../lib/session.svelte";

  let waid = $state(admin.waid);
  let error = $state("");
  let saved = $state(false);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    saved = false;
    const result = await api.PUT("/api/admin/waid", { body: { waid } });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    await refreshAdminSession();
    waid = admin.waid;
    saved = true;
  }
</script>

<h1>Account</h1>
<form class="row card" onsubmit={save} aria-label="Account WAID">
  <label>
    WAID
    <input bind:value={waid} maxlength="32" inputmode="numeric" />
  </label>
  <button type="submit" class="primary">Save WAID</button>
</form>
<p class="muted">
  Your WAID makes you the default Referee of the matches you create. Leave it empty to clear it.
</p>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if saved}<p role="status">WAID saved.</p>{/if}

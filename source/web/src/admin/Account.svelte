<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
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

<div class="mx-auto flex max-w-xl flex-col gap-4">
  <h1 class="m-0 text-xl">Account</h1>
  <form
    class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-6 shadow-card"
    onsubmit={save}
    aria-label="Account WAID"
  >
    <Field
      label="WAID"
      hint="Your WAID makes you the default Referee of the matches you create. Leave it empty to clear it."
    >
      <Input bind:value={waid} maxlength={32} inputmode="numeric" />
    </Field>
    <div><Button type="submit">Save WAID</Button></div>
    {#if error}<ErrorState title={error} />{/if}
    {#if saved}<p class="m-0 text-success-fg" role="status">WAID saved.</p>{/if}
  </form>
</div>

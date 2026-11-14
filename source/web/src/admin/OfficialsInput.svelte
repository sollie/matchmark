<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import Button from "../lib/ui/Button.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import Plus from "@lucide/svelte/icons/plus";
  import X from "@lucide/svelte/icons/x";

  let { role, waids = $bindable() }: { role: string; waids: string[] } = $props();

  let candidates = $state<{ username: string; waid: string }[]>([]);
  let picked = $state("");
  const pickable = $derived(candidates.filter((c) => !waids.includes(c.waid)));
  const lower = $derived(role.toLowerCase());

  onMount(async () => {
    const { data } = await api.GET("/api/officials/candidates");
    if (data) candidates = data.candidates;
  });

  function pick(waid: string) {
    picked = "";
    if (waid) waids.push(waid);
  }
</script>

<fieldset class="m-0 flex min-w-0 flex-col gap-2 rounded-lg border border-border p-3">
  <legend class="px-1 text-sm font-semibold text-muted">{role}s</legend>
  {#each waids as _, i (i)}
    <div class="flex items-end gap-2">
      <Field label={`${role} WAID ${i + 1}`} class="min-w-0 flex-1">
        <Input bind:value={waids[i]} maxlength={32} inputmode="numeric" />
      </Field>
      <Button variant="secondary" onclick={() => waids.splice(i, 1)}>
        <X class="size-4" aria-hidden="true" />
        Remove {lower}
        {i + 1}
      </Button>
    </div>
  {/each}
  <div class="flex flex-wrap items-center gap-2">
    {#if pickable.length > 0}
      <Select
        class="w-64"
        label={`Pick ${lower} from users`}
        placeholder="Pick a user…"
        bind:value={picked}
        onValueChange={pick}
        items={pickable.map((c) => ({ value: c.waid, label: `${c.username} (${c.waid})` }))}
      />
    {/if}
    <Button variant="secondary" onclick={() => waids.push("")}>
      <Plus class="size-4" aria-hidden="true" />
      Add {lower}
    </Button>
  </div>
</fieldset>

<script lang="ts">
  let { role, waids = $bindable() }: { role: string; waids: string[] } = $props();
</script>

<fieldset class="officials">
  <legend>{role}s</legend>
  {#each waids as _, i (i)}
    <span class="official">
      <label>
        {role} WAID {i + 1}
        <input bind:value={waids[i]} maxlength="32" inputmode="numeric" size="10" />
      </label>
      <button type="button" onclick={() => waids.splice(i, 1)}>
        Remove {role.toLowerCase()}
        {i + 1}
      </button>
    </span>
  {/each}
  <button type="button" onclick={() => waids.push("")}>Add {role.toLowerCase()}</button>
</fieldset>

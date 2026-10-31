<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  let {
    label,
    value = $bindable(""),
    ...rest
  }: { label: string; value?: string } & Omit<HTMLInputAttributes, "type" | "value"> = $props();

  const id = $props.id();
  let shown = $state(false);
</script>

<div class="field">
  <label for={id}>{label}</label>
  <div class="password-input">
    <input {id} type={shown ? "text" : "password"} bind:value {...rest} />
    <button
      type="button"
      class="toggle"
      aria-pressed={shown}
      aria-label={shown ? "Hide password" : "Show password"}
      onclick={() => (shown = !shown)}
    >
      <svg
        viewBox="0 0 24 24"
        width="18"
        height="18"
        aria-hidden="true"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12z" />
        <circle cx="12" cy="12" r="3" />
        {#if shown}<path d="M3 3l18 18" />{/if}
      </svg>
    </button>
  </div>
</div>

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  label {
    font-size: var(--text-sm);
    font-weight: 500;
  }
  .password-input {
    display: flex;
    gap: 0.3rem;
  }
  .password-input input {
    flex: 1;
    min-width: 0;
  }
  .toggle {
    padding: 0.3rem 0.5rem;
    line-height: 0;
  }
</style>

<script lang="ts">
  import { theme, setTheme, type Theme } from "./theme.svelte";

  const options: { value: Theme; label: string }[] = [
    { value: "auto", label: "Auto" },
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
  ];
</script>

<div
  class="flex flex-wrap items-center gap-2 border-b border-border px-3 py-2"
  role="radiogroup"
  aria-label="Mode"
>
  <span class="text-sm font-semibold text-muted" aria-hidden="true">Mode</span>
  <div class="flex flex-1 gap-1">
    {#each options as option (option.value)}
      <label
        class="relative flex min-h-11 flex-1 cursor-pointer items-center justify-center rounded-md border border-border px-2 text-fg has-checked:border-brand has-checked:bg-brand has-checked:text-on-brand has-focus-visible:outline-3 has-focus-visible:outline-offset-2 has-focus-visible:outline-focus"
      >
        <input
          class="absolute inset-0 m-0 cursor-pointer opacity-0"
          type="radio"
          name="matchmark-theme"
          value={option.value}
          checked={theme.value === option.value}
          onchange={() => setTheme(option.value)}
        />
        {option.label}
      </label>
    {/each}
  </div>
</div>

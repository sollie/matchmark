<script lang="ts">
  import { theme, setTheme, type Theme } from "./theme.svelte";

  const options: { value: Theme; label: string }[] = [
    { value: "auto", label: "Auto" },
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
  ];
</script>

<div class="theme-selector" role="radiogroup" aria-label="Theme">
  <span class="theme-title" aria-hidden="true">Theme</span>
  <div class="theme-options">
    {#each options as option (option.value)}
      <label>
        <input
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

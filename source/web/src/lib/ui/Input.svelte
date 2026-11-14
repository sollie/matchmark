<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";
  import { cn } from "./utils";
  import { controlClass } from "./control";

  let { value = $bindable(), class: className, ...rest }: HTMLInputAttributes = $props();
</script>

<input
  bind:value
  class={cn(controlClass, "placeholder:text-muted aria-invalid:border-danger", className)}
  {...rest}
/>

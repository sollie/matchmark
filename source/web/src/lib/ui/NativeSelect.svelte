<script lang="ts">
  import type { HTMLSelectAttributes } from "svelte/elements";
  import { cn } from "./utils";
  import { controlClass } from "./control";

  let { value = $bindable(), class: className, children, ...rest }: HTMLSelectAttributes = $props();
</script>

<select bind:value class={cn(controlClass, "cursor-pointer", className)} {...rest}>
  {@render children?.()}
</select>

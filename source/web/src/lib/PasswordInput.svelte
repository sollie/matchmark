<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";
  import Eye from "@lucide/svelte/icons/eye";
  import EyeOff from "@lucide/svelte/icons/eye-off";
  import Button from "./ui/Button.svelte";
  import Input from "./ui/Input.svelte";

  let {
    label,
    value = $bindable(""),
    ...rest
  }: { label: string; value?: string } & Omit<HTMLInputAttributes, "type" | "value"> = $props();

  const id = $props.id();
  let shown = $state(false);
</script>

<div class="flex flex-col gap-1">
  <label class="text-sm font-medium" for={id}>{label}</label>
  <div class="flex gap-1">
    <Input {id} type={shown ? "text" : "password"} bind:value {...rest} />
    <Button
      variant="secondary"
      size="icon"
      class="shrink-0"
      aria-pressed={shown}
      aria-label={shown ? "Hide password" : "Show password"}
      onclick={() => (shown = !shown)}
    >
      {#if shown}<EyeOff class="size-5" aria-hidden="true" />{:else}<Eye
          class="size-5"
          aria-hidden="true"
        />{/if}
    </Button>
  </div>
</div>

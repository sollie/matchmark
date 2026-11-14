<script lang="ts">
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import { liveFirstName, liveLastName, normalizeFirstName, normalizeLastName } from "../lib/names";

  let { first = $bindable(), last = $bindable() }: { first: string; last: string } = $props();

  function live(e: Event, transform: (s: string) => string, set: (s: string) => void) {
    const input = e.currentTarget as HTMLInputElement;
    const next = transform(input.value);
    if (next !== input.value) {
      const caret = input.selectionStart;
      input.value = next;
      input.setSelectionRange(caret, caret);
    }
    set(next);
  }
</script>

<Field label="First name">
  <Input
    value={first}
    maxlength={64}
    required
    oninput={(e) => live(e, liveFirstName, (s) => (first = s))}
    onblur={() => (first = normalizeFirstName(first))}
  />
</Field>
<Field label="Last name">
  <Input
    value={last}
    maxlength={64}
    required
    oninput={(e) => live(e, liveLastName, (s) => (last = s))}
    onblur={() => (last = normalizeLastName(last))}
  />
</Field>

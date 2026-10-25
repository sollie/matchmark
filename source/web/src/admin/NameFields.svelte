<script lang="ts">
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

<label>
  First name
  <input
    value={first}
    maxlength="64"
    required
    oninput={(e) => live(e, liveFirstName, (s) => (first = s))}
    onblur={() => (first = normalizeFirstName(first))}
  />
</label>
<label>
  Last name
  <input
    value={last}
    maxlength="64"
    required
    oninput={(e) => live(e, liveLastName, (s) => (last = s))}
    onblur={() => (last = normalizeLastName(last))}
  />
</label>

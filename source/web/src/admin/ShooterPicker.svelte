<script lang="ts">
  import { api } from "../lib/api/client";
  import type { components } from "../lib/api/schema";

  type Shooter = components["schemas"]["Shooter"];

  let {
    label = "Shooter",
    exclude = [],
    required = false,
    initial = null,
    onselect,
    onadd,
  }: {
    label?: string;
    exclude?: number[];
    required?: boolean;
    initial?: Shooter | null;
    onselect: (shooter: Shooter | null) => void;
    onadd?: (query: string) => void;
  } = $props();

  const uid = $props.id();
  const listId = `${uid}-list`;
  const optionId = (i: number) => `${uid}-opt-${i}`;

  let text = $state(initial ? `${initial.first_name} ${initial.last_name}` : "");
  let results = $state<Shooter[]>([]);
  let open = $state(false);
  let loading = $state(false);
  let searched = $state(false);
  let active = $state(-1);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let sequence = 0;

  const visible = $derived(results.filter((s) => !exclude.includes(s.id)));
  const optionCount = $derived(visible.length + (onadd ? 1 : 0));

  async function search(q: string) {
    const current = ++sequence;
    loading = true;
    const result = await api.GET("/api/shooters", { params: { query: { q: q.trim() || undefined, limit: Math.min(20 + exclude.length, 100) } } });
    if (current !== sequence) return;
    loading = false;
    results = result.data?.shooters ?? [];
    searched = true;
    active = visible.length > 0 ? 0 : -1;
  }

  function oninput() {
    onselect(null);
    open = true;
    active = -1;
    clearTimeout(timer);
    timer = setTimeout(() => search(text), 250);
  }

  function onfocus() {
    open = true;
    if (!searched) search(text);
  }

  function choose(s: Shooter) {
    clearTimeout(timer);
    sequence++;
    text = `${s.first_name} ${s.last_name}`;
    open = false;
    onselect(s);
  }

  function add() {
    open = false;
    onadd?.(text.trim());
  }

  function activate(i: number) {
    if (i < visible.length) choose(visible[i]);
    else add();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!open) {
        open = true;
        if (!searched) search(text);
        return;
      }
      if (optionCount === 0) return;
      const step = e.key === "ArrowDown" ? 1 : -1;
      active = (active + step + optionCount) % optionCount;
    } else if (e.key === "Enter" && open && active >= 0 && active < optionCount) {
      e.preventDefault();
      activate(active);
    } else if (e.key === "Escape" && open) {
      e.preventDefault();
      open = false;
    }
  }

  function onblur(e: FocusEvent) {
    const next = e.relatedTarget as Node | null;
    if (!next || !(e.currentTarget as HTMLElement).parentElement?.contains(next)) open = false;
  }
</script>

<div class="picker">
  <label>
    {label}
    <input
      type="text"
      role="combobox"
      autocomplete="off"
      aria-expanded={open}
      aria-controls={listId}
      aria-autocomplete="list"
      aria-activedescendant={open && active >= 0 ? optionId(active) : undefined}
      {required}
      bind:value={text}
      {oninput}
      {onfocus}
      {onkeydown}
      {onblur}
    />
  </label>
  <ul id={listId} role="listbox" aria-label={label} hidden={!open}>
    {#each visible as s, i (s.id)}
      <li id={optionId(i)} role="option" aria-selected={i === active} class:active={i === active} onmousedown={(e) => e.preventDefault()} onclick={() => choose(s)}>
        {s.first_name} {s.last_name} · {s.waid || "no WAID"} · {s.club} · {s.country}
      </li>
    {/each}
    {#if searched && !loading && visible.length === 0}
      <li class="empty" role="presentation">No match</li>
    {/if}
    {#if onadd}
      <li id={optionId(visible.length)} role="option" aria-selected={active === visible.length} class:active={active === visible.length} onmousedown={(e) => e.preventDefault()} onclick={add}>
        Search WA1500 / add new
      </li>
    {/if}
  </ul>
</div>

<style>
  .picker { position: relative; }
  ul { position: absolute; z-index: 10; left: 0; right: 0; margin: 0; padding: 0; list-style: none; background: var(--surface, #fff); border: 1px solid var(--border, #ccc); max-height: 16rem; overflow-y: auto; }
  li { padding: var(--space-2, 0.5rem) var(--space-3, 0.75rem); cursor: pointer; }
  li.active { background: var(--accent-soft, #e6efff); }
  li.empty { cursor: default; color: var(--muted, #666); }
  [hidden] { display: none; }
</style>

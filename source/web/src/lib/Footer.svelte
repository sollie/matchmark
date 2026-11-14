<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "./api/client";
  import { brand } from "./brand";

  let version = $state("");

  onMount(async () => {
    try {
      const { data } = await api.GET("/api/version");
      version = data?.version ?? "";
    } catch {
      version = "";
    }
  });
</script>

<footer
  class="flex flex-wrap justify-center gap-x-4 gap-y-1 border-t border-border px-4 py-3 text-sm text-muted print:hidden"
>
  <span>{brand.name}{version ? ` ${version}` : ""}</span>
  <span>© {brand.copyrightYear} {brand.copyrightHolder}</span>
  <a href={brand.repoUrl} target="_blank" rel="noopener noreferrer">Source code on GitHub</a>
</footer>

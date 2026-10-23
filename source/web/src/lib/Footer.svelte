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

<footer class="site-footer">
  <span>{brand.name}{version ? ` ${version}` : ""}</span>
  <span>© {brand.copyrightYear} {brand.copyrightHolder}</span>
  <a href={brand.repoUrl} target="_blank" rel="noopener noreferrer">Source code on GitHub</a>
</footer>

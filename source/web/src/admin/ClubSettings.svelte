<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { club, clubLogoUrl, refreshClub } from "../lib/club.svelte";

  let name = $state("");
  let loaded = $state(false);
  let error = $state("");
  let message = $state("");
  let fileInput = $state<HTMLInputElement>();

  onMount(async () => {
    await refreshClub();
    name = club.name;
    loaded = true;
  });

  async function saveName(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    message = "";
    const result = await api.PUT("/api/club", { body: { name } });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    message = "Club name saved.";
    await refreshClub();
    name = club.name;
  }

  async function uploadLogo(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    message = "";
    const file = fileInput?.files?.[0];
    if (!file) {
      error = "Choose a PNG or JPEG file first.";
      return;
    }
    const response = await fetch("/api/club/logo", {
      method: "PUT",
      credentials: "same-origin",
      headers: { "Content-Type": file.type },
      body: file,
    });
    if (!response.ok) {
      error =
        response.status === 422
          ? "The logo must be a PNG or JPEG of at most 512 KB and 2048 by 2048 pixels."
          : `Request failed (${response.status})`;
      return;
    }
    message = "Logo saved.";
    if (fileInput) fileInput.value = "";
    await refreshClub();
  }

  async function removeLogo() {
    error = "";
    message = "";
    const result = await api.DELETE("/api/club/logo");
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    message = "Logo removed.";
    await refreshClub();
  }
</script>

<h1>Club settings</h1>

{#if loaded}

{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}

<section aria-labelledby="club-name-heading">
  <h2 id="club-name-heading">Club name</h2>
  <form onsubmit={saveName}>
    <label>Name <input bind:value={name} maxlength="100" /></label>
    <button type="submit">Save name</button>
  </form>
</section>

<section aria-labelledby="club-logo-heading">
  <h2 id="club-logo-heading">Club logo</h2>
  {#if club.hasLogo}
    <p><img class="club-logo-preview" src={clubLogoUrl(club.version)} alt="Current club logo" /></p>
    <button type="button" onclick={removeLogo}>Remove logo</button>
  {:else}
    <p class="muted">No logo uploaded.</p>
  {/if}
  <form onsubmit={uploadLogo}>
    <label>Logo file <input type="file" accept="image/png,image/jpeg" bind:this={fileInput} /></label>
    <button type="submit">Upload logo</button>
  </form>
  <p class="muted">PNG or JPEG, at most 512 KB and 2048 by 2048 pixels.</p>
</section>
{/if}

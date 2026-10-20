<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";

  type User = components["schemas"]["User"];
  type Role = components["schemas"]["UserRole"];

  let users = $state<User[]>([]);
  let username = $state("");
  let role = $state<Role>("user");
  let password = $state("");
  let error = $state("");
  let created = $state("");

  async function load() {
    const result = await api.GET("/api/users");
    if (result.data) users = result.data.users;
    else error = failure(result);
  }

  onMount(load);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    created = "";
    const result = await api.POST("/api/users", { body: { username, role, password } });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    created = `User ${result.data?.username} created. They must change the password at first login.`;
    username = "";
    password = "";
    role = "user";
    await load();
  }
</script>

<h1>Users</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if created}<p role="status">{created}</p>{/if}

<form class="row card" onsubmit={create}>
  <label>
    New username
    <input bind:value={username} minlength="3" maxlength="32" pattern="[A-Za-z0-9._\-]+" autocomplete="off" required />
  </label>
  <label>
    Role
    <select bind:value={role}>
      <option value="user">User</option>
      <option value="admin">Admin</option>
    </select>
  </label>
  <label>
    Temporary password
    <input type="password" bind:value={password} minlength="16" maxlength="128" autocomplete="new-password" required />
  </label>
  <button type="submit" class="primary">Create user</button>
</form>
<p class="muted">
  Usernames use a-z, 0-9, '.', '_' and '-' (3 to 32 characters) and are never reused. The password is 16 to 128
  characters and must be changed at first login.
</p>

<div class="card scroll">
  <table>
    <thead>
      <tr><th>Username</th><th>Role</th><th>Status</th></tr>
    </thead>
    <tbody>
      {#each users as u (u.id)}
        <tr>
          <td>{u.username}</td>
          <td>{u.role === "admin" ? "Admin" : "User"}</td>
          <td>{!u.active ? "Deactivated" : u.must_change_password ? "Must change password" : "Active"}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

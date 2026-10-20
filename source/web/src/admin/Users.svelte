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
  let resetFor = $state<number | null>(null);
  let resetPassword = $state("");

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

  async function act(request: Promise<{ response: Response }>, message: string) {
    error = "";
    created = "";
    const result = await request;
    if (!result.response.ok) {
      error = failure(result as Parameters<typeof failure>[0]);
      return;
    }
    created = message;
    await load();
  }

  const params = (u: User) => ({ params: { path: { userID: u.id } } });

  const setActive = (u: User, active: boolean) =>
    act(
      api.POST(active ? "/api/users/{userID}/reactivate" : "/api/users/{userID}/deactivate", params(u)),
      `User ${u.username} ${active ? "reactivated" : "deactivated"}.`,
    );

  const setRole = (u: User, next: Role) =>
    act(
      api.PUT("/api/users/{userID}/role", { params: { path: { userID: u.id } }, body: { role: next } }),
      `User ${u.username} is now ${next === "admin" ? "an admin" : "a user"}.`,
    );

  async function reset(e: SubmitEvent, u: User) {
    e.preventDefault();
    await act(
      api.POST("/api/users/{userID}/reset-password", { params: { path: { userID: u.id } }, body: { password: resetPassword } }),
      `Password for ${u.username} reset. They must change it at next login.`,
    );
    if (!error) {
      resetFor = null;
      resetPassword = "";
    }
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
      <tr><th>Username</th><th>Role</th><th>Status</th><th>Actions</th></tr>
    </thead>
    <tbody>
      {#each users as u (u.id)}
        <tr>
          <td>{u.username}</td>
          <td>{u.role === "admin" ? "Admin" : "User"}</td>
          <td>{!u.active ? "Deactivated" : u.must_change_password ? "Must change password" : "Active"}</td>
          <td>
            {#if resetFor === u.id}
              <form class="row" onsubmit={(e) => reset(e, u)}>
                <label>
                  Temporary password for {u.username}
                  <input type="password" bind:value={resetPassword} minlength="16" maxlength="128" autocomplete="new-password" required />
                </label>
                <button type="submit" class="primary">Confirm reset</button>
                <button type="button" onclick={() => (resetFor = null)}>Cancel</button>
              </form>
            {:else}
              <button type="button" onclick={() => setActive(u, !u.active)}>{u.active ? "Deactivate" : "Reactivate"}</button>
              <button type="button" onclick={() => setRole(u, u.role === "admin" ? "user" : "admin")}>
                {u.role === "admin" ? "Make user" : "Make admin"}
              </button>
              <button type="button" onclick={() => { resetFor = u.id; resetPassword = ""; }}>Reset password</button>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

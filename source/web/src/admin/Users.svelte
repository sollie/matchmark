<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import PasswordInput from "../lib/PasswordInput.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import Table from "../lib/ui/Table.svelte";
  import { refreshAdminSession } from "../lib/session.svelte";
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
  let waidFor = $state<number | null>(null);
  let waidValue = $state("");

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

  const roleItems = [
    { value: "user", label: "User" },
    { value: "admin", label: "Admin" },
  ];

  const statusOf = (u: User) =>
    !u.active
      ? { label: "Deactivated", tone: "danger" as const }
      : u.must_change_password
        ? { label: "Must change password", tone: "warning" as const }
        : { label: "Active", tone: "success" as const };

  const params = (u: User) => ({ params: { path: { userID: u.id } } });

  const setActive = (u: User, active: boolean) =>
    act(
      api.POST(
        active ? "/api/users/{userID}/reactivate" : "/api/users/{userID}/deactivate",
        params(u),
      ),
      `User ${u.username} ${active ? "reactivated" : "deactivated"}.`,
    );

  const setBackup = (u: User, allowed: boolean) =>
    act(
      api.PUT("/api/users/{userID}/backup-permission", {
        params: { path: { userID: u.id } },
        body: { can_download_backup: allowed },
      }),
      `Backup download ${allowed ? "allowed" : "forbidden"} for ${u.username}.`,
    ).then(refreshAdminSession);

  const setRole = (u: User, next: Role) =>
    act(
      api.PUT("/api/users/{userID}/role", {
        params: { path: { userID: u.id } },
        body: { role: next },
      }),
      `User ${u.username} is now ${next === "admin" ? "an admin" : "a user"}.`,
    );

  async function saveWaid(e: SubmitEvent, u: User) {
    e.preventDefault();
    await act(
      api.PUT("/api/users/{userID}/waid", {
        params: { path: { userID: u.id } },
        body: { waid: waidValue },
      }),
      `WAID for ${u.username} saved.`,
    );
    if (!error) waidFor = null;
  }

  async function reset(e: SubmitEvent, u: User) {
    e.preventDefault();
    await act(
      api.POST("/api/users/{userID}/reset-password", {
        params: { path: { userID: u.id } },
        body: { password: resetPassword },
      }),
      `Password for ${u.username} reset. They must change it at next login.`,
    );
    if (!error) {
      resetFor = null;
      resetPassword = "";
    }
  }
</script>

<div class="flex flex-col gap-4">
  <h1 class="m-0 text-xl">Users</h1>
  {#if error}<ErrorState title={error} />{/if}
  {#if created}<p class="m-0 text-success-fg" role="status">{created}</p>{/if}

  <form
    class="grid gap-4 rounded-xl border border-border bg-surface p-4 shadow-card sm:grid-cols-2 lg:grid-cols-[1fr_12rem_1fr_auto] lg:items-start"
    onsubmit={create}
    aria-label="New user"
  >
    <Field label="New username">
      <Input
        bind:value={username}
        minlength={3}
        maxlength={32}
        pattern="[A-Za-z0-9._\-]+"
        autocomplete="off"
        required
      />
    </Field>
    <div class="flex flex-col gap-1">
      <span class="text-sm font-medium">Role</span>
      <Select label="Role" items={roleItems} bind:value={role} />
    </div>
    <PasswordInput
      label="Temporary password"
      bind:value={password}
      minlength={16}
      maxlength={128}
      autocomplete="new-password"
      required
    />
    <Button type="submit" class="self-end">Create user</Button>
  </form>
  <p class="m-0 text-sm text-muted">
    Usernames use a-z, 0-9, '.', '_' and '-' (3 to 32 characters) and are never reused. The password
    is 16 to 128 characters and must be changed at first login.
  </p>

  <Table label="Users" tableLabel="Users table">
    <thead>
      <tr>
        <th>Username</th><th>Role</th><th>WAID</th><th>Status</th><th>Backup</th><th>Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each users as u (u.id)}
        {@const status = statusOf(u)}
        <tr>
          <td class="font-medium">{u.username}</td>
          <td>{u.role === "admin" ? "Admin" : "User"}</td>
          <td>{u.waid ?? "-"}</td>
          <td><Badge tone={status.tone}>{status.label}</Badge></td>
          <td>{u.role !== "admin" ? "-" : u.can_download_backup ? "Allowed" : "Not allowed"}</td>
          <td>
            {#if waidFor === u.id}
              <form class="flex flex-wrap items-end gap-2" onsubmit={(e) => saveWaid(e, u)}>
                <Field label="WAID for {u.username}">
                  <Input bind:value={waidValue} maxlength={32} inputmode="numeric" />
                </Field>
                <Button type="submit">Save WAID</Button>
                <Button variant="secondary" onclick={() => (waidFor = null)}>Cancel</Button>
              </form>
            {:else if resetFor === u.id}
              <form class="flex flex-wrap items-end gap-2" onsubmit={(e) => reset(e, u)}>
                <PasswordInput
                  label="Temporary password for {u.username}"
                  bind:value={resetPassword}
                  minlength={16}
                  maxlength={128}
                  autocomplete="new-password"
                  required
                />
                <Button type="submit">Confirm reset</Button>
                <Button variant="secondary" onclick={() => (resetFor = null)}>Cancel</Button>
              </form>
            {:else}
              <div class="flex flex-wrap gap-2">
                <Button variant="secondary" size="sm" onclick={() => setActive(u, !u.active)}>
                  {u.active ? "Deactivate" : "Reactivate"}
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => setRole(u, u.role === "admin" ? "user" : "admin")}
                >
                  {u.role === "admin" ? "Make user" : "Make admin"}
                </Button>
                {#if u.role === "admin"}
                  <Button
                    variant="secondary"
                    size="sm"
                    onclick={() => setBackup(u, !u.can_download_backup)}
                  >
                    {u.can_download_backup ? "Forbid backup" : "Allow backup"}
                  </Button>
                {/if}
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => {
                    waidFor = u.id;
                    waidValue = u.waid ?? "";
                  }}>Set WAID</Button
                >
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => {
                    resetFor = u.id;
                    resetPassword = "";
                  }}>Reset password</Button
                >
              </div>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </Table>
</div>

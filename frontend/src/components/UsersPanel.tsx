import { FormEvent, useMemo, useState } from "react";

import type { AuthUser } from "../types";

interface UsersPanelProps {
  users: AuthUser[];
  currentUsername: string | null;
  onCreate: (input: { username: string; password: string; role: "admin" | "user" }) => Promise<void>;
  onResetPassword: (username: string, password: string) => Promise<void>;
  onChangeRole: (username: string, role: "admin" | "user") => Promise<void>;
  onDelete: (username: string) => Promise<void>;
}

export default function UsersPanel({
  users,
  currentUsername,
  onCreate,
  onResetPassword,
  onChangeRole,
  onDelete
}: UsersPanelProps) {
  const [createForm, setCreateForm] = useState({ username: "", password: "", role: "user" as "admin" | "user" });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const sortedUsers = useMemo(
    () => [...users].sort((a, b) => a.username.localeCompare(b.username)),
    [users]
  );

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!createForm.username.trim() || !createForm.password.trim()) {
      setError("Username and password are required");
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onCreate({
        username: createForm.username.trim(),
        password: createForm.password.trim(),
        role: createForm.role
      });
      setMessage(`User ${createForm.username.trim()} created`);
      setCreateForm({ username: "", password: "", role: "user" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setPending(false);
    }
  };

  const handleResetPassword = async (username: string) => {
    const password = window.prompt(`Enter a new password for ${username}`);
    if (!password) {
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onResetPassword(username, password);
      setMessage(`Password updated for ${username}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update password");
    } finally {
      setPending(false);
    }
  };

  const handleChangeRole = async (username: string, role: "admin" | "user") => {
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onChangeRole(username, role);
      setMessage(`Role updated for ${username}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update role");
    } finally {
      setPending(false);
    }
  };

  const handleDelete = async (username: string) => {
    if (!window.confirm(`Delete user ${username}?`)) {
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onDelete(username);
      setMessage(`User ${username} deleted`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete user");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="panel" aria-label="Users">
      <div>
        <h2>User Accounts</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>
          Manage administrator and standard user access.
        </p>
      </div>

      {message ? (
        <div className="notification success" role="status">
          {message}
        </div>
      ) : null}
      {error ? (
        <div className="notification error" role="alert">
          {error}
        </div>
      ) : null}

      <section>
        <strong>Existing users</strong>
        <div className="list">
          {sortedUsers.length === 0 ? (
            <span style={{ color: "#94a3b8", fontSize: "0.9rem" }}>No users found.</span>
          ) : (
            sortedUsers.map((user) => (
              <div
                key={user.username}
                style={{
                  border: "1px solid #e2e8f0",
                  borderRadius: "0.5rem",
                  padding: "0.6rem 0.75rem",
                  display: "flex",
                  flexDirection: "column",
                  gap: "0.5rem",
                  background: user.username === currentUsername ? "#eff6ff" : "#fff"
                }}
              >
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div>
                    <div style={{ fontWeight: 600 }}>{user.username}</div>
                    <div style={{ fontSize: "0.8rem", color: "#64748b" }}>Role: {user.role}</div>
                  </div>
                  <div className="flag-actions">
                    <button
                      type="button"
                      className="primary"
                      onClick={() => handleChangeRole(user.username, user.role === "admin" ? "user" : "admin")}
                      disabled={pending || user.username === currentUsername}
                    >
                      {user.role === "admin" ? "Make user" : "Make admin"}
                    </button>
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => handleResetPassword(user.username)}
                      disabled={pending}
                    >
                      Reset password
                    </button>
                    <button
                      type="button"
                      className="danger"
                      onClick={() => handleDelete(user.username)}
                      disabled={pending || user.username === currentUsername}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </section>

      <section>
        <strong>Create user</strong>
        <form onSubmit={handleCreate} className="form">
          <div className="form-field">
            <label htmlFor="new-user-username">Username</label>
            <input
              id="new-user-username"
              value={createForm.username}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, username: event.target.value }))}
              placeholder="jane.doe"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-user-password">Password</label>
            <input
              id="new-user-password"
              type="password"
              value={createForm.password}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, password: event.target.value }))}
              placeholder="••••••••"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-user-role">Role</label>
            <select
              id="new-user-role"
              value={createForm.role}
              onChange={(event) =>
                setCreateForm((prev) => ({ ...prev, role: event.target.value as "admin" | "user" }))
              }
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <button type="submit" className="primary-button" disabled={pending}>
            Create user
          </button>
        </form>
      </section>
    </div>
  );
}

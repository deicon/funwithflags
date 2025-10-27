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

type ModalState =
  | { type: "create" }
  | { type: "edit"; user: AuthUser }
  | { type: "delete"; user: AuthUser }
  | null;

const EMPTY_CREATE_FORM = { username: "", password: "", role: "user" as const };

export default function UsersPanel({
  users,
  currentUsername,
  onCreate,
  onResetPassword,
  onChangeRole,
  onDelete
}: UsersPanelProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState<"username" | "role">("username");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [modalState, setModalState] = useState<ModalState>(null);
  const [createForm, setCreateForm] = useState(EMPTY_CREATE_FORM);
  const [editForm, setEditForm] = useState<{ username: string; role: "admin" | "user"; password: string }>({
    username: "",
    role: "user",
    password: ""
  });

  const visibleUsers = useMemo(() => {
    const normalized = filter.trim().toLowerCase();
    const filtered = normalized
      ? users.filter((user) => user.username.toLowerCase().includes(normalized) || user.role.toLowerCase().includes(normalized))
      : users;

    const sorted = [...filtered].sort((a, b) => {
      let value = 0;
      if (sortBy === "username") {
        value = a.username.localeCompare(b.username);
      } else {
        value = a.role.localeCompare(b.role);
      }
      return sortDirection === "asc" ? value : -value;
    });

    return sorted;
  }, [filter, sortBy, sortDirection, users]);

  const toggleSort = (column: "username" | "role") => {
    if (sortBy === column) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(column);
    setSortDirection("asc");
  };

  const openCreateModal = () => {
    setMessage(null);
    setError(null);
    setCreateForm(EMPTY_CREATE_FORM);
    setModalState({ type: "create" });
  };

  const openEditModal = (user: AuthUser) => {
    setMessage(null);
    setError(null);
    setEditForm({ username: user.username, role: user.role, password: "" });
    setModalState({ type: "edit", user });
  };

  const openDeleteModal = (user: AuthUser) => {
    setMessage(null);
    setError(null);
    setModalState({ type: "delete", user });
  };

  const closeModal = () => {
    if (pending) {
      return;
    }
    setModalState(null);
  };

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
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setPending(false);
    }
  };

  const handleEdit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!modalState || modalState.type !== "edit") {
      return;
    }
    const { user } = modalState;
    const trimmedPassword = editForm.password.trim();
    const roleChanged = editForm.role !== user.role;
    const passwordChanged = !!trimmedPassword;

    if (!roleChanged && !passwordChanged) {
      setMessage("No changes to apply");
      setModalState(null);
      return;
    }

    setError(null);
    setMessage(null);
    setPending(true);
    try {
      if (roleChanged) {
        await onChangeRole(user.username, editForm.role);
      }
      if (passwordChanged) {
        await onResetPassword(user.username, trimmedPassword);
      }
      const updates: string[] = [];
      if (roleChanged) {
        updates.push("role");
      }
      if (passwordChanged) {
        updates.push("password");
      }
      setMessage(`Updated ${user.username}'s ${updates.join(" and ")}`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update user");
    } finally {
      setPending(false);
    }
  };

  const handleDelete = async () => {
    if (!modalState || modalState.type !== "delete") {
      return;
    }
    const { user } = modalState;
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onDelete(user.username);
      setMessage(`User ${user.username} deleted`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete user");
    } finally {
      setPending(false);
    }
  };

  const currentSortIndicator = (column: "username" | "role") => {
    if (sortBy !== column) {
      return "↕";
    }
    return sortDirection === "asc" ? "↑" : "↓";
  };

  return (
    <div className="panel" aria-label="Users">
      <div>
        <h2>User Accounts</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>Manage administrator and standard user access.</p>
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

      <div className="table-toolbar">
        <input
          className="toolbar-input"
          type="search"
          placeholder="Filter users…"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        <div className="toolbar-actions">
          <button type="button" className="button secondary" onClick={() => setFilter("")} disabled={!filter}>
            Clear
          </button>
          <button type="button" className="button primary" onClick={openCreateModal} disabled={pending}>
            Add User
          </button>
        </div>
      </div>

      <div className="table-wrapper">
        <table className="table" aria-label="Users">
          <thead>
            <tr>
              <th>
                <button
                  type="button"
                  className="table-sort"
                  onClick={() => toggleSort("username")}
                  aria-label={`Sort by username (${sortBy === "username" ? sortDirection : "unsorted"})`}
                >
                  Username <span aria-hidden="true">{currentSortIndicator("username")}</span>
                </button>
              </th>
              <th>
                <button
                  type="button"
                  className="table-sort"
                  onClick={() => toggleSort("role")}
                  aria-label={`Sort by role (${sortBy === "role" ? sortDirection : "unsorted"})`}
                >
                  Role <span aria-hidden="true">{currentSortIndicator("role")}</span>
                </button>
              </th>
              <th style={{ width: "160px" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {visibleUsers.length === 0 ? (
              <tr>
                <td colSpan={3} style={{ padding: "1.25rem", textAlign: "center", color: "#94a3b8", fontSize: "0.9rem" }}>
                  No users match the current filter.
                </td>
              </tr>
            ) : (
              visibleUsers.map((user) => (
                <tr key={user.username} style={{ background: user.username === currentUsername ? "#f1f5ff" : undefined }}>
                  <td>{user.username}</td>
                  <td style={{ textTransform: "capitalize" }}>{user.role}</td>
                  <td>
                    <div className="table-actions">
                      <button
                        type="button"
                        className="button secondary"
                        onClick={() => openEditModal(user)}
                        disabled={pending}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="button danger"
                        onClick={() => openDeleteModal(user)}
                        disabled={pending || user.username === currentUsername}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {modalState ? (
        <div className="modal-backdrop" role="presentation">
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="user-modal-title">
            {modalState.type === "create" ? (
              <>
                <div className="modal-header">
                  <h3 id="user-modal-title">Add User</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close add user dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <form onSubmit={handleCreate} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-user-username">Username</label>
                    <input
                      id="modal-user-username"
                      value={createForm.username}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, username: event.target.value }))}
                      placeholder="jane.doe"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-user-password">Password</label>
                    <input
                      id="modal-user-password"
                      type="password"
                      value={createForm.password}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, password: event.target.value }))}
                      placeholder="••••••••"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-user-role">Role</label>
                    <select
                      id="modal-user-role"
                      value={createForm.role}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, role: event.target.value as "admin" | "user" }))}
                    >
                      <option value="user">User</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button type="submit" className="button primary" disabled={pending}>
                      Create
                    </button>
                  </div>
                </form>
              </>
            ) : null}

            {modalState.type === "edit" ? (
              <>
                <div className="modal-header">
                  <h3 id="user-modal-title">Edit {modalState.user.username}</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close edit user dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <form onSubmit={handleEdit} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-edit-username">Username</label>
                    <input id="modal-edit-username" value={modalState.user.username} disabled />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-role">Role</label>
                    <select
                      id="modal-edit-role"
                      value={editForm.role}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, role: event.target.value as "admin" | "user" }))}
                    >
                      <option value="user">User</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-password">New password</label>
                    <input
                      id="modal-edit-password"
                      type="password"
                      value={editForm.password}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, password: event.target.value }))}
                      placeholder="Leave blank to keep current password"
                    />
                  </div>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button type="submit" className="button primary" disabled={pending}>
                      Save changes
                    </button>
                  </div>
                </form>
              </>
            ) : null}

            {modalState.type === "delete" ? (
              <>
                <div className="modal-header">
                  <h3 id="user-modal-title">Delete {modalState.user.username}</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close delete user dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <div className="modal-content">
                  <p style={{ marginTop: 0 }}>
                    Are you sure you want to delete <strong>{modalState.user.username}</strong>? This action cannot be undone.
                  </p>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button
                      type="button"
                      className="button danger"
                      onClick={handleDelete}
                      disabled={pending || modalState.user.username === currentUsername}
                    >
                      {pending ? "Deleting…" : "Delete"}
                    </button>
                  </div>
                </div>
              </>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

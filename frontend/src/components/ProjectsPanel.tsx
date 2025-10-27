import { FormEvent, useMemo, useState } from "react";

import type { Project } from "../types";

interface ProjectsPanelProps {
  projects: Project[];
  selectedProjectKey: string | null;
  onSelect: (project: Project) => void;
  onCreate: (input: { key: string; name: string; description?: string }) => Promise<void>;
  onUpdate: (key: string, input: { name: string; description?: string }) => Promise<void>;
  onDelete: (project: Project) => Promise<void>;
}

type ModalState =
  | { type: "create" }
  | { type: "edit"; project: Project }
  | { type: "delete"; project: Project }
  | null;

const EMPTY_CREATE_FORM = { key: "", name: "", description: "" };

export default function ProjectsPanel({
  projects,
  selectedProjectKey,
  onSelect,
  onCreate,
  onUpdate,
  onDelete
}: ProjectsPanelProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "key">("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [modalState, setModalState] = useState<ModalState>(null);
  const [createForm, setCreateForm] = useState(EMPTY_CREATE_FORM);
  const [editForm, setEditForm] = useState({ name: "", description: "" });

  const visibleProjects = useMemo(() => {
    const normalizedFilter = filter.trim().toLowerCase();
    const filtered = normalizedFilter
      ? projects.filter((project) => {
          const nameMatch = project.name.toLowerCase().includes(normalizedFilter);
          const keyMatch = project.key.toLowerCase().includes(normalizedFilter);
          const descriptionMatch = (project.description ?? "").toLowerCase().includes(normalizedFilter);
          return nameMatch || keyMatch || descriptionMatch;
        })
      : projects;

    const sorted = [...filtered].sort((a, b) => {
      const aValue = sortBy === "name" ? a.name : a.key;
      const bValue = sortBy === "name" ? b.name : b.key;
      const comparison = aValue.localeCompare(bValue);
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [filter, projects, sortBy, sortDirection]);

  const toggleSort = (column: "name" | "key") => {
    if (sortBy === column) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(column);
    setSortDirection("asc");
  };

  const currentSortIndicator = (column: "name" | "key") => {
    if (sortBy !== column) {
      return "↕";
    }
    return sortDirection === "asc" ? "↑" : "↓";
  };

  const openCreateModal = () => {
    setMessage(null);
    setError(null);
    setCreateForm({ ...EMPTY_CREATE_FORM });
    setModalState({ type: "create" });
  };

  const openEditModal = (project: Project) => {
    setMessage(null);
    setError(null);
    setEditForm({ name: project.name, description: project.description ?? "" });
    setModalState({ type: "edit", project });
  };

  const openDeleteModal = (project: Project) => {
    setMessage(null);
    setError(null);
    setModalState({ type: "delete", project });
  };

  const closeModal = () => {
    if (pending) {
      return;
    }
    setModalState(null);
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!createForm.key.trim() || !createForm.name.trim()) {
      setError("Project key and name are required");
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onCreate({
        key: createForm.key.trim(),
        name: createForm.name.trim(),
        description: createForm.description.trim() || undefined
      });
      setMessage(`Project ${createForm.key.trim()} created`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    } finally {
      setPending(false);
    }
  };

  const handleEdit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!modalState || modalState.type !== "edit") {
      return;
    }
    if (!editForm.name.trim()) {
      setError("Project name is required");
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onUpdate(modalState.project.key, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined
      });
      setMessage("Project updated");
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update project");
    } finally {
      setPending(false);
    }
  };

  const handleDelete = async () => {
    if (!modalState || modalState.type !== "delete") {
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onDelete(modalState.project);
      setMessage(`Project ${modalState.project.key} deleted`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete project");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="panel" aria-label="Projects">
      <div>
        <h2>Projects</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>
          Select a project to manage its stages and feature flags.
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

      <div className="table-toolbar">
        <input
          className="toolbar-input"
          type="search"
          placeholder="Filter projects…"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        <div className="toolbar-actions">
          <button type="button" className="button secondary" onClick={() => setFilter("")} disabled={!filter}>
            Clear
          </button>
          <button type="button" className="button primary" onClick={openCreateModal} disabled={pending}>
            Add Project
          </button>
        </div>
      </div>

      <div className="table-wrapper">
        <table className="table" aria-label="Projects">
          <thead>
            <tr>
              <th>
                <button
                  type="button"
                  className="table-sort"
                  onClick={() => toggleSort("name")}
                  aria-label={`Sort by name (${sortBy === "name" ? sortDirection : "unsorted"})`}
                >
                  Name <span aria-hidden="true">{currentSortIndicator("name")}</span>
                </button>
              </th>
              <th>
                <button
                  type="button"
                  className="table-sort"
                  onClick={() => toggleSort("key")}
                  aria-label={`Sort by key (${sortBy === "key" ? sortDirection : "unsorted"})`}
                >
                  Key <span aria-hidden="true">{currentSortIndicator("key")}</span>
                </button>
              </th>
              <th>Description</th>
              <th style={{ width: "160px" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {visibleProjects.length === 0 ? (
              <tr>
                <td colSpan={4} style={{ padding: "1.25rem", textAlign: "center", color: "#94a3b8", fontSize: "0.9rem" }}>
                  No projects match the current filter.
                </td>
              </tr>
            ) : (
              visibleProjects.map((project) => (
                <tr
                  key={project.key}
                  onClick={() => onSelect(project)}
                  aria-selected={project.key === selectedProjectKey}
                  style={{
                    background: project.key === selectedProjectKey ? "#f1f5ff" : undefined,
                    cursor: "pointer"
                  }}
                >
                  <td>{project.name}</td>
                  <td>{project.key}</td>
                  <td style={{ color: "#475569" }}>{project.description?.trim() || "—"}</td>
                  <td>
                    <div className="table-actions">
                      <button
                        type="button"
                        className="button secondary"
                        onClick={(event) => {
                          event.stopPropagation();
                          openEditModal(project);
                        }}
                        disabled={pending}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="button danger"
                        onClick={(event) => {
                          event.stopPropagation();
                          openDeleteModal(project);
                        }}
                        disabled={pending}
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
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="projects-modal-title">
            {modalState.type === "create" ? (
              <>
                <div className="modal-header">
                  <h3 id="projects-modal-title">Add Project</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close add project dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <form onSubmit={handleCreate} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-project-key">Key</label>
                    <input
                      id="modal-project-key"
                      value={createForm.key}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
                      placeholder="checkout"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-project-name">Name</label>
                    <input
                      id="modal-project-name"
                      value={createForm.name}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
                      placeholder="Checkout"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-project-description">Description</label>
                    <textarea
                      id="modal-project-description"
                      rows={2}
                      value={createForm.description}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
                      placeholder="Describe the project"
                    />
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
                  <h3 id="projects-modal-title">Edit {modalState.project.key}</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close edit project dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <form onSubmit={handleEdit} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-edit-project-key">Key</label>
                    <input id="modal-edit-project-key" value={modalState.project.key} disabled />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-project-name">Name</label>
                    <input
                      id="modal-edit-project-name"
                      value={editForm.name}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-project-description">Description</label>
                    <textarea
                      id="modal-edit-project-description"
                      rows={2}
                      value={editForm.description}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, description: event.target.value }))}
                      placeholder="Describe the project"
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
                  <h3 id="projects-modal-title">Delete {modalState.project.key}</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close delete project dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <div className="modal-content">
                  <p style={{ marginTop: 0 }}>
                    Are you sure you want to delete project <strong>{modalState.project.name}</strong>? This action cannot be undone.
                  </p>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button type="button" className="button danger" onClick={handleDelete} disabled={pending}>
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

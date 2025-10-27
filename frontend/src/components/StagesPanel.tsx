import { FormEvent, useMemo, useState } from "react";

import type { Project, Stage } from "../types";

interface StagesPanelProps {
  project: Project | null;
  stages: Stage[];
  selectedStageKey: string | null;
  onSelect: (stage: Stage) => void;
  onCreate: (input: { key: string; name: string; description?: string }) => Promise<void>;
  onUpdate: (key: string, input: { name: string; description?: string }) => Promise<void>;
  onDelete: (stage: Stage) => Promise<void>;
}

type ModalState =
  | { type: "create" }
  | { type: "edit"; stage: Stage }
  | { type: "delete"; stage: Stage }
  | null;

const EMPTY_CREATE_FORM = { key: "", name: "", description: "" };

export default function StagesPanel({
  project,
  stages,
  selectedStageKey,
  onSelect,
  onCreate,
  onUpdate,
  onDelete
}: StagesPanelProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "key">("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [modalState, setModalState] = useState<ModalState>(null);
  const [createForm, setCreateForm] = useState(EMPTY_CREATE_FORM);
  const [editForm, setEditForm] = useState({ name: "", description: "" });

  const visibleStages = useMemo(() => {
    const normalizedFilter = filter.trim().toLowerCase();
    const filtered = normalizedFilter
      ? stages.filter((stage) => {
          const nameMatch = stage.name.toLowerCase().includes(normalizedFilter);
          const keyMatch = stage.key.toLowerCase().includes(normalizedFilter);
          const descriptionMatch = (stage.description ?? "").toLowerCase().includes(normalizedFilter);
          return nameMatch || keyMatch || descriptionMatch;
        })
      : stages;

    const sorted = [...filtered].sort((a, b) => {
      const aValue = sortBy === "name" ? a.name : a.key;
      const bValue = sortBy === "name" ? b.name : b.key;
      const comparison = aValue.localeCompare(bValue);
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [filter, sortBy, sortDirection, stages]);

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
    if (!project) {
      setError("Select a project first");
      return;
    }
    setMessage(null);
    setError(null);
    setCreateForm({ ...EMPTY_CREATE_FORM });
    setModalState({ type: "create" });
  };

  const openEditModal = (stage: Stage) => {
    setMessage(null);
    setError(null);
    setEditForm({ name: stage.name, description: stage.description ?? "" });
    setModalState({ type: "edit", stage });
  };

  const openDeleteModal = (stage: Stage) => {
    setMessage(null);
    setError(null);
    setModalState({ type: "delete", stage });
  };

  const closeModal = () => {
    if (pending) {
      return;
    }
    setModalState(null);
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!project) {
      setError("Select a project first");
      return;
    }
    if (!createForm.key.trim() || !createForm.name.trim()) {
      setError("Stage key and name are required");
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
      setMessage(`Stage ${createForm.key.trim()} created`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create stage");
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
      setError("Stage name is required");
      return;
    }
    setError(null);
    setMessage(null);
    setPending(true);
    try {
      await onUpdate(modalState.stage.key, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined
      });
      setMessage("Stage updated");
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update stage");
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
      await onDelete(modalState.stage);
      setMessage(`Stage ${modalState.stage.key} deleted`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete stage");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="panel" aria-label="Stages">
      <div>
        <h2>Stages</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>
          Environments for <strong>{project?.name ?? "select a project"}</strong>.
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
          placeholder="Filter stages…"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        <div className="toolbar-actions">
          <button type="button" className="button secondary" onClick={() => setFilter("")} disabled={!filter}>
            Clear
          </button>
          <button type="button" className="button primary" onClick={openCreateModal} disabled={pending || !project}>
            Add Stage
          </button>
        </div>
      </div>

      <div className="table-wrapper">
        <table className="table" aria-label="Stages">
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
            {visibleStages.length === 0 ? (
              <tr>
                <td colSpan={4} style={{ padding: "1.25rem", textAlign: "center", color: "#94a3b8", fontSize: "0.9rem" }}>
                  {stages.length === 0 ? "No stages for this project." : "No stages match the current filter."}
                </td>
              </tr>
            ) : (
              visibleStages.map((stage) => (
                <tr
                  key={`${stage.projectKey}-${stage.key}`}
                  onClick={() => onSelect(stage)}
                  aria-selected={stage.key === selectedStageKey}
                  style={{
                    background: stage.key === selectedStageKey ? "#f1f5ff" : undefined,
                    cursor: "pointer"
                  }}
                >
                  <td>{stage.name}</td>
                  <td>{stage.key}</td>
                  <td style={{ color: "#475569" }}>{stage.description?.trim() || "—"}</td>
                  <td>
                    <div className="table-actions">
                      <button
                        type="button"
                        className="button secondary"
                        onClick={(event) => {
                          event.stopPropagation();
                          openEditModal(stage);
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
                          openDeleteModal(stage);
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
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="stages-modal-title">
            {modalState.type === "create" ? (
              <>
                <div className="modal-header">
                  <h3 id="stages-modal-title">Add Stage</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close add stage dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <form onSubmit={handleCreate} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-stage-key">Key</label>
                    <input
                      id="modal-stage-key"
                      value={createForm.key}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
                      placeholder="production"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-stage-name">Name</label>
                    <input
                      id="modal-stage-name"
                      value={createForm.name}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
                      placeholder="Production"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-stage-description">Description</label>
                    <textarea
                      id="modal-stage-description"
                      rows={2}
                      value={createForm.description}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
                      placeholder="Optional description"
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
                  <h3 id="stages-modal-title">Edit {modalState.stage.key}</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close edit stage dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <form onSubmit={handleEdit} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="modal-edit-stage-key">Key</label>
                    <input id="modal-edit-stage-key" value={modalState.stage.key} disabled />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-stage-name">Name</label>
                    <input
                      id="modal-edit-stage-name"
                      value={editForm.name}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="modal-edit-stage-description">Description</label>
                    <textarea
                      id="modal-edit-stage-description"
                      rows={2}
                      value={editForm.description}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, description: event.target.value }))}
                      placeholder="Optional description"
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
                  <h3 id="stages-modal-title">Delete {modalState.stage.key}</h3>
                  <button
                    type="button"
                    className="modal-close"
                    onClick={closeModal}
                    aria-label="Close delete stage dialog"
                    disabled={pending}
                  >
                    ✕
                  </button>
                </div>
                <div className="modal-content">
                  <p style={{ marginTop: 0 }}>
                    Are you sure you want to delete stage <strong>{modalState.stage.name}</strong>? This action cannot be undone.
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

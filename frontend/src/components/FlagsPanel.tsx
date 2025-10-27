import { FormEvent, useEffect, useMemo, useState } from "react";

import type { FeatureFlag, Project, Stage, Variation } from "../types";

interface FlagsPanelProps {
  projects: Project[];
  stages: Stage[];
  project: Project | null;
  stage: Stage | null;
  selectedProjectKey: string | null;
  selectedStageKey: string | null;
  flags: FeatureFlag[];
  onSelectProject: (project: Project) => void;
  onSelectStage: (stage: Stage) => void;
  onCreate: (input: {
    key: string;
    name: string;
    description?: string;
    enabled: boolean;
    active: boolean;
    defaultKey: string;
    validFrom: string;
    validTo?: string;
  }) => Promise<void>;
  onUpdate: (flag: FeatureFlag, updates: Partial<FeatureFlag>) => Promise<void>;
  onDelete: (flag: FeatureFlag) => Promise<void>;
}

type ModalState =
  | { type: "create" }
  | { type: "edit"; flag: FeatureFlag }
  | { type: "delete"; flag: FeatureFlag }
  | null;

const BOOLEAN_VARIATIONS: Variation[] = [
  { key: "control", type: "boolean", value: false, description: "Feature off" },
  { key: "enabled", type: "boolean", value: true, description: "Feature on" }
];

const nowIsoString = () => new Date().toISOString();

const toDateTimeLocalValue = (value: string) => {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const tzOffset = date.getTimezoneOffset();
  const localDate = new Date(date.getTime() - tzOffset * 60000);
  return localDate.toISOString().slice(0, 16);
};

const fromDateTimeLocalValue = (value: string) => {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString();
};

const formatDateTime = (value?: string) => {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const createDefaultFlagForm = () => ({
  key: "",
  name: "",
  description: "",
  enabled: true,
  active: true,
  defaultKey: "control",
  validFrom: nowIsoString(),
  validTo: ""
});

export default function FlagsPanel({
  projects,
  stages,
  project,
  stage,
  selectedProjectKey,
  selectedStageKey,
  flags,
  onSelectProject,
  onSelectStage,
  onCreate,
  onUpdate,
  onDelete
}: FlagsPanelProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState<"key" | "name">("key");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [modalState, setModalState] = useState<ModalState>(null);
  const [createForm, setCreateForm] = useState(() => createDefaultFlagForm());
  const [editForm, setEditForm] = useState({
    name: "",
    description: "",
    enabled: true,
    active: true,
    defaultKey: "",
    validFrom: "",
    validTo: ""
  });

  const visibleFlags = useMemo(() => {
    const normalizedFilter = filter.trim().toLowerCase();
    const filtered = normalizedFilter
      ? flags.filter((flag) => {
          const haystacks = [
            flag.key,
            flag.name,
            flag.description ?? "",
            flag.defaultKey,
            flag.enabled ? "enabled" : "disabled",
            flag.active ? "active" : "inactive"
          ];
          return haystacks.some((value) => value.toLowerCase().includes(normalizedFilter));
        })
      : flags;

    const sorted = [...filtered].sort((a, b) => {
      const aValue = sortBy === "key" ? a.key : a.name;
      const bValue = sortBy === "key" ? b.key : b.name;
      const comparison = aValue.localeCompare(bValue);
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [filter, flags, sortBy, sortDirection]);

  useEffect(() => {
    setMessage(null);
    setError(null);
    setFilter("");
    setModalState(null);
  }, [selectedProjectKey, selectedStageKey]);

  const toggleSort = (column: "key" | "name") => {
    if (sortBy === column) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(column);
    setSortDirection("asc");
  };

  const currentSortIndicator = (column: "key" | "name") => {
    if (sortBy !== column) {
      return "↕";
    }
    return sortDirection === "asc" ? "↑" : "↓";
  };

  const openCreateModal = () => {
    setMessage(null);
    setError(null);
    setCreateForm(createDefaultFlagForm());
    setModalState({ type: "create" });
  };

  const openEditModal = (flag: FeatureFlag) => {
    setMessage(null);
    setError(null);
    setEditForm({
      name: flag.name,
      description: flag.description ?? "",
      enabled: flag.enabled,
      active: flag.active,
      defaultKey: flag.defaultKey,
      validFrom: flag.validFrom,
      validTo: flag.validTo ?? ""
    });
    setModalState({ type: "edit", flag });
  };

  const openDeleteModal = (flag: FeatureFlag) => {
    setMessage(null);
    setError(null);
    setModalState({ type: "delete", flag });
  };

  const closeModal = () => {
    if (pending) {
      return;
    }
    setModalState(null);
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!project || !stage) {
      setError("Select a project and stage first");
      return;
    }
    if (!createForm.key.trim() || !createForm.name.trim()) {
      setError("Flag key and name are required");
      return;
    }
    setPending(true);
    setError(null);
    setMessage(null);
    try {
      await onCreate({
        key: createForm.key.trim(),
        name: createForm.name.trim(),
        description: createForm.description.trim() || undefined,
        enabled: createForm.enabled,
        active: createForm.active,
        defaultKey: createForm.defaultKey,
        validFrom: createForm.validFrom,
        validTo: createForm.validTo || undefined
      });
      setMessage(`Flag ${createForm.key.trim()} created`);
      setCreateForm(createDefaultFlagForm());
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create flag");
    } finally {
      setPending(false);
    }
  };

  const handleEdit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!modalState || modalState.type !== "edit") {
      return;
    }
    const { flag } = modalState;

    if (!editForm.name.trim()) {
      setError("Flag name is required");
      return;
    }

    setPending(true);
    setError(null);
    setMessage(null);
    try {
      await onUpdate(flag, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined,
        enabled: editForm.enabled,
        active: editForm.active,
        defaultKey: editForm.defaultKey,
        validFrom: editForm.validFrom,
        validTo: editForm.validTo || undefined
      });
      setMessage(`Flag ${flag.key} updated`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update flag");
    } finally {
      setPending(false);
    }
  };

  const handleDelete = async () => {
    if (!modalState || modalState.type !== "delete") {
      return;
    }
    setPending(true);
    setError(null);
    setMessage(null);
    try {
      await onDelete(modalState.flag);
      setMessage(`Flag ${modalState.flag.key} deleted`);
      setModalState(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete flag");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="panel" aria-label="Flags">
      <div>
        <h2>Flags</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>
          Manage feature flags for <strong>{project?.name ?? "select a project"}</strong> /{" "}
          <strong>{stage?.name ?? "stage"}</strong>.
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

      <div className="selection-grid">
        <div className="form-field">
          <label htmlFor="flag-project-select">Project</label>
          <select
            id="flag-project-select"
            value={selectedProjectKey ?? ""}
            onChange={(event) => {
              const selected = projects.find((item) => item.key === event.target.value);
              if (selected) {
                onSelectProject(selected);
              }
            }}
            disabled={projects.length === 0}
          >
            <option value="" disabled>
              Select a project
            </option>
            {projects.map((item) => (
              <option key={item.key} value={item.key}>
                {item.name}
              </option>
            ))}
          </select>
        </div>
        <div className="form-field">
          <label htmlFor="flag-stage-select">Stage</label>
          <select
            id="flag-stage-select"
            value={selectedStageKey ?? ""}
            onChange={(event) => {
              const selected = stages.find((item) => item.key === event.target.value);
              if (selected) {
                onSelectStage(selected);
              }
            }}
            disabled={!selectedProjectKey || stages.length === 0}
          >
            <option value="" disabled>
              {selectedProjectKey ? "Select a stage" : "Choose a project first"}
            </option>
            {stages.map((item) => (
              <option key={item.key} value={item.key}>
                {item.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="table-toolbar">
        <input
          className="toolbar-input"
          type="search"
          placeholder="Filter flags…"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        <div className="toolbar-actions">
          <button type="button" className="button secondary" onClick={() => setFilter("")} disabled={!filter}>
            Clear
          </button>
          <button
            type="button"
            className="button primary"
            onClick={openCreateModal}
            disabled={!project || !stage || pending}
          >
            Add Flag
          </button>
        </div>
      </div>

      <div className="table-wrapper">
        <table className="table" aria-label="Flags list">
          <thead>
            <tr>
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
              <th>Enabled</th>
              <th>Active</th>
              <th>Valid from</th>
              <th>Valid to</th>
              <th>Default variation</th>
              <th style={{ width: "180px" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {visibleFlags.length === 0 ? (
              <tr>
                <td colSpan={8} style={{ padding: "1.25rem", textAlign: "center", color: "#94a3b8", fontSize: "0.9rem" }}>
                  {filter ? "No flags match the current filter." : "No flags defined for this stage yet."}
                </td>
              </tr>
            ) : (
              visibleFlags.map((flag) => (
                <tr key={flag.id}>
                  <td>{flag.key}</td>
                  <td>{flag.name}</td>
                  <td>{flag.enabled ? "Yes" : "No"}</td>
                  <td>{flag.active ? "Yes" : "No"}</td>
                  <td>{formatDateTime(flag.validFrom)}</td>
                  <td>{formatDateTime(flag.validTo)}</td>
                  <td>{flag.defaultKey}</td>
                  <td>
                    <div className="table-actions">
                      <button type="button" className="button secondary" onClick={() => openEditModal(flag)} disabled={pending}>
                        Edit
                      </button>
                      <button type="button" className="button danger" onClick={() => openDeleteModal(flag)} disabled={pending}>
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
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="flag-modal-title">
            {modalState.type === "create" ? (
              <>
                <div className="modal-header">
                  <h3 id="flag-modal-title">Add Flag</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close add flag dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <form onSubmit={handleCreate} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="flag-create-key">Key</label>
                    <input
                      id="flag-create-key"
                      value={createForm.key}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
                      placeholder="checkout-flow"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-name">Name</label>
                    <input
                      id="flag-create-name"
                      value={createForm.name}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
                      placeholder="Checkout Flow"
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-description">Description</label>
                    <textarea
                      id="flag-create-description"
                      rows={2}
                      value={createForm.description}
                      onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-enabled">Enabled</label>
                    <select
                      id="flag-create-enabled"
                      value={createForm.enabled ? "true" : "false"}
                      onChange={(event) =>
                        setCreateForm((prev) => ({
                          ...prev,
                          enabled: event.target.value === "true"
                        }))
                      }
                    >
                      <option value="true">Yes</option>
                      <option value="false">No</option>
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-active">Active</label>
                    <select
                      id="flag-create-active"
                      value={createForm.active ? "true" : "false"}
                      onChange={(event) =>
                        setCreateForm((prev) => ({
                          ...prev,
                          active: event.target.value === "true"
                        }))
                      }
                    >
                      <option value="true">Yes</option>
                      <option value="false">No</option>
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-default">Default variation</label>
                    <select
                      id="flag-create-default"
                      value={createForm.defaultKey}
                      onChange={(event) =>
                        setCreateForm((prev) => ({
                          ...prev,
                          defaultKey: event.target.value
                        }))
                      }
                    >
                      {BOOLEAN_VARIATIONS.map((variation) => (
                        <option key={variation.key} value={variation.key}>
                          {variation.key}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-valid-from">Valid from</label>
                    <input
                      id="flag-create-valid-from"
                      type="datetime-local"
                      value={toDateTimeLocalValue(createForm.validFrom)}
                      onChange={(event) =>
                        setCreateForm((prev) => ({
                          ...prev,
                          validFrom: fromDateTimeLocalValue(event.target.value)
                        }))
                      }
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-create-valid-to">Valid to</label>
                    <input
                      id="flag-create-valid-to"
                      type="datetime-local"
                      value={toDateTimeLocalValue(createForm.validTo)}
                      onChange={(event) =>
                        setCreateForm((prev) => ({
                          ...prev,
                          validTo: fromDateTimeLocalValue(event.target.value)
                        }))
                      }
                    />
                  </div>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button type="submit" className="button primary" disabled={pending || !project || !stage}>
                      {pending ? "Creating…" : "Create"}
                    </button>
                  </div>
                </form>
              </>
            ) : null}

            {modalState?.type === "edit" ? (
              <>
                <div className="modal-header">
                  <h3 id="flag-modal-title">Edit {modalState.flag.key}</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close edit flag dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <form onSubmit={handleEdit} className="modal-content">
                  <div className="form-field">
                    <label htmlFor="flag-edit-key">Key</label>
                    <input id="flag-edit-key" value={modalState.flag.key} disabled />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-name">Name</label>
                    <input
                      id="flag-edit-name"
                      value={editForm.name}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-description">Description</label>
                    <textarea
                      id="flag-edit-description"
                      rows={2}
                      value={editForm.description}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, description: event.target.value }))}
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-enabled">Enabled</label>
                    <select
                      id="flag-edit-enabled"
                      value={editForm.enabled ? "true" : "false"}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, enabled: event.target.value === "true" }))}
                    >
                      <option value="true">Yes</option>
                      <option value="false">No</option>
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-active">Active</label>
                    <select
                      id="flag-edit-active"
                      value={editForm.active ? "true" : "false"}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, active: event.target.value === "true" }))}
                    >
                      <option value="true">Yes</option>
                      <option value="false">No</option>
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-default">Default variation</label>
                    <select
                      id="flag-edit-default"
                      value={editForm.defaultKey}
                      onChange={(event) => setEditForm((prev) => ({ ...prev, defaultKey: event.target.value }))}
                    >
                      {modalState.flag.variations.map((variation) => (
                        <option key={variation.key} value={variation.key}>
                          {variation.key}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-valid-from">Valid from</label>
                    <input
                      id="flag-edit-valid-from"
                      type="datetime-local"
                      value={toDateTimeLocalValue(editForm.validFrom)}
                      onChange={(event) =>
                        setEditForm((prev) => ({
                          ...prev,
                          validFrom: fromDateTimeLocalValue(event.target.value)
                        }))
                      }
                      required
                    />
                  </div>
                  <div className="form-field">
                    <label htmlFor="flag-edit-valid-to">Valid to</label>
                    <input
                      id="flag-edit-valid-to"
                      type="datetime-local"
                      value={toDateTimeLocalValue(editForm.validTo)}
                      onChange={(event) =>
                        setEditForm((prev) => ({
                          ...prev,
                          validTo: fromDateTimeLocalValue(event.target.value)
                        }))
                      }
                    />
                  </div>
                  <div className="modal-actions">
                    <button type="button" className="button secondary" onClick={closeModal} disabled={pending}>
                      Cancel
                    </button>
                    <button type="submit" className="button primary" disabled={pending}>
                      {pending ? "Saving…" : "Save changes"}
                    </button>
                  </div>
                </form>
              </>
            ) : null}

            {modalState?.type === "delete" ? (
              <>
                <div className="modal-header">
                  <h3 id="flag-modal-title">Delete {modalState.flag.key}</h3>
                  <button type="button" className="modal-close" onClick={closeModal} aria-label="Close delete flag dialog" disabled={pending}>
                    ✕
                  </button>
                </div>
                <div className="modal-content">
                  <p style={{ marginTop: 0 }}>
                    Are you sure you want to delete <strong>{modalState.flag.key}</strong>? This action cannot be undone.
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

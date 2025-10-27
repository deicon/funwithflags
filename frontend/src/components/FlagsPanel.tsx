import { FormEvent, useEffect, useMemo, useState } from "react";

import type { FeatureFlag, Project, Stage, Variation } from "../types";

interface FlagsPanelProps {
  project: Project | null;
  stage: Stage | null;
  flags: FeatureFlag[];
  onCreate: (input: {
    key: string;
    name: string;
    description?: string;
    enabled: boolean;
    defaultKey: string;
  }) => Promise<void>;
  onUpdate: (flag: FeatureFlag, updates: Partial<FeatureFlag>) => Promise<void>;
  onDelete: (flag: FeatureFlag) => Promise<void>;
}

const BOOLEAN_VARIATIONS: Variation[] = [
  { key: "control", type: "boolean", value: false, description: "Feature off" },
  { key: "enabled", type: "boolean", value: true, description: "Feature on" }
];

export default function FlagsPanel({ project, stage, flags, onCreate, onUpdate, onDelete }: FlagsPanelProps) {
  const [selectedFlagId, setSelectedFlagId] = useState<number | null>(null);
  const [createForm, setCreateForm] = useState({
    key: "",
    name: "",
    description: "",
    enabled: true,
    defaultKey: "control"
  });
  const [editForm, setEditForm] = useState({
    name: "",
    description: "",
    enabled: false,
    active: false,
    defaultKey: "",
    validFrom: "",
    validTo: ""
  });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const selectedFlag = useMemo(
    () => flags.find((flag) => flag.id === selectedFlagId) ?? null,
    [flags, selectedFlagId]
  );

  useEffect(() => {
    setSelectedFlagId(null);
    setCreateForm({
      key: "",
      name: "",
      description: "",
      enabled: true,
      defaultKey: "control"
    });
    setMessage(null);
    setError(null);
  }, [project?.key, stage?.key]);

  useEffect(() => {
    if (!selectedFlag) {
      setEditForm({ name: "", description: "", enabled: false, active: false, defaultKey: "", validFrom: "", validTo: "" });
      return;
    }
    setEditForm({
      name: selectedFlag.name,
      description: selectedFlag.description ?? "",
      enabled: selectedFlag.enabled,
      active: selectedFlag.active,
      defaultKey: selectedFlag.defaultKey,
      validFrom: selectedFlag.validFrom,
      validTo: selectedFlag.validTo ?? ""
    });
  }, [selectedFlag]);

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!project || !stage) {
      setError("Select a project and stage first");
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onCreate(createForm);
      setMessage(`Flag ${createForm.key.trim()} created`);
      setCreateForm({ key: "", name: "", description: "", enabled: true, defaultKey: "control" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create flag");
    }
  };

  const handleUpdate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedFlag) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onUpdate(selectedFlag, {
        name: editForm.name,
        description: editForm.description,
        enabled: editForm.enabled,
        active: editForm.active,
        defaultKey: editForm.defaultKey,
        validFrom: editForm.validFrom,
        validTo: editForm.validTo
      });
      setMessage("Flag updated");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update flag");
    }
  };

  const handleDelete = async () => {
    if (!selectedFlag) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onDelete(selectedFlag);
      setMessage("Flag deleted");
      setSelectedFlagId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete flag");
    }
  };

  return (
    <div className="panel" aria-label="Flags">
      <div>
        <h2>Flags</h2>
        <p style={{ margin: 0, color: "#64748b", fontSize: "0.85rem" }}>
          Manage feature flags for <strong>{project?.name ?? "select a project"}</strong> / <strong>{stage?.name ?? "stage"}</strong>.
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
        <strong>Existing flags</strong>
        <div style={{ maxHeight: "280px", overflow: "auto" }}>
          <table className="table" aria-label="Flags list">
            <thead>
              <tr>
                <th>Key</th>
                <th>Name</th>
                <th>Enabled</th>
                <th>Active</th>
              </tr>
            </thead>
            <tbody>
              {flags.length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ color: "#94a3b8", fontSize: "0.9rem" }}>
                    No flags defined.
                  </td>
                </tr>
              ) : (
                flags.map((flag) => (
                  <tr
                    key={flag.id}
                    onClick={() => setSelectedFlagId(flag.id)}
                    style={{
                      cursor: "pointer",
                      background: selectedFlagId === flag.id ? "#eff6ff" : undefined
                    }}
                  >
                    <td>{flag.key}</td>
                    <td>{flag.name}</td>
                    <td>{flag.enabled ? "Yes" : "No"}</td>
                    <td>{flag.active ? "Yes" : "No"}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <strong>Create flag</strong>
        <form onSubmit={handleCreate} className="form">
          <div className="form-field">
            <label htmlFor="new-flag-key">Key</label>
            <input
              id="new-flag-key"
              value={createForm.key}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
              placeholder="checkout-flow"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-flag-name">Name</label>
            <input
              id="new-flag-name"
              value={createForm.name}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
              placeholder="Checkout Flow"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-flag-description">Description</label>
            <textarea
              id="new-flag-description"
              rows={2}
              value={createForm.description}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
            />
          </div>
          <div className="form-field">
            <label>Enabled by default</label>
            <select
              value={createForm.enabled ? "true" : "false"}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, enabled: event.target.value === "true" }))}
            >
              <option value="true">Yes</option>
              <option value="false">No</option>
            </select>
          </div>
          <div className="form-field">
            <label>Default variation</label>
            <select
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
          <button type="submit" className="primary-button" disabled={!project || !stage}>
            Create flag
          </button>
        </form>
      </section>

      {selectedFlag ? (
        <section>
          <strong>Edit flag</strong>
          <form onSubmit={handleUpdate} className="form">
            <div className="form-field">
              <label htmlFor="edit-flag-name">Name</label>
              <input
                id="edit-flag-name"
                value={editForm.name}
                onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                required
              />
            </div>
            <div className="form-field">
              <label htmlFor="edit-flag-description">Description</label>
              <textarea
                id="edit-flag-description"
                rows={2}
                value={editForm.description}
                onChange={(event) => setEditForm((prev) => ({ ...prev, description: event.target.value }))}
              />
            </div>
            <div className="form-field">
              <label htmlFor="edit-flag-enabled">Enabled</label>
              <select
                id="edit-flag-enabled"
                value={editForm.enabled ? "true" : "false"}
                onChange={(event) => setEditForm((prev) => ({ ...prev, enabled: event.target.value === "true" }))}
              >
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
            </div>
            <div className="form-field">
              <label htmlFor="edit-flag-active">Active</label>
              <select
                id="edit-flag-active"
                value={editForm.active ? "true" : "false"}
                onChange={(event) => setEditForm((prev) => ({ ...prev, active: event.target.value === "true" }))}
              >
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
            </div>
            <div className="form-field">
              <label htmlFor="edit-flag-default">Default variation</label>
              <select
                id="edit-flag-default"
                value={editForm.defaultKey}
                onChange={(event) => setEditForm((prev) => ({ ...prev, defaultKey: event.target.value }))}
              >
                {selectedFlag.variations.map((variation) => (
                  <option key={variation.key} value={variation.key}>
                    {variation.key}
                  </option>
                ))}
              </select>
            </div>
            <div className="inline-actions">
              <button type="submit" className="primary">
                Save changes
              </button>
              <button type="button" className="secondary" onClick={handleDelete}>
                Delete
              </button>
            </div>
          </form>
        </section>
      ) : null}
    </div>
  );
}

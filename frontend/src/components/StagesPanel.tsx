import { FormEvent, useEffect, useMemo, useState } from "react";

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

export default function StagesPanel({
  project,
  stages,
  selectedStageKey,
  onSelect,
  onCreate,
  onUpdate,
  onDelete
}: StagesPanelProps) {
  const selectedStage = useMemo(
    () => stages.find((stage) => stage.key === selectedStageKey) ?? null,
    [selectedStageKey, stages]
  );

  const [createForm, setCreateForm] = useState({ key: "", name: "", description: "" });
  const [editForm, setEditForm] = useState({ name: "", description: "" });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedStage) {
      setEditForm({ name: "", description: "" });
      return;
    }
    setEditForm({ name: selectedStage.name, description: selectedStage.description ?? "" });
  }, [selectedStage]);

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!project) {
      setError("Select a project first");
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onCreate({
        key: createForm.key.trim(),
        name: createForm.name.trim(),
        description: createForm.description.trim() || undefined
      });
      setMessage(`Stage ${createForm.key.trim()} created`);
      setCreateForm({ key: "", name: "", description: "" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create stage");
    }
  };

  const handleUpdate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedStage) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onUpdate(selectedStage.key, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined
      });
      setMessage("Stage updated");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update stage");
    }
  };

  const handleDelete = async () => {
    if (!selectedStage) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onDelete(selectedStage);
      setMessage("Stage deleted");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete stage");
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

      <section>
        <strong>Available stages</strong>
        <div className="list">
          {stages.length === 0 ? (
            <span style={{ color: "#94a3b8", fontSize: "0.9rem" }}>No stages for this project.</span>
          ) : (
            stages.map((stage) => (
              <button
                key={`${stage.projectKey}-${stage.key}`}
                type="button"
                className={stage.key === selectedStageKey ? "active" : ""}
                onClick={() => onSelect(stage)}
              >
                <div style={{ fontWeight: 600 }}>{stage.name}</div>
                <div style={{ fontSize: "0.8rem", color: "#64748b" }}>{stage.key}</div>
              </button>
            ))
          )}
        </div>
      </section>

      <section>
        <strong>Create stage</strong>
        <form onSubmit={handleCreate} className="form">
          <div className="form-field">
            <label htmlFor="new-stage-key">Key</label>
            <input
              id="new-stage-key"
              value={createForm.key}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
              placeholder="production"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-stage-name">Name</label>
            <input
              id="new-stage-name"
              value={createForm.name}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
              placeholder="Production"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-stage-description">Description</label>
            <textarea
              id="new-stage-description"
              rows={2}
              value={createForm.description}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
              placeholder="Optional description"
            />
          </div>
          <button type="submit" className="primary-button" disabled={!project}>
            Create stage
          </button>
        </form>
      </section>

      {selectedStage ? (
        <section>
          <strong>Edit stage</strong>
          <form onSubmit={handleUpdate} className="form">
            <div className="form-field">
              <label htmlFor="edit-stage-name">Name</label>
              <input
                id="edit-stage-name"
                value={editForm.name}
                onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                required
              />
            </div>
            <div className="form-field">
              <label htmlFor="edit-stage-description">Description</label>
              <textarea
                id="edit-stage-description"
                rows={2}
                value={editForm.description}
                onChange={(event) => setEditForm((prev) => ({ ...prev, description: event.target.value }))}
              />
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

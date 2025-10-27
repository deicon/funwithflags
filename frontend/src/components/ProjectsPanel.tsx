import { FormEvent, useEffect, useMemo, useState } from "react";

import type { Project } from "../types";

interface ProjectsPanelProps {
  projects: Project[];
  selectedProjectKey: string | null;
  onSelect: (project: Project) => void;
  onCreate: (input: { key: string; name: string; description?: string }) => Promise<void>;
  onUpdate: (key: string, input: { name: string; description?: string }) => Promise<void>;
  onDelete: (project: Project) => Promise<void>;
}

export default function ProjectsPanel({
  projects,
  selectedProjectKey,
  onSelect,
  onCreate,
  onUpdate,
  onDelete
}: ProjectsPanelProps) {
  const selectedProject = useMemo(
    () => projects.find((project) => project.key === selectedProjectKey) ?? null,
    [projects, selectedProjectKey]
  );

  const [createForm, setCreateForm] = useState({ key: "", name: "", description: "" });
  const [editForm, setEditForm] = useState({ name: "", description: "" });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedProject) {
      setEditForm({ name: "", description: "" });
      return;
    }
    setEditForm({ name: selectedProject.name, description: selectedProject.description ?? "" });
  }, [selectedProject]);

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    setMessage(null);
    if (!createForm.key.trim() || !createForm.name.trim()) {
      setError("Project key and name are required");
      return;
    }
    try {
      await onCreate({
        key: createForm.key.trim(),
        name: createForm.name.trim(),
        description: createForm.description.trim() || undefined
      });
      setMessage(`Project ${createForm.key.trim()} created`);
      setCreateForm({ key: "", name: "", description: "" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    }
  };

  const handleUpdate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onUpdate(selectedProject.key, {
        name: editForm.name.trim(),
        description: editForm.description.trim() || undefined
      });
      setMessage("Project updated");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update project");
    }
  };

  const handleDelete = async () => {
    if (!selectedProject) {
      return;
    }
    setError(null);
    setMessage(null);
    try {
      await onDelete(selectedProject);
      setMessage("Project deleted");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete project");
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

      <section>
        <strong>Available projects</strong>
        <div className="list">
          {projects.length === 0 ? (
            <span style={{ color: "#94a3b8", fontSize: "0.9rem" }}>No projects yet.</span>
          ) : (
            projects.map((project) => (
              <button
                key={project.key}
                type="button"
                className={project.key === selectedProjectKey ? "active" : ""}
                onClick={() => onSelect(project)}
              >
                <div style={{ fontWeight: 600 }}>{project.name}</div>
                <div style={{ fontSize: "0.8rem", color: "#64748b" }}>{project.key}</div>
              </button>
            ))
          )}
        </div>
      </section>

      <section>
        <strong>Create project</strong>
        <form className="form" onSubmit={handleCreate}>
          <div className="form-field">
            <label htmlFor="new-project-key">Key</label>
            <input
              id="new-project-key"
              value={createForm.key}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, key: event.target.value }))}
              placeholder="checkout"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-project-name">Name</label>
            <input
              id="new-project-name"
              value={createForm.name}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
              placeholder="Checkout"
              required
            />
          </div>
          <div className="form-field">
            <label htmlFor="new-project-description">Description</label>
            <textarea
              id="new-project-description"
              rows={2}
              value={createForm.description}
              onChange={(event) => setCreateForm((prev) => ({ ...prev, description: event.target.value }))}
              placeholder="Describe the project"
            />
          </div>
          <button type="submit" className="primary-button">
            Create project
          </button>
        </form>
      </section>

      {selectedProject ? (
        <section>
          <strong>Edit project</strong>
          <form onSubmit={handleUpdate} className="form">
            <div className="form-field">
              <label htmlFor="edit-project-name">Name</label>
              <input
                id="edit-project-name"
                value={editForm.name}
                onChange={(event) => setEditForm((prev) => ({ ...prev, name: event.target.value }))}
                required
              />
            </div>
            <div className="form-field">
              <label htmlFor="edit-project-description">Description</label>
              <textarea
                id="edit-project-description"
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

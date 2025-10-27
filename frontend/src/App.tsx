import { useCallback, useEffect, useMemo, useState } from "react";

import LoginForm from "./components/LoginForm";
import ProjectsPanel from "./components/ProjectsPanel";
import StagesPanel from "./components/StagesPanel";
import FlagsPanel from "./components/FlagsPanel";
import UsersPanel from "./components/UsersPanel";
import { useAuth } from "./hooks/useAuth";
import type { AuthUser, FeatureFlag, Project, Stage } from "./types";

interface ProjectsResponse {
  projects: Project[];
}

interface StagesResponse {
  stages: Stage[];
}

interface FlagsResponse {
  flags: FeatureFlag[] | null;
}

export default function App() {
  const { isAuthenticated, user, logout, authorizedFetch } = useAuth();
  const [projects, setProjects] = useState<Project[]>([]);
  const [stages, setStages] = useState<Stage[]>([]);
  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [selectedProjectKey, setSelectedProjectKey] = useState<string | null>(null);
  const [selectedStageKey, setSelectedStageKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeSection, setActiveSection] = useState<"flags" | "projects-stages" | "users">("flags");

  const selectedProject = useMemo(
    () => projects.find((project) => project.key === selectedProjectKey) ?? null,
    [projects, selectedProjectKey]
  );
  const selectedStage = useMemo(
    () => stages.find((stage) => stage.key === selectedStageKey) ?? null,
    [stages, selectedStageKey]
  );

  const isAdmin = user?.role === "admin";

  const handleError = useCallback((message: string) => {
    setError(message);
    console.error(message);
  }, []);

  const loadProjects = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await authorizedFetch<ProjectsResponse>("/api/v1/admin/projects");
      setProjects(data.projects);
      if (data.projects.length > 0) {
        setSelectedProjectKey((prev) => prev ?? data.projects[0].key);
      } else {
        setSelectedProjectKey(null);
        setStages([]);
        setFlags([]);
      }
    } catch (err) {
      handleError(err instanceof Error ? err.message : "Failed to load projects");
      setProjects([]);
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch, handleError]);

  const loadStages = useCallback(
    async (projectKey: string) => {
      setLoading(true);
      setError(null);
      try {
        const data = await authorizedFetch<StagesResponse>(`/api/v1/admin/projects/${projectKey}/stages`);
        setStages(data.stages);
        if (data.stages.length > 0) {
          setSelectedStageKey((prev) => (prev && data.stages.some((stage) => stage.key === prev) ? prev : data.stages[0].key));
        } else {
          setSelectedStageKey(null);
          setFlags([]);
        }
      } catch (err) {
        handleError(err instanceof Error ? err.message : "Failed to load stages");
        setStages([]);
        setSelectedStageKey(null);
        setFlags([]);
      } finally {
        setLoading(false);
      }
    },
    [authorizedFetch, handleError]
  );

  const loadFlags = useCallback(
    async (projectKey: string, stageKey: string) => {
      setLoading(true);
      setError(null);
      try {
        const data = await authorizedFetch<FlagsResponse>(`/api/v1/admin/${projectKey}/${stageKey}/flags`);
        setFlags(Array.isArray(data.flags) ? data.flags : []);
      } catch (err) {
        handleError(err instanceof Error ? err.message : "Failed to load flags");
        setFlags([]);
      } finally {
        setLoading(false);
      }
    },
    [authorizedFetch, handleError]
  );

  useEffect(() => {
    if (!isAuthenticated) {
      setProjects([]);
      setStages([]);
      setFlags([]);
      setUsers([]);
      setSelectedProjectKey(null);
      setSelectedStageKey(null);
      setError(null);
      return;
    }
    loadProjects().catch((err) => handleError(err instanceof Error ? err.message : "Failed to load projects"));
  }, [handleError, isAuthenticated, loadProjects]);

  useEffect(() => {
    if (!isAuthenticated || !selectedProjectKey) {
      return;
    }
    loadStages(selectedProjectKey).catch((err) => handleError(err instanceof Error ? err.message : "Failed to load stages"));
  }, [handleError, isAuthenticated, loadStages, selectedProjectKey]);

  useEffect(() => {
    if (!isAuthenticated || !selectedProjectKey || !selectedStageKey) {
      return;
    }
    loadFlags(selectedProjectKey, selectedStageKey).catch((err) => handleError(err instanceof Error ? err.message : "Failed to load flags"));
  }, [handleError, isAuthenticated, loadFlags, selectedProjectKey, selectedStageKey]);

  const handleSelectProject = useCallback((project: Project) => {
    setSelectedProjectKey(project.key);
  }, []);

  const handleSelectStage = useCallback((stage: Stage) => {
    setSelectedStageKey(stage.key);
  }, []);

  const handleCreateProject = useCallback(
    async (input: { key: string; name: string; description?: string }) => {
      await authorizedFetch(`/api/v1/admin/projects`, {
        method: "POST",
        body: input
      });
      await loadProjects();
    },
    [authorizedFetch, loadProjects]
  );

  const handleUpdateProject = useCallback(
    async (key: string, input: { name: string; description?: string }) => {
      await authorizedFetch(`/api/v1/admin/projects/${key}`, {
        method: "PUT",
        body: input
      });
      await loadProjects();
    },
    [authorizedFetch, loadProjects]
  );

  const handleDeleteProject = useCallback(
    async (project: Project) => {
      if (!window.confirm(`Delete project ${project.key}?`)) {
        return;
      }
      await authorizedFetch(`/api/v1/admin/projects/${project.key}`, {
        method: "DELETE"
      });
      await loadProjects();
    },
    [authorizedFetch, loadProjects]
  );

  const handleCreateStage = useCallback(
    async (input: { key: string; name: string; description?: string }) => {
      if (!selectedProjectKey) {
        throw new Error("Select a project first");
      }
      await authorizedFetch(`/api/v1/admin/projects/${selectedProjectKey}/stages`, {
        method: "POST",
        body: input
      });
      await loadStages(selectedProjectKey);
    },
    [authorizedFetch, loadStages, selectedProjectKey]
  );

  const handleUpdateStage = useCallback(
    async (key: string, input: { name: string; description?: string }) => {
      if (!selectedProjectKey) {
        throw new Error("Select a project first");
      }
      await authorizedFetch(`/api/v1/admin/projects/${selectedProjectKey}/stages/${key}`, {
        method: "PUT",
        body: input
      });
      await loadStages(selectedProjectKey);
    },
    [authorizedFetch, loadStages, selectedProjectKey]
  );

  const handleDeleteStage = useCallback(
    async (stage: Stage) => {
      if (!selectedProjectKey) {
        throw new Error("Select a project first");
      }
      if (!window.confirm(`Delete stage ${stage.key}?`)) {
        return;
      }
      await authorizedFetch(`/api/v1/admin/projects/${selectedProjectKey}/stages/${stage.key}`, {
        method: "DELETE"
      });
      await loadStages(selectedProjectKey);
    },
    [authorizedFetch, loadStages, selectedProjectKey]
  );

  const loadUsers = useCallback(async () => {
    if (!isAdmin) {
      setUsers([]);
      return;
    }
    try {
      const data = await authorizedFetch<{ users: AuthUser[] }>("/api/v1/admin/users");
      setUsers(data.users);
    } catch (err) {
      handleError(err instanceof Error ? err.message : "Failed to load users");
      setUsers([]);
    }
  }, [authorizedFetch, handleError, isAdmin]);

  useEffect(() => {
    if (!isAuthenticated || !isAdmin) {
      setUsers([]);
      return;
    }
    loadUsers().catch((err) => handleError(err instanceof Error ? err.message : "Failed to load users"));
  }, [handleError, isAdmin, isAuthenticated, loadUsers]);

  useEffect(() => {
    if (!isAdmin && activeSection === "users") {
      setActiveSection("projects-stages");
    }
  }, [activeSection, isAdmin]);

  const handleCreateFlag = useCallback(
    async (input: {
      key: string;
      name: string;
      description?: string;
      enabled: boolean;
      active: boolean;
      defaultKey: string;
      validFrom: string;
      validTo?: string;
    }) => {
      if (!selectedProject || !selectedStage) {
        throw new Error("Select a project and stage first");
      }
      const body = {
        key: input.key.trim(),
        name: input.name.trim(),
        description: input.description?.trim() || undefined,
        enabled: input.enabled,
        active: input.active,
        defaultKey: input.defaultKey,
        validFrom: input.validFrom,
        ...(input.validTo ? { validTo: input.validTo } : {}),
        variations: [
          { key: "control", type: "boolean", value: false, description: "Feature off" },
          { key: "enabled", type: "boolean", value: true, description: "Feature on" }
        ],
        rules: []
      };

      await authorizedFetch(`/api/v1/admin/${selectedProject.key}/${selectedStage.key}/flags`, {
        method: "POST",
        body
      });
      await loadFlags(selectedProject.key, selectedStage.key);
    },
    [authorizedFetch, loadFlags, selectedProject, selectedStage]
  );

  const handleUpdateFlag = useCallback(
    async (flag: FeatureFlag, updates: Partial<FeatureFlag>) => {
      const payload: FeatureFlag = {
        ...flag,
        ...updates,
        description: updates.description ?? flag.description,
        variations: flag.variations,
        rules: flag.rules
      };

      await authorizedFetch(`/api/v1/admin/flags/${flag.id}`, {
        method: "PUT",
        body: payload
      });
      if (selectedProject && selectedStage) {
        await loadFlags(selectedProject.key, selectedStage.key);
      }
    },
    [authorizedFetch, loadFlags, selectedProject, selectedStage]
  );

  const handleDeleteFlag = useCallback(
    async (flag: FeatureFlag) => {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}`, {
        method: "DELETE"
      });
      if (selectedProject && selectedStage) {
        await loadFlags(selectedProject.key, selectedStage.key);
      }
    },
    [authorizedFetch, loadFlags, selectedProject, selectedStage]
  );

  const handleCreateUser = useCallback(
    async (input: { username: string; password: string; role: "admin" | "user" }) => {
      await authorizedFetch(`/api/v1/admin/users`, {
        method: "POST",
        body: input
      });
      await loadUsers();
    },
    [authorizedFetch, loadUsers]
  );

  const handleResetUserPassword = useCallback(
    async (username: string, password: string) => {
      await authorizedFetch(`/api/v1/admin/users/${username}`, {
        method: "PUT",
        body: { password }
      });
      await loadUsers();
    },
    [authorizedFetch, loadUsers]
  );

  const handleChangeUserRole = useCallback(
    async (username: string, role: "admin" | "user") => {
      await authorizedFetch(`/api/v1/admin/users/${username}`, {
        method: "PUT",
        body: { role }
      });
      await loadUsers();
    },
    [authorizedFetch, loadUsers]
  );

  const handleDeleteUser = useCallback(
    async (username: string) => {
      await authorizedFetch(`/api/v1/admin/users/${username}`, {
        method: "DELETE"
      });
      await loadUsers();
    },
    [authorizedFetch, loadUsers]
  );

  if (!isAuthenticated) {
    return <LoginForm />;
  }

  const sections: Array<{ key: "users" | "projects-stages" | "flags"; label: string; hidden?: boolean }> = [
    { key: "users", label: "Users", hidden: !isAdmin },
    { key: "projects-stages", label: "Projects & Stages" },
    { key: "flags", label: "Flags" }
  ];

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <h1 style={{ margin: 0, fontSize: "1.45rem" }}>FunWithFlags Admin</h1>
          <p style={{ margin: 0, fontSize: "0.85rem", color: "#cbd5f5" }}>
            Manage projects, stages, and feature flags.
          </p>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
          <span style={{ fontSize: "0.9rem" }}>
            Signed in as <strong>{user?.username}</strong> ({user?.role})
          </span>
          <button type="button" className="secondary-button" onClick={logout}>
            Logout
          </button>
        </div>
      </header>

      <div className="app-body">
        <nav className="sidebar" aria-label="Main navigation">
          <span className="sidebar-title">Manage</span>
          {sections
            .filter((section) => !section.hidden)
            .map((section) => (
              <button
                key={section.key}
                type="button"
                className={`nav-button${activeSection === section.key ? " active" : ""}`}
                onClick={() => setActiveSection(section.key)}
              >
                {section.label}
              </button>
            ))}
        </nav>
        <main className="app-content">
          {error ? (
            <div className="notification error" role="alert">
              {error}
            </div>
          ) : null}
          {loading ? (
            <div className="notification" role="status">
              Loading…
            </div>
          ) : null}

          {activeSection === "users" && isAdmin ? (
            <UsersPanel
              users={users}
              currentUsername={user?.username ?? null}
              onCreate={handleCreateUser}
              onResetPassword={handleResetUserPassword}
              onChangeRole={handleChangeUserRole}
              onDelete={handleDeleteUser}
            />
          ) : null}

          {activeSection === "projects-stages" ? (
            <div className="content-stack">
              <ProjectsPanel
                projects={projects}
                selectedProjectKey={selectedProjectKey}
                onSelect={handleSelectProject}
                onCreate={handleCreateProject}
                onUpdate={handleUpdateProject}
                onDelete={handleDeleteProject}
              />
              <StagesPanel
                project={selectedProject}
                stages={stages}
                selectedStageKey={selectedStageKey}
                onSelect={handleSelectStage}
                onCreate={handleCreateStage}
                onUpdate={handleUpdateStage}
                onDelete={handleDeleteStage}
              />
            </div>
          ) : null}

          {activeSection === "flags" ? (
            <FlagsPanel
              projects={projects}
              stages={stages}
              project={selectedProject}
              stage={selectedStage}
              selectedProjectKey={selectedProjectKey}
              selectedStageKey={selectedStageKey}
              flags={flags}
              onSelectProject={handleSelectProject}
              onSelectStage={handleSelectStage}
              onCreate={handleCreateFlag}
              onUpdate={handleUpdateFlag}
              onDelete={handleDeleteFlag}
            />
          ) : null}
        </main>
      </div>
    </div>
  );
}

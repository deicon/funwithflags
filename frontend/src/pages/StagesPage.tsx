import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { SearchInput } from "@/components/shared/SearchInput";
import { EmptyState } from "@/components/shared/EmptyState";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { PageBreadcrumb } from "@/components/shared/PageBreadcrumb";
import { Plus, Layers, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { Stage } from "@/types";

interface StagesResponse { stages: Stage[] }

export function StagesPage() {
  const { projectKey } = useParams<{ projectKey: string }>();
  const { authorizedFetch } = useAuth();
  const [stages, setStages] = useState<Stage[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  // Create/Edit modal
  const [modalOpen, setModalOpen] = useState(false);
  const [editingStage, setEditingStage] = useState<Stage | null>(null);
  const [formKey, setFormKey] = useState("");
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [saving, setSaving] = useState(false);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<Stage | null>(null);

  const fetchStages = useCallback(async () => {
    try {
      const data = await authorizedFetch<StagesResponse>(`/api/v1/admin/projects/${projectKey}/stages`);
      setStages(data.stages || []);
    } catch {
      toast.error("Failed to load stages");
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch, projectKey]);

  useEffect(() => {
    fetchStages();
  }, [fetchStages]);

  const filtered = useMemo(
    () => stages.filter((s) => {
      const q = search.toLowerCase();
      return s.name.toLowerCase().includes(q) || s.key.toLowerCase().includes(q);
    }),
    [stages, search]
  );

  const openCreate = () => {
    setEditingStage(null);
    setFormKey("");
    setFormName("");
    setFormDescription("");
    setModalOpen(true);
  };

  const openEdit = (s: Stage, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setEditingStage(s);
    setFormKey(s.key);
    setFormName(s.name);
    setFormDescription(s.description ?? "");
    setModalOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const body = { key: formKey, name: formName, description: formDescription };
      if (editingStage) {
        await authorizedFetch(`/api/v1/admin/projects/${projectKey}/stages/${editingStage.key}`, { method: "PUT", body });
      } else {
        await authorizedFetch(`/api/v1/admin/projects/${projectKey}/stages`, { method: "POST", body });
      }
      toast.success(editingStage ? "Stage updated" : "Stage created");
      setModalOpen(false);
      fetchStages();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save stage");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await authorizedFetch(`/api/v1/admin/projects/${projectKey}/stages/${deleteTarget.key}`, { method: "DELETE" });
      toast.success("Stage deleted");
      setDeleteTarget(null);
      fetchStages();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete stage");
    }
  };

  return (
    <>
      <PageBreadcrumb segments={[
        { label: "Projects", href: "/projects" },
        { label: projectKey!, },
      ]} />

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">{projectKey}</h1>
          <p className="text-sm text-muted-foreground mt-1">Select a stage to manage its flags</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" /> New Stage
        </Button>
      </div>

      <div className="mb-6 max-w-sm">
        <SearchInput placeholder="Search stages..." value={search} onChange={setSearch} />
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-36 rounded-lg" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Layers className="h-10 w-10" />}
          title={search ? "No matches" : "No stages yet"}
          description={search ? "Try a different search term." : "Create your first stage to get started."}
          actionLabel={!search ? "Create Stage" : undefined}
          onAction={!search ? openCreate : undefined}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((stage) => (
            <Link key={stage.key} to={`/projects/${projectKey}/stages/${stage.key}/flags`} className="block group">
              <Card className="h-full transition-colors hover:border-primary/50">
                <CardContent className="pt-5">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="text-[15px] font-semibold text-foreground group-hover:text-primary transition-colors">
                        {stage.name}
                      </div>
                      <div className="text-xs text-muted-foreground font-mono mt-0.5">{stage.key}</div>
                    </div>
                    <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={(e) => openEdit(stage, e)}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive"
                        onClick={(e) => { e.preventDefault(); e.stopPropagation(); setDeleteTarget(stage); }}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                  {stage.description && (
                    <p className="text-sm text-muted-foreground mt-3 line-clamp-2">{stage.description}</p>
                  )}
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingStage ? "Edit Stage" : "New Stage"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="stage-key">Key</Label>
              <Input
                id="stage-key"
                value={formKey}
                onChange={(e) => setFormKey(e.target.value)}
                disabled={!!editingStage}
                placeholder="production"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="stage-name">Name</Label>
              <Input
                id="stage-name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="Production"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="stage-desc">Description</Label>
              <Input
                id="stage-desc"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder="Optional description"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={handleSave} disabled={saving || !formKey || !formName}>
              {saving ? "Saving..." : editingStage ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete stage"
        description={`Are you sure you want to delete "${deleteTarget?.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        destructive
      />
    </>
  );
}

import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { PageBreadcrumb } from "@/components/shared/PageBreadcrumb";
import { Trash2, Pencil, Check, X } from "lucide-react";
import { toast } from "sonner";
import type { FeatureFlag } from "@/types";

export function FlagDetailPage() {
  const { projectKey, stageKey, flagKey } = useParams<{
    projectKey: string;
    stageKey: string;
    flagKey: string;
  }>();
  const navigate = useNavigate();
  const { authorizedFetch } = useAuth();
  const [flag, setFlag] = useState<FeatureFlag | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleteOpen, setDeleteOpen] = useState(false);

  // Inline editing state for details
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [savingDetails, setSavingDetails] = useState(false);

  const fetchFlag = useCallback(async () => {
    try {
      const data = await authorizedFetch<FeatureFlag>(`/api/v1/admin/${projectKey}/${stageKey}/flags/${flagKey}`);
      setFlag(data);
    } catch {
      toast.error("Flag not found");
      navigate(`/projects/${projectKey}/stages/${stageKey}/flags`);
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch, projectKey, stageKey, flagKey, navigate]);

  useEffect(() => {
    fetchFlag();
  }, [fetchFlag]);

  const updateFlag = async (updates: Partial<FeatureFlag>) => {
    if (!flag) return false;
    try {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}`, {
        method: "PUT",
        body: { ...flag, ...updates },
      });
      await fetchFlag();
      return true;
    } catch {
      toast.error("Failed to update flag");
      return false;
    }
  };

  const toggleEnabled = async () => {
    if (!flag) return;
    const prev = flag.enabled;
    setFlag({ ...flag, enabled: !prev });
    const ok = await updateFlag({ enabled: !prev });
    if (!ok) setFlag({ ...flag, enabled: prev });
  };

  const handleActivate = async () => {
    if (!flag) return;
    try {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}/activate`, { method: "POST" });
      toast.success("Flag activated");
      fetchFlag();
    } catch {
      toast.error("Failed to activate flag");
    }
  };

  const handleDeactivate = async () => {
    if (!flag) return;
    try {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}/deactivate`, { method: "POST" });
      toast.success("Flag deactivated");
      fetchFlag();
    } catch {
      toast.error("Failed to deactivate flag");
    }
  };

  const startEditDetails = () => {
    if (!flag) return;
    setEditName(flag.name);
    setEditDescription(flag.description || "");
    setEditing(true);
  };

  const saveDetails = async () => {
    setSavingDetails(true);
    const ok = await updateFlag({ name: editName, description: editDescription });
    if (ok) {
      setEditing(false);
      toast.success("Details updated");
    }
    setSavingDetails(false);
  };

  const handleDelete = async () => {
    if (!flag) return;
    try {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}`, { method: "DELETE" });
      toast.success("Flag deleted");
      navigate(`/projects/${projectKey}/stages/${stageKey}/flags`);
    } catch {
      toast.error("Failed to delete flag");
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-2 gap-4">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!flag) return null;

  return (
    <>
      <PageBreadcrumb segments={[
        { label: "Projects", href: "/projects" },
        { label: projectKey!, href: `/projects/${projectKey}/stages` },
        { label: stageKey!, href: `/projects/${projectKey}/stages/${stageKey}/flags` },
        { label: flag.key },
      ]} />

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold text-foreground">{flag.key}</h1>
            <Switch checked={flag.enabled} onCheckedChange={toggleEnabled} />
          </div>
          <p className="text-sm text-muted-foreground mt-1">{flag.description || "No description"}</p>
        </div>
        <Button variant="outline" className="text-destructive border-destructive/30" onClick={() => setDeleteOpen(true)}>
          <Trash2 className="h-4 w-4 mr-2" /> Delete
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Left Column */}
        <div className="space-y-4">
          {/* Details Card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <CardTitle className="text-sm font-semibold">Details</CardTitle>
              {!editing && (
                <Button variant="ghost" size="sm" onClick={startEditDetails}>
                  <Pencil className="h-3.5 w-3.5 mr-1" /> Edit
                </Button>
              )}
            </CardHeader>
            <CardContent className="space-y-3">
              {editing ? (
                <>
                  <div className="space-y-2">
                    <Label>Name</Label>
                    <Input value={editName} onChange={(e) => setEditName(e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Description</Label>
                    <Input value={editDescription} onChange={(e) => setEditDescription(e.target.value)} />
                  </div>
                  <div className="flex gap-2 pt-2">
                    <Button size="sm" onClick={saveDetails} disabled={savingDetails}>
                      <Check className="h-3.5 w-3.5 mr-1" /> Save
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => setEditing(false)}>
                      <X className="h-3.5 w-3.5 mr-1" /> Cancel
                    </Button>
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-1">Name</div>
                    <div className="text-sm text-foreground">{flag.name}</div>
                  </div>
                  <div>
                    <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-1">Key</div>
                    <code className="text-sm bg-muted px-2 py-0.5 rounded">{flag.key}</code>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Temporal Validity Card */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold">Temporal Validity</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-2">
                <Badge variant={flag.active ? "default" : "secondary"}>
                  {flag.active ? "Active" : "Inactive"}
                </Badge>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={flag.active ? handleDeactivate : handleActivate}
                >
                  {flag.active ? "Deactivate" : "Activate"}
                </Button>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-1">Valid From</div>
                  <div className="text-sm text-foreground">
                    {flag.validFrom ? new Date(flag.validFrom).toLocaleDateString() : "—"}
                  </div>
                </div>
                <div>
                  <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-1">Valid To</div>
                  <div className="text-sm text-foreground">
                    {flag.validTo ? new Date(flag.validTo).toLocaleDateString() : "—"}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Variations Card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <CardTitle className="text-sm font-semibold">Variations</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {(flag.variations || []).map((v) => (
                  <div
                    key={v.key}
                    className="flex items-center gap-3 bg-muted/50 border border-border rounded-lg px-3 py-2.5"
                  >
                    <div className={`w-1.5 h-1.5 rounded-full ${v.key === flag.defaultKey ? "bg-primary" : "bg-muted-foreground"}`} />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-foreground">{v.key}</div>
                      {v.description && (
                        <div className="text-xs text-muted-foreground">{v.description}</div>
                      )}
                    </div>
                    <Badge variant="secondary" className="text-[10px]">{v.type}</Badge>
                    {v.key === flag.defaultKey && (
                      <Badge className="text-[10px]">default</Badge>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column */}
        <div className="space-y-4">
          {/* Targeting Rules Card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <CardTitle className="text-sm font-semibold">Targeting Rules</CardTitle>
            </CardHeader>
            <CardContent>
              {(!flag.rules || flag.rules.length === 0) ? (
                <p className="text-sm text-muted-foreground">No targeting rules configured.</p>
              ) : (
                <div className="space-y-3">
                  {flag.rules.map((rule) => (
                    <div key={rule.id} className="bg-muted/50 border border-border rounded-lg p-3.5">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium text-foreground">
                          {rule.description || "Unnamed rule"}
                        </span>
                        {rule.variationKey && (
                          <span className="text-xs text-success">→ {rule.variationKey}</span>
                        )}
                      </div>
                      {/* Conditions */}
                      {rule.conditions && rule.conditions.length > 0 && (
                        <div className="text-xs text-muted-foreground space-y-1">
                          {rule.conditions.map((c, i) => (
                            <div key={i} className="flex items-center gap-1.5">
                              <code className="bg-muted px-1.5 py-0.5 rounded text-[11px]">{c.attribute}</code>
                              <span className="text-muted-foreground">{c.operator}</span>
                              <code className="bg-muted px-1.5 py-0.5 rounded text-[11px]">{String(c.value)}</code>
                            </div>
                          ))}
                        </div>
                      )}
                      {/* Rollout */}
                      {rule.rollout && (
                        <div className="mt-2">
                          <div className="text-xs text-muted-foreground mb-1">
                            Rollout by {rule.rollout.attribute}
                          </div>
                          <div className="space-y-1">
                            {rule.rollout.buckets?.map((b, i) => (
                              <div key={i} className="flex items-center gap-2">
                                <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                                  <div
                                    className="h-full bg-primary rounded-full"
                                    style={{ width: `${b.weight}%` }}
                                  />
                                </div>
                                <span className="text-[11px] text-muted-foreground w-16">
                                  {b.weight}% → {b.variationKey}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Info Card */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold">Info</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Created</span>
                  <span className="text-secondary-foreground">
                    {flag.createdAt ? new Date(flag.createdAt).toLocaleString() : "—"}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Updated</span>
                  <span className="text-secondary-foreground">
                    {flag.updatedAt ? new Date(flag.updatedAt).toLocaleString() : "—"}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Flag ID</span>
                  <code className="text-secondary-foreground text-xs">{flag.id}</code>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete flag"
        description={`Are you sure you want to delete "${flag.key}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        destructive
      />
    </>
  );
}

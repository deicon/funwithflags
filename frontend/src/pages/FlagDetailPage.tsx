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
import { Trash2, Pencil, Check, X, Plus, Clock, ChevronDown, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import type { FeatureFlag, FlagRange, RangeVersion, Rule } from "@/types";

interface RangesResponse { ranges: FlagRange[] | null }
interface VersionsResponse { versions: RangeVersion[] | null }

export function FlagDetailPage() {
  const { projectKey, stageKey, flagKey } = useParams<{
    projectKey: string;
    stageKey: string;
    flagKey: string;
  }>();
  const navigate = useNavigate();
  const { authorizedFetch } = useAuth();
  const [flag, setFlag] = useState<FeatureFlag | null>(null);
  const [ranges, setRanges] = useState<FlagRange[]>([]);
  const [versionsByRange, setVersionsByRange] = useState<Record<number, RangeVersion[]>>({});
  const [loading, setLoading] = useState(true);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [expandedRange, setExpandedRange] = useState<number | null>(null);

  // Inline editing state for details
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [savingDetails, setSavingDetails] = useState(false);

  // Create range state
  const [createRangeOpen, setCreateRangeOpen] = useState(false);
  const [rangeValidFrom, setRangeValidFrom] = useState("");
  const [rangeValidTo, setRangeValidTo] = useState("");
  const [savingRange, setSavingRange] = useState(false);

  const fetchFlag = useCallback(async () => {
    try {
      const data = await authorizedFetch<FeatureFlag>(`/api/v1/admin/${projectKey}/${stageKey}/flags/${flagKey}`);
      setFlag(data);
      return data;
    } catch {
      toast.error("Flag not found");
      navigate(`/projects/${projectKey}/stages/${stageKey}/flags`);
      return null;
    }
  }, [authorizedFetch, projectKey, stageKey, flagKey, navigate]);

  const fetchRanges = useCallback(async (flagId: number) => {
    try {
      const data = await authorizedFetch<RangesResponse>(`/api/v1/admin/flags/${flagId}/ranges`);
      const items = Array.isArray(data.ranges) ? data.ranges : [];
      setRanges(items);
      return items;
    } catch {
      toast.error("Failed to load ranges");
      return [];
    }
  }, [authorizedFetch]);

  const fetchVersions = useCallback(async (rangeId: number) => {
    try {
      const data = await authorizedFetch<VersionsResponse>(`/api/v1/admin/ranges/${rangeId}/versions`);
      const items = Array.isArray(data.versions) ? data.versions : [];
      setVersionsByRange((prev) => ({ ...prev, [rangeId]: items }));
      return items;
    } catch {
      toast.error("Failed to load versions");
      return [];
    }
  }, [authorizedFetch]);

  const loadAll = useCallback(async () => {
    setLoading(true);
    const f = await fetchFlag();
    if (f) {
      const rs = await fetchRanges(f.id);
      for (const r of rs) {
        await fetchVersions(r.id);
      }
      if (rs.length > 0) {
        setExpandedRange(rs[0].id);
      }
    }
    setLoading(false);
  }, [fetchFlag, fetchRanges, fetchVersions]);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

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

  // --- Range operations ---

  const handleCreateRange = async () => {
    if (!flag || !rangeValidFrom) return;
    setSavingRange(true);
    try {
      const body: Record<string, unknown> = {
        validFrom: new Date(rangeValidFrom).toISOString(),
        rules: [],
      };
      if (rangeValidTo) {
        body.validTo = new Date(rangeValidTo).toISOString();
      }
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}/ranges`, {
        method: "POST",
        body,
      });
      toast.success("Range created");
      setCreateRangeOpen(false);
      setRangeValidFrom("");
      setRangeValidTo("");
      const rs = await fetchRanges(flag.id);
      for (const r of rs) {
        await fetchVersions(r.id);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create range");
    } finally {
      setSavingRange(false);
    }
  };

  const handleActivateRange = async (rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/ranges/${rangeId}/activate`, { method: "POST" });
      toast.success("Range activated");
      if (flag) await fetchRanges(flag.id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to activate range");
    }
  };

  const handleDeactivateRange = async (rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/ranges/${rangeId}/deactivate`, { method: "POST" });
      toast.success("Range deactivated");
      if (flag) await fetchRanges(flag.id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to deactivate range");
    }
  };

  const handleDeleteRange = async (rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/ranges/${rangeId}`, { method: "DELETE" });
      toast.success("Range deleted");
      if (flag) {
        const rs = await fetchRanges(flag.id);
        // Clean up versions state
        setVersionsByRange((prev) => {
          const next = { ...prev };
          delete next[rangeId];
          return next;
        });
        if (expandedRange === rangeId) {
          setExpandedRange(rs.length > 0 ? rs[0].id : null);
        }
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete range");
    }
  };

  // --- Version operations ---

  const handlePublishVersion = async (versionId: number, rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/versions/${versionId}/publish`, { method: "POST" });
      toast.success("Version published");
      await fetchVersions(rangeId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to publish version");
    }
  };

  const handleDeleteVersion = async (versionId: number, rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/versions/${versionId}`, { method: "DELETE" });
      toast.success("Draft version deleted");
      await fetchVersions(rangeId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete version");
    }
  };

  const handleRollbackVersion = async (versionId: number, rangeId: number) => {
    try {
      await authorizedFetch(`/api/v1/admin/versions/${versionId}/rollback`, { method: "POST" });
      toast.success("Rollback draft created");
      await fetchVersions(rangeId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to rollback");
    }
  };

  const formatDate = (s: string) => {
    try { return new Date(s).toLocaleString(); } catch { return s; }
  };

  const formatShortDate = (s: string) => {
    try { return new Date(s).toLocaleDateString(); } catch { return s; }
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
                    {flag.createdAt ? formatDate(flag.createdAt) : "—"}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Updated</span>
                  <span className="text-secondary-foreground">
                    {flag.updatedAt ? formatDate(flag.updatedAt) : "—"}
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

        {/* Right Column — Ranges & Versions */}
        <div className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <CardTitle className="text-sm font-semibold">Temporal Ranges</CardTitle>
              <Button size="sm" onClick={() => setCreateRangeOpen(true)}>
                <Plus className="h-3.5 w-3.5 mr-1" /> Add Range
              </Button>
            </CardHeader>
            <CardContent>
              {ranges.length === 0 ? (
                <p className="text-sm text-muted-foreground">No ranges configured. Add a temporal range to define when this flag is evaluated.</p>
              ) : (
                <div className="space-y-3">
                  {ranges.map((rng) => (
                    <RangeCard
                      key={rng.id}
                      range={rng}
                      versions={versionsByRange[rng.id] || []}
                      expanded={expandedRange === rng.id}
                      onToggle={() => setExpandedRange(expandedRange === rng.id ? null : rng.id)}
                      onActivate={() => handleActivateRange(rng.id)}
                      onDeactivate={() => handleDeactivateRange(rng.id)}
                      onDelete={() => handleDeleteRange(rng.id)}
                      onPublishVersion={(vId) => handlePublishVersion(vId, rng.id)}
                      onDeleteVersion={(vId) => handleDeleteVersion(vId, rng.id)}
                      onRollbackVersion={(vId) => handleRollbackVersion(vId, rng.id)}
                      formatShortDate={formatShortDate}
                      formatDate={formatDate}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Create Range Modal */}
      {createRangeOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-card border border-border rounded-lg p-6 w-full max-w-md shadow-lg">
            <h3 className="text-lg font-semibold text-foreground mb-4">Add Temporal Range</h3>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="range-from">Valid From</Label>
                <Input
                  id="range-from"
                  type="datetime-local"
                  value={rangeValidFrom}
                  onChange={(e) => setRangeValidFrom(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="range-to">Valid To (optional)</Label>
                <Input
                  id="range-to"
                  type="datetime-local"
                  value={rangeValidTo}
                  onChange={(e) => setRangeValidTo(e.target.value)}
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <Button variant="outline" onClick={() => setCreateRangeOpen(false)}>Cancel</Button>
              <Button onClick={handleCreateRange} disabled={savingRange || !rangeValidFrom}>
                {savingRange ? "Creating..." : "Create"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete flag"
        description={`Are you sure you want to delete "${flag.key}"? This will remove all ranges and versions. This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        destructive
      />
    </>
  );
}

// --- Range Card Component ---

function RangeCard({
  range,
  versions,
  expanded,
  onToggle,
  onActivate,
  onDeactivate,
  onDelete,
  onPublishVersion,
  onDeleteVersion,
  onRollbackVersion,
  formatShortDate,
  formatDate,
}: {
  range: FlagRange;
  versions: RangeVersion[];
  expanded: boolean;
  onToggle: () => void;
  onActivate: () => void;
  onDeactivate: () => void;
  onDelete: () => void;
  onPublishVersion: (id: number) => void;
  onDeleteVersion: (id: number) => void;
  onRollbackVersion: (id: number) => void;
  formatShortDate: (s: string) => string;
  formatDate: (s: string) => string;
}) {
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  const sortedVersions = [...versions].sort((a, b) => b.version - a.version);
  const publishedVersion = sortedVersions.find((v) => v.status === "published");
  const draftVersion = sortedVersions.find((v) => v.status === "draft");

  return (
    <div className="bg-muted/50 border border-border rounded-lg overflow-hidden">
      {/* Range Header */}
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-muted/80 transition-colors"
      >
        {expanded ? <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" /> : <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />}
        <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-foreground">
            {formatShortDate(range.validFrom)}
            {" — "}
            {range.validTo ? formatShortDate(range.validTo) : "∞"}
          </div>
        </div>
        <Badge variant={range.active ? "default" : "secondary"} className="text-[10px]">
          {range.active ? "Active" : "Inactive"}
        </Badge>
      </button>

      {/* Expanded Content */}
      {expanded && (
        <div className="border-t border-border px-4 py-3 space-y-3">
          {/* Range Actions */}
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={range.active ? onDeactivate : onActivate}
            >
              {range.active ? "Deactivate" : "Activate"}
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="text-destructive border-destructive/30"
              onClick={() => setDeleteConfirm(true)}
            >
              <Trash2 className="h-3.5 w-3.5 mr-1" /> Delete Range
            </Button>
          </div>

          {/* Published Version */}
          {publishedVersion && (
            <div>
              <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-2">
                Published (v{publishedVersion.version})
                {publishedVersion.publishedAt && (
                  <span className="ml-2 normal-case">— {formatDate(publishedVersion.publishedAt)}</span>
                )}
              </div>
              <RulesDisplay rules={publishedVersion.rules} />
              <div className="mt-2">
                <Button size="sm" variant="outline" onClick={() => onRollbackVersion(publishedVersion.id)}>
                  Create rollback draft
                </Button>
              </div>
            </div>
          )}

          {/* Draft Version */}
          {draftVersion && (
            <div>
              <div className="flex items-center gap-2 mb-2">
                <div className="text-[11px] text-muted-foreground uppercase tracking-wide">
                  Draft (v{draftVersion.version})
                </div>
                <Badge variant="secondary" className="text-[10px]">draft</Badge>
              </div>
              <RulesDisplay rules={draftVersion.rules} />
              <div className="flex gap-2 mt-2">
                <Button size="sm" onClick={() => onPublishVersion(draftVersion.id)}>
                  Publish
                </Button>
                <Button size="sm" variant="outline" className="text-destructive border-destructive/30" onClick={() => onDeleteVersion(draftVersion.id)}>
                  Discard Draft
                </Button>
              </div>
            </div>
          )}

          {!publishedVersion && !draftVersion && (
            <p className="text-sm text-muted-foreground">No versions yet.</p>
          )}

          {/* Delete range confirmation */}
          <ConfirmDialog
            open={deleteConfirm}
            onOpenChange={setDeleteConfirm}
            title="Delete range"
            description="Are you sure you want to delete this range and all its versions? This cannot be undone."
            confirmLabel="Delete"
            onConfirm={() => { setDeleteConfirm(false); onDelete(); }}
            destructive
          />
        </div>
      )}
    </div>
  );
}

// --- Rules Display Component ---

function RulesDisplay({ rules }: { rules: Rule[] }) {
  if (!rules || rules.length === 0) {
    return <p className="text-xs text-muted-foreground">No targeting rules configured.</p>;
  }

  return (
    <div className="space-y-2">
      {rules.map((rule) => (
        <div key={rule.id} className="bg-card border border-border rounded-lg p-3">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs font-medium text-foreground">
              {rule.description || rule.id}
            </span>
            {rule.variationKey && (
              <span className="text-[11px] text-primary">→ {rule.variationKey}</span>
            )}
          </div>
          {/* Conditions */}
          {rule.conditions && rule.conditions.length > 0 && (
            <div className="text-xs text-muted-foreground space-y-1">
              {rule.conditions.map((c, i) => (
                <div key={i} className="flex items-center gap-1.5">
                  <code className="bg-muted px-1.5 py-0.5 rounded text-[11px]">{c.attribute}</code>
                  <span>{c.operator}</span>
                  <code className="bg-muted px-1.5 py-0.5 rounded text-[11px]">{String(c.value)}</code>
                </div>
              ))}
            </div>
          )}
          {/* Rollout */}
          {rule.rollout && (
            <div className="mt-1.5">
              <div className="text-[11px] text-muted-foreground mb-1">
                Rollout by {rule.rollout.attribute}
              </div>
              <div className="space-y-1">
                {rule.rollout.buckets?.map((b, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                      <div className="h-full bg-primary rounded-full" style={{ width: `${b.weight * 100}%` }} />
                    </div>
                    <span className="text-[11px] text-muted-foreground w-20">
                      {Math.round(b.weight * 100)}% → {b.variationKey}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

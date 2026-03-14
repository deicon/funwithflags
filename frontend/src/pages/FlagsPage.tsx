import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { SearchInput } from "@/components/shared/SearchInput";
import { EmptyState } from "@/components/shared/EmptyState";
import { PageBreadcrumb } from "@/components/shared/PageBreadcrumb";
import { Plus, Flag, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import type { FeatureFlag } from "@/types";

interface FlagsResponse { flags: FeatureFlag[] | null }

export function FlagsPage() {
  const { projectKey, stageKey } = useParams<{ projectKey: string; stageKey: string }>();
  const { authorizedFetch } = useAuth();
  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  // Create modal
  const [createOpen, setCreateOpen] = useState(false);
  const [formKey, setFormKey] = useState("");
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [saving, setSaving] = useState(false);

  const fetchFlags = useCallback(async () => {
    try {
      const data = await authorizedFetch<FlagsResponse>(`/api/v1/admin/${projectKey}/${stageKey}/flags`);
      setFlags(Array.isArray(data.flags) ? data.flags : []);
    } catch {
      toast.error("Failed to load flags");
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch, projectKey, stageKey]);

  useEffect(() => {
    fetchFlags();
  }, [fetchFlags]);

  const filtered = useMemo(
    () => flags.filter((f) => {
      const q = search.toLowerCase();
      return (
        f.key.toLowerCase().includes(q) ||
        f.name.toLowerCase().includes(q) ||
        (f.description && f.description.toLowerCase().includes(q))
      );
    }),
    [flags, search]
  );

  const toggleEnabled = async (flag: FeatureFlag) => {
    const prev = flag.enabled;
    // Optimistic update
    setFlags((fs) => fs.map((f) => (f.id === flag.id ? { ...f, enabled: !prev } : f)));
    try {
      await authorizedFetch(`/api/v1/admin/flags/${flag.id}`, {
        method: "PUT",
        body: { ...flag, enabled: !prev },
      });
      // Re-fetch to keep local state in sync with server (e.g. updatedAt)
      fetchFlags();
    } catch {
      // Rollback
      setFlags((fs) => fs.map((f) => (f.id === flag.id ? { ...f, enabled: prev } : f)));
      toast.error("Failed to toggle flag");
    }
  };

  const handleCreate = async () => {
    setSaving(true);
    try {
      await authorizedFetch(`/api/v1/admin/${projectKey}/${stageKey}/flags`, {
        method: "POST",
        body: {
          key: formKey,
          name: formName,
          description: formDescription,
          enabled: false,
          defaultKey: "control",
          variations: [
            { key: "control", type: "boolean", value: false, description: "Control" },
            { key: "enabled", type: "boolean", value: true, description: "Enabled" },
          ],
        },
      });
      toast.success("Flag created");
      setCreateOpen(false);
      setFormKey("");
      setFormName("");
      setFormDescription("");
      fetchFlags();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create flag");
    } finally {
      setSaving(false);
    }
  };

  const getTypeBadge = (flag: FeatureFlag) => {
    const defaultVar = flag.variations?.find((v) => v.key === flag.defaultKey);
    return defaultVar?.type || "boolean";
  };

  return (
    <>
      <PageBreadcrumb segments={[
        { label: "Projects", href: "/projects" },
        { label: projectKey!, href: `/projects/${projectKey}/stages` },
        { label: stageKey! },
      ]} />

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Feature Flags</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {stageKey} — {flags.length} flag{flags.length !== 1 ? "s" : ""}
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4 mr-2" /> New Flag
        </Button>
      </div>

      <div className="mb-6 max-w-sm">
        <SearchInput placeholder="Search flags..." value={search} onChange={setSearch} />
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Flag className="h-10 w-10" />}
          title={search ? "No matches" : "No flags yet"}
          description={search ? "Try a different search term." : "Create your first feature flag."}
          actionLabel={!search ? "Create Flag" : undefined}
          onAction={!search ? () => setCreateOpen(true) : undefined}
        />
      ) : (
        <div className="space-y-2">
          {filtered.map((flag) => (
            <Link
              key={flag.id}
              to={`/projects/${projectKey}/stages/${stageKey}/flags/${flag.key}`}
              className="block group"
            >
              <div className="flex items-center gap-4 bg-card border border-border rounded-lg px-5 py-3.5 transition-colors hover:border-primary/50">
                <div onClick={(e) => e.preventDefault()}>
                  <Switch
                    checked={flag.enabled}
                    onCheckedChange={() => toggleEnabled(flag)}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-semibold text-foreground">{flag.key}</div>
                  {flag.description && (
                    <div className="text-xs text-muted-foreground truncate">{flag.description}</div>
                  )}
                </div>
                <Badge variant="secondary" className="text-xs">
                  {getTypeBadge(flag)}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  Default: <span className="text-secondary-foreground">{flag.defaultKey}</span>
                </span>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create Modal */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New Flag</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="flag-key">Key</Label>
              <Input id="flag-key" value={formKey} onChange={(e) => setFormKey(e.target.value)} placeholder="my-feature" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="flag-name">Name</Label>
              <Input id="flag-name" value={formName} onChange={(e) => setFormName(e.target.value)} placeholder="My Feature" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="flag-desc">Description</Label>
              <Input id="flag-desc" value={formDescription} onChange={(e) => setFormDescription(e.target.value)} placeholder="Optional description" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button onClick={handleCreate} disabled={saving || !formKey || !formName}>
              {saving ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

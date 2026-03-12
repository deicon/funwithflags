import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { SearchInput } from "@/components/shared/SearchInput";
import { EmptyState } from "@/components/shared/EmptyState";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { PageBreadcrumb } from "@/components/shared/PageBreadcrumb";
import { Plus, Users, KeyRound, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { AuthUser } from "@/types";

interface UsersResponse { users: AuthUser[] }

export function UsersPage() {
  const { user: currentUser, authorizedFetch } = useAuth();
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  // Create modal
  const [createOpen, setCreateOpen] = useState(false);
  const [formUsername, setFormUsername] = useState("");
  const [formPassword, setFormPassword] = useState("");
  const [formRole, setFormRole] = useState("user");
  const [saving, setSaving] = useState(false);

  // Reset password modal
  const [resetTarget, setResetTarget] = useState<AuthUser | null>(null);
  const [newPassword, setNewPassword] = useState("");

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<AuthUser | null>(null);

  const fetchUsers = useCallback(async () => {
    try {
      const data = await authorizedFetch<UsersResponse>("/api/v1/admin/users");
      setUsers(data.users || []);
    } catch {
      toast.error("Failed to load users");
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const filtered = useMemo(
    () => users.filter((u) => u.username.toLowerCase().includes(search.toLowerCase())),
    [users, search]
  );

  const handleCreate = async () => {
    setSaving(true);
    try {
      await authorizedFetch("/api/v1/admin/users", {
        method: "POST",
        body: { username: formUsername, password: formPassword, role: formRole },
      });
      toast.success("User created");
      setCreateOpen(false);
      setFormUsername("");
      setFormPassword("");
      setFormRole("user");
      fetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setSaving(false);
    }
  };

  const handleResetPassword = async () => {
    if (!resetTarget) return;
    try {
      await authorizedFetch(`/api/v1/admin/users/${resetTarget.username}`, {
        method: "PUT",
        body: { password: newPassword },
      });
      toast.success("Password reset");
      setResetTarget(null);
      setNewPassword("");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to reset password");
    }
  };

  const handleRoleChange = async (u: AuthUser, newRole: string) => {
    try {
      await authorizedFetch(`/api/v1/admin/users/${u.username}`, {
        method: "PUT",
        body: { role: newRole },
      });
      toast.success("Role updated");
      fetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update role");
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await authorizedFetch(`/api/v1/admin/users/${deleteTarget.username}`, { method: "DELETE" });
      toast.success("User deleted");
      setDeleteTarget(null);
      fetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete user");
    }
  };

  const getInitial = (username: string) => username.charAt(0).toUpperCase();

  return (
    <>
      <PageBreadcrumb segments={[{ label: "Users" }]} />

      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-foreground">Users</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4 mr-2" /> New User
        </Button>
      </div>

      <div className="mb-6 max-w-sm">
        <SearchInput placeholder="Search users..." value={search} onChange={setSearch} />
      </div>

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-14 rounded-lg" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Users className="h-10 w-10" />}
          title={search ? "No matches" : "No users yet"}
          description={search ? "Try a different search term." : "Create your first user."}
          actionLabel={!search ? "Create User" : undefined}
          onAction={!search ? () => setCreateOpen(true) : undefined}
        />
      ) : (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          {/* Header */}
          <div className="grid grid-cols-[2fr_1fr_120px] px-5 py-2.5 border-b border-border text-[11px] uppercase tracking-wide text-muted-foreground">
            <div>Username</div>
            <div>Role</div>
            <div className="text-right">Actions</div>
          </div>
          {/* Rows */}
          {filtered.map((u) => (
            <div key={u.username} className="grid grid-cols-[2fr_1fr_120px] px-5 py-3.5 border-b border-border last:border-b-0 items-center">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center text-sm font-semibold text-foreground">
                  {getInitial(u.username)}
                </div>
                <span className="text-sm font-medium text-foreground">{u.username}</span>
              </div>
              <div>
                <Badge
                  variant={u.role === "admin" ? "default" : "secondary"}
                  className="text-xs cursor-pointer"
                  onClick={() => handleRoleChange(u, u.role === "admin" ? "user" : "admin")}
                >
                  {u.role}
                </Badge>
              </div>
              <div className="flex justify-end gap-1.5">
                <Button variant="secondary" size="sm" className="h-7 text-xs" onClick={() => { setResetTarget(u); setNewPassword(""); }}>
                  <KeyRound className="h-3 w-3 mr-1" /> Reset PW
                </Button>
                {u.username !== currentUser?.username && (
                  <Button variant="ghost" size="sm" className="h-7 text-xs text-destructive" onClick={() => setDeleteTarget(u)}>
                    <Trash2 className="h-3 w-3" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Modal */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New User</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>Username</Label>
              <Input value={formUsername} onChange={(e) => setFormUsername(e.target.value)} placeholder="username" />
            </div>
            <div className="space-y-2">
              <Label>Password</Label>
              <Input type="password" value={formPassword} onChange={(e) => setFormPassword(e.target.value)} placeholder="password" />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <select
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                value={formRole}
                onChange={(e) => setFormRole(e.target.value)}
              >
                <option value="user">user</option>
                <option value="admin">admin</option>
              </select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button onClick={handleCreate} disabled={saving || !formUsername || !formPassword}>
              {saving ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reset Password Modal */}
      <Dialog open={!!resetTarget} onOpenChange={() => setResetTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset Password for {resetTarget?.username}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>New Password</Label>
              <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="New password" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setResetTarget(null)}>Cancel</Button>
            <Button onClick={handleResetPassword} disabled={!newPassword}>Reset</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete user"
        description={`Are you sure you want to delete "${deleteTarget?.username}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        destructive
      />
    </>
  );
}

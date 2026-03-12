import { Link, Outlet, useLocation, Navigate } from "react-router-dom";
import { ThemeToggle } from "./ThemeToggle";
import { useAuth } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import { LogOut } from "lucide-react";

export function AppLayout() {
  const { user, isAuthenticated, logout } = useAuth();
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  const navItems = [
    { label: "Projects", href: "/projects" },
    { label: "Users", href: "/users" },
  ];

  return (
    <div className="min-h-screen bg-background">
      {/* Top nav */}
      <header className="bg-card border-b border-border">
        <div className="mx-auto max-w-7xl px-6 flex items-center h-14 gap-6">
          <Link to="/projects" className="text-[15px] font-bold text-foreground flex items-center gap-1.5">
            <span>🚩</span> FunWithFlags
          </Link>
          <nav className="flex gap-1 ml-4">
            {navItems.map((item) => {
              const isActive = location.pathname.startsWith(item.href);
              return (
                <Link
                  key={item.href}
                  to={item.href}
                  className={`text-[13px] px-3 py-1.5 rounded-md transition-colors ${
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <ThemeToggle />
            <span className="text-xs text-muted-foreground">{user?.username}</span>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={logout}>
              <LogOut className="h-4 w-4" />
              <span className="sr-only">Logout</span>
            </Button>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="mx-auto max-w-7xl px-6 py-6">
        <Outlet />
      </main>
    </div>
  );
}

# FunWithFlags UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Full frontend rewrite with shadcn/ui, Tailwind, React Router, and dual dark/light theme support.

**Architecture:** Replace the existing plain-CSS React SPA with a new build using shadcn/ui components, Tailwind CSS v4 for styling, and React Router v7 for URL-based navigation. Carry over the existing API client and auth context. Backend is unchanged.

**Tech Stack:** React 18, TypeScript, Vite, Tailwind CSS v4, shadcn/ui (Radix UI), React Router v7, Sonner (toasts), lucide-react (icons)

**Spec:** `docs/superpowers/specs/2026-03-12-ui-redesign-design.md`

### Critical API Pattern

The existing `authorizedFetch` (from `api/client.ts`) is a typed wrapper — **NOT** raw `fetch`. It:
- Returns `Promise<T>` (already-parsed JSON), not a `Response` object
- Throws an `Error` on non-OK HTTP responses (with the server's error message)
- Auto-sets `Content-Type: application/json`
- Auto-stringifies `body` if it's an object (no need for `JSON.stringify` or `headers`)
- Returns `undefined` for 204 No Content

**Correct pattern for all page components:**

```tsx
// Fetching lists — backend wraps lists in envelope objects:
// GET /api/v1/admin/projects returns { projects: [...] }
// GET .../stages returns { stages: [...] }
// GET .../flags returns { flags: [...] }
// GET .../users returns { users: [...] }
// GET .../flags/{key} returns the flag object directly (no wrapper)
interface ProjectsResponse { projects: Project[] }
const data = await authorizedFetch<ProjectsResponse>("/api/v1/admin/projects");
setProjects(data.projects || []);

// Mutations (body auto-stringified, Content-Type auto-set):
await authorizedFetch("/api/v1/admin/projects", {
  method: "POST",
  body: { key, name, description },  // object, NOT JSON.stringify
});

// Error handling — always use try/catch, NOT resp.ok:
try {
  const data = await authorizedFetch<ProjectsResponse>(path);
  setProjects(data.projects || []);
} catch (err) {
  toast.error(err instanceof Error ? err.message : "Failed to load");
}
```

**Every page component in this plan MUST follow this pattern.** Do NOT use `resp.ok`, `resp.json()`, or `headers: { "Content-Type": "application/json" }`.

### Other Notes

- `config.js` in `index.html` must be preserved — it provides runtime API base URL for Docker deployments
- `env.d.ts` should be preserved if it exists after shadcn init
- Default credentials in LoginPage (`admin`/`admin123`) are for dev convenience — acceptable since this is an internal tool
- Breadcrumbs use project/stage keys (not names) to avoid extra API calls — acceptable trade-off
- Variation/rule CRUD (add/edit/remove) on the flag detail page is deferred to a follow-up — this plan renders them read-only
- When shadcn init creates `tsconfig.app.json`, keep it and follow its structure
- To prevent theme flash on page load, add an inline script to `index.html` that sets the `dark` class before React mounts

---

## Chunk 1: Foundation & Theme System

### Task 1: Install dependencies and initialize Tailwind + shadcn/ui

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/tsconfig.json`
- Create: `frontend/src/lib/utils.ts`
- Create: `frontend/components.json`

- [ ] **Step 1: Install Tailwind CSS v4 and the Vite plugin**

```bash
cd frontend && npm install tailwindcss @tailwindcss/vite
```

- [ ] **Step 2: Update vite.config.ts to add Tailwind plugin**

Replace `frontend/vite.config.ts` with:

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 3: Update tsconfig.json to add path aliases**

Update `frontend/tsconfig.json` — add `baseUrl` and `paths` to `compilerOptions`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"]
}
```

- [ ] **Step 4: Install React Router, Sonner, and lucide-react**

```bash
cd frontend && npm install react-router-dom sonner lucide-react
```

- [ ] **Step 5: Initialize shadcn/ui**

```bash
cd frontend && npx shadcn@latest init
```

When prompted, select:
- Style: Default
- Base color: Slate
- CSS variables: Yes

This creates `frontend/components.json`, `frontend/src/lib/utils.ts`, and sets up the CSS file.

- [ ] **Step 6: Add required shadcn/ui components**

```bash
cd frontend && npx shadcn@latest add button card dialog input label switch table badge dropdown-menu separator skeleton toast breadcrumb select
```

- [ ] **Step 7: Verify the dev server starts**

```bash
cd frontend && npm run dev
```

Expected: Vite dev server starts on port 5173 without errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): initialize Tailwind v4, shadcn/ui, React Router, Sonner"
```

### Task 2: Set up the theme system with custom color palette

**Files:**
- Modify: `frontend/src/index.css` (or `globals.css` — created by shadcn init)
- Create: `frontend/src/context/ThemeContext.tsx`

- [ ] **Step 1: Replace the CSS variables in the shadcn globals CSS**

Find the CSS file created by shadcn init (typically `frontend/src/index.css` or `frontend/src/app/globals.css`). Replace the `:root` and `.dark` variable blocks with the custom palette from the spec.

The CSS variables should be (using HSL values as shadcn/ui expects):

```css
@import "tailwindcss";

@layer base {
  :root {
    /* Light theme: Teal on green-tinted gray */
    --background: 140 33% 97%;       /* #f0fdf4 */
    --foreground: 220 13% 9%;        /* #111827 */
    --card: 0 0% 100%;               /* #ffffff */
    --card-foreground: 220 13% 9%;   /* #111827 */
    --popover: 0 0% 100%;
    --popover-foreground: 220 13% 9%;
    --primary: 173 43% 30%;          /* #0d9488 */
    --primary-foreground: 0 0% 100%; /* #ffffff */
    --secondary: 220 9% 94%;         /* #f3f4f6 */
    --secondary-foreground: 220 9% 46%;  /* #6b7280 */
    --muted: 220 9% 94%;             /* #f3f4f6 */
    --muted-foreground: 218 11% 65%; /* #9ca3af */
    --accent: 220 9% 94%;
    --accent-foreground: 220 13% 9%;
    --destructive: 0 84% 60%;        /* #ef4444 */
    --destructive-foreground: 0 0% 100%;
    --success: 142 71% 45%;          /* #22c55e */
    --warning: 38 92% 50%;           /* #f59e0b */
    --border: 216 12% 84%;           /* #d1d5db */
    --input: 216 12% 84%;
    --ring: 173 43% 30%;
    --radius: 0.5rem;
  }

  .dark {
    /* Dark theme: Blue on charcoal */
    --background: 225 27% 7%;        /* #0f1117 */
    --foreground: 214 32% 91%;       /* #e6edf3 */
    --card: 216 21% 11%;             /* #161b22 */
    --card-foreground: 214 32% 91%;  /* #e6edf3 */
    --popover: 216 21% 11%;
    --popover-foreground: 214 32% 91%;
    --primary: 212 92% 58%;          /* #2f81f7 */
    --primary-foreground: 0 0% 100%; /* #ffffff */
    --secondary: 215 14% 15%;        /* #21262d */
    --secondary-foreground: 215 14% 52%; /* #7d8590 */
    --muted: 215 14% 15%;            /* #21262d */
    --muted-foreground: 215 8% 31%;  /* #484f58 */
    --accent: 215 14% 15%;
    --accent-foreground: 214 32% 91%;
    --destructive: 2 88% 62%;        /* #f85149 */
    --destructive-foreground: 0 0% 100%;
    --success: 139 55% 49%;          /* #3fb950 */
    --warning: 27 89% 56%;           /* #f0883e */
    --border: 215 14% 15%;           /* #21262d */
    --input: 215 14% 15%;
    --ring: 212 92% 58%;
    --radius: 0.5rem;
  }
}

@layer base {
  * {
    @apply border-border;
  }
  body {
    @apply bg-background text-foreground;
  }
}
```

- [ ] **Step 2: Create ThemeContext**

Create `frontend/src/context/ThemeContext.tsx`:

```tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

type Theme = "dark" | "light" | "system";

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

function getSystemTheme(): "dark" | "light" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const stored = localStorage.getItem("theme");
    return (stored as Theme) || "system";
  });

  useEffect(() => {
    const root = document.documentElement;
    const resolved = theme === "system" ? getSystemTheme() : theme;

    root.classList.remove("light", "dark");
    root.classList.add(resolved);
    localStorage.setItem("theme", theme);
  }, [theme]);

  useEffect(() => {
    if (theme !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => {
      const root = document.documentElement;
      root.classList.remove("light", "dark");
      root.classList.add(getSystemTheme());
    };
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) throw new Error("useTheme must be used within ThemeProvider");
  return context;
}
```

- [ ] **Step 3: Verify theme toggle works with a quick test**

Temporarily update `frontend/src/main.tsx` to wrap the app in `ThemeProvider` and render a test button that toggles the dark class. Open the browser and confirm background color changes.

- [ ] **Step 4: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): add custom dark/light theme system with CSS variables"
```

### Task 3: Delete old frontend code and set up clean entry point

**Files:**
- Delete: `frontend/src/styles.css`
- Delete: `frontend/src/components/LoginForm.tsx`
- Delete: `frontend/src/components/ProjectsPanel.tsx`
- Delete: `frontend/src/components/StagesPanel.tsx`
- Delete: `frontend/src/components/FlagsPanel.tsx`
- Delete: `frontend/src/components/UsersPanel.tsx`
- Delete: `frontend/src/App.tsx`
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/src/types.ts` (keep as-is, already correct)
- Keep: `frontend/src/api/client.ts` (carried over)
- Keep: `frontend/src/context/AuthContext.tsx` (carried over)
- Keep: `frontend/src/hooks/useAuth.ts` (carried over)

- [ ] **Step 1: Delete old component files and styles.css**

```bash
rm frontend/src/styles.css
rm frontend/src/components/LoginForm.tsx
rm frontend/src/components/ProjectsPanel.tsx
rm frontend/src/components/StagesPanel.tsx
rm frontend/src/components/FlagsPanel.tsx
rm frontend/src/components/UsersPanel.tsx
rm frontend/src/App.tsx
```

- [ ] **Step 2: Create directory structure for new components**

```bash
mkdir -p frontend/src/components/layout
mkdir -p frontend/src/components/shared
mkdir -p frontend/src/pages
```

- [ ] **Step 3: Update main.tsx with router and providers**

Replace `frontend/src/main.tsx` with:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { Toaster } from "sonner";
import "./index.css";

// Placeholder routes — will be replaced as pages are built
const router = createBrowserRouter([
  {
    path: "*",
    element: <div className="flex items-center justify-center min-h-screen text-foreground">Loading...</div>,
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <AuthProvider>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" />
      </AuthProvider>
    </ThemeProvider>
  </StrictMode>
);
```

- [ ] **Step 4: Update AuthContext imports**

The existing `AuthContext.tsx` imports from `../api/client`. Update its import to use the `@/` alias:

Change `import { login as apiLogin, refresh as apiRefresh, logout as apiLogout } from '../api/client';` to `import { login as apiLogin, refresh as apiRefresh, logout as apiLogout } from '@/api/client';`.

Similarly update `useAuth.ts` import from `'../context/AuthContext'` to `'@/context/AuthContext'`.

- [ ] **Step 5: Verify dev server starts cleanly**

```bash
cd frontend && npm run dev
```

Expected: Vite starts, browser shows "Loading..." text on a themed background.

- [ ] **Step 6: Commit**

```bash
git add -A frontend/src/
git commit -m "feat(frontend): clean slate — remove old components, set up router and providers"
```

---

## Chunk 2: Layout & Shared Components

### Task 4: Build AppLayout with top nav and breadcrumbs

**Files:**
- Create: `frontend/src/components/layout/AppLayout.tsx`
- Create: `frontend/src/components/layout/ThemeToggle.tsx`

- [ ] **Step 1: Create ThemeToggle component**

Create `frontend/src/components/layout/ThemeToggle.tsx`:

```tsx
import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/context/ThemeContext";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  const toggleTheme = () => {
    const resolved = theme === "system"
      ? (window.matchMedia("(prefers-color-scheme: dark)").matches ? "light" : "dark")
      : theme === "dark" ? "light" : "dark";
    setTheme(resolved);
  };

  const isDark = theme === "dark" || (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);

  return (
    <Button variant="ghost" size="icon" onClick={toggleTheme} className="h-8 w-8 rounded-full">
      {isDark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
      <span className="sr-only">Toggle theme</span>
    </Button>
  );
}
```

- [ ] **Step 2: Create AppLayout component**

Create `frontend/src/components/layout/AppLayout.tsx`:

```tsx
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
```

- [ ] **Step 3: Verify by wiring into the router temporarily**

Update `main.tsx` router to use `AppLayout` as the layout route with a placeholder child. Open browser and confirm the nav bar renders with theme toggle working.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/layout/
git commit -m "feat(frontend): add AppLayout with top nav and ThemeToggle"
```

### Task 5: Build shared components

**Files:**
- Create: `frontend/src/components/shared/SearchInput.tsx`
- Create: `frontend/src/components/shared/ConfirmDialog.tsx`
- Create: `frontend/src/components/shared/EmptyState.tsx`
- Create: `frontend/src/components/shared/PageBreadcrumb.tsx`

- [ ] **Step 1: Create SearchInput**

Create `frontend/src/components/shared/SearchInput.tsx`:

```tsx
import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";

interface SearchInputProps {
  placeholder?: string;
  value: string;
  onChange: (value: string) => void;
  debounceMs?: number;
}

export function SearchInput({ placeholder = "Search...", value, onChange, debounceMs = 300 }: SearchInputProps) {
  const [local, setLocal] = useState(value);

  useEffect(() => {
    setLocal(value);
  }, [value]);

  useEffect(() => {
    if (local === value) return;
    const timer = setTimeout(() => onChange(local), debounceMs);
    return () => clearTimeout(timer);
  }, [local, debounceMs, onChange, value]);

  return (
    <div className="relative">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      <Input
        placeholder={placeholder}
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        className="pl-9"
      />
    </div>
  );
}
```

- [ ] **Step 2: Create ConfirmDialog**

Create `frontend/src/components/shared/ConfirmDialog.tsx`:

```tsx
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmLabel?: string;
  onConfirm: () => void;
  destructive?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "Confirm",
  onConfirm,
  destructive = false,
}: ConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant={destructive ? "destructive" : "default"}
            onClick={() => {
              onConfirm();
              onOpenChange(false);
            }}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 3: Create EmptyState**

Create `frontend/src/components/shared/EmptyState.tsx`:

```tsx
import { Button } from "@/components/ui/button";
import type { ReactNode } from "react";

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({ icon, title, description, actionLabel, onAction }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      {icon && <div className="mb-4 text-muted-foreground">{icon}</div>}
      <h3 className="text-lg font-semibold text-foreground">{title}</h3>
      <p className="mt-1 text-sm text-muted-foreground max-w-sm">{description}</p>
      {actionLabel && onAction && (
        <Button className="mt-4" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Create PageBreadcrumb**

Create `frontend/src/components/shared/PageBreadcrumb.tsx`:

```tsx
import { Link } from "react-router-dom";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Fragment } from "react";

export interface BreadcrumbSegment {
  label: string;
  href?: string;
}

interface PageBreadcrumbProps {
  segments: BreadcrumbSegment[];
}

export function PageBreadcrumb({ segments }: PageBreadcrumbProps) {
  return (
    <Breadcrumb className="mb-4">
      <BreadcrumbList>
        {segments.map((segment, index) => (
          <Fragment key={index}>
            {index > 0 && <BreadcrumbSeparator />}
            <BreadcrumbItem>
              {segment.href ? (
                <BreadcrumbLink asChild>
                  <Link to={segment.href}>{segment.label}</Link>
                </BreadcrumbLink>
              ) : (
                <BreadcrumbPage>{segment.label}</BreadcrumbPage>
              )}
            </BreadcrumbItem>
          </Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/shared/
git commit -m "feat(frontend): add shared components — SearchInput, ConfirmDialog, EmptyState, PageBreadcrumb"
```

---

## Chunk 3: Auth & Login Page

### Task 6: Adapt auth context for React Router and build Login page

**Files:**
- Modify: `frontend/src/context/AuthContext.tsx` (update imports to use `@/` alias)
- Create: `frontend/src/pages/LoginPage.tsx`
- Modify: `frontend/src/main.tsx` (wire up routes)

- [ ] **Step 1: Create LoginPage**

Create `frontend/src/pages/LoginPage.tsx`:

```tsx
import { useState, type FormEvent } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function LoginPage() {
  const { isAuthenticated, login } = useAuth();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("admin123");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (isAuthenticated) {
    return <Navigate to="/projects" replace />;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(username, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <Card className="w-[380px]">
        <CardHeader className="text-center">
          <div className="text-2xl mb-1">🚩</div>
          <CardTitle className="text-xl">FunWithFlags</CardTitle>
          <CardDescription>Sign in to manage your feature flags</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Signing in..." : "Sign in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: Wire up the router with LoginPage and AppLayout**

Replace the router in `frontend/src/main.tsx`:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider, Navigate } from "react-router-dom";
import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { Toaster } from "sonner";
import { AppLayout } from "@/components/layout/AppLayout";
import { LoginPage } from "@/pages/LoginPage";
import "./index.css";

const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/projects" replace /> },
      { path: "projects", element: <div className="text-foreground">Projects — coming soon</div> },
      { path: "projects/:projectKey/stages", element: <div className="text-foreground">Stages — coming soon</div> },
      { path: "projects/:projectKey/stages/:stageKey/flags", element: <div className="text-foreground">Flags — coming soon</div> },
      { path: "projects/:projectKey/stages/:stageKey/flags/:flagKey", element: <div className="text-foreground">Flag Detail — coming soon</div> },
      { path: "users", element: <div className="text-foreground">Users — coming soon</div> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <AuthProvider>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" />
      </AuthProvider>
    </ThemeProvider>
  </StrictMode>
);
```

- [ ] **Step 3: Test the login flow end-to-end**

1. Start the backend: `docker compose up -d` (from project root)
2. Start the frontend: `cd frontend && npm run dev`
3. Open `http://localhost:5173` — should redirect to `/login`
4. Login with admin/admin123 — should redirect to `/projects` and show nav bar
5. Toggle dark/light theme — should work
6. Click logout — should return to login

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add LoginPage and wire up React Router with all route stubs"
```

---

## Chunk 4: Projects & Stages Pages

### Task 7: Build ProjectsPage

**Files:**
- Create: `frontend/src/pages/ProjectsPage.tsx`
- Modify: `frontend/src/main.tsx` (replace placeholder)

- [ ] **Step 1: Create ProjectsPage**

Create `frontend/src/pages/ProjectsPage.tsx`:

```tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
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
import { Plus, FolderOpen, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { Project } from "@/types";

interface ProjectsResponse { projects: Project[] }

export function ProjectsPage() {
  const { authorizedFetch } = useAuth();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  // Create/Edit modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [formKey, setFormKey] = useState("");
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [saving, setSaving] = useState(false);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null);

  const fetchProjects = useCallback(async () => {
    try {
      const data = await authorizedFetch<ProjectsResponse>("/api/v1/admin/projects");
      setProjects(data.projects || []);
    } catch {
      toast.error("Failed to load projects");
    } finally {
      setLoading(false);
    }
  }, [authorizedFetch]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const filtered = useMemo(
    () => projects.filter((p) => {
      const q = search.toLowerCase();
      return p.name.toLowerCase().includes(q) || p.key.toLowerCase().includes(q);
    }),
    [projects, search]
  );

  const openCreate = () => {
    setEditingProject(null);
    setFormKey("");
    setFormName("");
    setFormDescription("");
    setModalOpen(true);
  };

  const openEdit = (p: Project, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setEditingProject(p);
    setFormKey(p.key);
    setFormName(p.name);
    setFormDescription(p.description ?? "");
    setModalOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const body = { key: formKey, name: formName, description: formDescription };
      if (editingProject) {
        await authorizedFetch(`/api/v1/admin/projects/${editingProject.key}`, { method: "PUT", body });
      } else {
        await authorizedFetch("/api/v1/admin/projects", { method: "POST", body });
      }
      toast.success(editingProject ? "Project updated" : "Project created");
      setModalOpen(false);
      fetchProjects();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save project");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await authorizedFetch(`/api/v1/admin/projects/${deleteTarget.key}`, { method: "DELETE" });
      toast.success("Project deleted");
      setDeleteTarget(null);
      fetchProjects();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete project");
    }
  };

  return (
    <>
      <PageBreadcrumb segments={[{ label: "Projects" }]} />

      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-foreground">Projects</h1>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" /> New Project
        </Button>
      </div>

      <div className="mb-6 max-w-sm">
        <SearchInput placeholder="Search projects..." value={search} onChange={setSearch} />
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40 rounded-lg" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<FolderOpen className="h-10 w-10" />}
          title={search ? "No matches" : "No projects yet"}
          description={search ? "Try a different search term." : "Create your first project to get started."}
          actionLabel={!search ? "Create Project" : undefined}
          onAction={!search ? openCreate : undefined}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((project) => (
            <Link key={project.key} to={`/projects/${project.key}/stages`} className="block group">
              <Card className="h-full transition-colors hover:border-primary/50">
                <CardContent className="pt-5">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="text-[15px] font-semibold text-foreground group-hover:text-primary transition-colors">
                        {project.name}
                      </div>
                      <div className="text-xs text-muted-foreground font-mono mt-0.5">{project.key}</div>
                    </div>
                    <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={(e) => openEdit(project, e)}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive"
                        onClick={(e) => { e.preventDefault(); e.stopPropagation(); setDeleteTarget(project); }}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                  {project.description && (
                    <p className="text-sm text-muted-foreground mt-3 line-clamp-2">{project.description}</p>
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
            <DialogTitle>{editingProject ? "Edit Project" : "New Project"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="project-key">Key</Label>
              <Input
                id="project-key"
                value={formKey}
                onChange={(e) => setFormKey(e.target.value)}
                disabled={!!editingProject}
                placeholder="my-project"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-name">Name</Label>
              <Input
                id="project-name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="My Project"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-desc">Description</Label>
              <Input
                id="project-desc"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder="Optional description"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={handleSave} disabled={saving || !formKey || !formName}>
              {saving ? "Saving..." : editingProject ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete project"
        description={`Are you sure you want to delete "${deleteTarget?.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        destructive
      />
    </>
  );
}
```

- [ ] **Step 2: Wire into router**

In `main.tsx`, replace the projects placeholder:
```tsx
import { ProjectsPage } from "@/pages/ProjectsPage";
// ...
{ path: "projects", element: <ProjectsPage /> },
```

- [ ] **Step 3: Test Projects page**

1. Login and navigate to `/projects`
2. Verify card grid renders with existing projects
3. Create a new project via the modal
4. Edit a project
5. Delete a project
6. Test search filtering
7. Verify theme toggle applies to all elements

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add ProjectsPage with card grid, CRUD modals, search"
```

### Task 8: Build StagesPage

**Files:**
- Create: `frontend/src/pages/StagesPage.tsx`
- Modify: `frontend/src/main.tsx` (replace placeholder)

- [ ] **Step 1: Create StagesPage**

Create `frontend/src/pages/StagesPage.tsx`:

```tsx
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
```

- [ ] **Step 2: Wire into router**

In `main.tsx`, replace the stages placeholder:
```tsx
import { StagesPage } from "@/pages/StagesPage";
// ...
{ path: "projects/:projectKey/stages", element: <StagesPage /> },
```

- [ ] **Step 3: Test the Project → Stage navigation flow**

1. Click a project card → navigates to `/projects/{key}/stages`
2. Breadcrumb shows `Projects > {key}`
3. Create, edit, delete stages
4. Click a stage card → navigates to flags (placeholder)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add StagesPage with card grid, CRUD modals, search"
```

---

## Chunk 5: Flags Page & Flag Detail

### Task 9: Build FlagsPage

**Files:**
- Create: `frontend/src/pages/FlagsPage.tsx`
- Modify: `frontend/src/main.tsx` (replace placeholder)

- [ ] **Step 1: Create FlagsPage**

Create `frontend/src/pages/FlagsPage.tsx`:

```tsx
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
          active: false,
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
```

- [ ] **Step 2: Wire into router**

In `main.tsx`, replace the flags placeholder:
```tsx
import { FlagsPage } from "@/pages/FlagsPage";
// ...
{ path: "projects/:projectKey/stages/:stageKey/flags", element: <FlagsPage /> },
```

- [ ] **Step 3: Test the flags list**

1. Navigate Projects → Stage → Flags
2. Verify flags list with toggle switches
3. Toggle a flag enabled/disabled (should be instant, no page reload)
4. Create a new flag
5. Search filtering
6. Click a flag row → navigates to detail page (placeholder)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add FlagsPage with toggle switches, search, create modal"
```

### Task 10: Build FlagDetailPage

**Files:**
- Create: `frontend/src/pages/FlagDetailPage.tsx`
- Modify: `frontend/src/main.tsx` (replace placeholder)

- [ ] **Step 1: Create FlagDetailPage**

Create `frontend/src/pages/FlagDetailPage.tsx`:

```tsx
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

  const defaultVar = flag.variations?.find((v) => v.key === flag.defaultKey);

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
```

- [ ] **Step 2: Wire into router**

In `main.tsx`, replace the flag detail placeholder:
```tsx
import { FlagDetailPage } from "@/pages/FlagDetailPage";
// ...
{ path: "projects/:projectKey/stages/:stageKey/flags/:flagKey", element: <FlagDetailPage /> },
```

- [ ] **Step 3: Test the flag detail page**

1. Click a flag row from the flags list → navigates to detail page
2. Breadcrumb shows full path, each segment is clickable
3. Toggle enabled switch in header
4. Edit details inline (name, description)
5. View variations list with default badge
6. View targeting rules (if any exist)
7. Activate/deactivate temporal validity
8. Delete flag → navigates back to flags list
9. Test both dark and light themes

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add FlagDetailPage with inline edit, variations, rules, temporal validity"
```

---

## Chunk 6: Users Page & Final Cleanup

### Task 11: Build UsersPage

**Files:**
- Create: `frontend/src/pages/UsersPage.tsx`
- Modify: `frontend/src/main.tsx` (replace placeholder)

- [ ] **Step 1: Create UsersPage**

Create `frontend/src/pages/UsersPage.tsx`:

```tsx
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
```

- [ ] **Step 2: Wire into router**

In `main.tsx`, replace the users placeholder:
```tsx
import { UsersPage } from "@/pages/UsersPage";
// ...
{ path: "users", element: <UsersPage /> },
```

- [ ] **Step 3: Test Users page**

1. Navigate to Users page
2. Verify user list with avatars and role badges
3. Create a new user
4. Reset a user's password
5. Toggle role (click badge)
6. Delete a user (not self)
7. Search filtering

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add UsersPage with table, CRUD, password reset"
```

### Task 12: Final cleanup and verification

**Files:**
- Modify: `frontend/src/main.tsx` (remove all placeholder routes)
- Modify: `frontend/index.html` (add inline theme script to prevent flash — keep /config.js for Docker runtime config)

- [ ] **Step 1: Clean up main.tsx — ensure all routes use real page components**

Verify `main.tsx` has all routes wired to real components:
```tsx
import { LoginPage } from "@/pages/LoginPage";
import { ProjectsPage } from "@/pages/ProjectsPage";
import { StagesPage } from "@/pages/StagesPage";
import { FlagsPage } from "@/pages/FlagsPage";
import { FlagDetailPage } from "@/pages/FlagDetailPage";
import { UsersPage } from "@/pages/UsersPage";
```

Remove any remaining placeholder `<div>` elements from the router.

- [ ] **Step 2: Verify the full app end-to-end**

Run through the complete flow:
1. `docker compose up -d` (backend + database)
2. `cd frontend && npm run dev`
3. Open `http://localhost:5173`
4. Login → Projects → Create project → Stages → Create stage → Flags → Create flag → Flag detail → Edit → Delete
5. Users → Create user → Reset PW → Delete
6. Toggle dark/light theme on every page
7. Test breadcrumb navigation (click each segment)
8. Test browser back/forward buttons
9. Test deep links (paste a flag detail URL directly)

- [ ] **Step 3: Run the production build**

```bash
cd frontend && npm run build
```

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): complete UI redesign — all pages, dual theme, React Router"
```

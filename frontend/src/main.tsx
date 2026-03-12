import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider, Navigate } from "react-router-dom";
import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { AppLayout } from "@/components/layout/AppLayout";
import { LoginPage } from "@/pages/LoginPage";
import { ProjectsPage } from "@/pages/ProjectsPage";
import { Toaster } from "@/components/ui/sonner";
import "./styles.css";

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
      { path: "projects", element: <ProjectsPage /> },
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

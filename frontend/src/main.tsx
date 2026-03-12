import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider, Navigate } from "react-router-dom";
import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { AppLayout } from "@/components/layout/AppLayout";
import { LoginPage } from "@/pages/LoginPage";
import { ProjectsPage } from "@/pages/ProjectsPage";
import { StagesPage } from "@/pages/StagesPage";
import { FlagsPage } from "@/pages/FlagsPage";
import { FlagDetailPage } from "@/pages/FlagDetailPage";
import { UsersPage } from "@/pages/UsersPage";
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
      { path: "projects/:projectKey/stages", element: <StagesPage /> },
      { path: "projects/:projectKey/stages/:stageKey/flags", element: <FlagsPage /> },
      { path: "projects/:projectKey/stages/:stageKey/flags/:flagKey", element: <FlagDetailPage /> },
      { path: "users", element: <UsersPage /> },
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

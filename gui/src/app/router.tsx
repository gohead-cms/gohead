import { Routes, Route, Navigate, Outlet } from "react-router-dom";
import RequireAuth from "../features/auth/RequireAuth"; 
import Login from "../features/auth/Login";
import WorkspaceCanvas from "../features/workspace/WorkspaceCanvas";
import Layout from "./Layout"; // The main app shell

/**
 * A protected layout component that includes the main application shell (sidebar, header).
 * It uses an <Outlet /> to render nested child routes.
 */
function DashboardLayout() {
  return (
    <Layout>
      <Outlet />
    </Layout>
  );
}

/**
 * Defines all routes for the application. This component is rendered by App.tsx.
 */
export default function AppRouter() {
  return (
    <Routes>
      {/* Public Routes */}
      <Route path="/login" element={<Login />} />

      {/* Protected Routes */}
      <Route
        element={
          <RequireAuth>
            <DashboardLayout />
          </RequireAuth>
        }
      >
        {/* Redirect base path to the workspace */}
        <Route path="/" element={<Navigate to="/workspace" replace />} />

        {/* Feature Routes */}
        <Route path="/workspace" element={<WorkspaceCanvas />} />
        {/* Add future routes like /agents, /settings here */}
      </Route>

      {/* Fallback Route */}
      <Route path="*" element={<Navigate to="/login" />} />
    </Routes>
  );
}

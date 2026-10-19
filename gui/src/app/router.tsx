import { ReactNode } from "react";
import { Routes, Route, Navigate, Outlet } from "react-router-dom";
import { Center, Spinner } from "@chakra-ui/react";

import { Layout } from "./Layout";
import { LoginPage, RequireAuth } from "../features/auth";
import { MeProvider } from "../features/auth/MeProvider";
import { useMe } from "../features/auth/me";

import { WorkspaceCanvas } from "../features/workspace";
import { DashboardPage } from "../features/dashboard";
import { ContributionEditorPage, ContributionsPage } from "../features/contrib/components";
import { AgentsPage } from "../features/agents";
import { AgentEditorPage } from "../features/agents/components/AgentEditorPage";
import { ContentModelPage } from "../features/model/ContentModelPage";
import { TypeEditorPage } from "../features/model/TypeEditorPage";
import { MediaLibraryPage } from "../features/media/MediaLibraryPage";
import { ActivityPage } from "../features/activity/ActivityPage";
import { UsersPage } from "../features/settings/UsersPage";
import { TokensPage } from "../features/settings/TokensPage";
import { WebhooksPage } from "../features/settings/WebhooksPage";

// The main layout that includes the Sidebar and Header
function DashboardLayout() {
  return (
    <MeProvider>
      <Layout>
        <Outlet />
      </Layout>
    </MeProvider>
  );
}

/** Pages behind admin-only API endpoints: other roles go to the dashboard. */
function AdminOnly({ children }: { children: ReactNode }) {
  const { isAdmin, loading } = useMe();
  if (loading) {
    return (
      <Center py={20}>
        <Spinner />
      </Center>
    );
  }
  return isAdmin ? <>{children}</> : <Navigate to="/dashboard" replace />;
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />

      <Route
        element={
          <RequireAuth>
            <DashboardLayout />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />

        {/* Content */}
        <Route path="/contrib" element={<ContributionsPage />} />
        <Route path="/contrib/edit/:collectionName" element={<ContributionEditorPage />} />
        <Route path="/media" element={<MediaLibraryPage />} />

        {/* Content model */}
        <Route path="/model" element={<AdminOnly><ContentModelPage /></AdminOnly>} />
        <Route path="/model/:kind/new" element={<AdminOnly><TypeEditorPage /></AdminOnly>} />
        <Route path="/model/:kind/:name" element={<AdminOnly><TypeEditorPage /></AdminOnly>} />
        <Route path="/workspace" element={<AdminOnly><WorkspaceCanvas /></AdminOnly>} />
        <Route path="/collections" element={<Navigate to="/model" replace />} />

        {/* AI */}
        <Route path="/agents" element={<AdminOnly><AgentsPage /></AdminOnly>} />
        <Route path="/agents/new" element={<AdminOnly><AgentEditorPage /></AdminOnly>} />
        <Route path="/agents/:name/edit" element={<AdminOnly><AgentEditorPage /></AdminOnly>} />
        <Route path="/activity" element={<AdminOnly><ActivityPage /></AdminOnly>} />

        {/* Settings */}
        <Route path="/settings/users" element={<AdminOnly><UsersPage /></AdminOnly>} />
        <Route path="/settings/tokens" element={<AdminOnly><TokensPage /></AdminOnly>} />
        <Route path="/settings/webhooks" element={<AdminOnly><WebhooksPage /></AdminOnly>} />
      </Route>

      <Route path="*" element={<Navigate to="/" />} />
    </Routes>
  );
}

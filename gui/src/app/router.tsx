import { Routes, Route, Navigate, Outlet } from "react-router-dom";
import { Box, Heading } from "@chakra-ui/react";

// Layout and Auth
import { Layout } from "./Layout";
import { LoginPage, RequireAuth } from "../features/auth"; 

// Feature Pages
import { WorkspaceCanvas } from "../features/workspace";
import { CollectionsPage } from "../features/collections";
import { SettingsPage } from "../features/settings";
import { DashboardPage } from "../features/dashboard";
import { ContributionEditorPage, ContributionsPage } from "../features/contrib/components";
import { AgentsPage } from "../features/agents";

const PrimitivesPage = () => <Box p={8}><Heading>LLM Primitives</Heading></Box>;
const AutomationPage = () => <Box p={8}><Heading>Automation</Heading></Box>;

// The main layout that includes the Sidebar and Header
function DashboardLayout() {
  return (
    <Layout>
      <Outlet />
    </Layout>
  );
}

export function AppRouter() {
  return (
    // The <BrowserRouter> has been removed from this file
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      
      {/* Protected Routes */}
      <Route
        element={
          <RequireAuth>
            <DashboardLayout />
          </RequireAuth>
        }
      >
        {/* Redirect base path to the dashboard */}
        <Route path="/" element={<Navigate to="/dashboard" replace />} />

        <Route path="/dashboard" element={<DashboardPage />} />
        
        {/* Data Management */}
        <Route path="/collections" element={<CollectionsPage />} />
        <Route path="/workspace" element={<WorkspaceCanvas />} />
        <Route path="/contrib" element={<ContributionsPage />} />
        <Route path="/contrib/edit/:collectionName" element={<ContributionEditorPage />} />
        {/* Agent Management */}
        <Route path="/primitives" element={<PrimitivesPage />} />

        <Route path="/agents" element={<AgentsPage />} />

        {/* Automation */}
        <Route path="/automation" element={<AutomationPage />} />

        {/* Settings */}
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      
      {/* Fallback route */}
      <Route path="*" element={<Navigate to="/" />} />
    </Routes>
  );
}


import { BrowserRouter as Router, Routes, Route, Navigate, Outlet } from "react-router-dom";
import Collections from "./components/collections/CollectionPage";
import RequireAuth from "./components/RequireAuth";
import Login from "./pages/Login";
import PageShell from "./layouts/PageShell";
import SchemaStudio from "./components/collections/SchemaStudio";

// Dashboard layout, will render <Header />, <Sidebar /> and an <Outlet /> for nested content
function DashboardLayout() {
  return (
    <PageShell>
      <Outlet />
    </PageShell>
  );
}


export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Login />} />
        <Route
          element={
            <RequireAuth>
              <DashboardLayout />
            </RequireAuth>
          }
        >
          <Route path="/collections" element={<Collections />} />
          <Route path="/collections/studio" element={<SchemaStudio />} /> 
        </Route>
        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </Router>
  );
}

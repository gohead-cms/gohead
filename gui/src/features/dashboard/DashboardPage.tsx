import { useMe } from "../auth/me";
import { PageLoader } from "../../shared/ui/page-loader";
import { AdminDashboard } from "./components/AdminDashboard";
import { MemberDashboard } from "./components/MemberDashboard";

/** /dashboard: the instance overview for admins, shortcuts for other roles. */
export function DashboardPage() {
  const { me, loading, isAdmin, canWrite } = useMe();
  if (loading) return <PageLoader text="Loading dashboard..." />;
  return isAdmin ? <AdminDashboard /> : <MemberDashboard me={me} canWrite={canWrite} />;
}

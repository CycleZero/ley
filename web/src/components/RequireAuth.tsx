import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { PageLoading } from "./ui/Spinner";

export default function RequireAuth() {
  const { user, ready } = useAuthStore();
  if (!ready) return <PageLoading />;
  if (!user) return <Navigate to="/login" replace />;
  return <Outlet />;
}

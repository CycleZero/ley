import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { PageLoading } from "./ui/Spinner";

export default function RequireAdmin() {
  const { user, ready } = useAuthStore();
  if (!ready) return <PageLoading />;
  if (!user) return <Navigate to="/login" replace />;
  if (user.role !== "admin") return <Navigate to="/" replace />;
  return <Outlet />;
}

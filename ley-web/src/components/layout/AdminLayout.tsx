import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { FileText, FolderTree, Home, Image, LayoutDashboard, LogOut, Settings, Tags, ExternalLink } from "lucide-react";
import { useAuthStore } from "@/stores/auth";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/admin", label: "仪表盘", icon: LayoutDashboard, end: true },
  { to: "/admin/articles", label: "文章管理", icon: FileText },
  { to: "/admin/categories", label: "分类管理", icon: FolderTree },
  { to: "/admin/tags", label: "标签管理", icon: Tags },
  { to: "/admin/files", label: "文件管理", icon: Image },
  { to: "/admin/site", label: "站点配置", icon: Settings },
];

export default function AdminLayout() {
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const onLogout = async () => {
    await logout();
    navigate("/");
  };

  const nav = (
    <nav className="flex flex-col gap-1 text-sm">
      {navItems.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            cn(
              "inline-flex items-center gap-2.5 rounded-xl px-3 py-2 transition-colors",
              isActive
                ? "bg-[var(--accent-subtle)] text-[var(--accent)] font-medium"
                : "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]",
            )
          }
        >
          <Icon className="size-4" />
          {label}
        </NavLink>
      ))}
    </nav>
  );

  return (
    <div className="min-h-screen bg-[var(--bg-global)] flex">
      {/* 桌面端侧边栏 */}
      <aside className="hidden w-56 shrink-0 flex-col border-r border-[var(--border)] bg-[var(--bg-card)] p-4 md:flex">
        <a href="/admin" className="mb-6 flex items-center gap-2 px-3 text-lg font-semibold text-[var(--text-primary)]">
          <Home className="size-5 text-[var(--accent)]" />
          Ley 管理
        </a>
        {nav}
        <div className="mt-auto space-y-1 border-t border-[var(--border)] pt-4">
          <NavLink
            to="/"
            className="inline-flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
          >
            <ExternalLink className="size-4" /> 返回站点
          </NavLink>
          <button
            onClick={onLogout}
            className="inline-flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-red-500"
          >
            <LogOut className="size-4" /> 退出登录
          </button>
        </div>
      </aside>

      {/* 移动端顶部导航 */}
      <div className="fixed inset-x-0 top-0 z-40 border-b border-[var(--border)] bg-[var(--bg-card)] md:hidden">
        <div className="flex items-center justify-between px-4 h-12">
          <span className="text-base font-semibold text-[var(--text-primary)]">Ley 管理</span>
          <a href="/" className="text-xs text-[var(--text-tertiary)]">返回站点</a>
        </div>
        <div className="flex gap-1 overflow-x-auto px-3 pb-2">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "inline-flex shrink-0 items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs transition-colors",
                  isActive
                    ? "bg-[var(--accent-subtle)] text-[var(--accent)] font-medium"
                    : "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]",
                )
              }
            >
              <Icon className="size-3.5" />
              {label}
            </NavLink>
          ))}
        </div>
      </div>

      <main className="min-w-0 flex-1 p-4 md:p-8 pt-24 md:pt-8">
        <Outlet />
      </main>
    </div>
  );
}

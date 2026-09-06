import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { ExternalLink, FileText, FolderTree, Image, LayoutDashboard, LogOut, Settings, Tags } from "lucide-react";
import { useAuthStore } from "@/stores/auth";
import { Backdrop } from "@/components/layout/Backdrop";
import { SiteBrand } from "@/components/layout/SiteBrand";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/admin", label: "仪表盘", icon: LayoutDashboard, end: true },
  { to: "/admin/articles", label: "文章管理", icon: FileText },
  { to: "/admin/categories", label: "分类管理", icon: FolderTree },
  { to: "/admin/tags", label: "标签管理", icon: Tags },
  { to: "/admin/files", label: "文件管理", icon: Image },
  { to: "/admin/site", label: "站点配置", icon: Settings },
];

/** 「管理」后缀徽标：辅助 SiteBrand 区分前台/后台（不依赖站点配置） */
function AdminBadge() {
  return (
    <span className="rounded-md bg-[var(--accent-subtle)] px-1.5 py-0.5 text-[10px] font-semibold leading-none tracking-wide text-[var(--accent-blue)]">
      管理
    </span>
  );
}

export default function AdminLayout() {
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const onLogout = async () => {
    await logout();
    navigate("/");
  };

  const nav = (
    <nav aria-label="后台导航" className="flex flex-col gap-1 text-sm">
      {navItems.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            cn(
              "inline-flex items-center gap-2.5 rounded-xl px-3 py-2 transition-colors",
              isActive
                ? "bg-[var(--accent-blue)] font-medium text-white shadow-[0_0_14px_rgba(59,130,246,0.28)]"
                : "text-[var(--ink-2)] hover:bg-[var(--bg-hover)] hover:text-[var(--ink)]",
            )
          }
        >
          <Icon className="size-4" aria-hidden="true" />
          {label}
        </NavLink>
      ))}
    </nav>
  );

  return (
    <div className="flex min-h-screen">
      <Backdrop />

      {/* 桌面端侧栏（≥980px）：固定白色实卡 + 蓝灰描边 */}
      <aside className="sticky top-0 hidden h-screen w-56 shrink-0 flex-col overflow-y-auto border-r border-[var(--border-card)] bg-[var(--bg-card)] px-4 py-6 min-[980px]:flex">
        <div className="flex items-center gap-2 px-2 pb-8">
          <SiteBrand subtitle />
          <AdminBadge />
        </div>
        {nav}
        <div className="mt-auto space-y-1 border-t border-[var(--border-card)] pt-4">
          <NavLink
            to="/"
            title="返回站点"
            className="inline-flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-[var(--ink-3)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--ink)]"
          >
            <ExternalLink className="size-4" aria-hidden="true" /> 返回站点
          </NavLink>
          <button
            onClick={onLogout}
            title="退出登录"
            className="inline-flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-[var(--ink-3)] transition-colors hover:bg-[var(--bg-hover)] hover:text-red-500"
          >
            <LogOut className="size-4" aria-hidden="true" /> 退出登录
          </button>
        </div>
      </aside>

      {/* 移动端顶部横滑导航条（<980px） */}
      <div className="fixed inset-x-0 top-0 z-40 border-b border-[var(--border-card)] bg-[var(--bg-card)] min-[980px]:hidden">
        <div className="flex h-12 items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <SiteBrand />
            <AdminBadge />
          </div>
          <a href="/" title="返回站点" className="text-xs text-[var(--ink-3)] transition-colors hover:text-[var(--ink)]">
            返回站点
          </a>
        </div>
        <div className="flex gap-1.5 overflow-x-auto px-3 pb-2.5">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "inline-flex shrink-0 items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs transition-colors",
                  isActive
                    ? "bg-[var(--accent-blue)] font-medium text-white"
                    : "text-[var(--ink-2)] hover:bg-[var(--bg-hover)] hover:text-[var(--ink)]",
                )
              }
            >
              <Icon className="size-3.5" aria-hidden="true" />
              {label}
            </NavLink>
          ))}
        </div>
      </div>

      <main className="min-w-0 flex-1 p-4 pt-24 min-[980px]:p-8 min-[980px]:pt-8">
        <Outlet />
      </main>
    </div>
  );
}

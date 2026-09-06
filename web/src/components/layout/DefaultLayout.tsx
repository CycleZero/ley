import { Link, Outlet, useNavigate } from "react-router-dom";
import { LayoutDashboard, LogOut, Search, User } from "lucide-react";
import { useAuthStore, useIsAdmin, useIsLoggedIn } from "@/stores/auth";
import { useSiteConfig } from "@/hooks/use-site";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { cn } from "@/lib/utils";
import { Backdrop } from "./Backdrop";
import { NavRail } from "./NavRail";
import { SiteBrand } from "./SiteBrand";

/**
 * 内容列（视觉基准 v4 的 .page）：max-w 1264 居中；
 * 980–1480px 为左侧竖导航让位（--rail 右侧 ~102px，取 118px 留白），
 * 更宽视口回退居中对称留白。
 */
const CONTENT_COL =
  "mx-auto w-full max-w-[1264px] " +
  "max-[979px]:pl-4 max-[979px]:pr-4 " +
  "min-[980px]:max-[1480px]:pl-[118px] min-[980px]:max-[1480px]:pr-8 " +
  "min-[1481px]:pl-8 min-[1481px]:pr-8";

export default function DefaultLayout() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const loggedIn = useIsLoggedIn();
  const isAdmin = useIsAdmin();
  const navigate = useNavigate();
  const { data } = useSiteConfig();

  const onLogout = async () => {
    await logout();
    navigate("/");
  };

  // footer：site config 的 footer_text；空则中性占位，绝不硬编码站名
  const footerText = data?.config?.footer_text?.trim() || `© ${new Date().getFullYear()}`;

  return (
    <div className="relative flex min-h-screen flex-col">
      {/* 背景层：近白渐变 + 科技网格（T6，纯装饰不参与交互/读屏） */}
      <Backdrop />

      {/* 顶栏：近白半透明 + 蓝灰底边框 */}
      <header className="sticky top-0 z-40 h-16 border-b border-[var(--border)] bg-[var(--bg-card)]/80 backdrop-blur">
        <div className={cn(CONTENT_COL, "flex h-full items-center gap-4")}>
          <SiteBrand subtitle />
          <div className="ml-auto flex items-center gap-1">
            <Link
              to="/search"
              className="inline-flex size-8 items-center justify-center rounded-xl text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
              aria-label="搜索"
            >
              <Search className="size-4" />
            </Link>
            <ThemeToggle />
            {loggedIn && user ? (
              <div className="ml-1 flex items-center gap-1">
                {isAdmin && (
                  <Link
                    to="/admin"
                    className="inline-flex size-8 items-center justify-center rounded-xl text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                    title="管理后台"
                  >
                    <LayoutDashboard className="size-4" />
                  </Link>
                )}
                <Link
                  to="/profile"
                  className="inline-flex items-center gap-1.5 rounded-xl px-2 py-1 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                  title={user.username}
                >
                  {user.avatar ? (
                    <img src={user.avatar} alt="" className="size-6 rounded-full" />
                  ) : (
                    <span className="flex size-6 items-center justify-center rounded-full bg-[var(--accent-subtle)] text-xs font-medium text-[var(--accent)]">
                      {user.username?.charAt(0).toUpperCase()}
                    </span>
                  )}
                  <span className="hidden max-w-24 truncate sm:inline">{user.username}</span>
                </Link>
                <button
                  onClick={onLogout}
                  className="inline-flex size-8 items-center justify-center rounded-xl text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-red-500"
                  title="退出登录"
                >
                  <LogOut className="size-4" />
                </button>
              </div>
            ) : (
              <div className="ml-1 flex items-center gap-1 text-sm">
                <Link
                  to="/login"
                  className="inline-flex h-8 items-center gap-1.5 rounded-xl px-3 text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                >
                  <User className="size-3.5" /> 登录
                </Link>
                <Link
                  to="/register"
                  className="inline-flex h-8 items-center rounded-xl bg-[var(--accent)] px-3 font-medium text-white transition-colors hover:bg-[var(--accent-hover)]"
                >
                  注册
                </Link>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* 主导航：桌面左侧竖列胶囊 + 移动底部横栏（NavRail 内部 fixed 定位） */}
      <NavRail />

      <main className={cn(CONTENT_COL, "flex-1 pb-28 pt-6 min-[980px]:pb-20")}>
        <Outlet />
      </main>

      <footer className="mt-10 pb-28 text-center font-mono text-[11px] tracking-[0.16em] text-[var(--text-tertiary)] min-[980px]:pb-8">
        {footerText}
      </footer>
    </div>
  );
}

import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import { LayoutDashboard, LogOut, Search, User } from "lucide-react";
import { useAuthStore, useIsAdmin, useIsLoggedIn } from "@/stores/auth";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "首页", end: true },
  { to: "/articles", label: "文章" },
  { to: "/categories", label: "分类" },
  { to: "/tags", label: "标签" },
  { to: "/about", label: "关于" },
];

export default function DefaultLayout() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const loggedIn = useIsLoggedIn();
  const isAdmin = useIsAdmin();
  const navigate = useNavigate();

  const onLogout = async () => {
    await logout();
    navigate("/");
  };

  return (
    <div className="min-h-screen bg-[var(--bg-global)] flex flex-col">
      {/* 顶部导航 */}
      <header className="sticky top-0 z-40 h-14 border-b border-[var(--border)] bg-[var(--bg-card)]/80 backdrop-blur flex items-center gap-4 px-4 md:px-6">
        <Link to="/" className="text-lg font-semibold text-[var(--text-primary)]">
          Ley
        </Link>
        <nav className="ml-auto hidden items-center gap-1 text-sm md:flex">
          {navItems.map(({ to, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "rounded-xl px-3 py-1.5 transition-colors",
                  isActive
                    ? "bg-[var(--accent-subtle)] text-[var(--accent)] font-medium"
                    : "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]",
                )
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="flex items-center gap-1">
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
      </header>

      {/* 移动端导航 */}
      <nav className="sticky top-14 z-30 flex gap-1 overflow-x-auto border-b border-[var(--border)] bg-[var(--bg-card)] px-3 py-1.5 text-sm md:hidden">
        {navItems.map(({ to, label, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                "shrink-0 rounded-lg px-2.5 py-1.5 transition-colors",
                isActive
                  ? "bg-[var(--accent-subtle)] text-[var(--accent)] font-medium"
                  : "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]",
              )
            }
          >
            {label}
          </NavLink>
        ))}
      </nav>

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t border-[var(--border)] bg-[var(--bg-card)] py-6 text-center text-sm text-[var(--text-tertiary)]">
        © {new Date().getFullYear()} Ley · Powered by Go &amp; React
      </footer>
    </div>
  );
}

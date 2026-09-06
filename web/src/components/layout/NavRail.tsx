import { NavLink } from "react-router-dom";
import { FileText, House, LayoutGrid, Search, Tag, User } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * 竖排胶囊导航（视觉基准 v4 的 .side）：
 * - 桌面 ≥980px：左侧固定竖排胶囊（图标 + 小字标签）
 * - <980px：收为底部居中横向胶囊栏（避开顶部粘性 header）
 * active 状态：--accent-blue 底 + 白字；accessible names 与 DefaultLayout 导航一致。
 */

export interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  /** 精确匹配（用于首页 "/"） */
  end?: boolean;
}

const defaultNavItems: NavItem[] = [
  { to: "/", label: "首页", icon: House, end: true },
  { to: "/articles", label: "文章", icon: FileText },
  { to: "/categories", label: "分类", icon: LayoutGrid },
  { to: "/tags", label: "标签", icon: Tag },
  { to: "/about", label: "关于", icon: User },
  { to: "/search", label: "搜索", icon: Search },
];

export interface NavRailProps {
  /** 自定义导航项；缺省使用内置 6 项（首页/文章/分类/标签/关于/搜索） */
  items?: NavItem[];
  className?: string;
}

const CONTAINER_BASE =
  "fixed z-40 flex w-fit items-center gap-1 rounded-full border border-[var(--border-card)] bg-[rgba(255,255,255,0.95)] p-1.5 shadow-[var(--shadow-card)]";
// 移动端：底部居中；桌面：左侧竖排
const CONTAINER_POS =
  "bottom-[calc(env(safe-area-inset-bottom)+0.875rem)] left-0 right-0 mx-auto " +
  "min-[980px]:left-5 min-[980px]:right-auto min-[980px]:top-1/2 min-[980px]:bottom-auto " +
  "min-[980px]:mx-0 min-[980px]:-translate-y-1/2 min-[980px]:flex-col min-[980px]:gap-1.5 min-[980px]:p-2";

const ITEM_BASE =
  "flex w-14 flex-col items-center gap-1 rounded-2xl px-2 py-2 transition-colors " +
  "min-[980px]:w-16 min-[980px]:py-2.5";
const ITEM_ACTIVE =
  "bg-[var(--accent-blue)] text-white shadow-[0_0_16px_rgba(59,130,246,0.35)]";
const ITEM_INACTIVE =
  "text-[var(--ink-2)] hover:bg-[var(--bg-hover)] hover:text-[var(--ink)]";

export function NavRail({ items = defaultNavItems, className }: NavRailProps) {
  return (
    <nav aria-label="主导航" className={cn(CONTAINER_BASE, CONTAINER_POS, className)}>
      {items.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          title={label}
          className={({ isActive }) => cn(ITEM_BASE, isActive ? ITEM_ACTIVE : ITEM_INACTIVE)}
        >
          <Icon className="size-[18px]" strokeWidth={1.75} aria-hidden="true" />
          <span className="text-[10px] font-medium leading-none">{label}</span>
        </NavLink>
      ))}
    </nav>
  );
}

export default NavRail;

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * 页面头部标准组件：mono 大写 kicker + 标题 + 可选 description / actions（右侧）。
 * kicker：--font-mono 小字 + 0.1em 字距 + --ink-3 色；标题：--ink + 600 重力。
 */

export interface PageHeaderProps {
  /** mono 大写小 kicker（如 "PROJECT" / "N°01"） */
  kicker?: string;
  title: ReactNode;
  /** 标题下的一句话说明 */
  description?: ReactNode;
  /** 右侧操作区（按钮/连链接等） */
  actions?: ReactNode;
  className?: string;
}

export function PageHeader({ kicker, title, description, actions, className }: PageHeaderProps) {
  return (
    <header className={cn("flex flex-wrap items-end justify-between gap-x-6 gap-y-4", className)}>
      <div className="min-w-0">
        {kicker ? (
          <p className="font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-[var(--ink-3)]">
            {kicker}
          </p>
        ) : null}
        <h1 className="mt-1.5 text-2xl font-semibold tracking-[0.02em] text-[var(--ink)] md:text-3xl">
          {title}
        </h1>
        {description ? (
          <p className="mt-2 max-w-2xl text-sm text-[var(--ink-2)]">{description}</p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex shrink-0 items-center gap-2 self-center md:self-end">{actions}</div>
      ) : null}
    </header>
  );
}

export default PageHeader;

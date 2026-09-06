import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * Bento 面板统一卡片壳（Wave D T12）。
 * 视觉基准：/tmp/opencode/mock-d1-blue-tech/index-v4.html .card + .card-label
 * - 默认 .card（实白描边 22px 圆角，globals.css 已就绪）+ .card-hover 交互抬升
 * - 顶部 mono 小标签行：左侧 label（--ink-3 小字），右侧可选 action（链接等）
 * - grid span 由 T13（home.tsx）统一配置，本组件不写 span
 */
export function BentoCard({
  label,
  action,
  className,
  children,
}: {
  label?: string;
  action?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  const hasHeader = label !== undefined || action !== undefined;
  return (
    <article className={cn("card card-hover px-[19px] pb-4 pt-[18px]", className)}>
      {hasHeader && (
        <div className="mb-3 flex min-h-[16px] items-center justify-between gap-3">
          {label !== undefined && (
            <span className="text-[11.5px] font-semibold tracking-[0.12em] text-[var(--ink-3)]">
              {label}
            </span>
          )}
          {action !== undefined && (
            <span className="font-mono text-[10px] tracking-[0.1em]">{action}</span>
          )}
        </div>
      )}
      {children}
    </article>
  );
}

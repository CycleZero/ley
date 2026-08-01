import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

export function Empty({
  title = "暂无数据",
  description,
  action,
  icon,
}: {
  title?: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
      <div className="flex size-12 items-center justify-center rounded-2xl bg-[var(--bg-hover)] text-[var(--text-tertiary)]">
        {icon ?? <Inbox className="size-6" />}
      </div>
      <div>
        <p className="text-sm font-medium text-[var(--text-primary)]">{title}</p>
        {description && <p className="mt-1 text-xs text-[var(--text-tertiary)]">{description}</p>}
      </div>
      {action}
    </div>
  );
}

import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

/** 生成分页页码（含省略号） */
function pageList(current: number, total: number): Array<number | "…"> {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages: Array<number | "…"> = [1];
  const start = Math.max(2, current - 1);
  const end = Math.min(total - 1, current + 1);
  if (start > 2) pages.push("…");
  for (let i = start; i <= end; i++) pages.push(i);
  if (end < total - 1) pages.push("…");
  pages.push(total);
  return pages;
}

export function Pagination({
  page,
  total,
  pageSize,
  onChange,
}: {
  page: number;
  total: number;
  pageSize: number;
  onChange: (page: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  if (totalPages <= 1) return null;

  const btn =
    "inline-flex size-8 items-center justify-center rounded-lg text-sm transition-colors disabled:opacity-40 disabled:pointer-events-none";

  return (
    <nav className="flex items-center justify-center gap-1" aria-label="分页">
      <button className={cn(btn, "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]")} disabled={page <= 1} onClick={() => onChange(page - 1)} aria-label="上一页">
        <ChevronLeft className="size-4" />
      </button>
      {pageList(page, totalPages).map((p, i) =>
        p === "…" ? (
          <span key={`e${i}`} className="px-1 text-[var(--text-tertiary)]">
            …
          </span>
        ) : (
          <button
            key={p}
            onClick={() => onChange(p)}
            className={cn(
              btn,
              p === page
                ? "bg-[var(--accent)] text-white"
                : "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]",
            )}
          >
            {p}
          </button>
        ),
      )}
      <button className={cn(btn, "text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]")} disabled={page >= totalPages} onClick={() => onChange(page + 1)} aria-label="下一页">
        <ChevronRight className="size-4" />
      </button>
    </nav>
  );
}

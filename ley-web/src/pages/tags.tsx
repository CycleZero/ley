import { Link } from "react-router-dom";
import { useTags } from "@/hooks/use-tags";
import { Empty } from "@/components/ui/Empty";
import { Skeleton } from "@/components/ui/Skeleton";
import { cn } from "@/lib/utils";

/** 按文章数分级的字号映射（标签云） */
function sizeClass(count: number): string {
  if (count >= 20) return "text-lg";
  if (count >= 10) return "text-base";
  if (count >= 5) return "text-sm";
  return "text-xs";
}

/** 标签页：标签云 */
export default function TagsPage() {
  const { data, isLoading } = useTags();

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">标签</h1>
      <p className="mb-8 text-sm text-[var(--text-secondary)]">共 {data?.tags.length ?? 0} 个标签</p>

      {isLoading ? (
        <div className="flex flex-wrap gap-3">
          <Skeleton className="h-8 w-20" />
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-8 w-16" />
          <Skeleton className="h-8 w-24" />
        </div>
      ) : !data || data.tags.length === 0 ? (
        <Empty title="暂无标签" />
      ) : (
        <div className="card flex flex-wrap items-center gap-3 p-6">
          {data.tags.map((tag) => (
            <Link
              key={tag.id}
              to={`/tags/${encodeURIComponent(tag.name)}`}
              className={cn(
                "rounded-xl px-3 py-1.5 font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--accent-subtle)] hover:text-[var(--accent)]",
                sizeClass(tag.article_count),
              )}
            >
              #{tag.name}
              <span className="ml-1 text-xs text-[var(--text-tertiary)]">{tag.article_count}</span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

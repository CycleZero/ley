import { Link } from "react-router-dom";
import { useTags } from "@/hooks/use-tags";
import { PageHeader } from "@/components/ui/PageHeader";
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
      <PageHeader
        kicker="TAGS"
        title="标签"
        description={
          <>
            共 <span className="font-mono tabular-nums">{data?.tags.length ?? 0}</span> 个标签，点击标签查看归属文章
          </>
        }
      />

      {isLoading ? (
        <div className="mt-8 flex flex-wrap gap-3">
          <Skeleton className="h-8 w-20" />
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-8 w-16" />
          <Skeleton className="h-8 w-24" />
        </div>
      ) : !data || data.tags.length === 0 ? (
        <div className="mt-8">
          <Empty title="暂无标签" />
        </div>
      ) : (
        <div className="card mt-8 flex flex-wrap items-center gap-3 p-6">
          {data.tags.map((tag) => (
            <Link
              key={tag.id}
              to={`/tags/${encodeURIComponent(tag.name)}`}
              className={cn(
                "card-hover rounded-full border border-[var(--border-card)] bg-[var(--bg-card)] px-3 py-1.5 font-medium text-[var(--text-secondary)] hover:text-[var(--accent-blue)]",
                sizeClass(tag.article_count),
              )}
            >
              #{tag.name}
              <span className="ml-1 font-mono text-xs tabular-nums text-[var(--text-tertiary)]">
                {tag.article_count}
              </span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

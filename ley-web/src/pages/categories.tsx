import { Link } from "react-router-dom";
import { FolderOpen } from "lucide-react";
import { useCategories, type Category } from "@/hooks/use-categories";
import { Empty } from "@/components/ui/Empty";
import { Skeleton } from "@/components/ui/Skeleton";

function CategoryNode({ category, depth }: { category: Category; depth: number }) {
  return (
    <li>
      <Link
        to={`/articles?category=${category.id}`}
        className="flex items-center justify-between rounded-xl px-3 py-2.5 text-sm transition-colors hover:bg-[var(--bg-hover)]"
        style={{ marginLeft: depth > 0 ? `${depth * 1.25}rem` : undefined }}
      >
        <span className="inline-flex min-w-0 items-center gap-2 text-[var(--text-primary)]">
          <FolderOpen className="size-4 shrink-0 text-[var(--accent)]" />
          <span className="truncate">{category.name}</span>
        </span>
        <span className="ml-3 shrink-0 rounded-lg bg-[var(--bg-hover)] px-2 py-0.5 text-xs text-[var(--text-tertiary)]">
          {category.article_count} 篇
        </span>
      </Link>
      {category.children.length > 0 && (
        <ul className="mt-0.5 space-y-0.5">
          {category.children.map((child) => (
            <CategoryNode key={child.id} category={child} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  );
}

/** 分类页：树形展示全部分类 */
export default function CategoriesPage() {
  const { data, isLoading } = useCategories();

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">分类</h1>
      <p className="mb-8 text-sm text-[var(--text-secondary)]">按主题归档文章</p>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-11" />
          <Skeleton className="h-11" />
          <Skeleton className="h-11" />
        </div>
      ) : !data || data.categories.length === 0 ? (
        <Empty title="暂无分类" />
      ) : (
        <ul className="card divide-y divide-[var(--border-light)] p-2">
          {data.categories.map((c) => (
            <CategoryNode key={c.id} category={c} depth={0} />
          ))}
        </ul>
      )}
    </div>
  );
}

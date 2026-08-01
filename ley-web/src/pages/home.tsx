import { useMemo, useState } from "react";
import { useInfiniteArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { ArticleCard } from "@/components/article/ArticleCard";
import { ArticleListSkeleton } from "@/components/ui/Skeleton";
import { Empty } from "@/components/ui/Empty";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/utils";

/** 首页：文章流 + 分类快捷过滤（无限滚动加载） */
export default function HomePage() {
  const [categoryId, setCategoryId] = useState<number | undefined>(undefined);
  const { data: categories } = useCategories();
  const articles = useInfiniteArticles(categoryId ? { category_id: categoryId } : {});

  const all = useMemo(() => articles.data?.pages.flatMap((p) => p.articles) ?? [], [articles.data]);
  const total = articles.data?.pages[0]?.total ?? 0;

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      {/* 站点标题 */}
      <header className="mb-8 text-center">
        <h1 className="text-3xl font-bold tracking-tight text-[var(--text-primary)]">Ley</h1>
        <p className="mt-2 text-sm text-[var(--text-secondary)]">记录与分享 · 个人博客</p>
      </header>

      {/* 分类过滤 */}
      {categories && categories.categories.length > 0 && (
        <div className="mb-6 flex flex-wrap justify-center gap-2">
          <button
            onClick={() => setCategoryId(undefined)}
            className={cn(
              "rounded-xl px-3 py-1.5 text-sm transition-colors",
              categoryId === undefined
                ? "bg-[var(--accent)] text-white"
                : "bg-[var(--bg-card)] text-[var(--text-secondary)] border border-[var(--border)] hover:text-[var(--text-primary)]",
            )}
          >
            全部
          </button>
          {categories.categories.map((c) => (
            <button
              key={c.id}
              onClick={() => setCategoryId(c.id)}
              className={cn(
                "rounded-xl px-3 py-1.5 text-sm transition-colors",
                categoryId === c.id
                  ? "bg-[var(--accent)] text-white"
                  : "bg-[var(--bg-card)] text-[var(--text-secondary)] border border-[var(--border)] hover:text-[var(--text-primary)]",
              )}
            >
              {c.name}
              <span className="ml-1 text-xs opacity-60">{c.article_count}</span>
            </button>
          ))}
        </div>
      )}

      {/* 文章流 */}
      {articles.isLoading ? (
        <ArticleListSkeleton />
      ) : all.length === 0 ? (
        <Empty title="暂无文章" description="博客还在准备中，敬请期待" />
      ) : (
        <>
          <div className="space-y-4">
            {all.map((a) => (
              <ArticleCard key={a.id} article={a} />
            ))}
          </div>
          {total > all.length && (
            <div className="mt-8 text-center">
              <Button variant="secondary" onClick={() => articles.fetchNextPage()} disabled={!articles.hasNextPage || articles.isFetchingNextPage}>
                {articles.isFetchingNextPage ? "加载中…" : "加载更多"}
              </Button>
            </div>
          )}
          <p className="mt-6 text-center text-xs text-[var(--text-tertiary)]">
            共 {total} 篇文章
          </p>
        </>
      )}
    </div>
  );
}

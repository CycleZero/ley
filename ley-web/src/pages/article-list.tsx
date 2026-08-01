import { useSearchParams } from "react-router-dom";
import { useArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { ArticleCard } from "@/components/article/ArticleCard";
import { ArticleListSkeleton } from "@/components/ui/Skeleton";
import { Empty } from "@/components/ui/Empty";
import { Pagination } from "@/components/ui/Pagination";
import { Select } from "@/components/ui/Input";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 10;

/** 文章列表页：分页 + 分类筛选 */
export default function ArticleListPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, Number(searchParams.get("page") ?? 1) || 1);
  const categoryId = searchParams.get("category") ? Number(searchParams.get("category")) : undefined;

  const { data: categories } = useCategories();
  const { data, isLoading } = useArticles({
    category_id: categoryId,
    page,
    page_size: PAGE_SIZE,
  });

  const setPage = (p: number) => {
    const next = new URLSearchParams(searchParams);
    next.set("page", String(p));
    setSearchParams(next);
    window.scrollTo({ top: 0 });
  };

  const setCategory = (id: number | undefined) => {
    const next = new URLSearchParams(searchParams);
    if (id) next.set("category", String(id));
    else next.delete("category");
    next.delete("page");
    setSearchParams(next);
  };

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">文章</h1>
        {categories && categories.categories.length > 0 && (
          <Select
            className="w-40"
            value={categoryId ?? ""}
            onChange={(e) => setCategory(e.target.value ? Number(e.target.value) : undefined)}
          >
            <option value="">全部分类</option>
            {categories.categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.article_count})
              </option>
            ))}
          </Select>
        )}
      </div>

      {/* 分类 chips（移动端友好） */}
      {categories && categories.categories.length > 0 && (
        <div className="mb-6 flex flex-wrap gap-2 md:hidden">
          <button
            onClick={() => setCategory(undefined)}
            className={cn(
              "rounded-xl px-3 py-1 text-xs transition-colors border",
              categoryId === undefined
                ? "bg-[var(--accent)] text-white border-[var(--accent)]"
                : "bg-[var(--bg-card)] text-[var(--text-secondary)] border-[var(--border)]",
            )}
          >
            全部
          </button>
          {categories.categories.map((c) => (
            <button
              key={c.id}
              onClick={() => setCategory(c.id)}
              className={cn(
                "rounded-xl px-3 py-1 text-xs transition-colors border",
                categoryId === c.id
                  ? "bg-[var(--accent)] text-white border-[var(--accent)]"
                  : "bg-[var(--bg-card)] text-[var(--text-secondary)] border-[var(--border)]",
              )}
            >
              {c.name}
            </button>
          ))}
        </div>
      )}

      {isLoading ? (
        <ArticleListSkeleton />
      ) : !data || data.articles.length === 0 ? (
        <Empty title="暂无文章" description="换个分类看看吧" />
      ) : (
        <>
          <div className="space-y-4">
            {data.articles.map((a) => (
              <ArticleCard key={a.id} article={a} />
            ))}
          </div>
          <div className="mt-8">
            <Pagination page={page} total={data.total} pageSize={PAGE_SIZE} onChange={setPage} />
          </div>
        </>
      )}
    </div>
  );
}

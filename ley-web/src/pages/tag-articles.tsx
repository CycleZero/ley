import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { useArticles } from "@/hooks/use-articles";
import { useTags } from "@/hooks/use-tags";
import { ArticleCard } from "@/components/article/ArticleCard";
import { ArticleListSkeleton } from "@/components/ui/Skeleton";
import { Empty } from "@/components/ui/Empty";

/** 标签聚合页：查看某标签下的全部文章 */
export default function TagArticlesPage() {
  const { name } = useParams<{ name: string }>();
  const { data: tags } = useTags();
  const tag = tags?.tags.find((t) => t.name === name);

  const { data, isLoading } = useArticles({
    tags: tag ? [tag.name] : undefined,
    page_size: 50,
  });

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <Link
        to="/tags"
        className="mb-6 inline-flex items-center gap-1 text-sm text-[var(--text-tertiary)] hover:text-[var(--text-primary)] transition-colors"
      >
        <ArrowLeft className="size-4" /> 返回标签
      </Link>
      <h1 className="mb-8 text-2xl font-bold text-[var(--text-primary)]">
        标签 <span className="text-[var(--accent)]">#{name}</span>
        {data && <span className="ml-2 text-sm font-normal text-[var(--text-tertiary)]">共 {data.total} 篇</span>}
      </h1>

      {isLoading ? (
        <ArticleListSkeleton />
      ) : !data || data.articles.length === 0 ? (
        <Empty title="该标签下暂无文章" />
      ) : (
        <div className="space-y-4">
          {data.articles.map((a) => (
            <ArticleCard key={a.id} article={a} />
          ))}
        </div>
      )}
    </div>
  );
}

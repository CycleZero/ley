import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { useArticles } from "@/hooks/use-articles";
import { useTags } from "@/hooks/use-tags";
import { PageHeader } from "@/components/ui/PageHeader";
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
        className="mb-6 inline-flex items-center gap-1 text-sm text-[var(--text-tertiary)] transition-colors hover:text-[var(--accent)]"
      >
        <ArrowLeft className="size-4" /> 返回标签
      </Link>

      <PageHeader
        kicker="TAG ARCHIVE"
        title={<span className="text-[var(--accent)]">#{name}</span>}
        description={
          data ? (
            <>
              共 <span className="font-mono tabular-nums">{data.total}</span> 篇
            </>
          ) : null
        }
      />

      {isLoading ? (
        <div className="mt-8">
          <ArticleListSkeleton />
        </div>
      ) : !data || data.articles.length === 0 ? (
        <div className="mt-8">
          <Empty title="该标签下暂无文章" />
        </div>
      ) : (
        <div className="mt-8 space-y-4">
          {data.articles.map((a) => (
            <ArticleCard key={a.id} article={a} />
          ))}
        </div>
      )}
    </div>
  );
}

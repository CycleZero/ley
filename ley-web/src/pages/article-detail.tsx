import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Calendar, Eye, Heart } from "lucide-react";
import { useArticle, useLikeArticle, useUnlikeArticle } from "@/hooks/use-articles";
import { useAuthStore } from "@/stores/auth";
import { ArticleContent } from "@/components/article/ArticleContent";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Empty } from "@/components/ui/Empty";
import { Spinner } from "@/components/ui/Spinner";
import { formatDate, formatCount } from "@/lib/utils";

/** 文章详情页（CleanLayout：专注阅读） */
export default function ArticleDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  const { data, isLoading, isError } = useArticle(slug ?? "");
  const like = useLikeArticle();
  const unlike = useUnlikeArticle();

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Spinner className="size-8" />
      </div>
    );
  }

  if (isError || !data?.article) {
    return (
      <Empty
        title="文章不存在"
        description="它可能已被删除或地址有误"
        action={
          <Button variant="secondary" onClick={() => navigate("/articles")}>
            返回文章列表
          </Button>
        }
      />
    );
  }

  const article = data.article;

  const toggleLike = () => {
    if (!user) {
      navigate("/login");
      return;
    }
    if (article.is_liked) unlike.mutate(article.id);
    else like.mutate(article.id);
  };

  return (
    <article>
      {/* 返回 */}
      <Link
        to="/articles"
        className="mb-6 inline-flex items-center gap-1 text-sm text-[var(--text-tertiary)] hover:text-[var(--text-primary)] transition-colors"
      >
        <ArrowLeft className="size-4" /> 返回列表
      </Link>

      {/* 标题区 */}
      <header className="mb-8">
        <h1 className="text-3xl font-bold leading-snug tracking-tight text-[var(--text-primary)]">
          {article.title}
        </h1>
        <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-[var(--text-secondary)]">
          {article.author && (
            <span className="inline-flex items-center gap-2">
              {article.author.avatar ? (
                <img src={article.author.avatar} alt="" className="size-6 rounded-full" />
              ) : (
                <span className="flex size-6 items-center justify-center rounded-full bg-[var(--accent-subtle)] text-xs text-[var(--accent)]">
                  {article.author.username?.charAt(0).toUpperCase()}
                </span>
              )}
              <span className="font-medium">{article.author.username}</span>
            </span>
          )}
          <span className="inline-flex items-center gap-1">
            <Calendar className="size-4" /> {formatDate(article.published_at || article.created_at)}
          </span>
          <span className="inline-flex items-center gap-1">
            <Eye className="size-4" /> {formatCount(article.view_count)} 次浏览
          </span>
        </div>
        {article.category && (
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <Badge variant="accent">{article.category.name}</Badge>
            {article.tags.map((t) => (
              <Badge key={t.id}>#{t.name}</Badge>
            ))}
          </div>
        )}
      </header>

      {/* 封面 */}
      {article.cover_image && (
        <img
          src={article.cover_image}
          alt="封面"
          className="mb-8 w-full rounded-xl border border-[var(--border-light)] object-cover"
        />
      )}

      {/* 正文 */}
      <ArticleContent content={article.content} />

      {/* 点赞 */}
      <div className="mt-12 flex justify-center border-t border-[var(--border-light)] pt-8">
        <Button variant={article.is_liked ? "primary" : "secondary"} onClick={toggleLike} className="min-w-28">
          <Heart className={`size-4 ${article.is_liked ? "fill-current" : ""}`} />
          {article.is_liked ? "已喜欢" : "喜欢"} · {formatCount(article.like_count)}
        </Button>
      </div>
    </article>
  );
}

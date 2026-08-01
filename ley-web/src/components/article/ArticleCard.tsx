import { Link } from "react-router-dom";
import { Eye, Heart, Pin } from "lucide-react";
import type { Article } from "@/hooks/use-articles";
import { formatCount, timeAgo } from "@/lib/utils";
import { Badge } from "@/components/ui/Badge";

/** 文章卡片：首页文章流 / 列表页 / 后台列表共用 */
export function ArticleCard({ article }: { article: Article }) {
  return (
    <Link
      to={`/articles/${article.slug}`}
      className="card group block p-6 transition-colors hover:border-[var(--accent)]/40"
    >
      <div className="flex items-start gap-4">
        {/* 封面图（可选） */}
        {article.cover_image && (
          <img
            src={article.cover_image}
            alt=""
            loading="lazy"
            className="hidden sm:block size-24 rounded-xl object-cover border border-[var(--border-light)] shrink-0"
          />
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {article.is_top && (
              <span className="inline-flex items-center gap-0.5 rounded-lg bg-[var(--accent-subtle)] px-1.5 py-0.5 text-xs text-[var(--accent)]">
                <Pin className="size-3" /> 置顶
              </span>
            )}
            {article.category && (
              <Badge variant="accent">{article.category.name}</Badge>
            )}
          </div>
          <h2 className="mt-2 truncate text-lg font-semibold text-[var(--text-primary)] group-hover:text-[var(--accent)] transition-colors">
            {article.title}
          </h2>
          {article.excerpt && (
            <p className="mt-1.5 line-clamp-2 text-sm leading-relaxed text-[var(--text-secondary)]">
              {article.excerpt}
            </p>
          )}
          <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[var(--text-tertiary)]">
            {article.tags.length > 0 && (
              <span className="flex flex-wrap gap-1.5">
                {article.tags.slice(0, 4).map((t) => (
                  <span key={t.id} className="rounded-lg bg-[var(--bg-hover)] px-1.5 py-0.5">
                    #{t.name}
                  </span>
                ))}
              </span>
            )}
            <span className="ml-auto flex items-center gap-3">
              <span className="inline-flex items-center gap-1">
                <Eye className="size-3.5" /> {formatCount(article.view_count)}
              </span>
              <span className="inline-flex items-center gap-1">
                <Heart className="size-3.5" /> {formatCount(article.like_count)}
              </span>
              <time>{timeAgo(article.published_at || article.created_at)}</time>
            </span>
          </div>
        </div>
      </div>
    </Link>
  );
}

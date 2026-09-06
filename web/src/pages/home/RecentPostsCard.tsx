import { Link } from "react-router-dom";
import { Empty } from "@/components/ui/Empty";
import { Skeleton } from "@/components/ui/Skeleton";
import { useArticles } from "@/hooks/use-articles";
import { BentoCard } from "./BentoCard";

/** 日期格式化为 YYYY.MM.DD（mono 小字） */
function formatDot(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}.${p(d.getMonth() + 1)}.${p(d.getDate())}`;
}

/** 最近文章卡：useArticles({ page_size: 3 }) 前 3 篇，标题 + mono 日期，底部链到 /articles */
export function RecentPostsCard() {
  const { data, isLoading } = useArticles({ page_size: 3 });

  return (
    <BentoCard
      label="最近文章"
      action={
        <Link to="/articles" className="text-[var(--ink-3)] transition-colors hover:text-[var(--accent-blue)]">
          查看全部 →
        </Link>
      }
    >
      {isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      ) : !data || data.articles.length === 0 ? (
        <Empty title="暂无文章" description="写点东西再回来看看" />
      ) : (
        <div className="flex flex-col">
          {data.articles.map((a) => (
            <Link
              key={a.id}
              to={`/articles/${a.slug}`}
              className="group py-[8.5px] [&:not(:last-child)]:border-b [&:not(:last-child)]:border-dashed [&:not(:last-child)]:border-[rgba(59,130,246,0.14)]"
            >
              <div className="truncate text-[13.8px] font-semibold text-[var(--ink)] transition-colors group-hover:text-[var(--accent-blue)]">
                {a.title}
              </div>
              <div className="mt-[3px] font-mono text-[10px] text-[var(--ink-3)]">{formatDot(a.published_at)}</div>
            </Link>
          ))}
        </div>
      )}
    </BentoCard>
  );
}

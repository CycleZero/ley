import { Skeleton } from "@/components/ui/Skeleton";
import { useArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { useTags } from "@/hooks/use-tags";
import { cn } from "@/lib/utils";
import { BentoCard } from "./BentoCard";

const WINDOW = 12; // 近 12 月展示窗口
const FEED_SIZE = 60; // 拉取最近 60 篇用于月份分布（真实数据，非硬编码）

const pad = (n: number) => String(n).padStart(2, "0");

/** 近 12 个月发布量分布：输入文章列表，返回长度 12 的计数数组（旧 → 新） */
function monthlyCounts(articles: { published_at: string }[], now: Date): number[] {
  const buckets = new Map<string, number>();
  for (const a of articles) {
    const t = new Date(a.published_at);
    if (Number.isNaN(t.getTime())) continue;
    const key = `${t.getFullYear()}-${t.getMonth()}`;
    buckets.set(key, (buckets.get(key) ?? 0) + 1);
  }
  const counts = new Array<number>(WINDOW).fill(0);
  for (let i = WINDOW - 1; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    counts[WINDOW - 1 - i] = buckets.get(`${d.getFullYear()}-${d.getMonth()}`) ?? 0;
  }
  return counts;
}

/**
 * 站点统计卡：文章/分类/标签计数 + 近 12 月发布迷你柱状图。
 * 数字为 mono 数据（标签数用 --amber 渐变点缀）；最高柱用 --teal 尾色。
 */
export function StatsCard() {
  const articles = useArticles({ page_size: FEED_SIZE });
  const categories = useCategories();
  const tags = useTags();

  const isLoading = articles.isLoading || categories.isLoading || tags.isLoading;
  const total = articles.data?.total ?? 0;
  const catCount = categories.data?.categories.length ?? 0;
  const tagCount = tags.data?.tags.length ?? 0;
  const counts = monthlyCounts(articles.data?.articles ?? [], new Date());
  const max = Math.max(...counts, 1);
  const hotIndex = counts.reduce((hi, v, i) => (v > counts[hi] ? i : hi), 0); // 最高柱（首现最大）

  return (
    <BentoCard label="站点统计" action="LIVE">
      {isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-9 w-full" />
        </div>
      ) : (
        <>
          <div className="flex gap-[14px]">
            <StatNumber value={total} label="文章" />
            <StatNumber value={catCount} label="分类" />
            <StatNumber value={tagCount} label="标签" gold />
          </div>
          <div className="flex h-9 items-end gap-[5px] pt-[13px]" aria-hidden>
            {counts.map((v, i) => {
              const heightPct = v === 0 ? 4 : Math.max(12, Math.round((v / max) * 88));
              return (
                <span
                  key={i}
                  data-bar-count={v}
                  className={cn(
                    "flex-1 rounded-t-[3px]",
                    i === hotIndex && v > 0
                      ? "shadow-[0_0_10px_rgba(94,234,212,0.55)] opacity-100"
                      : "opacity-55",
                  )}
                  style={{
                    height: `${heightPct}%`,
                    background:
                      i === hotIndex && v > 0
                        ? "linear-gradient(180deg, var(--teal), var(--teal-ink))"
                        : "linear-gradient(180deg, #a9cdf7, var(--blue-soft))",
                  }}
                />
              );
            })}
          </div>
        </>
      )}
    </BentoCard>
  );
}

function StatNumber({ value, label, gold = false }: { value: number; label: string; gold?: boolean }) {
  return (
    <div className="flex-1 text-left">
      <div
        className={cn(
          "bg-clip-text font-mono text-[29px] font-bold leading-none text-transparent",
          "bg-gradient-to-b",
          gold
            ? "from-[var(--amber-deep)] to-[var(--amber)]"
            : "from-[var(--ink)] to-[var(--accent-blue-hover)]",
        )}
      >
        {pad(value)}
      </div>
      <div className="mt-[3px] text-[11px] text-[var(--ink-3)]">{label}</div>
    </div>
  );
}

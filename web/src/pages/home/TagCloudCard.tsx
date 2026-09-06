import { Empty } from "@/components/ui/Empty";
import { Skeleton } from "@/components/ui/Skeleton";
import { useTags } from "@/hooks/use-tags";
import { cn } from "@/lib/utils";
import { BentoCard } from "./BentoCard";

/** 胶囊尺寸分级：按 article_count（0-2 小 / 3-4 中 / 5+ 大） */
function sizeOf(count: number): "sm" | "md" | "lg" {
  if (count >= 5) return "lg";
  if (count >= 3) return "md";
  return "sm";
}

const SIZE_CLASSES: Record<"sm" | "md" | "lg", string> = {
  sm: "px-[10px] py-[4px] text-[10.5px]",
  md: "px-[13px] py-[6px] text-[12px]",
  lg: "px-[14px] py-[7px] text-[13px]",
};

const VARIANT_CLASSES = {
  hot: "border-[rgba(240,179,94,0.45)] bg-[rgba(240,179,94,0.14)] font-semibold text-[var(--amber-deep)]",
  teal: "border-[rgba(94,234,212,0.6)] bg-[rgba(94,234,212,0.12)] text-[var(--teal-ink)]",
  blue: "border-[var(--border)] bg-[var(--bg-card)] text-[var(--ink-2)]",
} as const;

/**
 * 标签云卡：useTags() 渲染标签胶囊 —— 最高计数一个琥珀 hot，其余按计数给薄荷 tint / 蓝色 tint，
 * 尺寸按 article_count 分级。
 */
export function TagCloudCard() {
  const { data, isLoading } = useTags();
  const tags = data?.tags ?? [];

  if (isLoading) {
    return (
      <BentoCard label="标签云">
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-7 w-14" />
          <Skeleton className="h-7 w-20" />
          <Skeleton className="h-7 w-16" />
        </div>
      </BentoCard>
    );
  }

  if (tags.length === 0) {
    return (
      <BentoCard label="标签云">
        <Empty title="暂无标签" description="打上标签便于检索" />
      </BentoCard>
    );
  }

  const sorted = [...tags].sort((a, b) => b.article_count - a.article_count);
  const hotId = sorted[0].id; // 计数最高的一个为 hot

  return (
    <BentoCard label="标签云" action={`${tags.length} TAGS`}>
      <div className="flex flex-wrap gap-2">
        {sorted.map((t) => {
          const size = sizeOf(t.article_count);
          const variant = t.id === hotId ? "hot" : t.article_count >= 3 ? "teal" : "blue";
          return (
            <span
              key={t.id}
              data-tag-variant={variant}
              data-tag-size={size}
              title={`${t.article_count} 篇文章`}
              className={cn(
                "rounded-full border transition-all duration-200 hover:-translate-y-[2px]",
                SIZE_CLASSES[size],
                VARIANT_CLASSES[variant],
                variant === "blue" &&
                  "hover:border-[rgba(59,130,246,0.4)] hover:text-[var(--accent-blue)] hover:shadow-[0_0_12px_rgba(59,130,246,0.24)]",
              )}
            >
              {t.name}
            </span>
          );
        })}
      </div>
    </BentoCard>
  );
}

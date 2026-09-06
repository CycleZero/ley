import { useSiteConfig } from "@/hooks/use-site";
import { cn } from "@/lib/utils";

/**
 * 首页 Hero（纯展示）：大标题 + 英文 kicker + 副标题 + 底部刻度尺。
 * 站点名/副标永远来自 useSiteConfig()（禁止硬编码站名）；
 * 标题为空或加载缺失时降级为中性占位「我的博客」，副标为空时用中性占位。
 * 不包含任何登录/交互逻辑；唯一装饰为 hero 底部刻度尺（spec §4.3）。
 * 布局对齐 docs/frontend-redesign-spec.md §4.1 上半留白：padding-top 88px、margin-bottom 48px。
 */

export interface HomeHeroProps {
  className?: string;
}

const FALLBACK_TITLE = "我的博客";
const FALLBACK_SUBTITLE = "记录 · 思考 · 分享";
const KICKER = "PERSONAL BLOG";

export function HomeHero({ className }: HomeHeroProps) {
  const { data } = useSiteConfig();
  const config = data?.config;
  const title = (config?.site_title ?? "").trim() || FALLBACK_TITLE;
  const subtitle = (config?.site_subtitle ?? "").trim() || FALLBACK_SUBTITLE;

  return (
    <section className={cn("relative mb-12 px-2.5 pb-[26px] pt-[88px]", className)}>
      {/* 英文 kicker：mono 小字 + 0.1em 字母间距 + 弱文字色 */}
      <p className="mb-[18px] font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--ink-3)]">
        {KICKER}
      </p>

      {/* 主标题：深蓝→主蓝 180° 渐变 clip 文字 + 轻 drop-shadow */}
      <h1 className="break-words text-[clamp(44px,6vw,72px)] font-extrabold leading-[1.04] tracking-[0.06em] text-transparent">
        <span className="bg-[linear-gradient(180deg,#123a6b_12%,#3b82f6_88%)] bg-clip-text drop-shadow-[0_4px_18px_rgba(125,180,240,0.38)]">
          {title}
        </span>
      </h1>

      {/* 副标题：ink-secondary */}
      <p className="mt-3.5 text-[15.5px] leading-[1.7] text-[var(--ink-2)]">{subtitle}</p>

      {/* 刻度尺（唯一装饰）：细线 + 等距主/次刻度（v4 .ruler，无 LAT 伪文本） */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-2.5 -bottom-1.5 h-3 opacity-60"
      >
        <div className="h-[6px] w-full bg-[repeating-linear-gradient(90deg,rgba(59,130,246,0.4)_0_1px,transparent_1px_28px)] bg-no-repeat bg-top" />
        <div className="h-[3.5px] w-full bg-[repeating-linear-gradient(90deg,rgba(59,130,246,0.22)_0_1px,transparent_1px_7px)] bg-no-repeat bg-top" />
      </div>
    </section>
  );
}

export default HomeHero;

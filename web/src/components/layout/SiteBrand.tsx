import { Link } from "react-router-dom";
import { useSiteConfig } from "@/hooks/use-site";
import { cn } from "@/lib/utils";

/**
 * 站点品牌：标题永远来自 useSiteConfig()（禁止硬编码站名）。
 * loading / 无数据 / 空标题时渲染中性占位「我的博客」。
 * subtitle 开启且配置存在 site_subtitle 时，附带 mono 小副标。
 */

export interface SiteBrandProps {
  /** 是否渲染 mono 小副标（site_subtitle） */
  subtitle?: boolean;
  className?: string;
}

const FALLBACK_TITLE = "我的博客";

export function SiteBrand({ subtitle = false, className }: SiteBrandProps) {
  const { data } = useSiteConfig();
  const config = data?.config;
  const title = (config?.site_title ?? "").trim() || FALLBACK_TITLE;

  return (
    <Link
      to="/"
      className={cn("inline-flex items-baseline gap-2 text-[var(--ink)]", className)}
    >
      <span className="text-lg font-semibold leading-none tracking-[0.02em]">{title}</span>
      {subtitle && config?.site_subtitle ? (
        <span className="font-mono text-[9px] uppercase tracking-[0.3em] text-[var(--ink-3)]">
          {config.site_subtitle}
        </span>
      ) : null}
    </Link>
  );
}

export default SiteBrand;

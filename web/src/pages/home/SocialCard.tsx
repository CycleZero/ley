import type { ReactNode } from "react";
import { Skeleton } from "@/components/ui/Skeleton";
import { useSiteConfig } from "@/hooks/use-site";
import { BentoCard } from "./BentoCard";

type SocialKind = "github" | "twitter" | "email";

interface SocialLink {
  key: SocialKind;
  label: string;
  href: string;
  external: boolean;
  icon: ReactNode;
}

/** 配置值可能是裸用户名/URL；规范化成可点击地址 */
function hrefFor(kind: SocialKind, value: string): string {
  const v = value.trim();
  if (/^https?:\/\//i.test(v)) return v;
  if (kind === "github") return `https://github.com/${v}`;
  if (kind === "twitter") return `https://x.com/${v}`;
  return `mailto:${v}`;
}

const GITHUB_ICON = (
  <svg viewBox="0 0 16 16" className="size-[26px] fill-current" aria-hidden>
    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8z" />
  </svg>
);

const TWITTER_ICON = (
  <svg viewBox="0 0 24 24" className="size-[26px] fill-current" aria-hidden>
    <path d="M23 4.9c-.8.4-1.7.6-2.6.8.9-.6 1.6-1.5 2-2.6-.9.5-1.9.9-2.9 1.1-.9-1-2.2-1.6-3.6-1.6-2.7 0-4.9 2.2-4.9 4.9 0 .4 0 .8.1 1.1-4.1-.2-7.7-2.2-10.1-5.2-.4.7-.7 1.5-.7 2.5 0 1.7.9 3.2 2.2 4.1-.8 0-1.6-.2-2.2-.6v.1c0 2.4 1.7 4.3 3.9 4.8-.4.1-.9.2-1.3.2-.3 0-.6 0-.9-.1.6 2 2.4 3.4 4.6 3.4-1.7 1.3-3.8 2.1-6.1 2.1-.4 0-.8 0-1.2-.1 2.2 1.4 4.8 2.2 7.5 2.2 9.1 0 14-7.5 14-14v-.6c1-.7 1.8-1.6 2.5-2.6z" />
  </svg>
);

const MAIL_ICON = (
  <svg viewBox="0 0 24 24" className="size-[26px]" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <rect x="3.4" y="4.6" width="17.2" height="14.8" rx="3" />
    <path d="m4.6 7 7.4 5.6L19.4 7" />
  </svg>
);

function collectLinks(config: {
  social_github: string;
  social_twitter: string;
  social_email: string;
}): SocialLink[] {
  const links: SocialLink[] = [];
  if (config.social_github?.trim()) {
    links.push({ key: "github", label: "GitHub", href: hrefFor("github", config.social_github), external: true, icon: GITHUB_ICON });
  }
  if (config.social_twitter?.trim()) {
    links.push({ key: "twitter", label: "Twitter", href: hrefFor("twitter", config.social_twitter), external: true, icon: TWITTER_ICON });
  }
  if (config.social_email?.trim()) {
    links.push({ key: "email", label: "邮件", href: hrefFor("email", config.social_email), external: false, icon: MAIL_ICON });
  }
  return links;
}

/**
 * 社交卡：useSiteConfig() 读取 social_github/twitter/email —— 只渲染配置提供的链接，
 * 无任何配置时不渲染 fake 链接（整卡隐藏）。白色圆角块 + hover 蓝光。
 */
export function SocialCard() {
  const { data, isLoading } = useSiteConfig();
  const cfg = data?.config;
  const links = cfg ? collectLinks(cfg) : [];

  if (isLoading) {
    return (
      <BentoCard label="找到我">
        <div className="flex gap-2">
          <Skeleton className="aspect-square flex-1 rounded-[14px]" />
          <Skeleton className="aspect-square flex-1 rounded-[14px]" />
          <Skeleton className="aspect-square flex-1 rounded-[14px]" />
        </div>
      </BentoCard>
    );
  }

  if (links.length === 0) return null; // 配置未提供任何社交 → 整卡隐藏，不出 fake 链接

  return (
    <BentoCard label="找到我" action={`${String(links.length).padStart(2, "0")}`}>
      <div className="grid grid-cols-3 gap-[9px]">
        {links.map((l) => (
          <a
            key={l.key}
            href={l.href}
            title={l.label}
            aria-label={l.label}
            {...(l.external ? { target: "_blank", rel: "noreferrer" } : {})}
            className="grid aspect-square place-items-center rounded-[14px] border border-[var(--border)] bg-[var(--bg-card)] text-[var(--ink-2)] shadow-[inset_0_1px_0_rgba(255,255,255,0.9),0_2px_8px_rgba(50,80,140,0.06)] transition-all duration-200 hover:-translate-y-[3px] hover:text-[var(--accent-blue)] hover:shadow-[0_0_18px_rgba(59,130,246,0.4),0_8px_18px_rgba(70,110,180,0.14)]"
          >
            {l.icon}
          </a>
        ))}
      </div>
    </BentoCard>
  );
}

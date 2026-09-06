import type { ReactNode } from "react";
import { useSiteConfig, type SiteConfig } from "@/hooks/use-site";
import { PageHeader } from "@/components/ui/PageHeader";
import { Card, CardHeader, CardTitle } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";

/* ---------- 内联 SVG 社交图标（描边/填充走 currentColor） ---------- */

function GitHubIcon() {
  return (
    <svg viewBox="0 0 16 16" className="size-[22px] fill-current" aria-hidden="true">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8z" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-[22px] fill-current" aria-hidden="true">
      <path d="M18.9 1.15h3.68l-8.04 9.19L24 22.85h-7.41l-5.8-7.58-6.64 7.58H.47l8.6-9.83L0 1.15h7.59l5.24 6.93zm-1.29 19.5h2.04L6.48 3.24H4.3z" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-[22px] stroke-current" fill="none" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect width="20" height="16" x="2" y="4" rx="3" />
      <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
    </svg>
  );
}

function RssIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-[22px] stroke-current" fill="none" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
      <path d="M4 11a9 9 0 0 1 9 9" />
      <path d="M4 4a16 16 0 0 1 16 16" />
      <circle cx="5" cy="19" r="1" />
    </svg>
  );
}

interface SocialItem {
  label: string;
  href: string;
  external?: boolean;
  icon: ReactNode;
}

function normalizeSocial(raw: string, prefix: string): string {
  return raw.startsWith("http") ? raw : `${prefix}${raw}`;
}

function buildSocials(config?: SiteConfig): SocialItem[] {
  const items: SocialItem[] = [];
  if (config?.social_github) {
    items.push({
      label: "GitHub",
      href: normalizeSocial(config.social_github, "https://github.com/"),
      external: true,
      icon: <GitHubIcon />,
    });
  }
  if (config?.social_twitter) {
    items.push({
      label: "X",
      href: normalizeSocial(config.social_twitter, "https://x.com/"),
      external: true,
      icon: <XIcon />,
    });
  }
  if (config?.social_email) {
    items.push({ label: "邮箱", href: `mailto:${config.social_email}`, icon: <MailIcon /> });
  }
  // RSS：/rss 为死链接（SPA 无此路由），按任务要求仅样式化保留，路由修复交由后续任务
  items.push({ label: "RSS", href: "/rss", icon: <RssIcon /> });
  return items;
}

/** 关于页：站点信息 + 联系方式 */
export default function AboutPage() {
  const { data, isLoading } = useSiteConfig();
  const config = data?.config;
  const siteName = config?.site_title || "未命名站点";
  const socials = buildSocials(config);

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <PageHeader kicker="ABOUT" title="关于" className="mb-8" />

      <div className="space-y-6">
        {/* 站点信息卡 */}
        <Card className="card-hover p-6">
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="mx-auto size-[46px] rounded-full" />
              <Skeleton className="mx-auto h-6 w-32" />
              <Skeleton className="mx-auto h-4 w-2/3" />
              <Skeleton className="mx-auto h-4 w-1/2" />
            </div>
          ) : (
            <div className="flex flex-col items-center text-center">
              <div className="avatar-grad grid size-[46px] place-items-center rounded-full text-lg font-extrabold text-white shadow-[0_0_0_3px_rgba(255,255,255,0.95),0_0_20px_rgba(167,139,250,0.4)]">
                {siteName.charAt(0).toUpperCase()}
              </div>
              <h2 className="mt-3 font-mono text-lg font-semibold text-[var(--ink)]">{siteName}</h2>
              {config?.site_subtitle && (
                <p className="mt-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--ink-3)]">
                  {config.site_subtitle}
                </p>
              )}
              <p className="mt-4 max-w-lg text-sm leading-relaxed text-[var(--ink-2)]">
                {config?.site_description || "一个用 Go 和 React 构建的个人博客平台。"}
              </p>
            </div>
          )}
        </Card>

        {/* 联系方式卡 */}
        <Card className="card-hover p-6">
          <CardHeader>
            <CardTitle>联系方式</CardTitle>
          </CardHeader>
          <div className="flex flex-wrap items-center gap-2.5">
            {socials.map((item) => (
              <a
                key={item.label}
                href={item.href}
                aria-label={item.label}
                title={item.label}
                target={item.external ? "_blank" : undefined}
                rel={item.external ? "noreferrer" : undefined}
                className="grid size-11 place-items-center rounded-xl border border-[var(--border-card)] bg-[var(--bg-card)] text-[var(--ink-2)] shadow-[0_2px_8px_rgba(50,80,140,0.06)] transition-all duration-200 hover:-translate-y-[3px] hover:text-[var(--accent)] hover:shadow-[0_0_18px_rgba(59,130,246,0.35),0_8px_18px_rgba(70,110,180,0.14)]"
              >
                {item.icon}
              </a>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

import { ExternalLink, Mail, Rss } from "lucide-react";
import { useSiteConfig } from "@/hooks/use-site";
import { Card, CardBody } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";

/** 关于页：站点介绍 + 社交链接 */
export default function AboutPage() {
  const { data, isLoading } = useSiteConfig();
  const config = data?.config;

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <h1 className="mb-8 text-2xl font-bold text-[var(--text-primary)]">关于</h1>

      <Card className="p-8">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-6 w-1/3" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
          </div>
        ) : (
          <>
            <h2 className="text-xl font-semibold text-[var(--text-primary)]">
              {config?.site_title || "Ley"}
            </h2>
            {config?.site_subtitle && (
              <p className="mt-1 text-sm text-[var(--text-secondary)]">{config.site_subtitle}</p>
            )}
            <div className="mt-6 text-sm leading-relaxed text-[var(--text-secondary)]">
              <p>{config?.site_description || "一个用 Go 和 React 构建的个人博客平台。"}</p>
            </div>
          </>
        )}
      </Card>

      <Card className="mt-6 p-8">
        <CardBody>
          <h3 className="text-base font-semibold text-[var(--text-primary)]">联系方式</h3>
          <div className="flex flex-wrap gap-2">
            {config?.social_github && (
              <a
                href={config.social_github.startsWith("http") ? config.social_github : `https://github.com/${config.social_github}`}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 rounded-xl bg-[var(--bg-hover)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--border)] hover:text-[var(--text-primary)]"
              >
                <ExternalLink className="size-4" /> GitHub
              </a>
            )}
            {config?.social_twitter && (
              <a
                href={config.social_twitter.startsWith("http") ? config.social_twitter : `https://x.com/${config.social_twitter}`}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 rounded-xl bg-[var(--bg-hover)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--border)] hover:text-[var(--text-primary)]"
              >
                <ExternalLink className="size-4" /> Twitter / X
              </a>
            )}
            {config?.social_email && (
              <a
                href={`mailto:${config.social_email}`}
                className="inline-flex items-center gap-2 rounded-xl bg-[var(--bg-hover)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--border)] hover:text-[var(--text-primary)]"
              >
                <Mail className="size-4" /> {config.social_email}
              </a>
            )}
            <a
              href="/rss"
              className="inline-flex items-center gap-2 rounded-xl bg-[var(--bg-hover)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--border)] hover:text-[var(--text-primary)]"
            >
              <Rss className="size-4" /> RSS
            </a>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}

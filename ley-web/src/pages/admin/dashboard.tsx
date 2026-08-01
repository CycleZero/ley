import { Link } from "react-router-dom";
import { Archive, FileText, FolderTree, Tags, Eye, Heart, PenLine } from "lucide-react";
import { useArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { useTags } from "@/hooks/use-tags";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatDate, formatCount } from "@/lib/utils";

function StatCard({
  label,
  value,
  icon: Icon,
  loading,
  to,
}: {
  label: string;
  value: number;
  icon: React.ElementType;
  loading?: boolean;
  to?: string;
}) {
  const body = (
    <Card className="p-5 transition-colors hover:border-[var(--accent)]/40">
      <div className="flex items-center justify-between">
        <span className="text-sm text-[var(--text-secondary)]">{label}</span>
        <span className="flex size-8 items-center justify-center rounded-xl bg-[var(--accent-subtle)] text-[var(--accent)]">
          <Icon className="size-4" />
        </span>
      </div>
      {loading ? (
        <Skeleton className="mt-2 h-8 w-16" />
      ) : (
        <p className="mt-1 text-2xl font-bold text-[var(--text-primary)]">{value}</p>
      )}
    </Card>
  );
  return to ? <Link to={to}>{body}</Link> : body;
}

const statusMeta: Record<string, { label: string; variant: "success" | "warning" | "default" }> = {
  published: { label: "已发布", variant: "success" },
  draft: { label: "草稿", variant: "warning" },
  archived: { label: "已归档", variant: "default" },
};

/** 后台仪表盘：内容统计 + 最近文章 */
export default function AdminDashboard() {
  const all = useArticles({ page_size: 1 });
  const published = useArticles({ status: "published", page_size: 1 });
  const drafts = useArticles({ status: "draft", page_size: 1 });
  const archived = useArticles({ status: "archived", page_size: 1 });
  const recent = useArticles({ page_size: 5 });
  const categories = useCategories();
  const tags = useTags();

  const articles = recent.data?.articles ?? [];

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">仪表盘</h1>
        <Link
          to="/admin/articles/new"
          className="inline-flex h-9 items-center gap-1.5 rounded-xl bg-[var(--accent)] px-4 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-hover)]"
        >
          <PenLine className="size-4" /> 写文章
        </Link>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
        <StatCard label="全部文章" value={all.data?.total ?? 0} icon={FileText} loading={all.isLoading} to="/admin/articles" />
        <StatCard label="已发布" value={published.data?.total ?? 0} icon={FileText} loading={published.isLoading} />
        <StatCard label="草稿" value={drafts.data?.total ?? 0} icon={PenLine} loading={drafts.isLoading} />
        <StatCard label="已归档" value={archived.data?.total ?? 0} icon={Archive} loading={archived.isLoading} />
        <StatCard label="分类" value={categories.data?.categories.length ?? 0} icon={FolderTree} loading={categories.isLoading} to="/admin/categories" />
        <StatCard label="标签" value={tags.data?.tags.length ?? 0} icon={Tags} loading={tags.isLoading} to="/admin/tags" />
      </div>

      {/* 最近文章 */}
      <Card className="mt-6 overflow-hidden">
        <div className="border-b border-[var(--border)] px-6 py-4">
          <h2 className="text-base font-semibold text-[var(--text-primary)]">最近文章</h2>
        </div>
        {recent.isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
        ) : articles.length === 0 ? (
          <p className="p-6 text-sm text-[var(--text-tertiary)]">还没有文章，点击右上角开始写作吧。</p>
        ) : (
          <ul className="divide-y divide-[var(--border-light)]">
            {articles.map((a) => (
              <li key={a.id} className="flex items-center gap-4 px-6 py-3.5">
                <div className="min-w-0 flex-1">
                  <Link to={`/admin/articles/edit/${a.id}`} className="truncate text-sm font-medium text-[var(--text-primary)] hover:text-[var(--accent)] transition-colors">
                    {a.title}
                  </Link>
                  <p className="mt-0.5 text-xs text-[var(--text-tertiary)]">
                    {formatDate(a.published_at || a.created_at)}
                  </p>
                </div>
                <span className="hidden items-center gap-1 text-xs text-[var(--text-tertiary)] sm:inline-flex">
                  <Eye className="size-3.5" /> {formatCount(a.view_count)}
                </span>
                <span className="hidden items-center gap-1 text-xs text-[var(--text-tertiary)] sm:inline-flex">
                  <Heart className="size-3.5" /> {formatCount(a.like_count)}
                </span>
                <Badge variant={statusMeta[a.status]?.variant ?? "default"}>
                  {statusMeta[a.status]?.label ?? a.status}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}

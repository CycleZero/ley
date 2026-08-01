import { useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { Archive, Eye, PenLine, Send, Trash2 } from "lucide-react";
import {
  useArticles,
  useArchiveArticle,
  useDeleteArticle,
  usePublishArticle,
  type Article,
} from "@/hooks/use-articles";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { Empty } from "@/components/ui/Empty";
import { Modal } from "@/components/ui/Modal";
import { Pagination } from "@/components/ui/Pagination";
import { Select } from "@/components/ui/Input";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatDate, formatCount } from "@/lib/utils";

const PAGE_SIZE = 10;

const statusMeta: Record<string, { label: string; variant: "success" | "warning" | "default" | "danger" }> = {
  published: { label: "已发布", variant: "success" },
  draft: { label: "草稿", variant: "warning" },
  archived: { label: "已归档", variant: "default" },
};

/** 后台文章管理：筛选 / 分页 / 发布 / 归档 / 删除 */
export default function ArticleManage() {
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<string>("");
  const [deleting, setDeleting] = useState<Article | null>(null);

  const { data, isLoading } = useArticles({
    status: status || undefined,
    page,
    page_size: PAGE_SIZE,
  });
  const publish = usePublishArticle();
  const archive = useArchiveArticle();
  const del = useDeleteArticle();

  const onPublish = (a: Article) => {
    publish.mutate(a.id, {
      onSuccess: () => toast.success(`《${a.title}》已发布`),
      onError: (e) => toast.error(e instanceof Error ? e.message : "发布失败"),
    });
  };
  const onArchive = (a: Article) => {
    archive.mutate(a.id, {
      onSuccess: () => toast.success(`《${a.title}》已归档`),
      onError: (e) => toast.error(e instanceof Error ? e.message : "归档失败"),
    });
  };
  const onDelete = () => {
    if (!deleting) return;
    del.mutate(deleting.id, {
      onSuccess: () => {
        toast.success("已删除");
        setDeleting(null);
      },
      onError: (e) => toast.error(e instanceof Error ? e.message : "删除失败"),
    });
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">文章管理</h1>
        <div className="flex items-center gap-2">
          <Select className="w-32" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1); }}>
            <option value="">全部状态</option>
            <option value="published">已发布</option>
            <option value="draft">草稿</option>
            <option value="archived">已归档</option>
          </Select>
          <Link
            to="/admin/articles/new"
            className="inline-flex h-9 items-center gap-1.5 rounded-xl bg-[var(--accent)] px-4 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-hover)]"
          >
            <PenLine className="size-4" /> 写文章
          </Link>
        </div>
      </div>

      <Card className="overflow-hidden">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
          </div>
        ) : !data || data.articles.length === 0 ? (
          <Empty title="暂无文章" description="点击右上角写文章" />
        ) : (
          <>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)] text-left text-xs text-[var(--text-tertiary)]">
                  <th className="px-6 py-3 font-medium">标题</th>
                  <th className="hidden px-4 py-3 font-medium md:table-cell">分类</th>
                  <th className="hidden px-4 py-3 font-medium sm:table-cell">浏览</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="hidden px-4 py-3 font-medium lg:table-cell">发布时间</th>
                  <th className="px-6 py-3 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--border-light)]">
                {data.articles.map((a) => (
                  <tr key={a.id} className="transition-colors hover:bg-[var(--bg-hover)]/60">
                    <td className="max-w-64 px-6 py-3.5">
                      <Link
                        to={`/admin/articles/edit/${a.id}`}
                        className="block truncate font-medium text-[var(--text-primary)] hover:text-[var(--accent)] transition-colors"
                        title={a.title}
                      >
                        {a.title}
                      </Link>
                    </td>
                    <td className="hidden px-4 py-3.5 text-[var(--text-secondary)] md:table-cell">
                      {a.category?.name ?? "—"}
                    </td>
                    <td className="hidden px-4 py-3.5 text-[var(--text-secondary)] sm:table-cell">
                      {formatCount(a.view_count)}
                    </td>
                    <td className="px-4 py-3.5">
                      <Badge variant={statusMeta[a.status]?.variant ?? "default"}>
                        {statusMeta[a.status]?.label ?? a.status}
                      </Badge>
                    </td>
                    <td className="hidden px-4 py-3.5 text-[var(--text-secondary)] lg:table-cell">
                      {formatDate(a.published_at || a.created_at)}
                    </td>
                    <td className="px-6 py-3.5">
                      <div className="flex justify-end gap-1">
                        <Link
                          to={`/articles/${a.slug}`}
                          className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                          title="查看"
                        >
                          <Eye className="size-4" />
                        </Link>
                        <Link
                          to={`/admin/articles/edit/${a.id}`}
                          className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                          title="编辑"
                        >
                          <PenLine className="size-4" />
                        </Link>
                        {a.status === "draft" && (
                          <button
                            onClick={() => onPublish(a)}
                            className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-[var(--accent-subtle)] hover:text-[var(--accent)]"
                            title="发布"
                          >
                            <Send className="size-4" />
                          </button>
                        )}
                        {a.status === "published" && (
                          <button
                            onClick={() => onArchive(a)}
                            className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                            title="归档"
                          >
                            <Archive className="size-4" />
                          </button>
                        )}
                        <button
                          onClick={() => setDeleting(a)}
                          className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-red-50 hover:text-red-500"
                          title="删除"
                        >
                          <Trash2 className="size-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="border-t border-[var(--border-light)] py-4">
              <Pagination page={page} total={data.total} pageSize={PAGE_SIZE} onChange={setPage} />
            </div>
          </>
        )}
      </Card>

      {/* 删除确认 */}
      <Modal
        open={!!deleting}
        onClose={() => setDeleting(null)}
        title="删除文章"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleting(null)}>
              取消
            </Button>
            <Button variant="danger" onClick={onDelete} disabled={del.isPending}>
              {del.isPending ? "删除中…" : "确认删除"}
            </Button>
          </>
        }
      >
        <p className="text-sm leading-relaxed text-[var(--text-secondary)]">
          确定要删除 <span className="font-medium text-[var(--text-primary)]">《{deleting?.title}》</span> 吗？
          删除后文章将不可恢复。
        </p>
      </Modal>
    </div>
  );
}

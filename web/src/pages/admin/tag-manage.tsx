import { useState } from "react";
import { toast } from "sonner";
import { Plus, Tags as TagsIcon, Trash2 } from "lucide-react";
import { useCreateTag, useDeleteTag, useTags } from "@/hooks/use-tags";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { Empty } from "@/components/ui/Empty";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Skeleton } from "@/components/ui/Skeleton";
import type { Tag } from "@/hooks/use-tags";

/** 后台标签管理：新增 / 删除 */
export default function TagManage() {
  const { data, isLoading } = useTags();
  const create = useCreateTag();
  const del = useDeleteTag();

  const [name, setName] = useState("");
  const [deleting, setDeleting] = useState<Tag | null>(null);

  const onCreate = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      toast.error("请输入标签名称");
      return;
    }
    try {
      await create.mutateAsync(trimmed);
      toast.success(`标签 #${trimmed} 已创建`);
      setName("");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "创建失败");
    }
  };

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await del.mutateAsync(deleting.id);
      toast.success("标签已删除");
      setDeleting(null);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "删除失败");
    }
  };

  return (
    <div>
      <PageHeader
        kicker="TAG MANAGE"
        title="标签管理"
        className="mb-6"
        actions={
          <div className="flex w-full max-w-sm items-center gap-2">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && onCreate()}
              placeholder="新标签名称，回车或点击添加"
            />
            <Button onClick={onCreate} disabled={create.isPending}>
              <Plus className="size-4" /> 添加
            </Button>
          </div>
        }
      />

      <Card className="overflow-hidden">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
          </div>
        ) : !data || data.tags.length === 0 ? (
          <Empty title="暂无标签" description="在上方输入名称创建标签" />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)] text-left text-xs text-[var(--text-tertiary)]">
                <th className="px-6 py-3 font-medium">名称</th>
                <th className="hidden px-4 py-3 font-medium md:table-cell">Slug</th>
                <th className="px-4 py-3 font-medium">文章数</th>
                <th className="px-6 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-light)]">
              {data.tags.map((tag) => (
                <tr key={tag.id} className="transition-colors hover:bg-[var(--bg-hover)]/60">
                  <td className="px-6 py-3.5">
                    <Badge className="rounded-full">
                      <TagsIcon className="size-3" /> #{tag.name}
                    </Badge>
                  </td>
                  <td className="hidden px-4 py-3.5 font-mono text-xs text-[var(--text-secondary)] md:table-cell">
                    {tag.slug}
                  </td>
                  <td className="px-4 py-3.5 font-mono tabular-nums text-[var(--text-secondary)]">
                    {tag.article_count}
                  </td>
                  <td className="px-6 py-3.5">
                    <div className="flex justify-end">
                      <button
                        onClick={() => setDeleting(tag)}
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
        )}
      </Card>

      <Modal
        open={!!deleting}
        onClose={() => setDeleting(null)}
        title="删除标签"
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
          确定要删除标签 <span className="font-medium text-[var(--text-primary)]">#{deleting?.name}</span> 吗？
          删除标签不会删除关联的文章。
        </p>
      </Modal>
    </div>
  );
}

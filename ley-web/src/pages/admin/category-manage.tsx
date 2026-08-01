import { useState } from "react";
import { toast } from "sonner";
import { FolderTree, Pencil, Plus, Trash2 } from "lucide-react";
import {
  useCategories,
  useCreateCategory,
  useDeleteCategory,
  useUpdateCategory,
  type Category,
} from "@/hooks/use-categories";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Empty } from "@/components/ui/Empty";
import { Field, Input, Select, Textarea } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Skeleton } from "@/components/ui/Skeleton";

interface FormState {
  id?: number;
  name: string;
  slug: string;
  description: string;
  parent_id: string;
  sort_order: string;
}

const emptyForm: FormState = { name: "", slug: "", description: "", parent_id: "", sort_order: "0" };

function flatten(categories: Category[]): Category[] {
  return categories.flatMap((c) => [c, ...flatten(c.children)]);
}

/** 后台分类管理：树形展示 + 增删改 */
export default function CategoryManage() {
  const { data, isLoading } = useCategories();
  const create = useCreateCategory();
  const update = useUpdateCategory();
  const del = useDeleteCategory();

  const [formOpen, setFormOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [deleting, setDeleting] = useState<Category | null>(null);

  const all = data?.categories ?? [];
  const flat = flatten(all);

  const openCreate = () => {
    setForm(emptyForm);
    setFormOpen(true);
  };
  const openEdit = (c: Category) => {
    setForm({
      id: c.id,
      name: c.name,
      slug: c.slug,
      description: c.description,
      parent_id: c.parent_id ? String(c.parent_id) : "",
      sort_order: String(c.sort_order),
    });
    setFormOpen(true);
  };

  const onSubmit = async () => {
    if (!form.name.trim()) {
      toast.error("请填写分类名称");
      return;
    }
    const payload = {
      name: form.name.trim(),
      slug: form.slug.trim() || undefined,
      description: form.description.trim() || undefined,
      parent_id: form.parent_id ? Number(form.parent_id) : undefined,
      sort_order: form.sort_order ? Number(form.sort_order) : 0,
    };
    try {
      if (form.id) {
        await update.mutateAsync({ id: form.id, ...payload });
        toast.success("分类已更新");
      } else {
        await create.mutateAsync(payload);
        toast.success("分类已创建");
      }
      setFormOpen(false);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "保存失败");
    }
  };

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await del.mutateAsync(deleting.id);
      toast.success("分类已删除");
      setDeleting(null);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "删除失败");
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">分类管理</h1>
        <Button onClick={openCreate}>
          <Plus className="size-4" /> 新建分类
        </Button>
      </div>

      <Card className="overflow-hidden">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
          </div>
        ) : flat.length === 0 ? (
          <Empty title="暂无分类" description="创建第一个分类开始整理文章" />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)] text-left text-xs text-[var(--text-tertiary)]">
                <th className="px-6 py-3 font-medium">名称</th>
                <th className="hidden px-4 py-3 font-medium md:table-cell">Slug</th>
                <th className="hidden px-4 py-3 font-medium sm:table-cell">排序</th>
                <th className="px-4 py-3 font-medium">文章数</th>
                <th className="px-6 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-light)]">
              {flat.map((c) => (
                <tr key={c.id} className="transition-colors hover:bg-[var(--bg-hover)]/60">
                  <td className="px-6 py-3.5">
                    <span className="inline-flex items-center gap-2 font-medium text-[var(--text-primary)]">
                      <FolderTree className="size-4 shrink-0 text-[var(--accent)]" />
                      {c.name}
                      {c.children.length > 0 && (
                        <span className="rounded-lg bg-[var(--bg-hover)] px-1.5 py-0.5 text-xs text-[var(--text-tertiary)]">
                          {c.children.length} 个子类
                        </span>
                      )}
                    </span>
                    {c.description && (
                      <span className="ml-2 hidden truncate text-xs text-[var(--text-tertiary)] lg:inline">
                        {c.description}
                      </span>
                    )}
                  </td>
                  <td className="hidden px-4 py-3.5 text-[var(--text-secondary)] md:table-cell">{c.slug}</td>
                  <td className="hidden px-4 py-3.5 text-[var(--text-secondary)] sm:table-cell">{c.sort_order}</td>
                  <td className="px-4 py-3.5 text-[var(--text-secondary)]">{c.article_count}</td>
                  <td className="px-6 py-3.5">
                    <div className="flex justify-end gap-1">
                      <button
                        onClick={() => openEdit(c)}
                        className="inline-flex size-8 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                        title="编辑"
                      >
                        <Pencil className="size-4" />
                      </button>
                      <button
                        onClick={() => setDeleting(c)}
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

      {/* 新建 / 编辑 */}
      <Modal
        open={formOpen}
        onClose={() => setFormOpen(false)}
        title={form.id ? "编辑分类" : "新建分类"}
        footer={
          <>
            <Button variant="secondary" onClick={() => setFormOpen(false)}>
              取消
            </Button>
            <Button onClick={onSubmit} disabled={create.isPending || update.isPending}>
              {create.isPending || update.isPending ? "保存中…" : "保存"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Field label="名称" error={form.name ? undefined : "必填"}>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="如：技术分享" />
          </Field>
          <Field label="Slug" hint="留空自动生成">
            <Input value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} placeholder="tech" />
          </Field>
          <Field label="父分类">
            <Select
              value={form.parent_id}
              onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
              disabled={!!form.id}
            >
              <option value="">无（顶级分类）</option>
              {flat
                .filter((c) => c.id !== form.id)
                .map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
            </Select>
          </Field>
          <Field label="排序" hint="数字越小越靠前">
            <Input
              type="number"
              value={form.sort_order}
              onChange={(e) => setForm({ ...form, sort_order: e.target.value })}
            />
          </Field>
          <Field label="描述">
            <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={2} />
          </Field>
        </div>
      </Modal>

      {/* 删除确认 */}
      <Modal
        open={!!deleting}
        onClose={() => setDeleting(null)}
        title="删除分类"
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
          确定要删除分类 <span className="font-medium text-[var(--text-primary)]">{deleting?.name}</span> 吗？
          <br />
          <span className="text-xs text-[var(--text-tertiary)]">包含子分类或文章的分类无法删除。</span>
        </p>
      </Modal>
    </div>
  );
}

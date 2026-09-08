import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { ArrowLeft, Eye, PenLine, Send, Trash2 } from "lucide-react";
import {
  useArticle,
  useCreateArticle,
  useDeleteArticle,
  useUpdateArticle,
} from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Field, Input, Select, Textarea } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Spinner } from "@/components/ui/Spinner";
import { ArticleContent } from "@/components/article/ArticleContent";
import { cn } from "@/lib/utils";

/** 后台文章编辑器：新建 / 编辑（Markdown 编辑 + 实时预览） */
export default function ArticleEditor() {
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id && id !== "new";
  const navigate = useNavigate();

  const { data: categories } = useCategories();
  const { data: articleData, isLoading: loadingArticle } = useArticle(id ?? "", isEdit);

  // 表单状态
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [content, setContent] = useState("");
  const [excerpt, setExcerpt] = useState("");
  const [coverImage, setCoverImage] = useState("");
  const [categoryId, setCategoryId] = useState<string>("");
  const [tagsInput, setTagsInput] = useState("");
  const [preview, setPreview] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const create = useCreateArticle();
  const update = useUpdateArticle();
  const del = useDeleteArticle();

  // 编辑模式：加载文章填充表单
  useEffect(() => {
    if (isEdit && articleData?.article) {
      const a = articleData.article;
      setTitle(a.title);
      setSlug(a.slug);
      setContent(a.content);
      setExcerpt(a.excerpt);
      setCoverImage(a.cover_image);
      setCategoryId(a.category ? String(a.category.id) : "");
      setTagsInput(a.tags.map((t) => t.name).join(", "));
    }
  }, [isEdit, articleData]);

  const tags = useMemo(
    () =>
      tagsInput
        .split(/[,，]/)
        .map((t) => t.trim())
        .filter(Boolean)
        .slice(0, 10),
    [tagsInput],
  );

  const saving = create.isPending || update.isPending;

  const save = async (status: "draft" | "published") => {
    if (!title.trim()) {
      toast.error("请填写文章标题");
      return;
    }
    const payload = {
      title: title.trim(),
      slug: slug.trim() || undefined,
      content,
      excerpt: excerpt.trim() || undefined,
      cover_image: coverImage.trim() || undefined,
      category_id: categoryId ? Number(categoryId) : undefined,
      tag_names: tags.length > 0 ? tags : undefined,
      status,
    };
    try {
      if (isEdit) {
        await update.mutateAsync({ id: Number(id), ...payload });
        toast.success(status === "published" ? "已发布" : "草稿已保存");
      } else {
        await create.mutateAsync(payload);
        toast.success(status === "published" ? "文章已发布" : "草稿已保存");
      }
      navigate("/admin/articles");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "保存失败");
    }
  };

  const onDelete = async () => {
    try {
      await del.mutateAsync(Number(id));
      toast.success("文章已删除");
      navigate("/admin/articles");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "删除失败");
    }
  };

  if (isEdit && loadingArticle) {
    return (
      <div className="flex justify-center py-24">
        <Spinner className="size-8" />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        className="mb-8"
        kicker="EDITOR"
        title={isEdit ? "编辑文章" : "写文章"}
        actions={
          <div className="flex items-center gap-2">
            <Link
              to="/admin/articles"
              aria-label="返回文章列表"
              className="inline-flex size-9 items-center justify-center rounded-xl text-[var(--ink-2)] transition-colors hover:bg-[var(--accent-subtle)] hover:text-[var(--accent)]"
            >
              <ArrowLeft className="size-4" />
            </Link>
            {isEdit && (
              <Button variant="ghost" onClick={() => setConfirmDelete(true)}>
                <Trash2 className="size-4" /> 删除
              </Button>
            )}
            <Button variant="secondary" onClick={() => save("draft")} disabled={saving}>
              {saving ? "保存中…" : "保存草稿"}
            </Button>
            <Button onClick={() => save("published")} disabled={saving}>
              <Send className="size-4" /> {isEdit ? "更新并发布" : "发布"}
            </Button>
          </div>
        }
      />

      <div className="space-y-6">
        {/* 大标题 */}
        <Card className="p-6">
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="文章标题"
            className="h-12 text-lg font-semibold"
          />
        </Card>

        {/* 元信息 */}
        <Card className="grid gap-5 p-6 sm:grid-cols-2">
          <Field label="Slug（留空自动生成）" hint="用于 URL，如 my-first-post">
            <Input
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="自动生成"
              className="font-mono"
            />
          </Field>
          <Field label="分类">
            <Select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
              <option value="">不分类</option>
              {categories?.categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="标签" hint="逗号分隔，最多 10 个，不存在会自动创建">
            <Input value={tagsInput} onChange={(e) => setTagsInput(e.target.value)} placeholder="Go, Kratos, 博客" />
          </Field>
          <Field label="封面图 URL">
            <Input
              value={coverImage}
              onChange={(e) => setCoverImage(e.target.value)}
              placeholder="https://…（可选）"
              className="font-mono"
            />
          </Field>
          <div className="sm:col-span-2">
            <Field label="摘要（留空自动截取正文开头）">
              <Textarea value={excerpt} onChange={(e) => setExcerpt(e.target.value)} rows={2} maxLength={300} />
            </Field>
          </div>
        </Card>

        {/* 正文编辑 / 预览 */}
        <Card className="overflow-hidden">
          <div className="flex items-center justify-between border-b border-[var(--border-card)] px-6 py-3.5">
            <span className="text-sm font-medium text-[var(--ink)]">正文（Markdown）</span>
            <div className="flex rounded-xl bg-[var(--bg-hover)] p-0.5">
              <button
                onClick={() => setPreview(false)}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-[10px] px-3 py-1.5 text-xs font-medium transition-colors",
                  !preview ? "bg-[var(--bg-card)] text-[var(--ink)] shadow-sm" : "text-[var(--ink-3)]",
                )}
              >
                <PenLine className="size-3.5" /> 编辑
              </button>
              <button
                onClick={() => setPreview(true)}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-[10px] px-3 py-1.5 text-xs font-medium transition-colors",
                  preview ? "bg-[var(--bg-card)] text-[var(--ink)] shadow-sm" : "text-[var(--ink-3)]",
                )}
              >
                <Eye className="size-3.5" /> 预览
              </button>
            </div>
          </div>
          {preview ? (
            <div className="min-h-96 px-6 py-5">
              {content.trim() ? (
                <ArticleContent content={content} />
              ) : (
                <p className="py-16 text-center text-sm text-[var(--ink-3)]">暂无内容</p>
              )}
            </div>
          ) : (
            <Textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={"支持 Markdown 语法：\n# 标题\n- 列表\n`代码`\n```go\nfmt.Println(\"hello\")\n```"}
              className="min-h-96 rounded-none border-0 bg-transparent focus:ring-0 px-6 py-5 font-mono text-sm leading-relaxed"
            />
          )}
        </Card>
      </div>

      {/* 删除确认 */}
      <Modal
        open={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        title="删除文章"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmDelete(false)}>
              取消
            </Button>
            <Button variant="danger" onClick={onDelete} disabled={del.isPending}>
              {del.isPending ? "删除中…" : "确认删除"}
            </Button>
          </>
        }
      >
        <p className="text-sm leading-relaxed text-[var(--ink-2)]">
          确定要删除这篇文章吗？删除后不可恢复。
        </p>
      </Modal>
    </div>
  );
}

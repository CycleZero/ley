import { useRef, useState } from "react";
import { toast } from "sonner";
import { FileImage, Trash2, Upload } from "lucide-react";
import { useDeleteFile, useFiles, useUploadFile, type FileInfo } from "@/hooks/use-files";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Empty } from "@/components/ui/Empty";
import { Modal } from "@/components/ui/Modal";
import { Pagination } from "@/components/ui/Pagination";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatDate, formatCount } from "@/lib/utils";

const PAGE_SIZE = 20;

/** 后台文件管理：上传 / 列表 / 删除 */
export default function FileManage() {
  const [page, setPage] = useState(1);
  const [deleting, setDeleting] = useState<FileInfo | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const { data, isLoading } = useFiles(page, PAGE_SIZE);
  const upload = useUploadFile();
  const del = useDeleteFile();

  const onPick = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    upload.mutate(
      { filename: file.name, content: file },
      {
        onSuccess: (res) => {
          toast.success(`已上传 ${res.file.filename}`);
          if (inputRef.current) inputRef.current.value = "";
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : "上传失败");
          if (inputRef.current) inputRef.current.value = "";
        },
      },
    );
  };

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await del.mutateAsync(deleting.id);
      toast.success("文件已删除");
      setDeleting(null);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "删除失败");
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">文件管理</h1>
        <div className="flex items-center gap-3">
          {data && <span className="text-sm text-[var(--text-tertiary)]">共 {data.total} 个文件</span>}
          <Button onClick={() => inputRef.current?.click()} disabled={upload.isPending}>
            <Upload className="size-4" /> {upload.isPending ? "上传中…" : "上传文件"}
          </Button>
          <input ref={inputRef} type="file" className="hidden" onChange={onPick} />
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      ) : !data || data.files.length === 0 ? (
        <Card>
          <Empty title="暂无文件" description="点击右上角上传文件（图片等）" />
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {data.files.map((f) => (
              <Card key={f.id} className="group overflow-hidden">
                <a href={f.url} target="_blank" rel="noreferrer" className="block">
                  {f.mime_type.startsWith("image/") ? (
                    <img
                      src={f.url}
                      alt={f.filename}
                      loading="lazy"
                      className="aspect-video w-full object-cover"
                    />
                  ) : (
                    <div className="flex aspect-video items-center justify-center bg-[var(--bg-hover)] text-[var(--text-tertiary)]">
                      <FileImage className="size-8" />
                    </div>
                  )}
                </a>
                <div className="p-3">
                  <a
                    href={f.url}
                    target="_blank"
                    rel="noreferrer"
                    className="block truncate text-sm font-medium text-[var(--text-primary)] hover:text-[var(--accent)] transition-colors"
                    title={f.filename}
                  >
                    {f.filename}
                  </a>
                  <div className="mt-1 flex items-center justify-between">
                    <span className="text-xs text-[var(--text-tertiary)]">
                      {formatCount(f.size)} B · {formatDate(f.created_at)}
                    </span>
                    <button
                      onClick={() => setDeleting(f)}
                      className="inline-flex size-7 items-center justify-center rounded-lg text-[var(--text-tertiary)] transition-colors hover:bg-red-50 hover:text-red-500"
                      title="删除"
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
          <div className="mt-6">
            <Pagination page={page} total={data.total} pageSize={PAGE_SIZE} onChange={setPage} />
          </div>
        </>
      )}

      <Modal
        open={!!deleting}
        onClose={() => setDeleting(null)}
        title="删除文件"
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
          确定要删除文件 <span className="font-medium text-[var(--text-primary)]">{deleting?.filename}</span> 吗？
          <br />
          <span className="text-xs text-[var(--text-tertiary)]">引用该文件的文章封面/图片将无法显示。</span>
        </p>
      </Modal>
    </div>
  );
}

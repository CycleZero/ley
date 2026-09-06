import { cn } from "@/lib/utils";

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "size-5 animate-spin rounded-full border-2 border-[var(--border)] border-t-[var(--accent)]",
        className,
      )}
      role="status"
      aria-label="加载中"
    />
  );
}

/** 页面级加载（配合 Suspense / 路由懒加载） */
export function PageLoading() {
  return (
    <div className="flex min-h-[50vh] items-center justify-center">
      <Spinner className="size-8" />
    </div>
  );
}

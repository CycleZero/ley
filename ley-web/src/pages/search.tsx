import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Search } from "lucide-react";
import { useSearchArticles } from "@/hooks/use-articles";
import { ArticleCard } from "@/components/article/ArticleCard";
import { ArticleListSkeleton } from "@/components/ui/Skeleton";
import { Empty } from "@/components/ui/Empty";
import { Pagination } from "@/components/ui/Pagination";
import { Input } from "@/components/ui/Input";

const PAGE_SIZE = 10;

/** 搜索页：关键词全文搜索（标题/摘要/正文） */
export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const keyword = searchParams.get("q") ?? "";
  const page = Math.max(1, Number(searchParams.get("page") ?? 1) || 1);

  const [input, setInput] = useState(keyword);
  const inputRef = useRef<HTMLInputElement>(null);

  // 搜索词变化时同步输入框
  useEffect(() => {
    setInput(keyword);
  }, [keyword]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const { data, isLoading, isFetching } = useSearchArticles(keyword, page, PAGE_SIZE);
  const searching = keyword.trim().length > 0;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const q = input.trim();
    const next = new URLSearchParams();
    if (q) next.set("q", q);
    next.set("page", "1");
    setSearchParams(next);
  };

  const setPage = (p: number) => {
    const next = new URLSearchParams(searchParams);
    next.set("page", String(p));
    setSearchParams(next);
    window.scrollTo({ top: 0 });
  };

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <h1 className="mb-6 text-2xl font-bold text-[var(--text-primary)]">搜索</h1>

      {/* 搜索框 */}
      <form onSubmit={submit} className="relative mb-8">
        <Search className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-[var(--text-tertiary)]" />
        <Input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="搜索文章标题、内容…"
          className="h-12 pl-10 text-base"
        />
      </form>

      {!searching ? (
        <Empty title="输入关键词开始搜索" description="支持匹配文章标题、摘要和正文" />
      ) : isLoading ? (
        <ArticleListSkeleton count={3} />
      ) : data && data.total > 0 ? (
        <>
          <p className="mb-4 text-sm text-[var(--text-tertiary)]">
            找到 <span className="font-medium text-[var(--text-primary)]">{data.total}</span> 篇与
            “<span className="font-medium text-[var(--accent)]">{keyword}</span>” 相关的文章
          </p>
          <div className="space-y-4">
            {data.articles.map((a) => (
              <ArticleCard key={a.id} article={a} />
            ))}
          </div>
          <div className="mt-8">
            <Pagination page={page} total={data.total} pageSize={PAGE_SIZE} onChange={setPage} />
          </div>
        </>
      ) : (
        <Empty
          title="没有找到相关文章"
          description={`没有与“${keyword}”匹配的结果，换个关键词试试`}
          action={
            <button
              onClick={() => {
                setInput("");
                setSearchParams(new URLSearchParams());
              }}
              className="text-sm text-[var(--accent)] hover:underline"
            >
              清除搜索
            </button>
          }
        />
      )}

      {/* 结果更新中的轻提示 */}
      {isFetching && searching && (
        <p className="mt-4 text-center text-xs text-[var(--text-tertiary)]">搜索中…</p>
      )}
    </div>
  );
}

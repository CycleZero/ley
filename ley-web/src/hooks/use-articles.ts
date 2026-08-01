import { apiClient } from "@/lib/api-client";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

// Types matching backend proto
export interface Article {
  id: number;
  title: string;
  slug: string;
  content: string;
  excerpt: string;
  cover_image: string;
  status: "draft" | "published" | "archived";
  author: { id: number; username: string; avatar: string };
  category: { id: number; name: string } | null;
  tags: { id: number; name: string }[];
  view_count: number;
  like_count: number;
  is_top: boolean;
  is_liked: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
}

interface ArticleListParams {
  status?: string;
  category_id?: number;
  tags?: string[];
  page?: number;
  page_size?: number;
}

export function useArticles(params: ArticleListParams = {}) {
  return useQuery({
    queryKey: ["articles", params],
    queryFn: () => apiClient("/articles", { query: params }) as Promise<{ articles: Article[]; total: number }>,
  });
}

export function useInfiniteArticles(params: ArticleListParams = {}) {
  return useInfiniteQuery({
    queryKey: ["articles", "infinite", params],
    queryFn: ({ pageParam }) =>
      apiClient("/articles", {
        query: { ...params, page: pageParam, page_size: params.page_size || 10 },
      }) as Promise<{ articles: Article[]; total: number }>,
    initialPageParam: 1,
    getNextPageParam: (last, _pages, lastPageParam) => {
      if ((last as { articles: Article[]; total: number }).articles.length < 10) return undefined;
      return (lastPageParam as number) + 1;
    },
  });
}

export function useArticle(identifier: string, enabled = true) {
  return useQuery({
    queryKey: ["article", identifier],
    queryFn: () => apiClient(`/articles/${identifier}`) as Promise<{ article: Article }>,
    enabled: !!identifier && enabled,
  });
}

export function useCreateArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      apiClient("/articles", { method: "POST", body: data }) as Promise<{ article: Article }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["articles"] }),
  });
}

export function useUpdateArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: number } & Record<string, unknown>) =>
      apiClient(`/articles/${id}`, { method: "PUT", body: data }) as Promise<{ article: Article }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["articles"] }),
  });
}

export function useDeleteArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/articles/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["articles"] }),
  });
}

export function usePublishArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/articles/${id}/publish`, { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["articles"] }),
  });
}

export function useArchiveArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/articles/${id}/archive`, { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["articles"] }),
  });
}

export function useLikeArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/articles/${id}/like`, { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["article"] }),
  });
}

export function useUnlikeArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/articles/${id}/like`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["article"] }),
  });
}

/** 关键词搜索（后端 LIKE 匹配：标题/摘要/正文） */
export function useSearchArticles(keyword: string, page = 1, pageSize = 10) {
  const trimmed = keyword.trim();
  return useQuery({
    queryKey: ["articles", "search", trimmed, page, pageSize],
    queryFn: () =>
      apiClient("/articles/search", {
        query: { keyword: trimmed, page, page_size: pageSize },
      }) as Promise<{ articles: Article[]; total: number }>,
    enabled: trimmed.length > 0,
  });
}

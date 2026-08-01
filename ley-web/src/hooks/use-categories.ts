import { apiClient } from "@/lib/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
  parent_id: number;
  sort_order: number;
  article_count: number;
  children: Category[];
}

export function useCategories() {
  return useQuery({
    queryKey: ["categories"],
    queryFn: () => apiClient("/categories") as Promise<{ categories: Category[] }>,
    staleTime: 60_000,
  });
}

export function useCreateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; slug?: string; description?: string; parent_id?: number; sort_order?: number }) =>
      apiClient("/categories", { method: "POST", body: data }) as Promise<{ category: Category }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  });
}

export function useUpdateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: number; name?: string; slug?: string; description?: string; parent_id?: number; sort_order?: number }) =>
      apiClient(`/categories/${id}`, { method: "PUT", body: data }) as Promise<{ category: Category }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  });
}

export function useDeleteCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/categories/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  });
}

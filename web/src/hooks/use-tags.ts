import { apiClient } from "@/lib/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface Tag {
  id: number;
  name: string;
  slug: string;
  article_count: number;
}

export function useTags() {
  return useQuery({
    queryKey: ["tags"],
    queryFn: () => apiClient("/tags") as Promise<{ tags: Tag[] }>,
    staleTime: 60_000,
  });
}

export function useCreateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiClient("/tags", { method: "POST", body: { name } }) as Promise<{ tag: Tag }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tags"] }),
  });
}

export function useDeleteTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/tags/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tags"] }),
  });
}

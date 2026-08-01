import { apiClient } from "@/lib/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface FileInfo {
  id: number;
  filename: string;
  mime_type: string;
  size: number;
  url: string;
  created_at: string;
}

export function useFiles(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["files", page, pageSize],
    queryFn: () =>
      apiClient("/files", { query: { page, page_size: pageSize } }) as Promise<{ files: FileInfo[]; total: number }>,
  });
}

export function useUploadFile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ filename, content }: { filename: string; content: Blob }) => {
      const form = new FormData();
      form.append("filename", filename);
      form.append("content", content);
      return apiClient("/files/upload", { method: "POST", body: form }) as Promise<{ file: FileInfo }>;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["files"] }),
  });
}

export function useDeleteFile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient(`/files/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["files"] }),
  });
}

export function usePresignedUploadUrl() {
  return useMutation({
    mutationFn: ({ filename, mime_type }: { filename: string; mime_type: string }) =>
      apiClient("/files/presigned-upload", {
        query: { filename, mime_type },
      }) as Promise<{ url: string; object_key: string }>,
  });
}

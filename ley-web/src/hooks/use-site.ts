import { apiClient } from "@/lib/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface SiteConfig {
  site_title: string;
  site_subtitle: string;
  site_description: string;
  site_logo: string;
  site_favicon: string;
  seo_keywords: string;
  seo_description: string;
  social_github: string;
  social_twitter: string;
  social_email: string;
  footer_text: string;
  icp_number: string;
  enable_likes: boolean;
}

export function useSiteConfig() {
  return useQuery({
    queryKey: ["site", "config"],
    queryFn: () => apiClient("/site/config") as Promise<{ config: SiteConfig }>,
  });
}

export function useUpdateSiteConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (config: Partial<SiteConfig>) =>
      apiClient("/site/config", { method: "PUT", body: { config } }) as Promise<{ config: SiteConfig }>,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["site"] }),
  });
}

import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/hooks/use-site", () => ({ useSiteConfig: vi.fn() }));

import { useSiteConfig } from "@/hooks/use-site";
import { SocialCard } from "./SocialCard";

const mockSite = useSiteConfig as ReturnType<typeof vi.fn>;

const emptyConfig = {
  site_title: "",
  site_subtitle: "",
  site_description: "",
  site_logo: "",
  site_favicon: "",
  seo_keywords: "",
  seo_description: "",
  social_github: "",
  social_twitter: "",
  social_email: "",
  footer_text: "",
  icp_number: "",
  enable_likes: true,
};

describe("SocialCard", () => {
  beforeEach(() => {
    mockSite.mockReset();
  });

  it("配置提供 GitHub/Twitter/邮件时渲染 3 个真实链接", () => {
    mockSite.mockReturnValue({
      data: { config: { ...emptyConfig, social_github: "poyuan", social_twitter: "https://x.com/poyuan", social_email: "hi@ley.dev" } },
      isLoading: false,
    });
    const { container } = render(<SocialCard />);
    const links = container.querySelectorAll("a");
    expect(links.length).toBe(3);
    expect(links[0].getAttribute("href")).toBe("https://github.com/poyuan");
    expect(links[1].getAttribute("href")).toBe("https://x.com/poyuan");
    expect(links[2].getAttribute("href")).toBe("mailto:hi@ley.dev");
    // mailto 不挂 target，外部链接才开新窗
    expect(links[0].getAttribute("target")).toBe("_blank");
    expect(links[2].getAttribute("target")).toBeNull();
  });

  it("配置未提供任何社交时整卡隐藏（无 fake 链接）", () => {
    mockSite.mockReturnValue({
      data: { config: { ...emptyConfig } },
      isLoading: false,
    });
    const { container } = render(<SocialCard />);
    expect(container.querySelector("a")).toBeNull();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("加载中显示骨架屏", () => {
    mockSite.mockReturnValue({ data: undefined, isLoading: true });
    const { container } = render(<SocialCard />);
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
  });
});

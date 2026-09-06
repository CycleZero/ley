import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/hooks/use-site", () => ({ useSiteConfig: vi.fn() }));

import { useSiteConfig } from "@/hooks/use-site";
import { AboutCard } from "./AboutCard";

const mockSite = useSiteConfig as ReturnType<typeof vi.fn>;

describe("AboutCard", () => {
  beforeEach(() => {
    mockSite.mockReset();
  });

  it("渲染配置中的名字、角色与简介", () => {
    mockSite.mockReturnValue({
      data: {
        config: {
          site_title: "蓝屿记",
          site_subtitle: "Full-stack Dev · Writer",
          site_description: "在蓝鲸般的天空里写诗。",
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
        },
      },
      isLoading: false,
    });
    render(<AboutCard />);
    expect(screen.getByText("蓝屿记")).toBeInTheDocument();
    expect(screen.getByText("Full-stack Dev · Writer")).toBeInTheDocument();
    expect(screen.getByText("在蓝鲸般的天空里写诗。")).toBeInTheDocument();
  });

  it("无配置时使用占位（不硬编码站点名）", () => {
    mockSite.mockReturnValue({ data: undefined, isLoading: false });
    render(<AboutCard />);
    expect(screen.getByText("未命名")).toBeInTheDocument();
  });

  it("加载中显示骨架屏", () => {
    mockSite.mockReturnValue({ data: undefined, isLoading: true });
    const { container } = render(<AboutCard />);
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
  });
});

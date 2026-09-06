import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { SiteBrand } from "./SiteBrand";
import { useSiteConfig } from "@/hooks/use-site";

vi.mock("@/hooks/use-site", () => ({
  useSiteConfig: vi.fn(),
  useUpdateSiteConfig: vi.fn(),
}));

const mockUseSiteConfig = vi.mocked(useSiteConfig);
type QueryResult = ReturnType<typeof useSiteConfig>;

function configResult(partial?: Partial<Record<string, unknown>>): QueryResult {
  return {
    data: {
      config: {
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
        ...partial,
      },
    },
  } as unknown as QueryResult;
}

function renderBrand(subtitle = false) {
  return render(
    <MemoryRouter>
      <SiteBrand subtitle={subtitle} />
    </MemoryRouter>,
  );
}

describe("SiteBrand", () => {
  beforeEach(() => {
    mockUseSiteConfig.mockReset();
  });

  it("mock useSiteConfig 返回 site_title=测试站 时渲染该文本", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "测试站" }));
    renderBrand();
    expect(screen.getByText("测试站")).toBeInTheDocument();
  });

  it("无 config 时渲染中性占位「我的博客」", () => {
    mockUseSiteConfig.mockReturnValue({ data: undefined } as unknown as QueryResult);
    renderBrand();
    expect(screen.getByText("我的博客")).toBeInTheDocument();
  });

  it("site_title 为空时同样渲染占位", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "   " }));
    renderBrand();
    expect(screen.getByText("我的博客")).toBeInTheDocument();
  });

  it("subtitle=true 且存在 site_subtitle 时渲染 mono 副标", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "测试站", site_subtitle: "BLUE ISLE" }));
    renderBrand(true);
    expect(screen.getByText("BLUE ISLE")).toBeInTheDocument();
  });

  it("subtitle=false 时不渲染副标", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "测试站", site_subtitle: "BLUE ISLE" }));
    renderBrand(false);
    expect(screen.queryByText("BLUE ISLE")).not.toBeInTheDocument();
  });
});

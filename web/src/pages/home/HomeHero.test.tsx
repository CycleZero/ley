import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { HomeHero } from "./HomeHero";
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

describe("HomeHero", () => {
  beforeEach(() => {
    mockUseSiteConfig.mockReset();
  });

  it("site_title=测试站 时渲染该标题", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "测试站" }));
    render(<HomeHero />);
    expect(screen.getByRole("heading", { level: 1, name: "测试站" })).toBeInTheDocument();
  });

  it("无 config 数据时渲染中性占位「我的博客」", () => {
    mockUseSiteConfig.mockReturnValue({ data: undefined } as unknown as QueryResult);
    render(<HomeHero />);
    expect(screen.getByRole("heading", { level: 1, name: "我的博客" })).toBeInTheDocument();
  });

  it("site_title 为空串/纯空格时同样渲染占位「我的博客」", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "   " }));
    render(<HomeHero />);
    expect(screen.getByRole("heading", { level: 1, name: "我的博客" })).toBeInTheDocument();
  });

  it("始终渲染英文 kicker PERSONAL BLOG", () => {
    mockUseSiteConfig.mockReturnValue(configResult());
    render(<HomeHero />);
    expect(screen.getByText("PERSONAL BLOG")).toBeInTheDocument();
  });

  it("存在 site_subtitle 时渲染副标", () => {
    mockUseSiteConfig.mockReturnValue(
      configResult({ site_title: "测试站", site_subtitle: "在蓝鲸般的天空里写诗" }),
    );
    render(<HomeHero />);
    expect(screen.getByText("在蓝鲸般的天空里写诗")).toBeInTheDocument();
  });

  it("site_subtitle 为空时渲染中性副标占位", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "测试站" }));
    render(<HomeHero />);
    expect(screen.getByText("记录 · 思考 · 分享")).toBeInTheDocument();
  });
});

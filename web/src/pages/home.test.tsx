import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { Article } from "@/hooks/use-articles";
import type { SiteConfig } from "@/hooks/use-site";

vi.mock("@/hooks/use-site", () => ({
  useSiteConfig: vi.fn(),
  useUpdateSiteConfig: vi.fn(),
}));
vi.mock("@/hooks/use-articles", () => ({
  useArticles: vi.fn(),
}));
vi.mock("@/hooks/use-tags", () => ({
  useTags: vi.fn(),
}));
vi.mock("@/hooks/use-categories", () => ({
  useCategories: vi.fn(),
}));

import { useArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { useTags } from "@/hooks/use-tags";
import { useSiteConfig } from "@/hooks/use-site";
import HomePage from "./home";

const mockUseSiteConfig = vi.mocked(useSiteConfig);
const mockUseArticles = vi.mocked(useArticles);
const mockUseTags = vi.mocked(useTags);
const mockUseCategories = vi.mocked(useCategories);

type QueryResult = ReturnType<typeof useSiteConfig>;

const BASE_CONFIG: SiteConfig = {
  site_title: "测试站",
  site_subtitle: "在蓝鲸般的天空里写诗",
  site_description: "一个测试用博客",
  site_logo: "",
  site_favicon: "",
  seo_keywords: "",
  seo_description: "",
  social_github: "tester",
  social_twitter: "",
  social_email: "hello@example.com",
  footer_text: "",
  icp_number: "",
  enable_likes: true,
};

function siteResult(config?: Partial<SiteConfig>, isLoading = false): QueryResult {
  return {
    data: { config: { ...BASE_CONFIG, ...config } },
    isLoading,
  } as unknown as QueryResult;
}

function articlesResult(articles: Article[], total = articles.length, isLoading = false): ReturnType<typeof useArticles> {
  return { data: { articles, total }, isLoading } as unknown as ReturnType<typeof useArticles>;
}

function tagsResult(isLoading = false): ReturnType<typeof useTags> {
  return {
    data: {
      tags: [
        { id: 1, name: "前端", slug: "fe", article_count: 5 },
        { id: 2, name: "React", slug: "react", article_count: 2 },
      ],
    },
    isLoading,
  } as unknown as ReturnType<typeof useTags>;
}

function categoriesResult(isLoading = false): ReturnType<typeof useCategories> {
  return {
    data: {
      categories: [
        { id: 1, name: "技术", slug: "tech", description: "", parent_id: 0, sort_order: 0, article_count: 3, children: [] },
      ],
    },
    isLoading,
  } as unknown as ReturnType<typeof useCategories>;
}

function makeArticle(id: number, title: string): Article {
  return {
    id,
    title,
    slug: `slug-${id}`,
    content: "",
    excerpt: "",
    cover_image: "",
    status: "published",
    author: { id: 1, username: "tester", avatar: "" },
    category: id % 2 === 0 ? { id: 2, name: "随记" } : null,
    tags: [],
    view_count: 0,
    like_count: 0,
    is_top: false,
    is_liked: false,
    published_at: "2026-07-03T09:00:00Z",
    created_at: "2026-07-03T09:00:00Z",
    updated_at: "2026-07-03T09:00:00Z",
  };
}

function mockData() {
  mockUseSiteConfig.mockReturnValue(siteResult());
  mockUseArticles.mockReturnValue(articlesResult([makeArticle(1, "第一篇"), makeArticle(2, "第二篇")]));
  mockUseTags.mockReturnValue(tagsResult());
  mockUseCategories.mockReturnValue(categoriesResult());
}

function renderHome() {
  return render(
    <MemoryRouter>
      <HomePage />
    </MemoryRouter>,
  );
}

describe("HomePage（Bento 首页）", () => {
  beforeEach(() => {
    mockUseSiteConfig.mockReset();
    mockUseArticles.mockReset();
    mockUseTags.mockReset();
    mockUseCategories.mockReset();
  });

  it("hero 渲染站点配置标题（站点名来自 useSiteConfig，非硬编码）", () => {
    mockData();
    renderHome();
    expect(screen.getByRole("heading", { level: 1, name: "测试站" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { level: 1, name: "Ley" })).not.toBeInTheDocument();
  });

  it("Bento 网格渲染全部 7 张面板卡", () => {
    mockData();
    renderHome();
    for (const label of ["问候", "站点统计", "最近文章", "日历", "标签云", "关于我", "找到我"]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    expect(screen.getAllByRole("article").length).toBeGreaterThanOrEqual(7);
    // 最近文章卡链到 /articles；社交卡渲染配置提供的链接
    expect(screen.getByRole("link", { name: "查看全部 →" })).toHaveAttribute("href", "/articles");
    expect(screen.getByTitle("GitHub")).toHaveAttribute("href", "https://github.com/tester");
  });

  it("bento 网格为 grid-cols-12 + gap-3.5，各卡 span 配置正确（含响应式断点）", () => {
    mockData();
    const { container } = renderHome();
    const grid = container.querySelector(".grid.grid-cols-12");
    expect(grid).not.toBeNull();
    expect(grid!.className).toContain("gap-3.5");
    // span4：问候/统计/社交；span6：最近文章；span3：日历/标签云；行3 关于我 span4 居中
    expect(screen.getByText("问候").closest("div.col-span-4")).not.toBeNull();
    expect(screen.getByText("站点统计").closest("div.col-span-4")).not.toBeNull();
    expect(screen.getByText("找到我").closest("div.col-span-4")).not.toBeNull();
    expect(screen.getByText("最近文章").closest("div.col-span-6")).not.toBeNull();
    expect(screen.getByText("日历").closest("div.col-span-3")).not.toBeNull();
    expect(screen.getByText("标签云").closest("div.col-span-3")).not.toBeNull();
    expect(screen.getByText("关于我").closest("div.col-span-4.col-start-5")).not.toBeNull();
    // 每格都带 <1120px 收 6 列 / <768px 单列 的断点类
    expect(screen.getByText("问候").closest("div.col-span-4")!.className).toContain("max-[1120px]:col-span-6");
    expect(screen.getByText("问候").closest("div.col-span-4")!.className).toContain("max-[767px]:col-span-12");
  });

  it("数据加载中显示 .animate-pulse 骨架屏", () => {
    mockUseSiteConfig.mockReturnValue({ data: undefined, isLoading: true } as unknown as QueryResult);
    mockUseArticles.mockReturnValue({ data: undefined, isLoading: true } as unknown as ReturnType<typeof useArticles>);
    mockUseTags.mockReturnValue({ data: undefined, isLoading: true } as unknown as ReturnType<typeof useTags>);
    mockUseCategories.mockReturnValue({ data: undefined, isLoading: true } as unknown as ReturnType<typeof useCategories>);
    renderHome();
    expect(document.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });

  it("不再渲染旧首页的文章流/分类 chips/加载更多（功能已迁 /articles）", () => {
    mockData();
    renderHome();
    expect(screen.queryByRole("button", { name: "加载更多" })).not.toBeInTheDocument();
    expect(screen.queryByText("全部")).not.toBeInTheDocument();
    expect(screen.queryByText(/共 \d+ 篇文章/)).not.toBeInTheDocument();
  });
});

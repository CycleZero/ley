import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import type { Article } from "@/hooks/use-articles";

const fetchNextPage = vi.fn();

vi.mock("@/hooks/use-articles", () => ({
  useInfiniteArticles: vi.fn(),
}));
vi.mock("@/hooks/use-categories", () => ({
  useCategories: vi.fn(),
}));

import { useInfiniteArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import HomePage from "./home";

const mockArticles = useInfiniteArticles as ReturnType<typeof vi.fn>;
const mockCategories = useCategories as ReturnType<typeof vi.fn>;

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

describe("HomePage", () => {
  beforeEach(() => {
    fetchNextPage.mockReset();
    mockArticles.mockReset();
    mockCategories.mockReset();
  });

  it("加载中显示骨架屏", () => {
    mockArticles.mockReturnValue({ data: undefined, isLoading: true });
    mockCategories.mockReturnValue({ data: undefined });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    // 骨架屏元素（animate-pulse）
    expect(document.querySelector(".animate-pulse")).not.toBeNull();
  });

  it("无文章时显示空态", () => {
    mockArticles.mockReturnValue({
      data: { pages: [{ articles: [], total: 0 }] },
      isLoading: false,
    });
    mockCategories.mockReturnValue({ data: { categories: [] } });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    expect(screen.getByText("暂无文章")).toBeInTheDocument();
  });

  it("渲染文章列表与总数", () => {
    mockArticles.mockReturnValue({
      data: { pages: [{ articles: [makeArticle(1, "第一篇"), makeArticle(2, "第二篇")], total: 2 }] },
      isLoading: false,
    });
    mockCategories.mockReturnValue({ data: { categories: [] } });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    expect(screen.getByText("第一篇")).toBeInTheDocument();
    expect(screen.getByText("第二篇")).toBeInTheDocument();
    expect(screen.getByText("共 2 篇文章")).toBeInTheDocument();
  });

  it("渲染分类过滤按钮并可切换", async () => {
    mockArticles.mockReturnValue({
      data: { pages: [{ articles: [], total: 0 }] },
      isLoading: false,
    });
    mockCategories.mockReturnValue({
      data: {
        categories: [
          { id: 1, name: "技术", slug: "tech", description: "", parent_id: 0, sort_order: 0, article_count: 3, children: [] },
        ],
      },
    });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    const chip = screen.getByRole("button", { name: /技术/ });
    await userEvent.click(chip);
    // 点击后重新查询（category_id=1）
    expect(mockArticles).toHaveBeenLastCalledWith({ category_id: 1 });
  });

  it("有更多时显示加载更多并可触发 fetchNextPage", async () => {
    mockArticles.mockReturnValue({
      data: { pages: [{ articles: [makeArticle(1, "第一篇")], total: 15 }] },
      isLoading: false,
      isFetchingNextPage: false,
      hasNextPage: true,
      fetchNextPage,
    });
    mockCategories.mockReturnValue({ data: { categories: [] } });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    const btn = screen.getByRole("button", { name: "加载更多" });
    await userEvent.click(btn);
    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("无更多时不显示加载更多", () => {
    mockArticles.mockReturnValue({
      data: { pages: [{ articles: [makeArticle(1, "第一篇")], total: 1 }] },
      isLoading: false,
      hasNextPage: false,
      fetchNextPage,
    });
    mockCategories.mockReturnValue({ data: { categories: [] } });
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("button", { name: "加载更多" })).not.toBeInTheDocument();
  });
});

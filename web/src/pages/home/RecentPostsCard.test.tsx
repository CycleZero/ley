import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/hooks/use-articles", () => ({ useArticles: vi.fn() }));

import { useArticles } from "@/hooks/use-articles";
import { RecentPostsCard } from "./RecentPostsCard";

const mockArticles = useArticles as ReturnType<typeof vi.fn>;

describe("RecentPostsCard", () => {
  beforeEach(() => {
    mockArticles.mockReset();
  });

  it("渲染最近 3 篇文章标题与 mono 日期", () => {
    mockArticles.mockReturnValue({
      data: {
        articles: [
          { id: 1, slug: "a-1", title: "第一篇", published_at: "2026-09-03T09:00:00Z" },
          { id: 2, slug: "a-2", title: "第二篇", published_at: "2026-08-27T09:00:00Z" },
          { id: 3, slug: "a-3", title: "第三篇", published_at: "2026-08-19T09:00:00Z" },
        ],
        total: 3,
      },
      isLoading: false,
    });
    render(
      <MemoryRouter>
        <RecentPostsCard />
      </MemoryRouter>,
    );
    expect(screen.getByText("第一篇")).toBeInTheDocument();
    expect(screen.getByText("第二篇")).toBeInTheDocument();
    expect(screen.getByText("第三篇")).toBeInTheDocument();
    expect(screen.getByText("2026.09.03")).toBeInTheDocument();
  });

  it("「查看全部 →」链到 /articles", () => {
    mockArticles.mockReturnValue({ data: { articles: [], total: 0 }, isLoading: false });
    render(
      <MemoryRouter>
        <RecentPostsCard />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /查看全部/ });
    expect(link.getAttribute("href")).toBe("/articles");
  });

  it("无文章时显示空态", () => {
    mockArticles.mockReturnValue({ data: { articles: [], total: 0 }, isLoading: false });
    render(
      <MemoryRouter>
        <RecentPostsCard />
      </MemoryRouter>,
    );
    expect(screen.getByText("暂无文章")).toBeInTheDocument();
  });

  it("加载中显示骨架屏", () => {
    mockArticles.mockReturnValue({ data: undefined, isLoading: true });
    const { container } = render(
      <MemoryRouter>
        <RecentPostsCard />
      </MemoryRouter>,
    );
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
  });
});

import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { Article } from "@/hooks/use-articles";

vi.mock("@/hooks/use-articles", () => ({
  useSearchArticles: vi.fn(),
}));

import { useSearchArticles } from "@/hooks/use-articles";
import SearchPage from "./search";

const mockSearch = useSearchArticles as ReturnType<typeof vi.fn>;

const article: Article = {
  id: 1,
  title: "Kratos 入门指南",
  slug: "kratos-intro",
  content: "",
  excerpt: "从零开始学 Kratos",
  cover_image: "",
  status: "published",
  author: { id: 1, username: "tester", avatar: "" },
  category: null,
  tags: [],
  view_count: 1,
  like_count: 0,
  is_top: false,
  is_liked: false,
  published_at: "2026-07-03T09:00:00Z",
  created_at: "2026-07-03T09:00:00Z",
  updated_at: "2026-07-03T09:00:00Z",
};

function renderPage(initialPath = "/search") {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/search" element={<SearchPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("SearchPage", () => {
  beforeEach(() => {
    mockSearch.mockReset();
  });

  it("未输入关键词时显示引导空态", () => {
    mockSearch.mockReturnValue({ data: undefined, isLoading: false, isFetching: false });
    renderPage();
    expect(screen.getByText("输入关键词开始搜索")).toBeInTheDocument();
  });

  it("输入关键词提交后显示结果", async () => {
    mockSearch.mockReturnValue({
      data: { articles: [article], total: 1 },
      isLoading: false,
      isFetching: false,
    });
    renderPage();
    await userEvent.type(screen.getByPlaceholderText("搜索文章标题、内容…"), "Kratos");
    // 表单无按钮，回车提交
    await userEvent.keyboard("{Enter}");

    await waitFor(() => {
      expect(screen.getByText("Kratos 入门指南")).toBeInTheDocument();
    });
    expect(mockSearch).toHaveBeenCalledWith("Kratos", 1, 10);
  });

  it("无结果显示空态", async () => {
    mockSearch.mockReturnValue({ data: { articles: [], total: 0 }, isLoading: false, isFetching: false });
    renderPage("/search?q=不存在");
    expect(await screen.findByText("没有找到相关文章")).toBeInTheDocument();
    expect(screen.getByText("清除搜索")).toBeInTheDocument();
  });

  it("结果统计显示总数", async () => {
    mockSearch.mockReturnValue({
      data: { articles: [article], total: 1 },
      isLoading: false,
      isFetching: false,
    });
    renderPage("/search?q=Kratos");
    const summary = await screen.findByText(/找到/);
    expect(summary.textContent).toContain("1");
    expect(summary.textContent).toContain("Kratos");
  });
});

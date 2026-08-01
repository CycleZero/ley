import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import type { Article } from "@/hooks/use-articles";

const likeMutate = vi.fn();
const unlikeMutate = vi.fn();

vi.mock("@/hooks/use-articles", () => ({
  useArticle: vi.fn(),
  useLikeArticle: () => ({ mutate: likeMutate, isPending: false }),
  useUnlikeArticle: () => ({ mutate: unlikeMutate, isPending: false }),
}));

import { useArticle } from "@/hooks/use-articles";
import ArticleDetailPage from "./article-detail";

const mockUseArticle = useArticle as ReturnType<typeof vi.fn>;

const article: Article = {
  id: 1,
  title: "Kratos 微服务入门",
  slug: "kratos-intro",
  content: "# 标题\n\n正文内容",
  excerpt: "",
  cover_image: "",
  status: "published",
  author: { id: 1, username: "tester", avatar: "" },
  category: { id: 1, name: "技术" },
  tags: [{ id: 1, name: "Go" }],
  view_count: 100,
  like_count: 5,
  is_top: false,
  is_liked: false,
  published_at: "2026-07-03T09:00:00Z",
  created_at: "2026-07-03T09:00:00Z",
  updated_at: "2026-07-03T09:00:00Z",
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/articles/kratos-intro"]}>
      <Routes>
        <Route path="/articles/:slug" element={<ArticleDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("ArticleDetailPage", () => {
  beforeEach(() => {
    likeMutate.mockReset();
    unlikeMutate.mockReset();
    useAuthStore.setState({ user: null, ready: true });
  });

  it("加载中显示 Spinner", () => {
    mockUseArticle.mockReturnValue({ data: undefined, isLoading: true, isError: false });
    renderPage();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("文章不存在时显示空态", () => {
    mockUseArticle.mockReturnValue({ data: undefined, isLoading: false, isError: true });
    renderPage();
    expect(screen.getByText("文章不存在")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "返回文章列表" })).toBeInTheDocument();
  });

  it("渲染标题、作者、分类、标签与 Markdown 正文", async () => {
    mockUseArticle.mockReturnValue({ data: { article }, isLoading: false, isError: false });
    renderPage();
    expect(screen.getByText("Kratos 微服务入门")).toBeInTheDocument();
    expect(screen.getByText("tester")).toBeInTheDocument();
    expect(screen.getByText("技术")).toBeInTheDocument();
    expect(screen.getByText("#Go")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { level: 1, name: "标题" })).toBeInTheDocument();
    expect(await screen.findByText("正文内容")).toBeInTheDocument();
  });

  it("未登录点击喜欢跳转登录页", async () => {
    mockUseArticle.mockReturnValue({ data: { article }, isLoading: false, isError: false });
    renderPage();
    await userEvent.click(screen.getByRole("button", { name: /喜欢/ }));
    // MemoryRouter 无 /login 路由 → 渲染 NotFound；验证 like 未被调用
    expect(likeMutate).not.toHaveBeenCalled();
  });

  it("已登录点击喜欢调用 like", async () => {
    useAuthStore.setState({ user: { ...article.author, email: "t@e.com", bio: "", role: "user" }, ready: true });
    mockUseArticle.mockReturnValue({ data: { article }, isLoading: false, isError: false });
    renderPage();
    await userEvent.click(screen.getByRole("button", { name: /喜欢/ }));
    expect(likeMutate).toHaveBeenCalledWith(1);
  });

  it("已喜欢状态点击调用 unlike", async () => {
    useAuthStore.setState({ user: { ...article.author, email: "t@e.com", bio: "", role: "user" }, ready: true });
    mockUseArticle.mockReturnValue({
      data: { article: { ...article, is_liked: true } },
      isLoading: false,
      isError: false,
    });
    renderPage();
    const btn = screen.getByRole("button", { name: /已喜欢/ });
    await userEvent.click(btn);
    expect(unlikeMutate).toHaveBeenCalledWith(1);
  });

  it("返回列表链接存在", () => {
    mockUseArticle.mockReturnValue({ data: { article }, isLoading: false, isError: false });
    renderPage();
    expect(screen.getByRole("link", { name: /返回列表/ })).toHaveAttribute("href", "/articles");
  });
});

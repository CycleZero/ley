import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ArticleCard } from "./ArticleCard";
import type { Article } from "@/hooks/use-articles";

function makeArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 1,
    title: "Kratos 微服务入门",
    slug: "kratos-intro",
    content: "",
    excerpt: "一篇介绍 Kratos 的文章",
    cover_image: "",
    status: "published",
    author: { id: 1, username: "tester", avatar: "" },
    category: { id: 1, name: "技术" },
    tags: [
      { id: 1, name: "Go" },
      { id: 2, name: "Kratos" },
    ],
    view_count: 120,
    like_count: 8,
    is_top: false,
    is_liked: false,
    published_at: "2026-07-03T09:00:00Z",
    created_at: "2026-07-03T09:00:00Z",
    updated_at: "2026-07-03T09:00:00Z",
    ...overrides,
  };
}

function renderCard(article: Article) {
  return render(
    <MemoryRouter>
      <ArticleCard article={article} />
    </MemoryRouter>,
  );
}

describe("ArticleCard", () => {
  it("渲染标题、摘要、分类与标签", () => {
    renderCard(makeArticle());
    expect(screen.getByText("Kratos 微服务入门")).toBeInTheDocument();
    expect(screen.getByText("一篇介绍 Kratos 的文章")).toBeInTheDocument();
    expect(screen.getByText("技术")).toBeInTheDocument();
    expect(screen.getByText("#Go")).toBeInTheDocument();
    expect(screen.getByText("#Kratos")).toBeInTheDocument();
  });

  it("链接指向 /articles/{slug}", () => {
    renderCard(makeArticle());
    const link = screen.getByRole("link", { name: /Kratos 微服务入门/ });
    expect(link).toHaveAttribute("href", "/articles/kratos-intro");
  });

  it("显示浏览量、点赞数与时间", () => {
    renderCard(makeArticle());
    expect(screen.getByText("120")).toBeInTheDocument();
    expect(screen.getByText("8")).toBeInTheDocument();
  });

  it("置顶文章显示置顶标记", () => {
    renderCard(makeArticle({ is_top: true }));
    expect(screen.getByText("置顶")).toBeInTheDocument();
  });

  it("非置顶不显示置顶标记", () => {
    renderCard(makeArticle({ is_top: false }));
    expect(screen.queryByText("置顶")).not.toBeInTheDocument();
  });

  it("有封面图时渲染图片", () => {
    const { container } = renderCard(makeArticle({ cover_image: "https://example.com/cover.png" }));
    const img = container.querySelector("img");
    expect(img).toHaveAttribute("src", "https://example.com/cover.png");
  });

  it("无封面图时不渲染图片", () => {
    const { container } = renderCard(makeArticle({ cover_image: "" }));
    expect(container.querySelector("img")).not.toBeInTheDocument();
  });

  it("无分类时不渲染分类 Badge", () => {
    renderCard(makeArticle({ category: null }));
    expect(screen.queryByText("技术")).not.toBeInTheDocument();
  });

  it("标签超过 4 个只显示前 4 个", () => {
    const tags = Array.from({ length: 6 }, (_, i) => ({ id: i + 1, name: `标签${i + 1}` }));
    renderCard(makeArticle({ tags }));
    expect(screen.getByText("#标签1")).toBeInTheDocument();
    expect(screen.queryByText("#标签5")).not.toBeInTheDocument();
  });

  it("无摘要时不渲染摘要段落", () => {
    renderCard(makeArticle({ excerpt: "" }));
    expect(screen.queryByText(/一篇介绍/)).not.toBeInTheDocument();
  });
});

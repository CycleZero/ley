import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/hooks/use-articles", () => ({ useArticles: vi.fn() }));
vi.mock("@/hooks/use-categories", () => ({ useCategories: vi.fn() }));
vi.mock("@/hooks/use-tags", () => ({ useTags: vi.fn() }));

import { useArticles } from "@/hooks/use-articles";
import { useCategories } from "@/hooks/use-categories";
import { useTags } from "@/hooks/use-tags";
import { StatsCard } from "./StatsCard";

const mockArticles = useArticles as ReturnType<typeof vi.fn>;
const mockCategories = useCategories as ReturnType<typeof vi.fn>;
const mockTags = useTags as ReturnType<typeof vi.fn>;

function mockData(overrides?: Partial<ReturnType<typeof mockArticles.mockReturnValue>>) {
  mockArticles.mockReturnValue({
    data: {
      articles: [
        { published_at: "2026-08-10T09:00:00Z" },
        { published_at: "2026-08-20T09:00:00Z" },
        { published_at: "2026-09-01T09:00:00Z" },
      ],
      total: 40,
    },
    isLoading: false,
    ...overrides,
  });
}

describe("StatsCard", () => {
  beforeEach(() => {
    mockArticles.mockReset();
    mockCategories.mockReset();
    mockTags.mockReset();
  });

  it("渲染文章/分类/标签计数（pad 两位）", () => {
    mockData();
    mockCategories.mockReturnValue({ data: { categories: [{ id: 1 }, { id: 2 }, { id: 3 }] }, isLoading: false });
    mockTags.mockReturnValue({ data: { tags: Array.from({ length: 8 }, (_, i) => ({ id: i, name: `t${i}`, slug: "", article_count: 0 })) }, isLoading: false });
    render(<StatsCard />);
    expect(screen.getByText("40")).toBeInTheDocument();
    expect(screen.getByText("03")).toBeInTheDocument();
    expect(screen.getByText("08")).toBeInTheDocument();
    expect(screen.getByText("文章")).toBeInTheDocument();
    expect(screen.getByText("标签")).toBeInTheDocument();
  });

  it("渲染 12 根迷你柱（近 12 月）", () => {
    mockData();
    mockCategories.mockReturnValue({ data: { categories: [] }, isLoading: false });
    mockTags.mockReturnValue({ data: { tags: [] }, isLoading: false });
    const { container } = render(<StatsCard />);
    expect(container.querySelectorAll("[data-bar-count]").length).toBe(12);
  });

  it("数据未加载时显示 animate-pulse 骨架", () => {
    mockArticles.mockReturnValue({ data: undefined, isLoading: true });
    mockCategories.mockReturnValue({ data: undefined, isLoading: true });
    mockTags.mockReturnValue({ data: undefined, isLoading: true });
    const { container } = render(<StatsCard />);
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
  });
});

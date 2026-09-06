import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/hooks/use-tags", () => ({ useTags: vi.fn() }));

import { useTags } from "@/hooks/use-tags";
import { TagCloudCard } from "./TagCloudCard";

const mockTags = useTags as ReturnType<typeof vi.fn>;

const sampleTags = [
  { id: 1, name: "前端", slug: "fe", article_count: 9 },
  { id: 2, name: "设计", slug: "design", article_count: 4 },
  { id: 3, name: "React", slug: "react", article_count: 2 },
  { id: 4, name: "AI", slug: "ai", article_count: 1 },
];

describe("TagCloudCard", () => {
  beforeEach(() => {
    mockTags.mockReset();
  });

  it("渲染标签文本与数量", () => {
    mockTags.mockReturnValue({ data: { tags: sampleTags }, isLoading: false });
    render(<TagCloudCard />);
    expect(screen.getByText("前端")).toBeInTheDocument();
    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("设计")).toBeInTheDocument();
  });

  it("最高计数的标签为琥珀 hot，其余按计数分级", () => {
    mockTags.mockReturnValue({ data: { tags: sampleTags }, isLoading: false });
    const { container } = render(<TagCloudCard />);
    expect(container.querySelector('[data-tag-variant="hot"]')?.textContent).toBe("前端");
    expect(container.querySelector('[data-tag-variant="teal"]')?.textContent).toBe("设计");
    expect(container.querySelector('[data-tag-size="sm"]')?.textContent).toBe("React");
  });

  it("无标签时显示空态", () => {
    mockTags.mockReturnValue({ data: { tags: [] }, isLoading: false });
    render(<TagCloudCard />);
    expect(screen.getByText("暂无标签")).toBeInTheDocument();
  });

  it("加载中显示骨架屏", () => {
    mockTags.mockReturnValue({ data: undefined, isLoading: true });
    const { container } = render(<TagCloudCard />);
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
  });
});

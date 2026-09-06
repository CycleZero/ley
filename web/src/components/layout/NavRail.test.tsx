import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { House } from "lucide-react";
import { NavRail } from "./NavRail";

function renderRail(initialEntries: string[] = ["/"]) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <NavRail />
    </MemoryRouter>,
  );
}

describe("NavRail", () => {
  it("渲染 6 个导航链接（首页/文章/分类/标签/关于/搜索）", () => {
    renderRail();
    expect(screen.getByRole("link", { name: "首页" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "文章" })).toHaveAttribute("href", "/articles");
    expect(screen.getByRole("link", { name: "分类" })).toHaveAttribute("href", "/categories");
    expect(screen.getByRole("link", { name: "标签" })).toHaveAttribute("href", "/tags");
    expect(screen.getByRole("link", { name: "关于" })).toHaveAttribute("href", "/about");
    expect(screen.getByRole("link", { name: "搜索" })).toHaveAttribute("href", "/search");
  });

  it("当前路由项高亮（aria-current=page）", () => {
    renderRail(["/articles"]);
    expect(screen.getByRole("link", { name: "文章" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "首页" })).not.toHaveAttribute("aria-current");
  });

  it("items prop 可覆盖默认导航项", () => {
    render(
      <MemoryRouter>
        <NavRail items={[{ to: "/custom", label: "自定义", icon: House }]} />
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "自定义" })).toHaveAttribute("href", "/custom");
    expect(screen.queryByRole("link", { name: "首页" })).not.toBeInTheDocument();
  });
});

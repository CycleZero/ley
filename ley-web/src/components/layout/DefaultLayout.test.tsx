import { beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import DefaultLayout from "./DefaultLayout";

const user = {
  id: 1,
  username: "tester",
  email: "t@e.com",
  avatar: "",
  bio: "",
  role: "user",
};

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <DefaultLayout />
    </MemoryRouter>,
  );
}

describe("DefaultLayout", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, ready: true });
    document.cookie = "ley_at=; path=/; max-age=0";
  });

  it("渲染主导航链接", () => {
    renderLayout();
    // 桌面 + 移动端导航各渲染一份
    expect(screen.getAllByRole("link", { name: "首页" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "文章" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "分类" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "标签" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "关于" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByRole("link", { name: "搜索" })).toBeInTheDocument();
  });

  it("未登录显示登录/注册按钮", () => {
    renderLayout();
    expect(screen.getByRole("link", { name: /登录/ })).toHaveAttribute("href", "/login");
    expect(screen.getByRole("link", { name: "注册" })).toHaveAttribute("href", "/register");
    expect(screen.queryByText("tester")).not.toBeInTheDocument();
  });

  it("已登录显示用户名与退出按钮，不显示注册", () => {
    document.cookie = "ley_at=at-1; path=/";
    useAuthStore.setState({ user, ready: true });
    renderLayout();
    expect(screen.getByText("tester")).toBeInTheDocument();
    expect(screen.getByTitle("退出登录")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "注册" })).not.toBeInTheDocument();
  });

  it("管理员显示后台入口", () => {
    document.cookie = "ley_at=at-1; path=/";
    useAuthStore.setState({ user: { ...user, role: "admin" }, ready: true });
    renderLayout();
    expect(screen.getByTitle("管理后台")).toBeInTheDocument();
  });

  it("非管理员不显示后台入口", () => {
    document.cookie = "ley_at=at-1; path=/";
    useAuthStore.setState({ user, ready: true });
    renderLayout();
    expect(screen.queryByTitle("管理后台")).not.toBeInTheDocument();
  });

  it("主题切换按钮存在", () => {
    renderLayout();
    expect(screen.getByLabelText("切换到暗色模式")).toBeInTheDocument();
  });
});

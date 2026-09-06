import { beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import RequireAuth from "./RequireAuth";
import RequireAdmin from "./RequireAdmin";

const user = {
  id: 1,
  username: "tester",
  email: "t@e.com",
  avatar: "",
  bio: "",
  role: "user" as const,
};

function renderAuth(guard: React.ComponentType) {
  const Guard = guard;
  return render(
    <MemoryRouter initialEntries={["/protected"]}>
      <Routes>
        <Route element={<Guard />}>
          <Route path="/protected" element={<div>受保护内容</div>} />
        </Route>
        <Route path="/login" element={<div>登录页</div>} />
        <Route path="/" element={<div>首页</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RequireAuth", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, ready: false });
  });

  it("ready 前显示加载", () => {
    renderAuth(RequireAuth);
    expect(screen.getByRole("status", { name: "加载中" })).toBeInTheDocument();
  });

  it("未登录重定向到 /login", () => {
    useAuthStore.setState({ user: null, ready: true });
    renderAuth(RequireAuth);
    expect(screen.getByText("登录页")).toBeInTheDocument();
    expect(screen.queryByText("受保护内容")).not.toBeInTheDocument();
  });

  it("已登录渲染子路由", () => {
    useAuthStore.setState({ user, ready: true });
    renderAuth(RequireAuth);
    expect(screen.getByText("受保护内容")).toBeInTheDocument();
  });
});

describe("RequireAdmin", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, ready: false });
  });

  it("ready 前显示加载", () => {
    renderAuth(RequireAdmin);
    expect(screen.getByRole("status", { name: "加载中" })).toBeInTheDocument();
  });

  it("未登录重定向到 /login", () => {
    useAuthStore.setState({ user: null, ready: true });
    renderAuth(RequireAdmin);
    expect(screen.getByText("登录页")).toBeInTheDocument();
  });

  it("普通用户重定向到首页", () => {
    useAuthStore.setState({ user, ready: true });
    renderAuth(RequireAdmin);
    expect(screen.getByText("首页")).toBeInTheDocument();
    expect(screen.queryByText("受保护内容")).not.toBeInTheDocument();
  });

  it("管理员渲染子路由", () => {
    useAuthStore.setState({ user: { ...user, role: "admin" }, ready: true });
    renderAuth(RequireAdmin);
    expect(screen.getByText("受保护内容")).toBeInTheDocument();
  });
});

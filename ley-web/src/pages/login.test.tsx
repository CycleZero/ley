import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { Toaster } from "sonner";
import type { Mock } from "vitest";
import { apiClient } from "@/lib/api-client";

vi.mock("@/lib/api-client", () => ({
  api: vi.fn(),
  apiClient: vi.fn(),
}));

import LoginPage from "./login";

const mockApi = apiClient as unknown as Mock;

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <Toaster />
      <LoginPage />
    </MemoryRouter>,
  );
}

describe("LoginPage", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("渲染表单元素", () => {
    renderPage();
    expect(screen.getByLabelText("用户名 / 邮箱")).toBeInTheDocument();
    expect(screen.getByLabelText("密码")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "登 录" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /注册/ })).toHaveAttribute("href", "/register");
  });

  it("空账号提交显示校验错误", async () => {
    renderPage();
    await userEvent.click(screen.getByRole("button", { name: "登 录" }));
    expect(await screen.findByText("请输入用户名或邮箱")).toBeInTheDocument();
    expect(mockApi).not.toHaveBeenCalled();
  });

  it("密码过短显示校验错误", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名 / 邮箱"), "tester");
    await userEvent.type(screen.getByLabelText("密码"), "123");
    await userEvent.click(screen.getByRole("button", { name: "登 录" }));
    expect(await screen.findByText("密码至少 6 位")).toBeInTheDocument();
    expect(mockApi).not.toHaveBeenCalled();
  });

  it("登录成功写入 cookie 并跳转首页", async () => {
    mockApi.mockResolvedValueOnce({
      user: { id: 1, username: "tester", email: "t@e.com", avatar: "", bio: "", role: "user" },
      token_pair: { access_token: "at-1", refresh_token: "rt-1", expires_in: 900 },
    });
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名 / 邮箱"), "tester");
    await userEvent.type(screen.getByLabelText("密码"), "password123");
    await userEvent.click(screen.getByRole("button", { name: "登 录" }));

    await waitFor(() => {
      expect(mockApi).toHaveBeenCalledWith("/auth/login", {
        method: "POST",
        body: { account: "tester", password: "password123" },
      });
    });
    expect(document.cookie).toContain("ley_at=at-1");
  });

  it("登录失败显示错误提示", async () => {
    mockApi.mockRejectedValueOnce(new Error("账号或密码错误"));
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名 / 邮箱"), "tester");
    await userEvent.type(screen.getByLabelText("密码"), "wrongpass1");
    await userEvent.click(screen.getByRole("button", { name: "登 录" }));
    // sonner toast 渲染在 body
    expect(await screen.findByText("账号或密码错误")).toBeInTheDocument();
  });
});

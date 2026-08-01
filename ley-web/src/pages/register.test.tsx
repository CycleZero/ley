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

import RegisterPage from "./register";

const mockApi = apiClient as unknown as Mock;

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/register"]}>
      <Toaster />
      <RegisterPage />
    </MemoryRouter>,
  );
}

async function fillValidForm() {
  await userEvent.type(screen.getByLabelText("用户名"), "tester");
  await userEvent.type(screen.getByLabelText("邮箱"), "t@example.com");
  await userEvent.type(screen.getByLabelText("密码"), "Password123");
  await userEvent.type(screen.getByLabelText("确认密码"), "Password123");
}

describe("RegisterPage", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("渲染表单元素", () => {
    renderPage();
    expect(screen.getByLabelText("用户名")).toBeInTheDocument();
    expect(screen.getByLabelText("邮箱")).toBeInTheDocument();
    expect(screen.getByLabelText("密码")).toBeInTheDocument();
    expect(screen.getByLabelText("确认密码")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /登录/ })).toHaveAttribute("href", "/login");
  });

  it("用户名太短报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "ab");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("用户名至少 3 个字符")).toBeInTheDocument();
  });

  it("用户名含非法字符报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "bad name!");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("用户名只能包含字母、数字、下划线和连字符")).toBeInTheDocument();
  });

  it("邮箱格式错误报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "tester");
    await userEvent.type(screen.getByLabelText("邮箱"), "not-an-email");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("邮箱格式不正确")).toBeInTheDocument();
  });

  it("密码缺少大写字母报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "tester");
    await userEvent.type(screen.getByLabelText("邮箱"), "t@example.com");
    await userEvent.type(screen.getByLabelText("密码"), "password123");
    await userEvent.type(screen.getByLabelText("确认密码"), "password123");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("密码必须包含大写字母")).toBeInTheDocument();
  });

  it("密码缺少数字报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "tester");
    await userEvent.type(screen.getByLabelText("邮箱"), "t@example.com");
    await userEvent.type(screen.getByLabelText("密码"), "Passwordabc");
    await userEvent.type(screen.getByLabelText("确认密码"), "Passwordabc");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("密码必须包含数字")).toBeInTheDocument();
  });

  it("两次密码不一致报错", async () => {
    renderPage();
    await userEvent.type(screen.getByLabelText("用户名"), "tester");
    await userEvent.type(screen.getByLabelText("邮箱"), "t@example.com");
    await userEvent.type(screen.getByLabelText("密码"), "Password123");
    await userEvent.type(screen.getByLabelText("确认密码"), "Password124");
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));
    expect(await screen.findByText("两次输入的密码不一致")).toBeInTheDocument();
  });

  it("注册成功调用 API 并写入 cookie", async () => {
    mockApi.mockResolvedValueOnce({
      user: { id: 1, username: "tester", email: "t@example.com", avatar: "", bio: "", role: "user" },
      token_pair: { access_token: "at-1", refresh_token: "rt-1", expires_in: 900 },
    });
    renderPage();
    await fillValidForm();
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));

    await waitFor(() => {
      expect(mockApi).toHaveBeenCalledWith("/auth/register", {
        method: "POST",
        body: { username: "tester", email: "t@example.com", password: "Password123" },
      });
    });
    expect(document.cookie).toContain("ley_at=at-1");
  });

  it("注册失败显示错误提示且不写 cookie", async () => {
    mockApi.mockRejectedValueOnce(new Error("用户名已存在"));
    renderPage();
    await fillValidForm();
    await userEvent.click(screen.getByRole("button", { name: "注 册" }));

    expect(await screen.findByText("用户名已存在")).toBeInTheDocument();
    expect(document.cookie).not.toContain("ley_at");
  });
});

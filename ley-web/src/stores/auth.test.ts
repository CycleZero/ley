import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Mock } from "vitest";
import { apiClient } from "@/lib/api-client";

// mock api-client：auth store 的唯一数据来源
vi.mock("@/lib/api-client", () => ({
  api: vi.fn(),
  apiClient: vi.fn(),
}));

import { useAuthStore } from "./auth";

const mockApi = apiClient as unknown as Mock;

const user = {
  id: 1,
  username: "tester",
  email: "tester@example.com",
  avatar: "",
  bio: "",
  role: "admin",
};

const tokenPair = {
  access_token: "at-1",
  refresh_token: "rt-1",
  expires_in: 900,
};

function setCookie(name: string, value: string) {
  document.cookie = `${name}=${value}; path=/`;
}

describe("auth store", () => {
  beforeEach(() => {
    mockApi.mockReset();
    useAuthStore.setState({ user: null, ready: false });
  });

  it("login 写入 cookie 并保存 user", async () => {
    mockApi.mockResolvedValueOnce({ user, token_pair: tokenPair });

    await useAuthStore.getState().login("tester", "password123");

    expect(document.cookie).toContain("ley_at=at-1");
    expect(document.cookie).toContain("ley_rt=rt-1");
    expect(useAuthStore.getState().user).toEqual(user);
    // 请求打到 /auth/login
    expect(mockApi).toHaveBeenCalledWith("/auth/login", { method: "POST", body: { account: "tester", password: "password123" } });
  });

  it("register 写入 cookie 并保存 user", async () => {
    mockApi.mockResolvedValueOnce({ user, token_pair: tokenPair });

    await useAuthStore.getState().register("tester", "t@e.com", "Password123");

    expect(document.cookie).toContain("ley_at=at-1");
    expect(useAuthStore.getState().user?.username).toBe("tester");
  });

  it("logout 调用后端、清除 cookie 与 user（即使后端失败也清本地）", async () => {
    mockApi.mockRejectedValueOnce(new Error("网络错误"));
    setCookie("ley_at", "at-1");
    setCookie("ley_rt", "rt-1");
    useAuthStore.setState({ user });

    await useAuthStore.getState().logout();

    expect(document.cookie).not.toContain("ley_at");
    expect(document.cookie).not.toContain("ley_rt");
    expect(useAuthStore.getState().user).toBeNull();
  });

  it("init 无 token 直接 ready", async () => {
    await useAuthStore.getState().init();
    expect(useAuthStore.getState().ready).toBe(true);
    expect(useAuthStore.getState().user).toBeNull();
    expect(mockApi).not.toHaveBeenCalled();
  });

  it("init 有 token 时拉取 /users/me", async () => {
    setCookie("ley_at", "at-1");
    mockApi.mockResolvedValueOnce({ user });

    await useAuthStore.getState().init();

    expect(mockApi).toHaveBeenCalledWith("/users/me");
    expect(useAuthStore.getState().user).toEqual(user);
    expect(useAuthStore.getState().ready).toBe(true);
  });

  it("init 遇 401 时尝试 refresh 并轮换 cookie", async () => {
    setCookie("ley_at", "expired");
    setCookie("ley_rt", "rt-1");
    mockApi.mockRejectedValueOnce(new Error("unauthorized"));
    mockApi.mockResolvedValueOnce({
      user,
      token_pair: { access_token: "new-at", refresh_token: "new-rt" },
    });

    await useAuthStore.getState().init();

    expect(mockApi).toHaveBeenCalledWith("/auth/refresh", { method: "POST", body: { refresh_token: "rt-1" } });
    expect(document.cookie).toContain("new-at");
    expect(useAuthStore.getState().user).toEqual(user);
  });

  it("init refresh 也失败时清 cookie 并置未登录", async () => {
    setCookie("ley_at", "expired");
    setCookie("ley_rt", "rt-1");
    mockApi.mockRejectedValueOnce(new Error("unauthorized"));
    mockApi.mockRejectedValueOnce(new Error("refresh failed"));

    await useAuthStore.getState().init();

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().ready).toBe(true);
    expect(document.cookie).not.toContain("ley_at");
  });

  it("updateProfile 更新 user", async () => {
    mockApi.mockResolvedValueOnce({ user: { ...user, bio: "新简介" } });

    await useAuthStore.getState().updateProfile({ bio: "新简介" });

    expect(mockApi).toHaveBeenCalledWith("/users/me", { method: "PUT", body: { bio: "新简介" } });
    expect(useAuthStore.getState().user?.bio).toBe("新简介");
  });
});

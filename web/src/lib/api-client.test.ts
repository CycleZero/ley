import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api-client";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function setCookie(name: string, value: string) {
  document.cookie = `${name}=${value}; path=/`;
}

describe("api-client", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  it("有 token 时注入 Authorization header 并解包 data", async () => {
    setCookie("ley_at", "token-123");
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, msg: "ok", data: { ok: true } }));

    const data = await api<{ ok: boolean }>("/test");
    expect(data).toEqual({ ok: true });

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("/api/v1/test");
    const headers = init?.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer token-123");
  });

  it("无 token 时不带 Authorization", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, msg: "ok", data: null }));

    await api("/test");
    const [, init] = fetchMock.mock.calls[0];
    const headers = init?.headers as Headers;
    expect(headers.get("Authorization")).toBeNull();
  });

  it("业务错误（code !== 0）抛出 msg", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: 1001, msg: "参数错误", data: null }));

    await expect(api("/test")).rejects.toThrow("参数错误");
  });

  it("401 时用 refresh token 刷新并重试原请求", async () => {
    setCookie("ley_at", "expired-at");
    setCookie("ley_rt", "rt-1");
    const fetchMock = vi.mocked(fetch);
    // 1) 原请求 401
    fetchMock.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));
    // 2) refresh 成功，返回新 token 对
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        code: 0,
        msg: "ok",
        data: { token_pair: { access_token: "new-at", refresh_token: "new-rt" } },
      }),
    );
    // 3) 重试成功
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, msg: "ok", data: { ok: true } }));

    const data = await api<{ ok: boolean }>("/test");
    expect(data).toEqual({ ok: true });

    // refresh 请求打向 /auth/refresh
    const refreshUrl = String(fetchMock.mock.calls[1][0]);
    expect(refreshUrl).toBe("/api/v1/auth/refresh");
    // cookie 已轮换
    expect(document.cookie).toContain("new-at");
    expect(document.cookie).toContain("new-rt");
    // 重试请求携带新 token
    const [, retryInit] = fetchMock.mock.calls[2];
    const headers = retryInit?.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer new-at");
  });

  it("401 且刷新失败时清除 cookie 并抛错", async () => {
    setCookie("ley_at", "expired-at");
    setCookie("ley_rt", "bad-rt");
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));
    fetchMock.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 })); // refresh 也 401

    await expect(api("/test")).rejects.toThrow();
    expect(document.cookie).not.toContain("ley_at");
    expect(document.cookie).not.toContain("ley_rt");
  });

  it("401 只重试一次（_retry 标记）", async () => {
    setCookie("ley_at", "expired-at");
    setCookie("ley_rt", "rt-1");
    const fetchMock = vi.mocked(fetch);
    // 原请求 401 → refresh 成功 → 重试仍 401 → 不再重试，直接抛错
    fetchMock.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        code: 0,
        msg: "ok",
        data: { token_pair: { access_token: "new-at", refresh_token: "new-rt" } },
      }),
    );
    fetchMock.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));

    await expect(api("/test")).rejects.toThrow();
    // 总共 3 次请求（原请求 + refresh + 1 次重试），没有第 4 次
    expect(fetchMock.mock.calls).toHaveLength(3);
  });
});

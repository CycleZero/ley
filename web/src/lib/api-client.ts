import { ofetch } from "ofetch";
import type { FetchOptions } from "ofetch";

// 统一响应格式 (Gateway wrapresp)
interface GatewayResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// 请求选项附加重试标记（TS 不识别，用弱类型包装）
interface RetriableOptions extends Omit<FetchOptions<"json">, "responseType"> {
  _retry?: boolean;
}

function getCookie(name: string): string | undefined {
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match?.[1];
}

function setTokenCookies(accessToken: string, refreshToken: string) {
  document.cookie = `ley_at=${accessToken}; path=/; max-age=900; SameSite=Lax`;
  document.cookie = `ley_rt=${refreshToken}; path=/; max-age=604800; SameSite=Lax`;
}

function clearTokenCookies() {
  document.cookie = "ley_at=; path=/; max-age=0";
  document.cookie = "ley_rt=; path=/; max-age=0";
}

let refreshPromise: Promise<boolean> | null = null;

/** 用 refresh token 换取新 token 对（单飞：并发 401 只刷新一次） */
function refreshToken(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const refreshToken = getCookie("ley_rt");
      if (!refreshToken) return false;
      try {
        const res = await rawClient<GatewayResponse<{ token_pair: { access_token: string; refresh_token: string } }>>(
          "/auth/refresh",
          { method: "POST", body: { refresh_token: refreshToken } },
        );
        if (res.code !== 0) return false;
        setTokenCookies(res.data.token_pair.access_token, res.data.token_pair.refresh_token);
        return true;
      } catch {
        return false;
      } finally {
        refreshPromise = null;
      }
    })();
  }
  return refreshPromise;
}

// 底层客户端：只负责 Token 注入
const rawClient = ofetch.create({
  baseURL: "/api/v1",
  async onRequest({ options }) {
    const token = getCookie("ley_at");
    if (token) {
      const headers = new Headers(options.headers);
      headers.set("Authorization", `Bearer ${token}`);
      options.headers = headers;
    }
  },
});

/**
 * 统一请求入口：Bearer 注入 + 401 单飞刷新重试 + 解包 Gateway 统一响应
 *
 * 页面/组件禁止直接使用 ofetch/fetch，一律走此函数（经 hooks/use-*.ts）。
 * 返回解包后的 data（code !== 0 时抛错）。
 */
export async function api<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const opts = options as RetriableOptions;

  let res: GatewayResponse<T>;
  try {
    res = await rawClient<GatewayResponse<T>>(path, opts);
  } catch (err) {
    // 401 → 刷新 token 后重试一次
    const status = (err as { response?: { status?: number } } | null)?.response?.status;
    if (status === 401 && !opts._retry) {
      opts._retry = true;
      const ok = await refreshToken();
      if (ok) {
        const newToken = getCookie("ley_at");
        if (newToken) {
          const headers = new Headers(opts.headers);
          headers.set("Authorization", `Bearer ${newToken}`);
          opts.headers = headers;
          res = await rawClient<GatewayResponse<T>>(path, opts);
        } else {
          clearTokenCookies();
          throw err;
        }
      } else {
        // 刷新失败 → 清除 cookie，页面自行处理重定向
        clearTokenCookies();
        throw err;
      }
    } else {
      throw err;
    }
  }

  if (res.code !== 0) {
    throw new Error(res.msg || "请求失败");
  }
  return res.data;
}

// 兼容旧命名（auth store 等处的调用）
export const apiClient = api;

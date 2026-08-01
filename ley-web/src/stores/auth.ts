import { create } from "zustand";
import { apiClient } from "@/lib/api-client";

interface User {
  id: number;
  username: string;
  email: string;
  avatar: string;
  bio: string;
  role: string;
}

function getCookie(name: string): string | undefined {
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match?.[1];
}

interface AuthState {
  user: User | null;
  ready: boolean; // init 完成后为 true
  login: (account: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  init: () => Promise<void>;
  updateProfile: (data: { avatar?: string; bio?: string }) => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  ready: false,

  login: async (account, password) => {
    const res = (await apiClient("/auth/login", {
      method: "POST",
      body: { account, password },
    })) as { user: User; token_pair: { access_token: string; refresh_token: string; expires_in: number } };
    document.cookie = `ley_at=${res.token_pair.access_token}; path=/; max-age=900; SameSite=Lax`;
    document.cookie = `ley_rt=${res.token_pair.refresh_token}; path=/; max-age=604800; SameSite=Lax`;
    set({ user: res.user });
  },

  register: async (username, email, password) => {
    const res = (await apiClient("/auth/register", {
      method: "POST",
      body: { username, email, password },
    })) as { user: User; token_pair: { access_token: string; refresh_token: string; expires_in: number } };
    document.cookie = `ley_at=${res.token_pair.access_token}; path=/; max-age=900; SameSite=Lax`;
    document.cookie = `ley_rt=${res.token_pair.refresh_token}; path=/; max-age=604800; SameSite=Lax`;
    set({ user: res.user });
  },

  logout: async () => {
    const rt = getCookie("ley_rt");
    try {
      await apiClient("/auth/logout", {
        method: "POST",
        body: { refresh_token: rt },
      });
    } catch {
      // 即使服务端失败也清除本地状态
    }
    document.cookie = "ley_at=; path=/; max-age=0";
    document.cookie = "ley_rt=; path=/; max-age=0";
    set({ user: null });
  },

  init: async () => {
    const token = getCookie("ley_at");
    if (!token) {
      set({ ready: true });
      return;
    }
    try {
      const res = (await apiClient("/users/me")) as { user: User };
      set({ user: res.user, ready: true });
    } catch {
      // access token 过期，尝试 refresh
      const rt = getCookie("ley_rt");
      if (!rt) {
        set({ ready: true });
        return;
      }
      try {
        const refreshRes = (await apiClient("/auth/refresh", {
          method: "POST",
          body: { refresh_token: rt },
        })) as { user: User; token_pair: { access_token: string; refresh_token: string } };
        document.cookie = `ley_at=${refreshRes.token_pair.access_token}; path=/; max-age=900; SameSite=Lax`;
        document.cookie = `ley_rt=${refreshRes.token_pair.refresh_token}; path=/; max-age=604800; SameSite=Lax`;
        set({ user: refreshRes.user, ready: true });
      } catch {
        document.cookie = "ley_at=; path=/; max-age=0";
        document.cookie = "ley_rt=; path=/; max-age=0";
        set({ ready: true });
      }
    }
  },

  updateProfile: async (data) => {
    const res = (await apiClient("/users/me", {
      method: "PUT",
      body: data,
    })) as { user: User };
    set({ user: res.user });
  },
}));

// Computed helpers (in store for convenience)
export const useIsLoggedIn = () => {
  const user = useAuthStore((s) => s.user);
  return !!getCookie("ley_at") && !!user;
};
export const useIsAdmin = () => {
  const user = useAuthStore((s) => s.user);
  return user?.role === "admin";
};

import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Shiki 代码高亮在测试 worker 中初始化较慢，放宽异步查询超时
configure({ asyncUtilTimeout: 15_000 });

// 每个用例后清理 DOM 与残留 mock
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  document.cookie = "ley_at=; path=/; max-age=0";
  document.cookie = "ley_rt=; path=/; max-age=0";
  localStorage.clear();
});

// jsdom 无 matchMedia（ThemeToggle 等组件会用到，可选链兜底，这里补全避免误伤）
if (!window.matchMedia) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

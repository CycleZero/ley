import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { useSiteConfig } from "@/hooks/use-site";
import DefaultLayout from "./DefaultLayout";

vi.mock("@/hooks/use-site", () => ({
  useSiteConfig: vi.fn(),
  useUpdateSiteConfig: vi.fn(),
}));

const mockUseSiteConfig = vi.mocked(useSiteConfig);
type QueryResult = ReturnType<typeof useSiteConfig>;

const DEFAULT_CONFIG = {
  site_title: "测试站",
  site_subtitle: "副标",
  site_description: "",
  site_logo: "",
  site_favicon: "",
  seo_keywords: "",
  seo_description: "",
  social_github: "",
  social_twitter: "",
  social_email: "",
  footer_text: "© 测试站",
  icp_number: "",
  enable_likes: true,
};

function configResult(partial?: Partial<Record<string, unknown>>): QueryResult {
  return {
    data: { config: { ...DEFAULT_CONFIG, ...partial } },
  } as unknown as QueryResult;
}

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
    mockUseSiteConfig.mockReset();
    mockUseSiteConfig.mockReturnValue(configResult());
  });

  it("渲染主导航链接", () => {
    renderLayout();
    // 主导航由 NavRail 渲染（桌面左侧 + 移动底部各一份）
    expect(screen.getAllByRole("link", { name: "首页" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "文章" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "分类" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "标签" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: "关于" }).length).toBeGreaterThanOrEqual(1);
    // 搜索：NavRail 内一项 + 顶栏 icon 一项
    expect(screen.getAllByRole("link", { name: "搜索" }).length).toBeGreaterThanOrEqual(1);
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

  it("品牌显示 useSiteConfig 的 site_title", () => {
    renderLayout();
    expect(screen.getByText("测试站")).toBeInTheDocument();
  });

  it("site_title 为空时品牌回退「我的博客」", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ site_title: "" }));
    renderLayout();
    expect(screen.getByText("我的博客")).toBeInTheDocument();
  });

  it("footer 显示配置的 footer_text，不硬编码站名", () => {
    renderLayout();
    expect(screen.getByText("© 测试站")).toBeInTheDocument();
  });

  it("footer_text 为空时回退中性版权占位", () => {
    mockUseSiteConfig.mockReturnValue(configResult({ footer_text: "" }));
    renderLayout();
    expect(screen.getByText(`© ${new Date().getFullYear()}`)).toBeInTheDocument();
  });

  it("点击退出登录调用 logout 并跳转首页", () => {
    document.cookie = "ley_at=at-1; path=/";
    useAuthStore.setState({ user, ready: true });
    const logoutSpy = vi.spyOn(useAuthStore.getState(), "logout").mockResolvedValue(undefined);
    renderLayout();
    fireEvent.click(screen.getByTitle("退出登录"));
    expect(logoutSpy).toHaveBeenCalled();
  });
});

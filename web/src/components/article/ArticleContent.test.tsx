import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { ArticleContent } from "./ArticleContent";

// react-markdown + 异步 rehype 插件（Shiki）依赖 Suspense 挂起/恢复
function renderContent(content: string) {
  return render(
    <Suspense fallback={<div>加载中</div>}>
      <ArticleContent content={content} />
    </Suspense>,
  );
}

describe("ArticleContent", () => {
  it("渲染标题与段落", async () => {
    renderContent("# 标题\n\n这是正文段落");
    expect(await screen.findByRole("heading", { level: 1, name: "标题" })).toBeInTheDocument();
    expect(await screen.findByText("这是正文段落")).toBeInTheDocument();
  });

  it("渲染链接与加粗", async () => {
    renderContent("**加粗** 和 [链接](https://example.com)");
    expect(await screen.findByText("加粗")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "链接" })).toHaveAttribute("href", "https://example.com");
  });

  it("渲染 GFM 表格", async () => {
    renderContent("| a | b |\n|---|---|\n| 1 | 2 |");
    expect(await screen.findByRole("table")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("渲染任务列表（GFM checkbox）", async () => {
    renderContent("- [x] 已完成\n- [ ] 未完成");
    const checkboxes = await screen.findAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    expect(checkboxes[0]).toBeChecked();
    expect(checkboxes[1]).not.toBeChecked();
  });

  it("代码块使用 Shiki 高亮", async () => {
    renderContent("```go\npackage main\n```");
    const pre = await screen.findByText((_, el) => el?.tagName === "PRE" && el.className.includes("shiki"));
    expect(pre.className).toContain("shiki");
    // Shiki 会把代码拆成多个 span，textContent 应包含原文
    expect(pre.textContent).toContain("package main");
    // 语法 token 有主题着色（inline style 带 shiki 色变量）
    const token = pre.querySelector("span[style]");
    expect(token).not.toBeNull();
    expect(token!.getAttribute("style")).toContain("--shiki");
  });

  it("行内代码渲染", async () => {
    renderContent("运行 `go run` 命令");
    const inline = await screen.findByText("go run");
    expect(inline.tagName).toBe("CODE");
    expect(inline.closest("pre")).toBeNull();
  });

  it("引用块渲染", async () => {
    renderContent("> 引用内容");
    expect(await screen.findByText("引用内容")).toBeInTheDocument();
  });

  it("列表渲染", async () => {
    renderContent("- 第一项\n- 第二项");
    expect(await screen.findByText("第一项")).toBeInTheDocument();
    expect(screen.getByText("第二项")).toBeInTheDocument();
  });

  it("空内容安全渲染", async () => {
    const { container } = renderContent("");
    await waitFor(() => {
      expect(container.querySelector(".markdown-body")).toBeInTheDocument();
    });
  });
});

import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeToggle } from "./ThemeToggle";

describe("ThemeToggle", () => {
  it("初始读取 localStorage 偏好", () => {
    localStorage.setItem("ley_theme", "dark");
    render(<ThemeToggle />);
    expect(document.documentElement.dataset.theme).toBe("dark");
  });

  it("无偏好时跟随系统（默认 light）", () => {
    render(<ThemeToggle />);
    expect(document.documentElement.dataset.theme).toBe("light");
  });

  it("点击切换主题并持久化", async () => {
    render(<ThemeToggle />);
    await userEvent.click(screen.getByRole("button"));
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(localStorage.getItem("ley_theme")).toBe("dark");

    await userEvent.click(screen.getByRole("button"));
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(localStorage.getItem("ley_theme")).toBe("light");
  });

  it("aria-label 随主题变化", async () => {
    render(<ThemeToggle />);
    expect(screen.getByLabelText("切换到暗色模式")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button"));
    expect(screen.getByLabelText("切换到亮色模式")).toBeInTheDocument();
  });
});

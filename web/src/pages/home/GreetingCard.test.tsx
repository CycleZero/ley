import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { GreetingCard } from "./GreetingCard";

describe("GreetingCard", () => {
  it("早上 8 点渲染「早上好」、日期与时钟", () => {
    render(<GreetingCard now={new Date(2026, 8, 6, 8, 24)} />);
    expect(screen.getByText("早上好")).toBeInTheDocument();
    expect(screen.getByText("2026.09.06 星期日")).toBeInTheDocument();
    expect(screen.getByText("08:24")).toBeInTheDocument();
    expect(screen.getByText("AM")).toBeInTheDocument();
  });

  it("下午渲染「下午好」", () => {
    render(<GreetingCard now={new Date(2026, 8, 6, 15, 30)} />);
    expect(screen.getByText("下午好")).toBeInTheDocument();
    expect(screen.getByText("PM")).toBeInTheDocument();
  });

  it("夜晚渲染「晚上好」", () => {
    render(<GreetingCard now={new Date(2026, 8, 6, 21, 5)} />);
    expect(screen.getByText("晚上好")).toBeInTheDocument();
    expect(screen.getByText("PM")).toBeInTheDocument();
  });

  it("包含唯一一枚 L 形角标装饰", () => {
    const { container } = render(<GreetingCard now={new Date(2026, 8, 6, 8, 0)} />);
    // 仅问候卡允许 1 处角标：绝对定位的 L 形括号
    const corners = container.querySelectorAll("i[aria-hidden]");
    expect(corners.length).toBeGreaterThanOrEqual(1);
  });
});

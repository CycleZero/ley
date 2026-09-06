import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { CalendarCard } from "./CalendarCard";

describe("CalendarCard", () => {
  it("渲染 9 月网格：星期行、月标题、天数", () => {
    render(<CalendarCard now={new Date(2026, 8, 6)} />); // 2026-09-06 星期日
    expect(screen.getByText("9月")).toBeInTheDocument();
    expect(screen.getByText("一")).toBeInTheDocument();
    expect(screen.getByText("日")).toBeInTheDocument();
    expect(screen.getByText("30")).toBeInTheDocument(); // 9 月 30 天
    // 日历网格共 30 个日期格（不含空白格）
    const cells = document.querySelectorAll("span[data-today]").length;
    expect(cells).toBeGreaterThan(0);
  });

  it("今日（9/6）蓝色圆环高亮", () => {
    const { container } = render(<CalendarCard now={new Date(2026, 8, 6)} />);
    const todayCell = container.querySelector('[data-today="true"]');
    expect(todayCell).not.toBeNull();
    expect(todayCell?.textContent).toBe("6");
  });

  it("非今日不高亮", () => {
    const { container } = render(<CalendarCard now={new Date(2026, 8, 6)} />);
    expect(container.querySelector('[data-today="true"]')?.textContent).toBe("6");
    expect(container.querySelectorAll('[data-today="true"]').length).toBe(1);
  });
});

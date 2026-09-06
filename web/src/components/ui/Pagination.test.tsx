import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Pagination } from "./Pagination";

describe("Pagination", () => {
  it("总页数 <= 1 时不渲染", () => {
    const { container } = render(<Pagination page={1} total={5} pageSize={10} onChange={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("渲染页码并高亮当前页", () => {
    render(<Pagination page={2} total={50} pageSize={10} onChange={vi.fn()} />);
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    // 当前页高亮
    expect(screen.getByText("2").className).toContain("text-white");
  });

  it("点击页码触发 onChange", async () => {
    const onChange = vi.fn();
    render(<Pagination page={1} total={50} pageSize={10} onChange={onChange} />);
    await userEvent.click(screen.getByText("3"));
    expect(onChange).toHaveBeenCalledWith(3);
  });

  it("上一页/下一页按钮", async () => {
    const onChange = vi.fn();
    render(<Pagination page={3} total={50} pageSize={10} onChange={onChange} />);
    await userEvent.click(screen.getByLabelText("上一页"));
    expect(onChange).toHaveBeenCalledWith(2);
    await userEvent.click(screen.getByLabelText("下一页"));
    expect(onChange).toHaveBeenCalledWith(4);
  });

  it("第一页时上一页禁用", () => {
    render(<Pagination page={1} total={50} pageSize={10} onChange={vi.fn()} />);
    expect(screen.getByLabelText("上一页")).toBeDisabled();
  });

  it("页数多时显示省略号", () => {
    render(<Pagination page={5} total={200} pageSize={10} onChange={vi.fn()} />);
    // 20 页 → 1 … 4 5 6 … 20
    expect(screen.getAllByText("…").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("20")).toBeInTheDocument();
  });
});

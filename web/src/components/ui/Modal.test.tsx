import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Modal } from "./Modal";

describe("Modal", () => {
  it("open=false 时不渲染", () => {
    render(<Modal open={false} onClose={vi.fn()} title="标题" />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("open=true 时渲染标题、内容与 footer", () => {
    render(
      <Modal open onClose={vi.fn()} title="删除确认" footer={<button>确定</button>}>
        <p>确定删除吗？</p>
      </Modal>,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("删除确认")).toBeInTheDocument();
    expect(screen.getByText("确定删除吗？")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "确定" })).toBeInTheDocument();
  });

  it("点击关闭按钮触发 onClose", async () => {
    const onClose = vi.fn();
    render(<Modal open onClose={onClose} title="标题" />);
    await userEvent.click(screen.getByLabelText("关闭"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("点击遮罩触发 onClose", async () => {
    const onClose = vi.fn();
    render(<Modal open onClose={onClose} title="标题" />);
    // 遮罩是 dialog 的第一个子元素（absolute inset-0）
    const backdrop = screen.getByRole("dialog").firstElementChild!;
    await userEvent.click(backdrop as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("按 Esc 触发 onClose", async () => {
    const onClose = vi.fn();
    render(<Modal open onClose={onClose} title="标题" />);
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("open=false 时 Esc 不触发", async () => {
    const onClose = vi.fn();
    render(<Modal open={false} onClose={onClose} title="标题" />);
    await userEvent.keyboard("{Escape}");
    expect(onClose).not.toHaveBeenCalled();
  });
});

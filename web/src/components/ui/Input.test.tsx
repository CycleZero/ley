import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Field, Input, Select, Textarea } from "./Input";

describe("Input", () => {
  it("渲染并透传 placeholder/value", () => {
    render(<Input placeholder="请输入" value="内容" onChange={() => {}} />);
    const input = screen.getByPlaceholderText("请输入");
    expect(input).toHaveValue("内容");
  });

  it("支持受控输入", async () => {
    const onChange = vi.fn();
    render(<Input value="" onChange={onChange} aria-label="用户名" />);
    await userEvent.type(screen.getByLabelText("用户名"), "abc");
    expect(onChange).toHaveBeenCalled();
  });

  it("disabled 生效", () => {
    render(<Input disabled aria-label="禁用" />);
    expect(screen.getByLabelText("禁用")).toBeDisabled();
  });
});

describe("Textarea", () => {
  it("渲染并透传 rows", () => {
    render(<Textarea rows={4} aria-label="正文" />);
    expect(screen.getByLabelText("正文")).toHaveAttribute("rows", "4");
  });
});

describe("Select", () => {
  it("渲染选项并支持选择", async () => {
    const onChange = vi.fn();
    render(
      <Select value="" onChange={onChange} aria-label="分类">
        <option value="">全部</option>
        <option value="1">技术</option>
      </Select>,
    );
    await userEvent.selectOptions(screen.getByLabelText("分类"), "1");
    expect(onChange).toHaveBeenCalled();
  });
});

describe("Field", () => {
  it("渲染 label 与 hint", () => {
    render(
      <Field label="用户名" hint="3-32 位">
        <Input aria-label="用户名" />
      </Field>,
    );
    expect(screen.getByText("用户名")).toBeInTheDocument();
    expect(screen.getByText("3-32 位")).toBeInTheDocument();
  });

  it("有 error 时优先显示错误而非 hint", () => {
    render(
      <Field label="密码" error="密码太短" hint="至少 8 位">
        <Input aria-label="密码" />
      </Field>,
    );
    expect(screen.getByText("密码太短")).toBeInTheDocument();
    expect(screen.queryByText("至少 8 位")).not.toBeInTheDocument();
  });
});

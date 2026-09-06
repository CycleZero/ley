import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "./Button";
import { Badge } from "./Badge";
import { Spinner } from "./Spinner";
import { Empty } from "./Empty";
import { Card, CardTitle, CardBody } from "./Card";

describe("Button", () => {
  it("渲染 children 与默认类型 button", () => {
    render(<Button>保存</Button>);
    const btn = screen.getByRole("button", { name: "保存" });
    expect(btn).toHaveAttribute("type", "button");
  });

  it("type 可覆盖为 submit", () => {
    render(<Button type="submit">提交</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("type", "submit");
  });

  it("disabled 时不可点击", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick} disabled>确定</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("点击触发 onClick", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>确定</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("variant 应用对应 class", () => {
    render(<Button variant="danger">删除</Button>);
    expect(screen.getByRole("button").className).toContain("bg-red-500");
  });
});

describe("Badge", () => {
  it("渲染文本与默认样式", () => {
    render(<Badge>草稿</Badge>);
    const badge = screen.getByText("草稿");
    expect(badge.className).toContain("rounded-lg");
  });

  it("accent variant 应用 accent 色", () => {
    render(<Badge variant="accent">已发布</Badge>);
    expect(screen.getByText("已发布").className).toContain("accent");
  });
});

describe("Spinner", () => {
  it("渲染加载指示并带 aria-label", () => {
    render(<Spinner />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByLabelText("加载中")).toBeInTheDocument();
  });
});

describe("Empty", () => {
  it("默认文案", () => {
    render(<Empty />);
    expect(screen.getByText("暂无数据")).toBeInTheDocument();
  });

  it("自定义标题/描述/操作", () => {
    render(
      <Empty
        title="没有文章"
        description="去写一篇吧"
        action={<button>写文章</button>}
      />,
    );
    expect(screen.getByText("没有文章")).toBeInTheDocument();
    expect(screen.getByText("去写一篇吧")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "写文章" })).toBeInTheDocument();
  });
});

describe("Card", () => {
  it("组合渲染", () => {
    render(
      <Card>
        <CardTitle>标题</CardTitle>
        <CardBody>内容</CardBody>
      </Card>,
    );
    expect(screen.getByText("标题")).toBeInTheDocument();
    expect(screen.getByText("内容")).toBeInTheDocument();
  });
});

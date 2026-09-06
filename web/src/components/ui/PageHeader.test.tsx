import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { PageHeader } from "./PageHeader";

describe("PageHeader", () => {
  it("渲染 kicker/title/description/actions", () => {
    render(
      <PageHeader
        kicker="PROJECT"
        title="站点统计"
        description="数据面板说明"
        actions={<button type="button">刷新</button>}
      />,
    );
    expect(screen.getByText("PROJECT")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "站点统计" })).toBeInTheDocument();
    expect(screen.getByText("数据面板说明")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "刷新" })).toBeInTheDocument();
  });

  it("kicker/description/actions 可选，不传则不渲染", () => {
    render(<PageHeader title="只有标题" />);
    expect(screen.getByRole("heading", { name: "只有标题" })).toBeInTheDocument();
    expect(screen.queryByText("PROJECT")).not.toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});

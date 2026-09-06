import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { Backdrop } from "./Backdrop";

describe("Backdrop", () => {
  it("根元素含 aria-hidden 且 pointer-events-none", () => {
    const { container } = render(<Backdrop particles={false} />);
    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-hidden", "true");
    expect(root).toHaveClass("pointer-events-none");
  });

  it("particles=false 时不渲染粒子", () => {
    const { container } = render(<Backdrop particles={false} />);
    expect(container.querySelectorAll(".ley-particle")).toHaveLength(0);
  });

  it("默认渲染粒子，数量上限 16", () => {
    const { container } = render(<Backdrop />);
    const count = container.querySelectorAll(".ley-particle").length;
    expect(count).toBeGreaterThan(0);
    expect(count).toBeLessThanOrEqual(16);
  });
});

import { describe, expect, it } from "vitest";
import { cn, formatDate, timeAgo, formatCount } from "./utils";

describe("cn", () => {
  it("合并多个类名", () => {
    expect(cn("a", "b", "c")).toBe("a b c");
  });
  it("过滤 falsy 值", () => {
    expect(cn("a", false, null, undefined, "")).toBe("a");
  });
  it("无参返回空字符串", () => {
    expect(cn()).toBe("");
  });
});

describe("formatDate", () => {
  it("格式化 Date", () => {
    const d = new Date(2026, 6, 3, 9, 5); // 2026-07-03 09:05
    expect(formatDate(d)).toBe("2026-07-03 09:05");
  });
  it("格式化时间戳字符串", () => {
    expect(formatDate("2026-07-03T09:05:00Z")).toBe("2026-07-03 17:05");
  });
  it("非法输入原样返回", () => {
    expect(formatDate("not-a-date")).toBe("not-a-date");
  });
  it("月份与日期补零", () => {
    expect(formatDate(new Date(2026, 0, 2, 1, 3))).toBe("2026-01-02 01:03");
  });
});

describe("timeAgo", () => {
  it("刚刚", () => {
    expect(timeAgo(new Date())).toBe("刚刚");
  });
  it("n 分钟前", () => {
    const d = new Date(Date.now() - 5 * 60_000);
    expect(timeAgo(d)).toBe("5 分钟前");
  });
  it("n 小时前", () => {
    const d = new Date(Date.now() - 3 * 3_600_000);
    expect(timeAgo(d)).toBe("3 小时前");
  });
  it("n 天前", () => {
    const d = new Date(Date.now() - 2 * 86_400_000);
    expect(timeAgo(d)).toBe("2 天前");
  });
  it("超过 30 天显示日期", () => {
    const d = new Date(Date.now() - 40 * 86_400_000);
    expect(timeAgo(d)).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/);
  });
});

describe("formatCount", () => {
  it("千以下原样", () => {
    expect(formatCount(999)).toBe("999");
  });
  it("千级缩写", () => {
    expect(formatCount(1_200)).toBe("1.2k");
  });
  it("万级缩写", () => {
    expect(formatCount(12_345)).toBe("12.3k");
  });
  it("百万级缩写", () => {
    expect(formatCount(2_500_000)).toBe("2.5m");
  });
});

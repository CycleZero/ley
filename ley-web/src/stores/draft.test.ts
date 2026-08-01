import { describe, expect, it } from "vitest";
import { useDraftStore } from "./draft";

describe("draft store", () => {
  it("初始状态为空", () => {
    const s = useDraftStore.getState();
    expect(s.title).toBe("");
    expect(s.content).toBe("");
    expect(s.tagNames).toEqual([]);
    expect(s.categoryId).toBeNull();
  });

  it("setField 更新对应字段", () => {
    useDraftStore.getState().setField("title", "我的文章");
    useDraftStore.getState().setField("content", "正文内容");
    useDraftStore.getState().setField("tagNames", ["Go", "React"]);

    const s = useDraftStore.getState();
    expect(s.title).toBe("我的文章");
    expect(s.content).toBe("正文内容");
    expect(s.tagNames).toEqual(["Go", "React"]);
  });

  it("reset 恢复初始状态", () => {
    useDraftStore.getState().setField("title", "标题");
    useDraftStore.getState().setField("categoryId", 3);

    useDraftStore.getState().reset();

    const s = useDraftStore.getState();
    expect(s.title).toBe("");
    expect(s.categoryId).toBeNull();
    expect(s.coverImage).toBe("");
  });
});

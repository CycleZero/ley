import { create } from "zustand";

interface DraftState {
  title: string;
  content: string;
  excerpt: string;
  coverImage: string;
  categoryId: number | null;
  tagNames: string[];
  setField: (field: string, value: unknown) => void;
  reset: () => void;
}

const initial = {
  title: "",
  content: "",
  excerpt: "",
  coverImage: "",
  categoryId: null as number | null,
  tagNames: [] as string[],
};

export const useDraftStore = create<DraftState>((set) => ({
  ...initial,
  setField: (field, value) => set({ [field]: value }),
  reset: () => set(initial),
}));

import { create } from "zustand";

interface Toast {
  id: number;
  message: string;
  type: "success" | "error" | "info";
}

interface UIState {
  toasts: Toast[];
  headerVisible: boolean;
  mobileSidebarOpen: boolean;
  pageTitle: string;
  toast: (message: string, type?: Toast["type"]) => void;
  removeToast: (id: number) => void;
  setHeaderVisible: (v: boolean) => void;
  toggleMobileSidebar: () => void;
  setPageTitle: (title: string) => void;
}

let toastId = 0;

export const useUIStore = create<UIState>((set) => ({
  toasts: [],
  headerVisible: true,
  mobileSidebarOpen: false,
  pageTitle: "Ley",

  toast: (message, type = "info") => {
    const id = ++toastId;
    set((s) => ({ toasts: [...s.toasts, { id, message, type }] }));
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    }, 3000);
  },

  removeToast: (id) => {
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
  },

  setHeaderVisible: (v) => set({ headerVisible: v }),
  toggleMobileSidebar: () => set((s) => ({ mobileSidebarOpen: !s.mobileSidebarOpen })),
  setPageTitle: (title) => set({ pageTitle: title }),
}));

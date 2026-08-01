import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Suspense, useEffect, useState } from "react";
import { RouterProvider } from "react-router-dom";
import { Toaster } from "sonner";
import { router } from "./routes.tsx"; // TSX file with JSX elements
import { useAuthStore } from "./stores/auth";
import "./lib/i18n";
import "./styles/globals.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  const [ready, setReady] = useState(false);
  const init = useAuthStore((s) => s.init);

  useEffect(() => {
    init().then(() => setReady(true));
  }, [init]);

  if (!ready) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[var(--bg-global)]">
        <div className="size-8 border-2 border-[var(--accent)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-screen bg-[var(--bg-global)]">
          <div className="size-8 border-2 border-[var(--accent)] border-t-transparent rounded-full animate-spin" />
        </div>
      }
    >
      <RouterProvider router={router} />
    </Suspense>
  );
}

export default function Root() {
  return (
    <QueryClientProvider client={queryClient}>
      <Toaster richColors position="top-center" />
      <App />
    </QueryClientProvider>
  );
}

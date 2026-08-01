import { Outlet } from "react-router-dom";

export default function CleanLayout() {
  return (
    <div className="min-h-screen bg-[var(--bg-global)]">
      <header className="sticky top-0 z-40 h-14 border-b border-[var(--border)] bg-[var(--bg-card)]/80 backdrop-blur flex items-center px-6">
        <a href="/" className="text-lg font-medium text-[var(--text-primary)]">
          Ley
        </a>
      </header>
      <main className="mx-auto max-w-3xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  );
}

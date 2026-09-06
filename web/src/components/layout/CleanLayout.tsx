import { Outlet } from "react-router-dom";
import { Backdrop } from "./Backdrop";
import { SiteBrand } from "./SiteBrand";

export default function CleanLayout() {
  return (
    <div className="min-h-screen">
      <Backdrop />
      <header className="sticky top-0 z-40 flex h-14 items-center border-b border-[var(--border)] bg-[var(--bg-card)]/80 px-6 backdrop-blur">
        <SiteBrand />
      </header>
      <main className="mx-auto w-full max-w-3xl px-6 py-10">
        <Outlet />
      </main>
    </div>
  );
}

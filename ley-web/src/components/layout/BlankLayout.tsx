import { Outlet } from "react-router-dom";

export default function BlankLayout() {
  return (
    <div className="min-h-screen bg-[var(--bg-global)] flex items-center justify-center p-4">
      <div className="w-full max-w-sm card p-8">
        <Outlet />
      </div>
    </div>
  );
}

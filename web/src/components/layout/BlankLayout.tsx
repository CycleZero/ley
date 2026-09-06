import { Outlet } from "react-router-dom";
import { Backdrop } from "./Backdrop";
import { SiteBrand } from "./SiteBrand";

export default function BlankLayout() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <Backdrop />
      <SiteBrand className="mb-8" />
      <div className="w-full max-w-sm card p-8">
        <Outlet />
      </div>
    </div>
  );
}

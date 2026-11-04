import type { ComponentType } from "react";
import { Link, createBrowserRouter } from "react-router-dom";
import AppLayout from "@/app/app-layout";
import OrdersLayout from "@/app/orders/layout";

type PageModule = { default: ComponentType };

function lazyPage(load: () => Promise<PageModule>) {
  return async () => ({ Component: (await load()).default });
}

function RouteLoading() {
  return (
    <main className="home-shell" aria-busy="true">
      <section className="home-hero" aria-live="polite">
        <h1>正在加载页面</h1>
        <p className="home-copy">请稍候…</p>
      </section>
    </main>
  );
}

function NotFoundPage() {
  return (
    <main className="home-shell">
      <section className="home-hero" aria-labelledby="not-found-title">
        <h1 id="not-found-title">页面不存在</h1>
        <p className="home-copy">请检查地址或返回工作台。</p>
        <Link className="home-link primary" to="/">
          返回工作台
        </Link>
      </section>
    </main>
  );
}

export const router = createBrowserRouter([
  {
    Component: AppLayout,
    HydrateFallback: RouteLoading,
    children: [
      { index: true, lazy: lazyPage(() => import("@/app/home-page")) },
      {
        path: "orders",
        Component: OrdersLayout,
        children: [
          { index: true, lazy: lazyPage(() => import("@/app/orders/page")) },
          { path: "new", lazy: lazyPage(() => import("@/app/orders/new-page")) },
          { path: ":orderId", lazy: lazyPage(() => import("@/app/orders/detail-page")) },
        ],
      },
      { path: "admin/audit-events", lazy: lazyPage(() => import("@/app/admin/audit-events/page")) },
      { path: "console", lazy: lazyPage(() => import("@/app/console/page")) },
      { path: "issues", lazy: lazyPage(() => import("@/app/issues/page")) },
      { path: "*", Component: NotFoundPage },
    ],
  },
]);

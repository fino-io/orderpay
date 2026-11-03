import type { ComponentType } from "react";
import { Link, createBrowserRouter } from "react-router-dom";
import AppLayout from "@/app/app-layout";

type PageModule = { default: ComponentType };

function lazyPage(load: () => Promise<PageModule>) {
  return async () => ({ Component: (await load()).default });
}

function RouteLoading() {
  return (
    <main className="home-shell" aria-busy="true">
      <section className="home-hero" aria-live="polite">
        <p className="eyebrow">FINO</p>
        <h1>Loading page</h1>
        <p className="home-copy">Loading…</p>
      </section>
    </main>
  );
}

function NotFoundPage() {
  return (
    <main className="home-shell">
      <section className="home-hero" aria-labelledby="not-found-title">
        <p className="eyebrow">404</p>
        <h1 id="not-found-title">Page not found.</h1>
        <p className="home-copy">The requested page does not exist.</p>
        <Link className="home-link primary" to="/">Return home</Link>
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
      { path: "console", lazy: lazyPage(() => import("@/app/console/page")) },
      { path: "issues", lazy: lazyPage(() => import("@/app/issues/page")) },
      { path: "*", Component: NotFoundPage },
    ],
  },
]);

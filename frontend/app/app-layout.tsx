import { useEffect } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import { ThemeToggle } from "@/app/theme-toggle";

const projectTitle = "orderpay | fino";
const projectName = "orderpay";
const monitorHref = "/monitor/d/service-overview/service-overview?from=now-1h&to=now&var-job=All&refresh=30s";
const logsHref = "/monitor/d/service-logs/service-logs?from=now-15m&to=now&var-service=All&var-level=All";
const tracePane = {
  trace: {
    datasource: "tempo",
    queries: [{ refId: "A", query: "{}", queryType: "traceql", limit: 20, tableType: "traces", datasource: { type: "tempo", uid: "tempo" } }],
    range: { from: "now-15m", to: "now" },
    compact: false,
  },
};
const traceHref = `/monitor/explore?schemaVersion=1&panes=${encodeURIComponent(JSON.stringify(tracePane))}&orgId=1`;

export default function AppLayout() {
  const { pathname } = useLocation();

  useEffect(() => {
    const pageTitle = pathname === "/issues" ? "Issues" : pathname === "/console" ? "API Console" : "";
    document.title = pageTitle ? `${pageTitle} | ${projectTitle}` : projectTitle;
  }, [pathname]);

  return (
    <>
      <header className="site-header">
        <div className="site-header__inner">
          <div className="site-branding">
            <Link className="brand" to="/" aria-label="Fino home">
              <img className="fino-logo" src="/fino-logo.png" alt="fino" />
            </Link>
            <Link className="product-label" to="/" title="Home">{projectName}</Link>
            <NavLink className="console-link" to="/console">API Console</NavLink>
            <NavLink className="console-link" to="/issues">Issues</NavLink>
            <a className="console-link observability-link" href={monitorHref} target="_blank" rel="noreferrer">Monitor</a>
            <a className="console-link observability-link" href={logsHref} target="_blank" rel="noreferrer">Logs</a>
            <a className="console-link observability-link" href={traceHref} target="_blank" rel="noreferrer">Trace</a>
          </div>
          <nav className="site-nav" aria-label="Main navigation">
            <ThemeToggle />
          </nav>
        </div>
      </header>
      <Outlet />
    </>
  );
}

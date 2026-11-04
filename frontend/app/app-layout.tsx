import { useEffect, useState, type FormEvent } from "react";
import { useIsMutating } from "@tanstack/react-query";
import { Link, NavLink, Outlet, useBeforeUnload, useBlocker, useLocation, useNavigate } from "react-router-dom";
import { ThemeToggle } from "@/app/theme-toggle";
import { AccessProvider, useAccess } from "@/lib/access";
import { ConfirmDialog } from "@/app/business-ui";
import "./business.css";

const projectTitle = "orderpay | fino";
const projectName = "orderpay";
const monitorHref = "/monitor/d/service-overview/service-overview?from=now-1h&to=now&var-job=All&refresh=30s";
const logsHref = "/monitor/d/service-logs/service-logs?from=now-15m&to=now&var-service=All&var-level=All";
const tracePane = {
  trace: {
    datasource: "tempo",
    queries: [
      {
        refId: "A",
        query: "{}",
        queryType: "traceql",
        limit: 20,
        tableType: "traces",
        datasource: { type: "tempo", uid: "tempo" },
      },
    ],
    range: { from: "now-15m", to: "now" },
    compact: false,
  },
};
const traceHref = `/monitor/explore?schemaVersion=1&panes=${encodeURIComponent(JSON.stringify(tracePane))}&orgId=1`;

export default function AppLayout() {
  return (
    <AccessProvider>
      <WorkspaceShell />
    </AccessProvider>
  );
}

function WorkspaceShell() {
  const { connection, changeConnection, notice, dirty } = useAccess();
  const { pathname, search } = useLocation();
  const navigate = useNavigate();
  const pending = useIsMutating() > 0;
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [token, setToken] = useState("");
  const [connectionError, setConnectionError] = useState("");
  const [mode, setMode] = useState<"merchant" | "audit">("merchant");
  const toolsPage = pathname === "/console" || pathname === "/issues";
  const blocker = useBlocker(() => dirty.current || pending);

  useBeforeUnload((event) => {
    if (dirty.current || pending) {
      event.preventDefault();
      event.returnValue = "";
    }
  });

  useEffect(() => {
    const pageTitle =
      pathname === "/issues"
        ? "Issues"
        : pathname === "/console"
          ? "API Console"
          : pathname === "/admin/audit-events"
            ? "审计事件"
            : pathname === "/orders/new"
              ? "创建订单"
              : pathname.startsWith("/orders/")
                ? "订单详情"
                : "订单";
    document.title = pageTitle ? `${pageTitle} | ${projectTitle}` : projectTitle;
  }, [pathname]);

  function applyConnection(event: FormEvent) {
    event.preventDefault();
    if (token.trim().startsWith("{")) {
      setConnectionError(
        "请只填写 JSON 中对应身份的密钥，不要粘贴整段 JSON。商户密钥用于商户模式，管理员密钥用于审计模式。",
      );
      return;
    }
    setConnectionError("");
    if (changeConnection(token.trim(), mode)) {
      setToken("");
      setSettingsOpen(false);
      const keepTarget = !connection && mode === "merchant" && pathname.startsWith("/orders");
      navigate(keepTarget ? pathname + search : mode === "merchant" ? "/orders" : "/admin/audit-events");
    }
  }

  return (
    <>
      <header className="site-header">
        <div className="site-header__inner">
          <div className="site-branding">
            <Link className="brand" to="/" aria-label="Orderpay 首页">
              <img className="fino-logo" src="/fino-logo.png" alt="fino" />
            </Link>
            <Link className="product-label" to="/" title="Home">
              {projectName}
            </Link>
            <NavLink className="console-link" to="/console">
              API Console
            </NavLink>
            <NavLink className="console-link" to="/issues">
              Issues
            </NavLink>
            <a className="console-link" href={monitorHref} target="_blank" rel="noreferrer">
              Monitor
            </a>
            <a className="console-link" href={logsHref} target="_blank" rel="noreferrer">
              Logs
            </a>
            <a className="console-link" href={traceHref} target="_blank" rel="noreferrer">
              Trace
            </a>
          </div>
          <nav className="site-nav" aria-label="显示和连接设置">
            <ThemeToggle />
            <button
              className="secondary"
              type="button"
              aria-expanded={settingsOpen || !connection}
              aria-controls="connection-settings"
              onClick={() => {
                setSettingsOpen(!settingsOpen);
                setMode(connection?.mode ?? "merchant");
                setToken("");
              }}
            >
              连接设置
            </button>
          </nav>
        </div>
      </header>
      {(settingsOpen || !connection) && (
        <section className="connection-settings" id="connection-settings" aria-labelledby="connection-title">
          <h2 id="connection-title">连接本地服务</h2>
          <p className="muted">凭证仅保存在当前页面内存。工作模式只选择页面，权限由服务端验证。</p>
          <form onSubmit={applyConnection} className="connection-form">
            <label>
              工作模式
              <select value={mode} onChange={(event) => setMode(event.target.value as "merchant" | "audit")}>
                <option value="merchant">商户</option>
                <option value="audit">审计</option>
              </select>
            </label>
            <label>
              服务凭证
              <input
                type="password"
                value={token}
                onChange={(event) => {
                  setToken(event.target.value);
                  setConnectionError("");
                }}
                aria-describedby="credential-help"
                aria-invalid={!!connectionError}
                required
                minLength={32}
                autoComplete="off"
                spellCheck={false}
              />
            </label>
            <button type="submit" disabled={pending}>
              应用连接
            </button>
            {connection && (
              <button
                className="secondary"
                type="button"
                disabled={pending}
                onClick={() => {
                  if (changeConnection("", connection.mode)) setToken("");
                }}
              >
                清除连接
              </button>
            )}
          </form>
          <p
            id="credential-help"
            className={connectionError ? "field-error" : "muted"}
            role={connectionError ? "alert" : undefined}
          >
            {connectionError ||
              "只填写对应的单个密钥，不要粘贴整段 JSON。商户密钥选择商户模式，管理员密钥选择审计模式；审计连接后需点击查询。"}
          </p>
        </section>
      )}
      {notice && (
        <p className="shell-notice work-notice warning" role="alert">
          {notice}
        </p>
      )}
      {connection || toolsPage ? (
        <Outlet key={connection?.scope ?? "tools"} />
      ) : (
        <main className="work-page">
          <h1>订单工作台</h1>
          <p className="muted">配置连接后可管理订单，或切换至审计模式查询业务事件。</p>
        </main>
      )}
      {blocker.state === "blocked" && (
        <ConfirmDialog
          title={pending ? "操作正在处理中" : "有未保存的内容"}
          confirmLabel="放弃内容并离开"
          busy={pending}
          onClose={() => blocker.reset()}
          onConfirm={() => {
            dirty.current = false;
            blocker.proceed();
          }}
        >
          <p>{pending ? "请等待操作结果，切换页面不会撤销服务端处理。" : "离开当前页面会丢弃尚未保存的订单内容。"}</p>
        </ConfirmDialog>
      )}
    </>
  );
}

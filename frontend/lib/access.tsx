import { createContext, useContext, useMemo, useRef, useState, type ReactNode } from "react";
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError, type Connection } from "@/api/business";

type Access = {
  connection: Connection | null;
  notice: string;
  dirty: React.RefObject<boolean>;
  changeConnection: (token: string, mode: Connection["mode"]) => boolean;
};

const AccessContext = createContext<Access | null>(null);

export function AccessProvider({ children }: { children: ReactNode }) {
  const [connection, setConnection] = useState<Connection | null>(null);
  const [notice, setNotice] = useState("");
  const dirty = useRef(false);
  const activeScope = useRef("");
  const scope = connection?.scope ?? "";
  const client = useMemo(() => {
    const onError = (error: Error) => {
      if (error instanceof ApiError && error.status === 401 && activeScope.current === scope) {
        activeScope.current = "";
        dirty.current = false;
        void client.cancelQueries();
        client.clear();
        setNotice("连接已失效，请重新配置凭证。未保存内容已清除。");
        setConnection(null);
      }
    };
    return new QueryClient({
      queryCache: new QueryCache({ onError }),
      mutationCache: new MutationCache({ onError }),
      defaultOptions: {
        queries: {
          retry: false,
          networkMode: "always",
          staleTime: Infinity,
          refetchOnWindowFocus: false,
          refetchOnReconnect: false,
          refetchOnMount: false,
        },
        mutations: { retry: false, networkMode: "always" },
      },
    });
  }, [scope]);

  function changeConnection(token: string, mode: Connection["mode"]) {
    if (client.isMutating()) {
      setNotice("正在确认操作结果，请等待后再切换连接。");
      return false;
    }
    if (dirty.current && !window.confirm("有尚未保存的订单内容，切换连接会丢弃这些内容。确认继续？")) return false;
    void client.cancelQueries();
    client.clear();
    dirty.current = false;
    const next = token ? { token, mode, scope: crypto.randomUUID() } : null;
    activeScope.current = next?.scope ?? "";
    setConnection(next);
    setNotice("");
    return true;
  }

  return (
    <AccessContext.Provider value={{ connection, notice, dirty, changeConnection }}>
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    </AccessContext.Provider>
  );
}

export function useAccess() {
  const context = useContext(AccessContext);
  if (!context) throw new Error("AccessProvider is required");
  return context;
}

export function AccessBoundary({ mode, children }: { mode: Connection["mode"]; children: ReactNode }) {
  const { connection } = useAccess();
  if (!connection || connection.mode !== mode) {
    return (
      <main className="work-page">
        <h1>工作模式不匹配</h1>
        <p>请通过连接设置选择{mode === "merchant" ? "商户" : "审计"}模式并配置对应凭证。选择模式不会授予访问权限。</p>
      </main>
    );
  }
  return children;
}

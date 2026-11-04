import { useEffect, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useAccess } from "@/lib/access";
import { apiScope } from "@/api/business";
import { CopyValue, ErrorNotice, LoadingRows, Pagination, StatusBadge } from "@/app/business-ui";
import { orderListOptions } from "./api";
import { firstPage, useOrderWorkspace } from "./layout";
import { isUUID, money, time } from "./model";

export default function OrdersPage() {
  const { connection } = useAccess();
  const { position, setPosition } = useOrderWorkspace();
  const client = useQueryClient();
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const [orderId, setOrderId] = useState("");
  const [lookupError, setLookupError] = useState("");
  const status = [1, 2, 3].includes(Number(params.get("status"))) ? Number(params.get("status")) : 0;
  const page = position.status === status ? position : firstPage(status);
  const token = page.tokens[page.index];
  const query = useQuery(orderListOptions(connection!, status, token));

  useEffect(() => {
    window.scrollTo(0, page.scroll);
  }, [status]);

  function locate(event: FormEvent) {
    event.preventDefault();
    const id = orderId.trim();
    if (!isUUID(id)) {
      setLookupError("请输入完整的订单 UUID。");
      return;
    }
    rememberPosition();
    navigate(`/orders/${id}`);
  }

  function rememberPosition() {
    setPosition({ ...page, scroll: window.scrollY });
  }

  function refresh() {
    void client.invalidateQueries({ queryKey: [...apiScope(connection!), "orders"], refetchType: "none" });
    setPosition(firstPage(status));
    if (!token) void query.refetch();
  }

  const rows = query.data?.orders ?? [];
  const nextToken = query.data?.nextPageToken;

  return (
    <main className="work-page">
      <div className="page-heading">
        <div>
          <h1>订单</h1>
          <p className="muted">管理当前商户订单</p>
        </div>
        <Link className="work-button" to="/orders/new">
          创建订单
        </Link>
      </div>
      <div className="work-toolbar">
        <div className="status-filters" role="group" aria-label="订单状态筛选">
          {["全部", "待支付", "已支付", "已取消"].map((label, value) => (
            <button
              key={value}
              type="button"
              className="secondary"
              aria-pressed={status === value}
              onClick={() => {
                setPosition(firstPage(value));
                setParams(value ? { status: String(value) } : {});
              }}
            >
              {label}
            </button>
          ))}
        </div>
        <button className="secondary" type="button" onClick={refresh} disabled={query.isFetching}>
          刷新
        </button>
      </div>
      <form className="lookup-form" onSubmit={locate}>
        <label htmlFor="order-lookup">按订单 ID 定位</label>
        <input
          id="order-lookup"
          value={orderId}
          onChange={(event) => {
            setOrderId(event.target.value);
            setLookupError("");
          }}
          placeholder="输入完整订单 UUID"
          aria-invalid={!!lookupError}
          aria-describedby="lookup-help"
        />
        <button className="secondary" type="submit">
          查看
        </button>
        <p id="lookup-help" className={lookupError ? "field-error" : "muted"}>
          {lookupError || "请输入系统订单 ID，不支持按商户订单号搜索。"}
        </p>
      </form>
      <ErrorNotice error={query.error}>
        {!!token && (
          <button type="button" className="secondary" onClick={() => setPosition(firstPage(status))}>
            返回第一页
          </button>
        )}
      </ErrorNotice>
      <p className="query-status muted" role="status">
        {query.isFetching
          ? "正在读取订单…"
          : query.dataUpdatedAt
            ? `${query.isError ? "数据可能已过期 · " : ""}最后获取：${time(query.dataUpdatedAt)}`
            : ""}
      </p>
      {query.isPending && query.isFetching ? (
        <LoadingRows />
      ) : (
        query.data &&
        (rows.length ? (
          <div className="table-region">
            <table className="work-table orders-table">
              <caption className="sr-only">当前商户订单，创建时间从早到晚</caption>
              <thead>
                <tr>
                  <th scope="col">商户订单号 / 订单 ID</th>
                  <th scope="col" className="amount">
                    金额（CNY）
                  </th>
                  <th scope="col">订单状态</th>
                  <th scope="col" className="optional-column">
                    创建时间
                  </th>
                  <th scope="col">操作</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((order) => (
                  <tr key={order.id}>
                    <td>
                      <Link className="order-number" to={`/orders/${order.id}`} onClick={rememberPosition}>
                        {order.merchantOrderNo}
                      </Link>
                      <div className="row-secondary">
                        <CopyValue value={order.id} short />
                      </div>
                    </td>
                    <td className="amount">{money(order.amountMinor)}</td>
                    <td>
                      <StatusBadge status={order.status} />
                    </td>
                    <td className="optional-column muted">{time(order.createTime)}</td>
                    <td>
                      <Link className="action-link" to={`/orders/${order.id}`} onClick={rememberPosition}>
                        查看
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="work-empty">
            <h2>{status ? "没有符合当前状态的订单" : "还没有订单"}</h2>
            {status ? (
              <button
                className="secondary"
                onClick={() => {
                  setPosition(firstPage(0));
                  setParams({});
                }}
              >
                查看全部
              </button>
            ) : (
              <Link className="action-link" to="/orders/new">
                创建订单
              </Link>
            )}
          </div>
        ))
      )}
      <Pagination
        previous={page.index > 0}
        next={!!nextToken && !query.isError}
        busy={query.isFetching}
        onPrevious={() => setPosition({ ...page, index: page.index - 1, scroll: 0 })}
        onNext={() => {
          if (nextToken)
            setPosition({
              ...page,
              tokens: [...page.tokens.slice(0, page.index + 1), nextToken],
              index: page.index + 1,
              scroll: 0,
            });
        }}
      />
    </main>
  );
}

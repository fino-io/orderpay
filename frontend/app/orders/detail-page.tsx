import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useLocation, useParams } from "react-router-dom";
import { ApiError, apiScope } from "@/api/business";
import { useAccess } from "@/lib/access";
import { ConfirmDialog, CopyValue, ErrorNotice, LoadingRows, StatusBadge } from "@/app/business-ui";
import { cancelOrder, orderDetailOptions } from "./api";
import { useOrderWorkspace } from "./layout";
import { isUUID, money, orderAction, time } from "./model";

export default function OrderDetailPage() {
  const { orderId = "" } = useParams();
  if (!isUUID(orderId))
    return (
      <main className="work-page">
        <h1>订单 ID 格式有误</h1>
        <Link to="/orders">返回订单列表</Link>
      </main>
    );
  return <OrderDetail key={orderId} orderId={orderId} />;
}

function OrderDetail({ orderId }: { orderId: string }) {
  const { connection } = useAccess();
  const { resetList, position } = useOrderWorkspace();
  const location = useLocation();
  const client = useQueryClient();
  const query = useQuery({ ...orderDetailOptions(connection!, orderId), refetchOnMount: "always" });
  const [confirming, setConfirming] = useState(false);
  const [message, setMessage] = useState<string>(location.state?.message ?? "");
  const [paymentStale, setPaymentStale] = useState(false);

  function resetOrders() {
    resetList();
    void client.invalidateQueries({ queryKey: [...apiScope(connection!), "orders"], refetchType: "none" });
  }
  const mutation = useMutation({
    mutationFn: () => cancelOrder(connection!, orderId),
    onSuccess: async (order) => {
      await client.cancelQueries({ queryKey: orderDetailOptions(connection!, orderId).queryKey });
      client.setQueryData(orderDetailOptions(connection!, orderId).queryKey, (detail) =>
        detail ? { ...detail, order } : { order },
      );
      setPaymentStale(true);
      setConfirming(false);
      setMessage("订单已取消。");
      resetOrders();
      const result = await query.refetch();
      if (!result.error) setPaymentStale(false);
    },
    onError: async (error) => {
      setConfirming(false);
      if (error instanceof ApiError && [401, 403].includes(error.status)) return;
      const result = await query.refetch();
      if (!result.error && result.data?.order.status === "ORDER_STATUS_CANCELLED") {
        setMessage("已核对：订单已取消。");
        resetOrders();
      }
    },
  });
  const detail = query.data;
  const action = detail ? orderAction(detail) : null;

  async function refresh() {
    mutation.reset();
    const result = await query.refetch();
    if (!result.error) setPaymentStale(false);
  }

  const cancellationVerified = detail?.order.status === "ORDER_STATUS_CANCELLED" && !query.isError;

  return (
    <main className="work-page">
      <Link className="back-link" to={position.status ? `/orders?status=${position.status}` : "/orders"}>
        返回订单
      </Link>
      <div className="page-heading">
        <div>
          <h1>{detail?.order.merchantOrderNo ?? "订单详情"}</h1>
          <CopyValue value={orderId} />
        </div>
        <button className="secondary" type="button" onClick={refresh} disabled={query.isFetching || mutation.isPending}>
          刷新
        </button>
      </div>
      {message && (
        <div className="work-notice success" role="status">
          {message}
        </div>
      )}
      <ErrorNotice error={query.error} />
      <ErrorNotice error={cancellationVerified ? null : mutation.error} />
      {mutation.error && !cancellationVerified && (
        <p className="work-notice warning" role="alert">
          {mutation.error instanceof ApiError && mutation.error.status === 409
            ? "状态已变化，请核对详情；读取失败时请手动刷新。"
            : "取消请求未得到确定结果，请核对最新详情，勿将请求失败视为订单未取消。"}
        </p>
      )}
      {query.dataUpdatedAt > 0 && (
        <p className="query-status muted">
          {query.isFetching ? "正在刷新 · " : query.isError ? "数据可能已过期 · " : ""}最后获取：
          {time(query.dataUpdatedAt)}
        </p>
      )}
      {query.isPending && <LoadingRows />}
      {detail && (
        <>
          <div className="order-summary">
            <StatusBadge status={detail.order.status} />
            <strong>
              {money(detail.order.amountMinor)} <small>CNY</small>
            </strong>
          </div>
          {action?.warning && !paymentStale && (
            <div className="work-notice warning" role="alert">
              {action.warning}
            </div>
          )}
          <div className="detail-columns">
            <section className="detail-section" aria-labelledby="order-info">
              <h2 id="order-info">订单信息</h2>
              <dl>
                <dt>商户订单号</dt>
                <dd>{detail.order.merchantOrderNo}</dd>
                <dt>订单描述</dt>
                <dd className="preserve-text">{detail.order.description}</dd>
                <dt>创建时间</dt>
                <dd>{time(detail.order.createTime)}</dd>
                <dt>更新时间</dt>
                <dd>{time(detail.order.updateTime)}</dd>
              </dl>
            </section>
            <section className="detail-section" aria-labelledby="payment-info">
              <h2 id="payment-info">支付信息</h2>
              {paymentStale && <p className="work-notice warning">支付信息待刷新，以下为上次读取结果。</p>}
              {detail.payment ? (
                <dl>
                  <dt>支付状态</dt>
                  <dd>
                    <StatusBadge status={detail.payment.status} />
                  </dd>
                  <dt>支付 ID</dt>
                  <dd>
                    <CopyValue value={detail.payment.id} />
                  </dd>
                  <dt>受理金额</dt>
                  <dd>{money(detail.payment.amountMinor)} CNY</dd>
                  <dt>渠道引用</dt>
                  <dd>
                    {detail.payment.providerReference ? (
                      <CopyValue value={detail.payment.providerReference} />
                    ) : (
                      "暂未取得"
                    )}
                  </dd>
                  <dt>原因码</dt>
                  <dd>{detail.payment.reasonCode || "—"}</dd>
                  <dt>创建时间</dt>
                  <dd>{time(detail.payment.createTime)}</dd>
                  <dt>更新时间</dt>
                  <dd>{time(detail.payment.updateTime)}</dd>
                </dl>
              ) : (
                <p className="muted">未创建支付记录</p>
              )}
              <p className="payment-disabled">支付功能暂未开放</p>
            </section>
          </div>
          <div className="detail-actions">
            <p className="muted">{paymentStale ? "订单已取消，支付信息待刷新。" : action?.explanation}</p>
            {detail.order.status === "ORDER_STATUS_PENDING_PAYMENT" && (
              <button
                className="secondary danger-button"
                type="button"
                disabled={!action?.canCancel || query.isFetching || query.isError || mutation.isPending}
                onClick={() => setConfirming(true)}
              >
                取消订单
              </button>
            )}
          </div>
          {confirming && (
            <ConfirmDialog
              title="确认取消订单"
              confirmLabel="确认取消"
              busy={mutation.isPending}
              onClose={() => setConfirming(false)}
              onConfirm={() => {
                if (action?.canCancel && !query.isError && !query.isFetching) mutation.mutate();
              }}
            >
              <p>{detail.order.merchantOrderNo}</p>
              <p className="dialog-amount">{money(detail.order.amountMinor)} CNY</p>
              <p>取消后不能恢复，服务端会检查支付状态。</p>
            </ConfirmDialog>
          )}
        </>
      )}
    </main>
  );
}

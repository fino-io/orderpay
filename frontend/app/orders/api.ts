import { queryOptions } from "@tanstack/react-query";
import { apiScope, businessRequest, type Connection } from "@/api/business";
import { parseDetail, parseOrder, record, text, type OrderInput } from "./model";

export function orderListOptions(connection: Connection, status: number, pageToken: string) {
  return queryOptions({
    queryKey: [...apiScope(connection), "orders", status, pageToken],
    refetchOnMount: true,
    queryFn: async ({ signal }) => {
      const body = record(
        await businessRequest(
          connection,
          "listOrders",
          { status: status || undefined, pageSize: 20, pageToken: pageToken || undefined },
          signal,
        ),
      );
      const rows = body.orders ?? [];
      if (!Array.isArray(rows)) throw new Error("订单列表格式有误。");
      return { orders: rows.map(parseOrder), nextPageToken: text(body.nextPageToken ?? "") };
    },
  });
}

export function orderDetailOptions(connection: Connection, orderId: string) {
  return queryOptions({
    queryKey: [...apiScope(connection), "order", orderId],
    queryFn: async ({ signal }) => {
      const detail = parseDetail(await businessRequest(connection, "getOrder", { orderId }, signal));
      if (detail.order.id !== orderId) throw new Error("服务返回的订单与查询不一致。");
      return detail;
    },
  });
}

export async function createOrder(connection: Connection, order: OrderInput) {
  const body = record(await businessRequest(connection, "createOrder", { order }));
  return parseOrder(body.order);
}

export async function cancelOrder(connection: Connection, orderId: string) {
  const body = record(await businessRequest(connection, "cancelOrder", { orderId }));
  const order = parseOrder(body.order);
  if (order.id !== orderId || order.status !== "ORDER_STATUS_CANCELLED") throw new Error("取消结果需要重新核对。");
  return order;
}

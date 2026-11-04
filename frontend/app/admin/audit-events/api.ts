import { queryOptions } from "@tanstack/react-query";
import { apiScope, businessRequest, type Connection } from "@/api/business";
import { record, text, timestamp } from "@/app/orders/model";

export type AuditFilter = { merchantId: string; resourceId: string };
export type AuditEvent = {
  id: string;
  merchantId: string;
  actorId: string;
  action: string;
  resourceId: string;
  requestId: string;
  beforeStatus: string;
  afterStatus: string;
  reasonCode: string;
  evidenceReference: string;
  createTime: string;
};

export function auditOptions(connection: Connection, filter: AuditFilter, pageToken: string) {
  return queryOptions({
    queryKey: [...apiScope(connection), "audits", filter, pageToken],
    queryFn: async ({ signal }) => {
      const body = record(
        await businessRequest(
          connection,
          "listAuditEvents",
          {
            merchantId: filter.merchantId || undefined,
            resourceId: filter.resourceId || undefined,
            pageSize: 20,
            pageToken: pageToken || undefined,
          },
          signal,
        ),
      );
      const rows = body.events ?? [];
      if (!Array.isArray(rows)) throw new Error("审计列表格式有误。");
      const events = rows.map((value): AuditEvent => {
        const row = record(value);
        return {
          id: text(row.id),
          merchantId: text(row.merchantId ?? ""),
          actorId: text(row.actorId ?? ""),
          action: text(row.action),
          resourceId: text(row.resourceId ?? ""),
          requestId: text(row.requestId ?? ""),
          beforeStatus: text(row.beforeStatus ?? ""),
          afterStatus: text(row.afterStatus ?? ""),
          reasonCode: text(row.reasonCode ?? ""),
          evidenceReference: text(row.evidenceReference ?? ""),
          createTime: timestamp(row.createTime),
        };
      });
      return { events, nextPageToken: text(body.nextPageToken ?? "") };
    },
  });
}

const actions: Record<string, string> = {
  ORDER_CREATED: "创建订单",
  ORDER_CANCELLED: "取消订单",
  PAYMENT_ACCEPTED: "受理支付",
  PAYMENT_CANCELLED: "关闭支付",
  AUDIT_QUERIED: "查询审计",
  PAYMENT_RESULT_CONFLICT: "支付结果冲突",
  RECONCILIATION_CLOSED: "关闭核查",
};
export const actionLabel = (value: string) => (Object.hasOwn(actions, value) ? actions[value] : value);

export type Order = {
  id: string;
  merchantOrderNo: string;
  amountMinor: number;
  description: string;
  status: string;
  createTime: string;
  updateTime: string;
};

export type Payment = {
  id: string;
  orderId: string;
  amountMinor: number;
  status: string;
  providerReference: string;
  reasonCode: string;
  createTime: string;
  updateTime: string;
};

export type OrderInput = Pick<Order, "merchantOrderNo" | "amountMinor" | "description">;
export type OrderDetail = { order: Order; payment?: Payment };

export function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value))
    throw new Error("服务返回的数据格式有误，请刷新或联系管理员。");
  return value as Record<string, unknown>;
}

export function text(value: unknown): string {
  if (typeof value !== "string") throw new Error("服务返回的字段格式有误，请联系管理员。");
  return value;
}

function amount(value: unknown) {
  if (typeof value !== "number" || !Number.isInteger(value) || value < 1 || value > 100000000)
    throw new Error("服务返回的金额有误。");
  return value;
}

export function timestamp(value: unknown) {
  const result = text(value);
  if (!Number.isFinite(Date.parse(result))) throw new Error("服务返回的时间格式有误。");
  return result;
}

export function parseOrder(value: unknown): Order {
  const row = record(value);
  if (!isUUID(row.id)) throw new Error("服务返回的订单 ID 有误。");
  return {
    id: row.id,
    merchantOrderNo: text(row.merchantOrderNo),
    description: text(row.description),
    amountMinor: amount(row.amountMinor),
    status: text(row.status ?? "ORDER_STATUS_UNSPECIFIED"),
    createTime: timestamp(row.createTime),
    updateTime: timestamp(row.updateTime),
  };
}

export function parseDetail(value: unknown): OrderDetail {
  const body = record(value);
  const order = parseOrder(body.order);
  if (body.payment == null) return { order };
  const row = record(body.payment);
  if (!isUUID(row.id) || !isUUID(row.orderId)) throw new Error("服务返回的支付 ID 有误。");
  return {
    order,
    payment: {
      id: row.id,
      orderId: row.orderId,
      amountMinor: amount(row.amountMinor),
      status: text(row.status ?? "PAYMENT_STATUS_UNSPECIFIED"),
      providerReference: text(row.providerReference ?? ""),
      reasonCode: text(row.reasonCode ?? ""),
      createTime: timestamp(row.createTime),
      updateTime: timestamp(row.updateTime),
    },
  };
}

export function isUUID(value: unknown): value is string {
  return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
}

export function parseMoney(value: string) {
  if (!/^\d{1,7}(\.\d{1,2})?$/.test(value)) throw new Error("请输入金额，最多两位小数。");
  const [whole, fraction = ""] = value.split(".");
  const minor = Number(whole) * 100 + Number(fraction.padEnd(2, "0"));
  if (minor < 1 || minor > 100000000) throw new Error("金额范围为 0.01～1,000,000.00 元。");
  return minor;
}

const currency = new Intl.NumberFormat("zh-CN", { style: "currency", currency: "CNY" });
const dateTime = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  timeZoneName: "short",
});

export const money = (minor: number) => currency.format(minor / 100);
export const time = (value: string | number) => dateTime.format(new Date(value));

const states: Record<string, string> = {
  ORDER_STATUS_PENDING_PAYMENT: "待支付",
  ORDER_STATUS_PAID: "已支付",
  ORDER_STATUS_CANCELLED: "已取消",
  PAYMENT_STATUS_CREATED: "支付已受理",
  PAYMENT_STATUS_PENDING: "支付结果待确认",
  PAYMENT_STATUS_SUCCEEDED: "支付成功",
  PAYMENT_STATUS_DECLINED: "支付已关闭，未支付",
};
export const knownStatus = (status: string) => Object.hasOwn(states, status);
export const statusLabel = (status: string) => (knownStatus(status) ? states[status] : status ? "状态未知" : "—");

export function orderAction({ order, payment }: OrderDetail): {
  canCancel: boolean;
  warning: string;
  explanation: string;
} {
  const deny = (warning: string) => ({ canCancel: false, warning, explanation: "请联系管理员核对，当前不能取消。" });
  if (!knownStatus(order.status) || (payment && !knownStatus(payment.status)))
    return deny("状态未知，请刷新或联系管理员。");
  if (payment && (payment.orderId !== order.id || payment.amountMinor !== order.amountMinor))
    return deny("订单与支付信息不一致。");
  if (payment?.reasonCode) {
    const reasons: Record<string, string> = {
      RECONCILIATION_REQUIRED: "支付结果需要核查。",
      CHANNEL_RESULT_CONFLICT: "支付结果存在冲突。",
      COMPENSATION_REQUIRED: "需要外部补偿处理。",
    };
    if (Object.hasOwn(reasons, payment.reasonCode)) return deny(reasons[payment.reasonCode]);
    if (!(
      payment.reasonCode === "ORDER_CANCELLED" &&
      order.status === "ORDER_STATUS_CANCELLED" &&
      payment.status === "PAYMENT_STATUS_DECLINED"
    )) {
      return deny(`支付原因需要核对：${payment.reasonCode}`);
    }
  }
  if (order.status === "ORDER_STATUS_PENDING_PAYMENT") {
    if (!payment || ["PAYMENT_STATUS_CREATED", "PAYMENT_STATUS_DECLINED"].includes(payment.status)) {
      return {
        canCancel: true,
        warning: "",
        explanation:
          payment?.status === "PAYMENT_STATUS_CREATED"
            ? "服务端将检查支付是否已准备提交，取消可能被拒绝。"
            : "取消后订单不能恢复。",
      };
    }
    if (payment.status === "PAYMENT_STATUS_PENDING")
      return { canCancel: false, warning: "", explanation: "支付结果待确认，当前不能取消。" };
  }
  if (order.status === "ORDER_STATUS_PAID" && payment?.status === "PAYMENT_STATUS_SUCCEEDED")
    return { canCancel: false, warning: "", explanation: "已确认收款。" };
  if (order.status === "ORDER_STATUS_CANCELLED" && (!payment || payment.status === "PAYMENT_STATUS_DECLINED"))
    return { canCancel: false, warning: "", explanation: "订单已取消。" };
  return deny("订单与支付状态组合需要核对。");
}

// HTTP sample verified against the current generated transport and SQLite workflow.
export const order = {
  id: "01a1a401-78aa-7407-968b-3c4722018026",
  merchantOrderNo: "SO-20261104-001",
  amountMinor: 1990,
  description: "测试订单",
  status: "ORDER_STATUS_PENDING_PAYMENT",
  createTime: "2026-11-04T00:42:02.282264Z",
  updateTime: "2026-11-04T00:42:02.282264Z",
};

export const payment = {
  id: "01a1a401-78aa-7407-968b-3c4722018027",
  orderId: order.id,
  amountMinor: 1990,
  status: "PAYMENT_STATUS_CREATED",
  providerReference: "",
  reasonCode: "",
  createTime: order.createTime,
  updateTime: order.updateTime,
};

export const event = {
  id: "01a1a401-78aa-7999-a668-27482a0d3bed",
  merchantId: "019a4a70-0000-7000-8000-000000000001",
  actorId: "019a4a70-0000-7000-8000-000000000001",
  action: "ORDER_CREATED",
  resourceId: order.id,
  requestId: "68a15a56-1316-42c3-b784-b8652f41fd7c",
  afterStatus: order.status,
  createTime: order.createTime,
};

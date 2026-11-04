import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, businessRequest, responseData, type Connection } from "@/api/business";
import { orderAction, parseDetail, parseMoney, statusLabel } from "@/app/orders/model";
import { order, payment } from "./fixtures";

vi.mock("@/lib/config", () => ({ config: { apiBase: "http://orderpay.test", appId: 1 } }));
afterEach(() => vi.unstubAllGlobals());

describe("金额和响应边界", () => {
  it("用整数分表达小数金额，并限制输入范围", () => {
    expect(parseMoney("0.01")).toBe(1);
    expect(parseMoney("19.90")).toBe(1990);
    expect(parseMoney("1.1")).toBe(110);
    expect(parseMoney("1000000.00")).toBe(100000000);
    for (const value of ["0", "-1", "1.001", "1e2", "1000000.01", "Infinity", ""])
      expect(() => parseMoney(value)).toThrow();
  });

  it("解析真实 HTTP 样本，缺失支付正常，错误金额和时间不降级为默认值", () => {
    expect(parseDetail({ order }).payment).toBeUndefined();
    expect(parseDetail({ order, payment }).payment?.status).toBe("PAYMENT_STATUS_CREATED");
    expect(() => parseDetail({ order: { ...order, amountMinor: undefined } })).toThrow();
    expect(() => parseDetail({ order: { ...order, createTime: "not-a-time" } })).toThrow();
  });

  it("即使 HTTP 为 200，也识别 envelope 内的业务错误", () => {
    expect(responseData({ code: "200", data: { order } }, 200)).toEqual({ order });
    expect(() => responseData({ code: "100060401", message: "authentication required" }, 200)).toThrow(ApiError);
    try {
      responseData({ error: { code: "100061012" } }, 200);
    } catch (error) {
      expect((error as ApiError).status).toBe(404);
    }
    try {
      responseData({ code: "100061064" }, 503);
    } catch (error) {
      expect((error as Error).message).toContain("渠道");
    }
    try {
      responseData({}, 503);
    } catch (error) {
      expect((error as Error).message).not.toContain("渠道");
    }
  });

  it("复用生成 body 映射、透传取消信号，并保留 HTTP 错误码", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ order }), { status: 200 }));
    vi.stubGlobal("fetch", fetch);
    const connection: Connection = { token: "test-private-key", scope: "scope", mode: "merchant" };
    const controller = new AbortController();
    await businessRequest(
      connection,
      "createOrder",
      { order: { merchantOrderNo: "test", amountMinor: 1990, description: "test" } },
      controller.signal,
    );
    const [url, init] = fetch.mock.calls[0];
    expect(String(url)).toBe("http://orderpay.test/orderpay/v1/orders");
    expect(JSON.parse(init.body)).toEqual({ merchantOrderNo: "test", amountMinor: 1990, description: "test" });
    expect(init.headers.get("Authorization")).toBe("Bearer test-private-key");
    controller.abort();
    expect(init.signal.aborted).toBe(true);
    fetch.mockResolvedValue(new Response('{"code":"100061062","message":"conflict"}', { status: 409 }));
    await expect(businessRequest(connection, "cancelOrder", { orderId: order.id })).rejects.toMatchObject({
      status: 409,
      code: "100061062",
    });
  });
});

describe("取消候选和异常优先", () => {
  it("仅开放待支付且无支付、CREATED 或正常 DECLINED 的取消候选", () => {
    expect(orderAction({ order }).canCancel).toBe(true);
    expect(orderAction({ order, payment }).canCancel).toBe(true);
    expect(orderAction({ order, payment: { ...payment, status: "PAYMENT_STATUS_DECLINED" } }).canCancel).toBe(true);
    expect(orderAction({ order, payment: { ...payment, status: "PAYMENT_STATUS_PENDING" } }).canCancel).toBe(false);
  });

  it("核查、未知原因、金额和状态冲突均不能被成功状态掩盖", () => {
    const paid = { ...order, status: "ORDER_STATUS_PAID" };
    const succeeded = { ...payment, status: "PAYMENT_STATUS_SUCCEEDED" };
    expect(orderAction({ order: paid, payment: succeeded }).warning).toBe("");
    for (const reasonCode of [
      "RECONCILIATION_REQUIRED",
      "CHANNEL_RESULT_CONFLICT",
      "COMPENSATION_REQUIRED",
      "FUTURE_REASON",
    ]) {
      expect(orderAction({ order: paid, payment: { ...succeeded, reasonCode } }).warning).not.toBe("");
      expect(orderAction({ order, payment: { ...payment, reasonCode } }).canCancel).toBe(false);
    }
    expect(orderAction({ order, payment: { ...payment, amountMinor: 1 } }).canCancel).toBe(false);
    expect(orderAction({ order, payment: { ...payment, orderId: payment.id } }).canCancel).toBe(false);
    expect(orderAction({ order: { ...order, status: "ORDER_STATUS_FUTURE" } }).canCancel).toBe(false);
    expect(statusLabel("constructor")).toBe("状态未知");
    expect(orderAction({ order, payment: { ...payment, reasonCode: "constructor" } }).warning).toContain("constructor");
    expect(orderAction({ order, payment: succeeded }).warning).not.toBe("");
  });
});

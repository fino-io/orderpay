import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import AppLayout from "@/app/app-layout";
import HomePage from "@/app/home-page";
import OrdersLayout from "@/app/orders/layout";
import OrdersPage from "@/app/orders/page";
import NewOrderPage from "@/app/orders/new-page";
import OrderDetailPage from "@/app/orders/detail-page";
import AuditEventsPage from "@/app/admin/audit-events/page";
import { event, order, payment } from "./fixtures";

vi.mock("@/lib/config", () => ({ config: { apiBase: "http://orderpay.test", appId: 1 } }));
const merchantKey = "local-merchant-key-for-test-only-0001";
const otherKey = "local-merchant-key-for-test-only-0002";
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
const routers: ReturnType<typeof createMemoryRouter>[] = [];

beforeEach(() => {
  vi.stubGlobal(
    "matchMedia",
    vi.fn(() => ({ matches: false })),
  );
  vi.spyOn(window, "scrollTo").mockImplementation(() => {});
  HTMLDialogElement.prototype.showModal = function () {
    this.setAttribute("open", "");
  };
  HTMLDialogElement.prototype.close = function () {
    this.removeAttribute("open");
  };
});

afterEach(() => {
  cleanup();
  for (const router of routers.splice(0)) router.dispose();
  vi.unstubAllGlobals();
});

function workspace(path = "/orders") {
  const router = createMemoryRouter(
    [
      {
        Component: AppLayout,
        children: [
          { index: true, Component: HomePage },
          {
            path: "orders",
            Component: OrdersLayout,
            children: [
              { index: true, Component: OrdersPage },
              { path: "new", Component: NewOrderPage },
              { path: ":orderId", Component: OrderDetailPage },
            ],
          },
          { path: "admin/audit-events", Component: AuditEventsPage },
        ],
      },
    ],
    { initialEntries: [path] },
  );
  routers.push(router);
  render(<RouterProvider router={router} />);
  return router;
}

function connect(mode = "merchant", token = merchantKey) {
  fireEvent.change(screen.getByLabelText("工作模式"), { target: { value: mode } });
  fireEvent.change(screen.getByLabelText("服务凭证"), { target: { value: token } });
  fireEvent.click(screen.getByRole("button", { name: "应用连接" }));
}

it("审计连接、输入、展开、聚焦与返回页面不自动查询", async () => {
  const fetch = vi.fn().mockImplementation(() => Promise.resolve(json({ events: [event] })));
  vi.stubGlobal("fetch", fetch);
  const router = workspace();
  connect("audit");
  await screen.findByRole("heading", { name: "审计事件" });
  expect(fetch).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText("商户 ID（可选）"), { target: { value: event.merchantId } });
  expect(fetch).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "查询" }));
  await screen.findByText("创建订单");
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(String(fetch.mock.calls[0][0])).toContain(`merchantId=${event.merchantId}`);
  fireEvent.click(screen.getByRole("button", { name: "详情" }));
  expect(screen.getByText(event.requestId)).toBeTruthy();
  fireEvent(window, new Event("focus"));
  await act(async () => {
    await router.navigate("/orders");
  });
  await screen.findByRole("heading", { name: "工作模式不匹配" });
  await act(async () => {
    await router.navigate("/admin/audit-events");
  });
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(screen.getByText(/填写条件后点击查询/)).toBeTruthy();
});

it("更换连接取消旧读取，迟到的旧响应不能覆盖新商户或清除新连接", async () => {
  let finishOld!: (response: Response) => void;
  let oldSignal!: AbortSignal;
  const fetch = vi.fn().mockImplementation((_url: unknown, init: RequestInit) => {
    if (new Headers(init.headers).get("Authorization") === `Bearer ${merchantKey}`) {
      oldSignal = init.signal!;
      return new Promise<Response>((resolve) => {
        finishOld = resolve;
      });
    }
    return Promise.resolve(json({ orders: [{ ...order, merchantOrderNo: "商户 B 的订单" }] }));
  });
  vi.stubGlobal("fetch", fetch);
  workspace();
  connect();
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));
  fireEvent.click(screen.getByRole("button", { name: "连接设置" }));
  connect("merchant", otherKey);
  await screen.findByRole("link", { name: "商户 B 的订单" });
  expect(oldSignal.aborted).toBe(true);
  await act(async () => {
    finishOld(json({ code: "100060401", message: "authentication required" }, 401));
  });
  expect(screen.getByRole("link", { name: "商户 B 的订单" })).toBeTruthy();
  expect(screen.queryByText(/连接已失效/)).toBeNull();
  expect(Object.values(localStorage).join("")).not.toContain(merchantKey);
  expect(Object.values(localStorage).join("")).not.toContain(otherKey);
});

it("创建校验金额，超时类错误后仅重试原订单内容", async () => {
  let posts = 0;
  const fetch = vi.fn().mockImplementation((url: URL, init: RequestInit) => {
    if (init.method === "POST")
      return Promise.resolve(++posts === 1 ? json({ code: "100061032" }, 500) : json({ order }));
    return Promise.resolve(json(url.pathname.endsWith(order.id) ? { order } : { orders: posts > 1 ? [order] : [] }));
  });
  vi.stubGlobal("fetch", fetch);
  workspace();
  connect();
  await screen.findByText("还没有订单");
  fireEvent.click(screen.getAllByRole("link", { name: "创建订单" })[0]);
  await screen.findByRole("heading", { name: "创建订单" });
  fireEvent.change(screen.getByLabelText("商户订单号"), { target: { value: order.merchantOrderNo } });
  fireEvent.change(screen.getByLabelText("订单描述"), { target: { value: order.description } });
  fireEvent.change(screen.getByLabelText("金额（CNY，元）"), { target: { value: "1.001" } });
  fireEvent.click(screen.getByRole("button", { name: "创建订单" }));
  expect(posts).toBe(0);
  expect(screen.getByText("请输入金额，最多两位小数。")).toBeTruthy();
  fireEvent.change(screen.getByLabelText("金额（CNY，元）"), { target: { value: "19.90" } });
  fireEvent.click(screen.getByRole("button", { name: "创建订单" }));
  await screen.findByRole("button", { name: "重试原请求" });
  expect(screen.getByLabelText("商户订单号").matches(":disabled")).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "重试原请求" }));
  await screen.findByRole("heading", { name: order.merchantOrderNo });
  const bodies = fetch.mock.calls.filter(([, init]) => init.method === "POST").map(([, init]) => JSON.parse(init.body));
  expect(bodies).toEqual(
    Array(2).fill({ merchantOrderNo: order.merchantOrderNo, amountMinor: 1990, description: order.description }),
  );
  fireEvent.click(screen.getByRole("link", { name: "返回订单" }));
  await screen.findByRole("link", { name: order.merchantOrderNo });
  expect(
    fetch.mock.calls.filter(([url, init]) => init.method === "GET" && url.pathname === "/orderpay/v1/orders"),
  ).toHaveLength(2);
});

it("游标翻页与状态筛选贯穿 API，刷新从已缓存的第一页重新读取", async () => {
  const fetch = vi.fn().mockImplementation((url: URL) =>
    Promise.resolve(
      json({
        orders: [{ ...order, merchantOrderNo: url.searchParams.has("pageToken") ? "第二页订单" : "第一页订单" }],
        nextPageToken: url.searchParams.has("pageToken") ? "" : "opaque-next-cursor",
      }),
    ),
  );
  vi.stubGlobal("fetch", fetch);
  workspace();
  connect();
  await screen.findByRole("link", { name: "第一页订单" });
  fireEvent.click(screen.getByRole("button", { name: "下一页" }));
  await screen.findByRole("link", { name: "第二页订单" });
  expect(String(fetch.mock.calls[1][0])).toContain("pageToken=opaque-next-cursor");
  fireEvent.click(screen.getByRole("button", { name: "刷新" }));
  await screen.findByRole("link", { name: "第一页订单" });
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3));
  expect(String(fetch.mock.calls[2][0])).not.toContain("pageToken=");
  fireEvent.click(screen.getByRole("button", { name: "待支付" }));
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(4));
  expect(String(fetch.mock.calls[3][0])).toContain("status=1");
  expect(String(fetch.mock.calls[3][0])).toContain("pageSize=20");
  expect(String(fetch.mock.calls[3][0])).not.toContain("pageToken=");
});

it("取消成功后重读失败保留已确认结果，并标记旧支付信息", async () => {
  let reads = 0;
  const fetch = vi.fn().mockImplementation((_url: unknown, init: RequestInit) => {
    if (init.method === "POST") return Promise.resolve(json({ order: { ...order, status: "ORDER_STATUS_CANCELLED" } }));
    return Promise.resolve(++reads === 1 ? json({ order, payment }) : json({ code: "100061011" }, 500));
  });
  vi.stubGlobal("fetch", fetch);
  workspace(`/orders/${order.id}`);
  connect();
  await screen.findByRole("heading", { name: order.merchantOrderNo });
  fireEvent.click(screen.getByRole("button", { name: "取消订单" }));
  fireEvent.click(screen.getByRole("button", { name: "确认取消" }));
  await screen.findByText("支付信息待刷新，以下为上次读取结果。");
  expect(screen.getByText("订单已取消。")).toBeTruthy();
  expect(screen.getByText("已取消", { selector: ".status-badge" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "取消订单" })).toBeNull();
  expect(screen.queryByText("订单与支付状态组合需要核对。")).toBeNull();
});

it("取消竞争返回 409 后读取 PENDING，禁用再次取消", async () => {
  let reads = 0;
  const fetch = vi.fn().mockImplementation((_url: unknown, init: RequestInit) => {
    if (init.method === "POST") return Promise.resolve(json({ code: "100061062" }, 409));
    return Promise.resolve(
      json({
        order,
        payment: { ...payment, status: ++reads === 1 ? "PAYMENT_STATUS_CREATED" : "PAYMENT_STATUS_PENDING" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetch);
  workspace(`/orders/${order.id}`);
  connect();
  await screen.findByRole("heading", { name: order.merchantOrderNo });
  fireEvent.click(screen.getByRole("button", { name: "取消订单" }));
  fireEvent.click(screen.getByRole("button", { name: "确认取消" }));
  await screen.findByText("支付结果待确认，当前不能取消。");
  expect((screen.getByRole("button", { name: "取消订单" }) as HTMLButtonElement).disabled).toBe(true);
  expect(fetch.mock.calls.filter(([, init]) => init.method === "POST")).toHaveLength(1);
});

it("写入返回 401 清除连接，不再用失效凭证补查详情", async () => {
  const fetch = vi
    .fn()
    .mockImplementation((_url: unknown, init: RequestInit) =>
      Promise.resolve(init.method === "POST" ? json({ code: "100060401" }, 401) : json({ order, payment })),
    );
  vi.stubGlobal("fetch", fetch);
  workspace(`/orders/${order.id}`);
  connect();
  await screen.findByRole("heading", { name: order.merchantOrderNo });
  fireEvent.click(screen.getByRole("button", { name: "取消订单" }));
  fireEvent.click(screen.getByRole("button", { name: "确认取消" }));
  await screen.findByText("连接已失效，请重新配置凭证。未保存内容已清除。");
  expect(screen.getByLabelText("服务凭证")).toBeTruthy();
  expect(fetch).toHaveBeenCalledTimes(2);
});

it("未保存表单离开时确认，返回可保留输入", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation(() => Promise.resolve(json({ orders: [] }))),
  );
  workspace();
  connect();
  await screen.findByText("还没有订单");
  fireEvent.click(screen.getAllByRole("link", { name: "创建订单" })[0]);
  await screen.findByRole("heading", { name: "创建订单" });
  fireEvent.change(screen.getByLabelText("商户订单号"), { target: { value: "未提交的订单" } });
  fireEvent.click(screen.getByRole("link", { name: "返回列表" }));
  await screen.findByRole("dialog", { name: "有未保存的内容" });
  fireEvent.click(screen.getByRole("button", { name: "返回" }));
  expect((screen.getByLabelText("商户订单号") as HTMLInputElement).value).toBe("未提交的订单");
});

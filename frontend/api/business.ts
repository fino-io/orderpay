import { apiRoutes, requestApi, type ApiRouteName } from "@/api";
import { config } from "@/lib/config";

export type Connection = { token: string; scope: string; mode: "merchant" | "audit" };

export class ApiError extends Error {
  constructor(
    public status: number,
    public code = "",
  ) {
    super(errorMessage(status, code));
  }
}

function errorMessage(status: number, code: string) {
  if (code === "100061064") return "支付渠道尚未启用。";
  switch (status) {
    case 400:
      return "请求参数有误，请核对输入；分页出错时可返回第一页。";
    case 401:
      return "连接已失效，请重新配置凭证。";
    case 403:
      return "当前凭证无权访问此页面，请核对工作模式和凭证。";
    case 404:
      return "订单不存在或当前连接无权访问。";
    case 409:
      return "订单内容或状态冲突，请核对订单。";
    case 429:
      return "请求过于频繁，请稍后重试。";
    default:
      return status >= 500 ? "服务暂时不可用，请稍后重试。" : "未能连接服务，请检查网络和 API 配置。";
  }
}

export function responseData(value: unknown, status: number): unknown {
  const body = value && typeof value === "object" ? (value as Record<string, unknown>) : {};
  const error = body.error && typeof body.error === "object" ? (body.error as Record<string, unknown>) : body;
  const code = error.code === undefined ? "" : String(error.code);
  if (status >= 400 || (code && code !== "200")) {
    const businessStatus = Number(code.slice(-4));
    const domainStatus: Record<string, number> = {
      "100060601": 400,
      "100061001": 400,
      "100061022": 400,
      "100061031": 400,
      "100061012": 404,
      "100061062": 409,
      "100061063": 409,
      "100061064": 503,
    };
    const mapped =
      domainStatus[code] ?? ([400, 401, 403, 404, 409, 429].includes(businessStatus) ? businessStatus : 500);
    throw new ApiError(status >= 400 ? status : mapped, code);
  }
  return "data" in body ? body.data : value;
}

export async function businessRequest(
  connection: Connection,
  route: ApiRouteName,
  payload: Record<string, unknown>,
  signal?: AbortSignal,
) {
  let status = 0;
  let response: unknown;
  const timeout = AbortSignal.timeout(20000);
  try {
    const result = await requestApi(apiRoutes[route], payload, {
      apiBase: config.apiBase,
      signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
      headers: { Authorization: `Bearer ${connection.token}` },
      onResponseMeta: (meta) => {
        status = meta.status;
      },
      onResponseText: (text) => {
        try {
          response = JSON.parse(text);
        } catch {
          response = undefined;
        }
      },
    });
    return responseData(result, status);
  } catch (error) {
    if (signal?.aborted || error instanceof ApiError) throw error;
    if (status >= 400) responseData(response, status);
    if (status > 0) throw new Error("服务返回的数据格式有误，请检查 API 地址或网关转发配置。");
    throw new ApiError(status);
  }
}

export function apiScope(connection: Connection) {
  return [config.apiBase, connection.scope];
}

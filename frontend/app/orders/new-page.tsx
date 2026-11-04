import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { ApiError, apiScope } from "@/api/business";
import { useAccess } from "@/lib/access";
import { ErrorNotice } from "@/app/business-ui";
import { createOrder } from "./api";
import { useOrderWorkspace } from "./layout";
import { money, parseMoney, type OrderInput } from "./model";

export default function NewOrderPage() {
  const { connection, dirty } = useAccess();
  const { resetList } = useOrderWorkspace();
  const client = useQueryClient();
  const navigate = useNavigate();
  const [form, setForm] = useState({ merchantOrderNo: "", amount: "", description: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [snapshot, setSnapshot] = useState<OrderInput | null>(null);
  const mutation = useMutation({ mutationFn: (order: OrderInput) => createOrder(connection!, order) });
  const changed = Object.values(form).some(Boolean);
  const uncertain =
    !!snapshot &&
    !!mutation.error &&
    !(mutation.error instanceof ApiError && [400, 403, 404, 409, 429].includes(mutation.error.status));

  useEffect(() => {
    dirty.current = changed && !mutation.isSuccess;
    return () => {
      dirty.current = false;
    };
  }, [changed, mutation.isSuccess, dirty]);
  useEffect(() => {
    if (!mutation.isSuccess) return;
    dirty.current = false;
    resetList();
    void client.invalidateQueries({ queryKey: [...apiScope(connection!), "orders"], refetchType: "none" });
    navigate(`/orders/${mutation.data.id}`, { replace: true, state: { message: "订单已确认。" } });
  }, [mutation.isSuccess]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (mutation.isPending) return;
    if (uncertain && snapshot) {
      mutation.mutate(snapshot);
      return;
    }
    const nextErrors: Record<string, string> = {};
    if (!form.merchantOrderNo.trim() || [...form.merchantOrderNo].length > 128)
      nextErrors.merchantOrderNo = "请输入商户订单号，最多 128 个字符。";
    if (!form.description.trim() || [...form.description].length > 200)
      nextErrors.description = "请输入订单描述，最多 200 个字符。";
    let amountMinor = 0;
    try {
      amountMinor = parseMoney(form.amount);
    } catch (error) {
      nextErrors.amount = (error as Error).message;
    }
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) {
      document.getElementById(`new-${Object.keys(nextErrors)[0]}`)?.focus();
      return;
    }
    const input = { merchantOrderNo: form.merchantOrderNo, description: form.description, amountMinor };
    setSnapshot(input);
    mutation.mutate(input);
  }

  function update(name: keyof typeof form, value: string) {
    setForm({ ...form, [name]: value });
    setErrors({ ...errors, [name]: "" });
    mutation.reset();
  }

  let preview = "—";
  try {
    preview = money(parseMoney(form.amount));
  } catch {
    /* Invalid drafts are validated on submit. */
  }

  return (
    <main className="work-page">
      <Link className="back-link" to="/orders">
        返回订单
      </Link>
      <div className="page-heading">
        <div>
          <h1>创建订单</h1>
          <p className="muted">订单内容创建后不可修改</p>
        </div>
      </div>
      <form className="order-form" onSubmit={submit} noValidate>
        <fieldset disabled={mutation.isPending || uncertain}>
          <label htmlFor="new-merchantOrderNo">商户订单号</label>
          <input
            id="new-merchantOrderNo"
            value={form.merchantOrderNo}
            onChange={(event) => update("merchantOrderNo", event.target.value)}
            required
            aria-invalid={!!errors.merchantOrderNo}
            aria-describedby="order-no-help"
          />
          <p id="order-no-help" className={errors.merchantOrderNo ? "field-error" : "muted"}>
            {errors.merchantOrderNo || "使用业务系统中的唯一订单号，最多 128 字符。"}
          </p>
          <label htmlFor="new-amount">金额（CNY，元）</label>
          <input
            id="new-amount"
            inputMode="decimal"
            value={form.amount}
            placeholder="0.00"
            onChange={(event) => update("amount", event.target.value)}
            required
            aria-invalid={!!errors.amount}
            aria-describedby="amount-help"
          />
          <p id="amount-help" className={errors.amount ? "field-error" : "muted"}>
            {errors.amount || "0.01～1,000,000.00 元，最多两位小数。"}
          </p>
          <label htmlFor="new-description">订单描述</label>
          <textarea
            id="new-description"
            rows={3}
            value={form.description}
            onChange={(event) => update("description", event.target.value)}
            required
            aria-invalid={!!errors.description}
            aria-describedby="description-help"
          />
          <p id="description-help" className={errors.description ? "field-error" : "muted"}>
            {errors.description || `${[...form.description].length} / 200 字符`}
          </p>
        </fieldset>
        <div className="order-review">
          <span className="muted">确认订单金额</span>
          <strong>{preview} CNY</strong>
          <p>创建后订单号、金额和描述不可修改。</p>
        </div>
        {mutation.error instanceof ApiError && mutation.error.status === 409 ? (
          <div className="work-notice danger" role="alert">
            同一商户订单号的内容与已有订单不同，请核对原单。
          </div>
        ) : (
          <ErrorNotice error={mutation.error} />
        )}
        {uncertain && (
          <div className="work-notice warning" role="alert">
            <p>提交结果尚未确认。可重试原请求，系统会按相同订单号和内容处理。</p>
            <button
              type="button"
              className="secondary"
              onClick={() => {
                if (window.confirm("原请求可能已成功，请先核对原单。确认结束原请求重试并修改表单？")) {
                  setSnapshot(null);
                  mutation.reset();
                }
              }}
            >
              返回修改
            </button>
          </div>
        )}
        <div className="button-row">
          <button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "创建中…" : uncertain ? "重试原请求" : "创建订单"}
          </button>
          <Link className="action-link" to="/orders">
            返回列表
          </Link>
        </div>
      </form>
    </main>
  );
}

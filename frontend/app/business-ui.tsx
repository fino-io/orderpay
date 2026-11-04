import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { ApiError } from "@/api/business";
import { knownStatus, statusLabel } from "@/app/orders/model";

export function ErrorNotice({ error, children }: { error: Error | null; children?: ReactNode }) {
  if (!error) return null;
  return (
    <div className="work-notice danger" role="alert">
      <p>{error.message}</p>
      {error instanceof ApiError && error.code && <small>错误码：{error.code}</small>}
      {children}
    </div>
  );
}

export function CopyValue({ value, short = false }: { value: string; short?: boolean }) {
  const [feedback, setFeedback] = useState("");
  if (!value) return <span className="muted">—</span>;
  return (
    <span className="copy-value">
      <span className="identifier" title={value}>
        {short && value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value}
      </span>
      <button
        className="text-button"
        type="button"
        aria-label={`复制 ${value}`}
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value);
            setFeedback("已复制");
          } catch {
            setFeedback("复制失败，请选择完整值手动复制");
          }
        }}
      >
        复制
      </button>
      <span className="copy-feedback" role="status">
        {feedback}
      </span>
      {short && <span className="sr-only">完整值：{value}</span>}
    </span>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const tone =
    !knownStatus(status) || status === "PAYMENT_STATUS_PENDING"
      ? "warning"
      : ["ORDER_STATUS_PAID", "PAYMENT_STATUS_SUCCEEDED"].includes(status)
        ? "success"
        : "neutral";
  return (
    <span className={`status-badge ${tone}`} title={status}>
      {statusLabel(status)}
    </span>
  );
}

export function Pagination({
  previous,
  next,
  busy,
  onPrevious,
  onNext,
}: {
  previous: boolean;
  next: boolean;
  busy: boolean;
  onPrevious: () => void;
  onNext: () => void;
}) {
  return (
    <nav className="work-pagination" aria-label="列表分页">
      <span className="muted">创建时间从早到晚 · 每页 20 条</span>
      <div className="button-row">
        <button type="button" className="secondary" disabled={!previous || busy} onClick={onPrevious}>
          上一页
        </button>
        <button type="button" className="secondary" disabled={!next || busy} onClick={onNext}>
          下一页
        </button>
      </div>
    </nav>
  );
}

export function ConfirmDialog({
  title,
  children,
  confirmLabel,
  busy = false,
  onConfirm,
  onClose,
}: {
  title: string;
  children: ReactNode;
  confirmLabel: string;
  busy?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const element = ref.current!;
    const previous = document.activeElement as HTMLElement | null;
    element.showModal();
    return () => {
      element.close();
      previous?.focus();
    };
  }, []);

  return (
    <dialog
      ref={ref}
      className="work-dialog"
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <h2 id={titleId}>{title}</h2>
      <div className="dialog-body">{children}</div>
      <div className="button-row dialog-actions">
        <button className="secondary" type="button" autoFocus disabled={busy} onClick={onClose}>
          返回
        </button>
        <button type="button" disabled={busy} onClick={onConfirm}>
          {busy ? "处理中…" : confirmLabel}
        </button>
      </div>
    </dialog>
  );
}

export function LoadingRows() {
  return (
    <div className="loading-rows" role="status" aria-label="正在加载">
      <span />
      <span />
      <span />
    </div>
  );
}

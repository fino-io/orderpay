import { Fragment, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AccessBoundary, useAccess } from "@/lib/access";
import { CopyValue, ErrorNotice, LoadingRows, Pagination } from "@/app/business-ui";
import { isUUID, statusLabel, time } from "@/app/orders/model";
import { actionLabel, auditOptions, type AuditFilter } from "./api";

const emptyFilter: AuditFilter = { merchantId: "", resourceId: "" };

export default function AuditEventsPage() {
  return (
    <AccessBoundary mode="audit">
      <AuditEvents />
    </AccessBoundary>
  );
}

function AuditEvents() {
  const { connection } = useAccess();
  const client = useQueryClient();
  const [draft, setDraft] = useState(emptyFilter);
  const [errors, setErrors] = useState<Partial<AuditFilter>>({});
  const [request, setRequest] = useState({ filter: emptyFilter, tokens: [""], index: 0 });
  const [queried, setQueried] = useState(false);
  const [selected, setSelected] = useState("");
  const query = useQuery({
    ...auditOptions(connection!, request.filter, request.tokens[request.index]),
    enabled: false,
  });

  function run(filter: AuditFilter, tokens = [""], index = 0) {
    if (query.isFetching) return;
    setRequest({ filter, tokens, index });
    setQueried(true);
    setSelected("");
    void client.fetchQuery({ ...auditOptions(connection!, filter, tokens[index]), staleTime: 0 }).catch(() => {});
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    const filter = { merchantId: draft.merchantId.trim(), resourceId: draft.resourceId.trim() };
    const nextErrors: Partial<AuditFilter> = {};
    if (filter.merchantId && !isUUID(filter.merchantId)) nextErrors.merchantId = "请输入完整商户 UUID。";
    if (filter.resourceId && !isUUID(filter.resourceId)) nextErrors.resourceId = "请输入完整资源 UUID。";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) {
      document.getElementById(`audit-${Object.keys(nextErrors)[0]}`)?.focus();
      return;
    }
    run(filter);
  }

  const data = queried ? query.data : undefined;
  const nextToken = data?.nextPageToken;

  return (
    <main className="work-page">
      <div className="page-heading">
        <div>
          <h1>审计事件</h1>
          <p className="muted">按商户或资源核对业务事件</p>
        </div>
      </div>
      <form className="audit-filters" onSubmit={submit} noValidate>
        <label htmlFor="audit-merchantId">
          商户 ID（可选）
          <input
            id="audit-merchantId"
            value={draft.merchantId}
            onChange={(event) => setDraft({ ...draft, merchantId: event.target.value })}
            aria-invalid={!!errors.merchantId}
            aria-describedby="audit-merchant-error"
          />
          <span id="audit-merchant-error" className="field-error">
            {errors.merchantId}
          </span>
        </label>
        <label htmlFor="audit-resourceId">
          资源 ID（可选）
          <input
            id="audit-resourceId"
            value={draft.resourceId}
            onChange={(event) => setDraft({ ...draft, resourceId: event.target.value })}
            aria-invalid={!!errors.resourceId}
            aria-describedby="audit-resource-error"
          />
          <span id="audit-resource-error" className="field-error">
            {errors.resourceId}
          </span>
        </label>
        <div className="button-row">
          <button type="submit" disabled={query.isFetching}>
            {query.isFetching ? "查询中…" : "查询"}
          </button>
          <button
            type="button"
            className="secondary"
            disabled={query.isFetching}
            onClick={() => {
              setDraft(emptyFilter);
              setErrors({});
              setQueried(false);
              setSelected("");
              setRequest({ filter: emptyFilter, tokens: [""], index: 0 });
            }}
          >
            重置
          </button>
        </div>
      </form>
      <div className="audit-help muted">
        <p>每次查询或翻页均会留下审计记录。资源 ID 精确匹配，订单事件不自动包含支付事件。</p>
        {queried && (
          <p>
            当前查询：商户 {request.filter.merchantId || "全部"} · 资源 {request.filter.resourceId || "全部"}
          </p>
        )}
      </div>
      {queried && (
        <ErrorNotice error={query.error}>
          {request.index > 0 && (
            <button type="button" className="secondary" onClick={() => run(request.filter)}>
              返回第一页
            </button>
          )}
        </ErrorNotice>
      )}
      {queried && query.isFetching && <LoadingRows />}
      {!queried ? (
        <div className="work-empty">填写条件后点击查询，也可不填条件查询全部事件。</div>
      ) : (
        data && (
          <>
            <p className="query-status muted" role="status">
              {query.isError ? "数据可能已过期 · " : ""}最后获取：{time(query.dataUpdatedAt)}
            </p>
            {data.events.length ? (
              <div className="table-region">
                <table className="work-table audit-table">
                  <caption className="sr-only">审计事件，创建时间从早到晚</caption>
                  <thead>
                    <tr>
                      <th scope="col">发生时间</th>
                      <th scope="col">操作</th>
                      <th scope="col">商户 / 资源</th>
                      <th scope="col">状态变化</th>
                      <th scope="col">操作者</th>
                      <th scope="col">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.events.map((event) => (
                      <Fragment key={event.id}>
                        <tr>
                          <td>{time(event.createTime)}</td>
                          <td title={event.action}>{actionLabel(event.action)}</td>
                          <td>
                            <CopyValue value={event.merchantId} short />
                            <br />
                            <CopyValue value={event.resourceId} short />
                          </td>
                          <td>
                            {statusLabel(event.beforeStatus)} → {statusLabel(event.afterStatus)}
                          </td>
                          <td>
                            <CopyValue value={event.actorId} short />
                          </td>
                          <td>
                            <button
                              className="text-button"
                              type="button"
                              aria-expanded={selected === event.id}
                              aria-controls={`event-${event.id}`}
                              onClick={() => setSelected(selected === event.id ? "" : event.id)}
                            >
                              详情
                            </button>
                          </td>
                        </tr>
                        <tr id={`event-${event.id}`} hidden={selected !== event.id}>
                          <td colSpan={6}>
                            <dl className="audit-detail">
                              <dt>事件 ID</dt>
                              <dd>
                                <CopyValue value={event.id} />
                              </dd>
                              <dt>操作</dt>
                              <dd>{event.action}</dd>
                              <dt>商户 ID</dt>
                              <dd>
                                <CopyValue value={event.merchantId} />
                              </dd>
                              <dt>资源 ID</dt>
                              <dd>
                                <CopyValue value={event.resourceId} />
                              </dd>
                              <dt>操作者</dt>
                              <dd>
                                <CopyValue value={event.actorId} />
                              </dd>
                              <dt>请求关联 ID</dt>
                              <dd>
                                <CopyValue value={event.requestId} />
                              </dd>
                              <dt>变更前状态</dt>
                              <dd>{event.beforeStatus || "—"}</dd>
                              <dt>变更后状态</dt>
                              <dd>{event.afterStatus || "—"}</dd>
                              <dt>原因码</dt>
                              <dd>{event.reasonCode || "—"}</dd>
                              <dt>证据引用</dt>
                              <dd>
                                <CopyValue value={event.evidenceReference} />
                              </dd>
                            </dl>
                          </td>
                        </tr>
                      </Fragment>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="work-empty">没有符合当前条件的事件。可修改条件后重新查询。</div>
            )}
            <Pagination
              previous={request.index > 0}
              next={!!nextToken && !query.isError}
              busy={query.isFetching}
              onPrevious={() => run(request.filter, request.tokens, request.index - 1)}
              onNext={() => {
                if (nextToken)
                  run(request.filter, [...request.tokens.slice(0, request.index + 1), nextToken], request.index + 1);
              }}
            />
          </>
        )
      )}
    </main>
  );
}

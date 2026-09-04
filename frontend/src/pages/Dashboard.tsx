import { useCallback, useEffect, useState } from 'react'
import PageLayout from '../components/PageLayout.tsx'

type Message = {
  id: number
  body: string
  createdAt: string
}

type Item = {
  id: number
  externalId: string
  title: string
  status: string
  createdAt: string
  processedAt: string | null
}

type JobLog = {
  id: number
  jobId: string
  jobType: string
  detail: string
  createdAt: string
}

type ExampleResponse = {
  service: string
  database: {
    host: string
    connected: boolean
    messages: Message[]
    items: Item[]
    jobLogs: JobLog[]
    error?: string
  }
  queue: {
    addr: string
    connected: boolean
    length: number
    error?: string
  }
}

type LoadState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ok'; data: ExampleResponse }
  | { status: 'error'; message: string }

function StatusBadge({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${
        ok ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'
      }`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${ok ? 'bg-emerald-500' : 'bg-rose-500'}`}
      />
      {label}
    </span>
  )
}

export default function Dashboard() {
  const [state, setState] = useState<LoadState>({ status: 'idle' })
  const [enqueueState, setEnqueueState] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle')
  const [enqueueMessage, setEnqueueMessage] = useState('')

  const load = useCallback(async () => {
    setState({ status: 'loading' })
    try {
      const res = await fetch('/api/example')
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `HTTP ${res.status}`)
      }
      const data = (await res.json()) as ExampleResponse
      setState({ status: 'ok', data })
    } catch (error) {
      const message =
        error instanceof Error ? error.message : '不明なエラーが発生しました'
      setState({ status: 'error', message })
    }
  }, [])

  const enqueue = useCallback(async () => {
    setEnqueueState('loading')
    setEnqueueMessage('')
    try {
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ note: 'frontend からのジョブ' }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `HTTP ${res.status}`)
      }
      setEnqueueState('ok')
      setEnqueueMessage('Queue にジョブを投入しました。Worker が処理します。')
      await load()
    } catch (error) {
      setEnqueueState('error')
      setEnqueueMessage(
        error instanceof Error ? error.message : 'ジョブ投入に失敗しました',
      )
    }
  }, [load])

  useEffect(() => {
    void load()
  }, [load])

  const data = state.status === 'ok' ? state.data : null
  const messages = data?.database.messages ?? []
  const items = data?.database.items ?? []
  const jobLogs = data?.database.jobLogs ?? []

  return (
    <PageLayout
      title="開発環境コンソール"
      description="Frontend は Internal API だけを呼びます。External API は Batch からのみ利用します。"
      width="wide"
      panel={false}
      actions={
        <div className="flex flex-col gap-2 sm:flex-row">
          <button
            type="button"
            onClick={() => void enqueue()}
            disabled={enqueueState === 'loading'}
            className="inline-flex items-center justify-center rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm font-medium text-slate-900 transition hover:bg-stone-50 disabled:cursor-not-allowed disabled:text-slate-400"
          >
            {enqueueState === 'loading' ? '投入中...' : 'ジョブを投入'}
          </button>
          <button
            type="button"
            onClick={() => void load()}
            disabled={state.status === 'loading'}
            className="inline-flex items-center justify-center rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-700 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {state.status === 'loading' ? '確認中...' : '状態を再読込'}
          </button>
        </div>
      }
    >

        <section className="mb-6 rounded-2xl border border-stone-200 bg-white p-4 shadow-sm sm:p-5">
          <h2 className="text-sm font-semibold text-slate-900">接続経路</h2>
          <p className="mt-2 overflow-x-auto font-mono text-xs leading-6 text-slate-600 sm:text-sm">
            Browser → frontend → internal-api:8080 → db:3306 / queue:6379
          </p>
          <p className="mt-1 overflow-x-auto font-mono text-xs leading-6 text-slate-600 sm:text-sm">
            docker compose run --rm batch → external-api:8081 / queue:6379 / db:3306
          </p>
        </section>

        {state.status === 'loading' && (
          <p className="mb-6 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
            Internal API に問い合わせています...
          </p>
        )}

        {state.status === 'error' && (
          <p className="mb-6 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-900">
            API の取得に失敗しました: {state.message}
          </p>
        )}

        {enqueueMessage && (
          <p
            className={`mb-6 rounded-xl border px-4 py-3 text-sm ${
              enqueueState === 'error'
                ? 'border-rose-200 bg-rose-50 text-rose-900'
                : 'border-emerald-200 bg-emerald-50 text-emerald-900'
            }`}
          >
            {enqueueMessage}
          </p>
        )}

        <div className="grid gap-4 md:grid-cols-3">
          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-base font-semibold">Internal API</h2>
              <StatusBadge
                ok={state.status === 'ok'}
                label={state.status === 'ok' ? '接続中' : '未確認'}
              />
            </div>
            <p className="mt-3 text-sm text-slate-600">
              Vite proxy 経由で <code>/api/*</code> を{' '}
              <code>internal-api:8080</code> へ転送しています。
            </p>
          </article>

          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-base font-semibold">MySQL</h2>
              <StatusBadge
                ok={Boolean(data?.database.connected)}
                label={data?.database.connected ? '接続中' : '未接続'}
              />
            </div>
            <p className="mt-3 text-sm text-slate-600">
              Internal API と Worker は <code>db:3306</code> で接続します。
            </p>
            <p className="mt-4 font-mono text-xs text-slate-500">
              {data?.database.host ?? 'db:3306'}
            </p>
            {data?.database.error && (
              <p className="mt-2 text-xs text-rose-700">{data.database.error}</p>
            )}
          </article>

          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-base font-semibold">Redis Queue</h2>
              <StatusBadge
                ok={Boolean(data?.queue.connected)}
                label={data?.queue.connected ? '接続中' : '未接続'}
              />
            </div>
            <p className="mt-3 text-sm text-slate-600">
              待機中のジョブ数: {data?.queue.connected ? data.queue.length : '-'}
            </p>
            <p className="mt-4 font-mono text-xs text-slate-500">
              {data?.queue.addr ?? 'queue:6379'}
            </p>
            {data?.queue.error && (
              <p className="mt-2 text-xs text-rose-700">{data.queue.error}</p>
            )}
          </article>
        </div>

        <section className="mt-6 rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
          <h2 className="text-base font-semibold">Batch の手動実行</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Batch は指定時刻に自動実行されません。開発環境では、次のコマンドを実行した時点を「その時刻が来た」とみなします。
          </p>
          <pre className="mt-3 overflow-x-auto rounded-xl bg-slate-900 px-4 py-3 text-sm text-slate-100">
            docker compose run --rm batch
          </pre>
          <p className="mt-3 text-sm text-slate-600">
            Batch は External API からデータを取得し、MySQL へ保存したあと Queue へジョブを投入します。実処理は Worker が行います。
          </p>
        </section>

        <section className="mt-6 grid gap-4 lg:grid-cols-2">
          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <h2 className="text-base font-semibold">Batch が取り込んだ外部データ</h2>
            {state.status === 'ok' && items.length === 0 && (
              <p className="mt-3 text-sm text-slate-500">
                まだ取り込まれていません。Batch を実行してください。
              </p>
            )}
            {items.length > 0 && (
              <ul className="mt-4 space-y-3">
                {items.map((item) => (
                  <li
                    key={item.id}
                    className="rounded-xl border border-stone-100 bg-stone-50 px-4 py-3"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-sm text-slate-800">{item.title}</p>
                      <StatusBadge
                        ok={item.status === 'processed'}
                        label={item.status}
                      />
                    </div>
                    <p className="mt-1 text-xs text-slate-500">
                      {item.externalId} / #{item.id}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </article>

          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <h2 className="text-base font-semibold">Worker の処理ログ</h2>
            {state.status === 'ok' && jobLogs.length === 0 && (
              <p className="mt-3 text-sm text-slate-500">
                まだジョブは処理されていません。
              </p>
            )}
            {jobLogs.length > 0 && (
              <ul className="mt-4 space-y-3">
                {jobLogs.map((log) => (
                  <li
                    key={log.id}
                    className="rounded-xl border border-stone-100 bg-stone-50 px-4 py-3"
                  >
                    <p className="text-sm text-slate-800">{log.detail}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {log.jobType} / {log.jobId}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </article>
        </section>

        <section className="mt-6 rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
          <h2 className="text-base font-semibold">MySQL のメッセージ</h2>
          {state.status === 'ok' && messages.length === 0 && (
            <p className="mt-3 text-sm text-slate-500">メッセージはまだありません。</p>
          )}
          {messages.length > 0 && (
            <ul className="mt-4 space-y-3">
              {messages.map((message) => (
                <li
                  key={message.id}
                  className="rounded-xl border border-stone-100 bg-stone-50 px-4 py-3"
                >
                  <p className="text-sm text-slate-800">{message.body}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    #{message.id} / {message.createdAt}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </section>
    </PageLayout>
  )
}

import { useCallback, useEffect, useState } from 'react'

type Message = {
  id: number
  body: string
  createdAt: string
}

type ExampleResponse = {
  service: string
  database: {
    host: string
    connected: boolean
    messages: Message[]
    error?: string
  }
  externalApi: {
    url: string
    ok: boolean
    message?: string
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

export default function App() {
  const [state, setState] = useState<LoadState>({ status: 'idle' })

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

  useEffect(() => {
    void load()
  }, [load])

  const data = state.status === 'ok' ? state.data : null
  const messages = data?.database.messages ?? []

  return (
    <div className="min-h-svh px-4 py-8 sm:px-6 lg:px-8">
      <main className="mx-auto max-w-5xl">
        <header className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-sm font-medium tracking-wide text-teal-800">
              Docker Compose 開発環境
            </p>
            <h1 className="mt-1 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
              開発環境コンソール
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
              Frontend から相対パス <code className="rounded bg-white px-1.5 py-0.5">/api/example</code>{' '}
              を呼び出し、Internal API・MySQL・External API の接続を確認します。
            </p>
          </div>
          <button
            type="button"
            onClick={() => void load()}
            disabled={state.status === 'loading'}
            className="inline-flex items-center justify-center rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-700 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {state.status === 'loading' ? '確認中...' : '接続を再確認'}
          </button>
        </header>

        <section className="mb-6 rounded-2xl border border-stone-200 bg-white p-4 shadow-sm sm:p-5">
          <h2 className="text-sm font-semibold text-slate-900">接続経路</h2>
          <p className="mt-2 overflow-x-auto font-mono text-xs leading-6 text-slate-600 sm:text-sm">
            Browser → localhost:5173 → frontend → /api/* → internal-api:8080 → db:3306 / external-api:8081
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

        {state.status === 'idle' && (
          <p className="mb-6 rounded-xl border border-stone-200 bg-white px-4 py-3 text-sm text-slate-600">
            接続確認をまだ実行していません。
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
            <p className="mt-4 font-mono text-xs text-slate-500">
              {data?.service ?? 'internal-api'}
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
              Internal API は Compose のサービス名 <code>db:3306</code> で接続します。
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
              <h2 className="text-base font-semibold">External API</h2>
              <StatusBadge
                ok={Boolean(data?.externalApi.ok)}
                label={data?.externalApi.ok ? '応答あり' : '未応答'}
              />
            </div>
            <p className="mt-3 text-sm text-slate-600">
              Internal API から <code>http://external-api:8081</code> を呼び出します。
            </p>
            <p className="mt-4 font-mono text-xs text-slate-500">
              {data?.externalApi.url ?? 'http://external-api:8081'}
            </p>
            {data?.externalApi.error && (
              <p className="mt-2 text-xs text-rose-700">{data.externalApi.error}</p>
            )}
          </article>
        </div>

        <section className="mt-6 grid gap-4 lg:grid-cols-2">
          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <h2 className="text-base font-semibold">MySQL の初期メッセージ</h2>
            {state.status === 'ok' && messages.length === 0 && (
              <p className="mt-3 text-sm text-slate-500">
                メッセージはまだありません。
              </p>
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
          </article>

          <article className="rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
            <h2 className="text-base font-semibold">External API の応答</h2>
            {data?.externalApi.message ? (
              <p className="mt-4 rounded-xl border border-teal-100 bg-teal-50 px-4 py-3 text-sm text-teal-950">
                {data.externalApi.message}
              </p>
            ) : (
              <p className="mt-3 text-sm text-slate-500">
                External API からのメッセージはまだありません。
              </p>
            )}
          </article>
        </section>
      </main>
    </div>
  )
}

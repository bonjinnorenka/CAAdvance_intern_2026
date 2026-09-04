import { Link } from 'react-router-dom'

const items = [
  {
    to: '/create-report',
    title: 'レポート作成画面',
    description: '広告アカウントと期間を指定して、レポート作成を依頼します。',
  },
  {
    to: '/history',
    title: 'レポート履歴',
    description: '作成済みレポートの一覧と、CSV のダウンロードを確認します。',
  },
  {
    to: '/admin',
    title: '管理者ページ',
    description: 'ユーザーと広告アカウントの権限を管理します。',
  },
  {
    to: '/dashboard',
    title: 'ダッシュボード',
    description: '開発環境の接続状態、Queue、Batch の状況を確認します。',
  },
] as const

export default function Menu() {
  return (
    <div className="min-h-svh px-4 py-8 sm:px-6 lg:px-8">
      <main className="mx-auto max-w-3xl">
        <header className="mb-8">
          <p className="text-sm font-medium tracking-wide text-teal-800">
            広告配信レポートシステム
          </p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
            メニュー
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            利用する画面を選んでください。
          </p>
        </header>

        <ul className="grid gap-4 sm:grid-cols-2">
          {items.map((item) => (
            <li key={item.to}>
              <Link
                to={item.to}
                className="block h-full rounded-2xl border border-stone-200 bg-white p-5 shadow-sm transition hover:border-teal-300 hover:bg-stone-50"
              >
                <h2 className="text-base font-semibold text-slate-900">{item.title}</h2>
                <p className="mt-2 text-sm leading-6 text-slate-600">{item.description}</p>
              </Link>
            </li>
          ))}
        </ul>
      </main>
    </div>
  )
}

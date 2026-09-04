import { Link } from 'react-router-dom'
import PageLayout from '../components/PageLayout.tsx'

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
    <PageLayout
      title="メニュー"
      description="利用する画面を選んでください。"
      backTo={null}
      width="medium"
      panel={false}
    >
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
    </PageLayout>
  )
}

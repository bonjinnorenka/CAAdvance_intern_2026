import { Link } from 'react-router-dom'

type PlaceholderPageProps = {
  title: string
  description: string
}

export default function PlaceholderPage({ title, description }: PlaceholderPageProps) {
  return (
    <div className="min-h-svh px-4 py-8 sm:px-6 lg:px-8">
      <main className="mx-auto max-w-3xl">
        <p className="mb-4">
          <Link
            to="/"
            className="text-sm font-medium text-teal-800 transition hover:text-teal-950"
          >
            ← メニューへ
          </Link>
        </p>
        <h1 className="text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
          {title}
        </h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">{description}</p>
        <section className="mt-6 rounded-2xl border border-stone-200 bg-white p-5 shadow-sm">
          <p className="text-sm text-slate-600">この画面はこれから実装します。</p>
        </section>
      </main>
    </div>
  )
}

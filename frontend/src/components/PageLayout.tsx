import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

type PageLayoutProps = {
  title: string
  description?: string
  backTo?: string | null
  backLabel?: string
  width?: 'default' | 'medium' | 'wide'
  panel?: boolean
  actions?: ReactNode
  children: ReactNode
}

const widthClass = {
  default: 'max-w-[640px]',
  medium: 'max-w-3xl',
  wide: 'max-w-5xl',
} as const

export default function PageLayout({
  title,
  description,
  backTo = '/',
  backLabel = '← メニューへ',
  width = 'default',
  panel = true,
  actions,
  children,
}: PageLayoutProps) {
  const header = (
    <header className={actions ? 'mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between' : 'mb-8'}>
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-slate-900">{title}</h1>
        {description && (
          <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
        )}
      </div>
      {actions}
    </header>
  )

  return (
    <div className="min-h-svh bg-[#f0f2f5] px-4 py-10 sm:px-6">
      <div className={`mx-auto ${widthClass[width]}`}>
        {backTo && (
          <p className="mb-4">
            <Link
              to={backTo}
              className="text-sm font-medium text-slate-500 transition hover:text-slate-800"
            >
              {backLabel}
            </Link>
          </p>
        )}
        {panel ? (
          <section className="rounded-2xl bg-white px-8 py-8 shadow-[0_8px_30px_rgba(15,23,42,0.08)] sm:px-10 sm:py-9">
            {header}
            {children}
          </section>
        ) : (
          <>
            {header}
            {children}
          </>
        )}
      </div>
    </div>
  )
}

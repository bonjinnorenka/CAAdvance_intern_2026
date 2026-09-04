import { useCallback, useEffect, useState, type FormEvent } from 'react'
import PageLayout from '../components/PageLayout.tsx'

const currentUserId = '1'

type AdAccount = {
  id: string
  name: string
}

type ApiErrorBody = {
  error?: {
    code?: string
    message?: string
    unauthorized_account_ids?: string[]
  }
}

type ReportCreateAccepted = {
  job_id: number
  status: string
}

export default function CreateReport() {
  const [accounts, setAccounts] = useState<AdAccount[]>([])
  const [loadError, setLoadError] = useState('')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [marginRate, setMarginRate] = useState('')
  const [submitError, setSubmitError] = useState('')
  const [submitOk, setSubmitOk] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const loadAccounts = useCallback(async () => {
    setLoadError('')
    try {
      const res = await fetch('/me/ad_accounts', {
        headers: { 'X-User-Id': currentUserId },
      })
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as ApiErrorBody | null
        throw new Error(body?.error?.message || `HTTP ${res.status}`)
      }
      const data = (await res.json()) as AdAccount[]
      setAccounts(data)
    } catch (error) {
      setAccounts([])
      setLoadError(
        error instanceof Error ? error.message : '広告アカウントの取得に失敗しました',
      )
    }
  }, [])

  useEffect(() => {
    void loadAccounts()
  }, [loadAccounts])

  const toggleAccount = (id: string) => {
    setSelectedIds((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    )
  }

  const validate = () => {
    if (selectedIds.length === 0) {
      return '広告アカウントを1件以上選択してください'
    }
    if (!dateFrom || !dateTo) {
      return 'レポート期間を入力してください'
    }
    if (dateFrom > dateTo) {
      return '開始日は終了日以前である必要があります'
    }
    if (marginRate === '') {
      return 'マージン料率を入力してください'
    }
    const margin = Number(marginRate)
    if (!Number.isInteger(margin) || margin < 0 || margin >= 100) {
      return 'マージン料率は 0 以上 100 未満の整数で入力してください'
    }
    return ''
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitError('')
    setSubmitOk('')

    const message = validate()
    if (message) {
      setSubmitError(message)
      return
    }

    setSubmitting(true)
    try {
      const res = await fetch('/report', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-Id': currentUserId,
        },
        body: JSON.stringify({
          ad_account_ids: selectedIds,
          date_from: dateFrom,
          date_to: dateTo,
          margin_rate: Number(marginRate),
        }),
      })
      const body = (await res.json().catch(() => null)) as
        | (ReportCreateAccepted & ApiErrorBody)
        | null
      if (!res.ok) {
        throw new Error(body?.error?.message || `HTTP ${res.status}`)
      }
      setSubmitOk(`レポート作成を受け付けました（ジョブ ID: ${body?.job_id ?? '-'}）`)
    } catch (error) {
      setSubmitError(
        error instanceof Error ? error.message : 'レポート作成に失敗しました',
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <PageLayout
      title="レポート作成"
      description="広告アカウントと期間、マージン料率を指定してレポートを作成します。"
    >
      <form onSubmit={(event) => void onSubmit(event)}>
          <div className="space-y-7">
            <fieldset>
              <legend className="flex flex-wrap items-baseline gap-x-2">
                <span className="text-sm font-semibold text-slate-900">
                  広告アカウント選択
                  <span className="ml-0.5 text-red-500">*</span>
                </span>
                <span className="text-xs text-red-500">複数選択可能</span>
              </legend>

              <div className="mt-2 max-h-56 overflow-y-auto rounded-lg border border-slate-200">
                {loadError && (
                  <p className="px-4 py-3 text-sm text-red-600">{loadError}</p>
                )}
                {!loadError && accounts.length === 0 && (
                  <p className="px-4 py-3 text-sm text-slate-500">
                    利用できる広告アカウントがありません。
                  </p>
                )}
                {accounts.map((account) => (
                  <label
                    key={account.id}
                    className="flex cursor-pointer items-center justify-between gap-3 border-b border-slate-100 px-4 py-3 last:border-b-0 hover:bg-slate-50"
                  >
                    <span className="flex min-w-0 items-center gap-3">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(account.id)}
                        onChange={() => toggleAccount(account.id)}
                        className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="truncate text-sm text-slate-800">{account.name}</span>
                    </span>
                    <span className="shrink-0 text-sm text-slate-400">{account.id}</span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset>
              <legend className="text-sm font-semibold text-slate-900">
                レポート期間
                <span className="ml-0.5 text-red-500">*</span>
              </legend>
              <div className="mt-2 flex flex-col items-stretch gap-3 sm:flex-row sm:items-end">
                <label className="min-w-0 flex-1">
                  <span className="mb-1.5 block text-xs text-slate-500">開始日</span>
                  <input
                    type="date"
                    value={dateFrom}
                    onChange={(event) => setDateFrom(event.target.value)}
                    placeholder="年 / 月 / 日"
                    className="w-full rounded-lg border border-slate-300 px-3 py-2.5 text-sm text-slate-800 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                  />
                </label>
                <span className="hidden pb-2.5 text-slate-400 sm:block">~</span>
                <label className="min-w-0 flex-1">
                  <span className="mb-1.5 block text-xs text-slate-500">終了日</span>
                  <input
                    type="date"
                    value={dateTo}
                    onChange={(event) => setDateTo(event.target.value)}
                    placeholder="年 / 月 / 日"
                    className="w-full rounded-lg border border-slate-300 px-3 py-2.5 text-sm text-slate-800 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                  />
                </label>
              </div>
            </fieldset>

            <div>
              <label htmlFor="margin-rate" className="text-sm font-semibold text-slate-900">
                マージン料率
                <span className="ml-0.5 text-red-500">*</span>
              </label>
              <div className="relative mt-2 w-40">
                <input
                  id="margin-rate"
                  type="number"
                  min={0}
                  max={99}
                  step={1}
                  inputMode="numeric"
                  placeholder="0.0"
                  value={marginRate}
                  onChange={(event) => setMarginRate(event.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2.5 pr-8 text-sm text-slate-800 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
                <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-slate-500">
                  %
                </span>
              </div>
            </div>
          </div>

          {submitError && (
            <p className="mt-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {submitError}
            </p>
          )}
          {submitOk && (
            <p className="mt-6 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
              {submitOk}
            </p>
          )}

          <div className="mt-8 flex justify-end">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300"
            >
              {submitting ? '作成中...' : '作成'}
            </button>
          </div>
      </form>
    </PageLayout>
  )
}

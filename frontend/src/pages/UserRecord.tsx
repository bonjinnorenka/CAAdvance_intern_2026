import { useCallback, useEffect, useState } from 'react'
import PageLayout from '../components/PageLayout.tsx'

const currentUserId = '1'

type ReportStatus = 'queued' | 'processing' | 'completed' | 'failed' | string

type Report = {
  id: number
  name: string
  status: ReportStatus
  reason: string | null
  created_at: string
}

type ApiErrorBody = {
  error?: {
    code?: string
    message?: string
  }
}

function parseCreatedAt(iso: string) {
  const match = iso.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})/)
  if (!match) {
    return { title: iso, filename: `${iso}.csv` }
  }
  const [, year, month, day, hour, minute, second] = match
  return {
    title: `${year}-${month}-${day} ${hour}:${minute}:${second}`,
    filename: `${year}-${month}-${day}_${hour}-${minute}-${second}.csv`,
  }
}

function statusClass(status: ReportStatus) {
  if (status === 'completed') {
    return 'bg-green-100 text-green-800'
  }
  if (status === 'failed') {
    return 'bg-red-100 text-red-800'
  }
  if (status === 'queued') {
    return 'bg-yellow-100 text-yellow-800'
  }
  return 'bg-slate-100 text-slate-700'
}

export default function UserRecord() {
  const [reports, setReports] = useState<Report[]>([])
  const [loadError, setLoadError] = useState('')
  const [actionError, setActionError] = useState('')
  const [downloadingId, setDownloadingId] = useState<number | null>(null)

  const loadReports = useCallback(async () => {
    setLoadError('')
    try {
      const res = await fetch('/me/reports', {
        headers: { 'X-User-Id': currentUserId },
      })
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as ApiErrorBody | null
        throw new Error(body?.error?.message || `HTTP ${res.status}`)
      }
      const data = (await res.json()) as Report[]
      setReports(data)
    } catch (error) {
      setReports([])
      setLoadError(
        error instanceof Error ? error.message : 'レポート履歴の取得に失敗しました',
      )
    }
  }, [])

  useEffect(() => {
    void loadReports()
  }, [loadReports])

  const download = async (report: Report) => {
    setActionError('')
    if (report.status !== 'completed') {
      setActionError('完了したレポートのみダウンロードできます')
      return
    }

    setDownloadingId(report.id)
    try {
      const res = await fetch(`/report?id=${report.id}`, {
        headers: { 'X-User-Id': currentUserId },
      })
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as ApiErrorBody | null
        throw new Error(body?.error?.message || `HTTP ${res.status}`)
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = parseCreatedAt(report.created_at).filename
      document.body.append(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
    } catch (error) {
      setActionError(
        error instanceof Error ? error.message : 'CSV のダウンロードに失敗しました',
      )
    } finally {
      setDownloadingId(null)
    }
  }

  return (
    <PageLayout
      title="履歴"
      description="作成したレポートの一覧から CSV をダウンロードできます。"
      width="medium"
      panel={false}
    >
        {loadError && (
          <p className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {loadError}
          </p>
        )}
        {actionError && (
          <p className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {actionError}
          </p>
        )}

        {!loadError && reports.length === 0 && (
          <p className="rounded-xl border border-slate-200 bg-white px-4 py-6 text-center text-sm text-slate-500">
            レポート履歴はまだありません。
          </p>
        )}

        <ul className="space-y-3">
          {reports.map((report) => {
            const { title } = parseCreatedAt(report.created_at)
            const canDownload = report.status === 'completed'
            return (
              <li key={report.id}>
                <button
                  type="button"
                  onClick={() => void download(report)}
                  disabled={downloadingId === report.id}
                  className="flex w-full items-center justify-between rounded-xl border border-slate-200 bg-white px-5 py-4 text-left shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-wait"
                >
                  <div>
                    <p className="text-base font-bold text-slate-900">{title}</p>
                    <span
                      className={`mt-2 inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${statusClass(report.status)}`}
                    >
                      {report.status}
                    </span>
                    {report.status === 'failed' && report.reason && (
                      <p className="mt-2 text-xs text-slate-500">{report.reason}</p>
                    )}
                  </div>
                  <span
                    className={`text-xl leading-none ${canDownload ? 'text-slate-500' : 'text-slate-300'}`}
                    aria-hidden="true"
                  >
                    ↓
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
    </PageLayout>
  )
}

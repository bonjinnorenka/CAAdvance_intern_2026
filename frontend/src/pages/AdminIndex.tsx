import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import PageLayout from '../components/PageLayout.tsx'

const currentUserId = '1'

type UserSummary = {
  id: number
  name: string
  role: string
  created_at: string
}

type ApiErrorBody = {
  error?: {
    code?: string
    message?: string
  }
}

function formatDate(iso: string) {
  const match = iso.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!match) {
    return iso
  }
  return `${match[1]}-${match[2]}-${match[3]}`
}

export default function AdminIndex() {
  const [users, setUsers] = useState<UserSummary[]>([])
  const [loadError, setLoadError] = useState('')

  const loadUsers = useCallback(async () => {
    setLoadError('')
    try {
      const res = await fetch('/users', {
        headers: { 'X-User-Id': currentUserId },
      })
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as ApiErrorBody | null
        throw new Error(body?.error?.message || `HTTP ${res.status}`)
      }
      const data = (await res.json()) as UserSummary[]
      setUsers(data)
    } catch (error) {
      setUsers([])
      setLoadError(
        error instanceof Error ? error.message : 'ユーザー一覧の取得に失敗しました',
      )
    }
  }, [])

  useEffect(() => {
    void loadUsers()
  }, [loadUsers])

  return (
    <PageLayout
      title="ユーザー一覧"
      description="ユーザーを選ぶと詳細を確認・編集できます。"
      width="medium"
      panel={false}
    >
        {loadError && (
          <p className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {loadError}
          </p>
        )}

        {!loadError && users.length === 0 && (
          <p className="rounded-xl border border-slate-200 bg-white px-4 py-6 text-center text-sm text-slate-500">
            ユーザーはまだいません。
          </p>
        )}

        <ul className="space-y-3">
          {users.map((user) => (
            <li key={user.id}>
              <Link
                to={`/admin/users/${user.id}`}
                className="flex items-start justify-between rounded-xl border border-slate-200 bg-white px-5 py-4 shadow-sm transition hover:border-slate-300 hover:bg-slate-50"
              >
                <span className="text-lg font-bold text-slate-900">{user.name}</span>
                <span className="flex flex-col items-end gap-3">
                  <span className="text-lg leading-none text-slate-300" aria-hidden="true">
                    ›
                  </span>
                  <span className="text-xs text-slate-400">{formatDate(user.created_at)}</span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
    </PageLayout>
  )
}

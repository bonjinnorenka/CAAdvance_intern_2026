import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import PageLayout from '../components/PageLayout.tsx'

const currentUserId = '1'

type AdAccount = {
  id: string
  name: string
}

type UserDetail = {
  id: number
  name: string
  role: 'admin' | 'member' | string
  created_at: string
  updated_at: string
  ad_account_ids: string[]
}

type ApiErrorBody = {
  error?: {
    code?: string
    message?: string
  }
}

async function readApiError(res: Response) {
  const body = (await res.json().catch(() => null)) as ApiErrorBody | null
  return body?.error?.message || `HTTP ${res.status}`
}

export default function AccountManageScreen() {
  const { userId } = useParams()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [role, setRole] = useState('member')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [accounts, setAccounts] = useState<AdAccount[]>([])
  const [loadError, setLoadError] = useState('')
  const [actionError, setActionError] = useState('')
  const [actionOk, setActionOk] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(async () => {
    if (!userId) {
      setLoadError('ユーザー ID が不正です')
      setLoading(false)
      return
    }

    setLoading(true)
    setLoadError('')
    try {
      const headers = { 'X-User-Id': currentUserId }
      const [userRes, accountRes] = await Promise.all([
        fetch(`/user?id=${encodeURIComponent(userId)}`, { headers }),
        fetch('/ad_accounts', { headers }),
      ])
      if (!userRes.ok) {
        throw new Error(await readApiError(userRes))
      }
      if (!accountRes.ok) {
        throw new Error(await readApiError(accountRes))
      }
      const user = (await userRes.json()) as UserDetail
      const allAccounts = (await accountRes.json()) as AdAccount[]
      setName(user.name)
      setRole(user.role)
      setSelectedIds(user.ad_account_ids ?? [])
      setAccounts(allAccounts)
    } catch (error) {
      setLoadError(
        error instanceof Error ? error.message : 'ユーザー情報の取得に失敗しました',
      )
    } finally {
      setLoading(false)
    }
  }, [userId])

  useEffect(() => {
    void load()
  }, [load])

  const toggleAccount = (id: string) => {
    setSelectedIds((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    )
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!userId) {
      return
    }
    setActionError('')
    setActionOk('')

    const trimmed = name.trim()
    if (!trimmed) {
      setActionError('ユーザー名を入力してください')
      return
    }

    setSaving(true)
    try {
      const res = await fetch(`/user?id=${encodeURIComponent(userId)}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-User-Id': currentUserId,
        },
        body: JSON.stringify({
          name: trimmed,
          role,
          ad_account_ids: selectedIds,
        }),
      })
      if (!res.ok) {
        throw new Error(await readApiError(res))
      }
      setName(trimmed)
      setActionOk('ユーザー情報を更新しました')
    } catch (error) {
      setActionError(
        error instanceof Error ? error.message : 'ユーザー情報の更新に失敗しました',
      )
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async () => {
    if (!userId) {
      return
    }
    if (!window.confirm('このユーザーを削除しますか？')) {
      return
    }

    setActionError('')
    setActionOk('')
    setDeleting(true)
    try {
      const res = await fetch(`/user?id=${encodeURIComponent(userId)}`, {
        method: 'DELETE',
        headers: { 'X-User-Id': currentUserId },
      })
      if (!res.ok) {
        throw new Error(await readApiError(res))
      }
      navigate('/admin')
    } catch (error) {
      setActionError(
        error instanceof Error ? error.message : 'ユーザーの削除に失敗しました',
      )
      setDeleting(false)
    }
  }

  return (
    <PageLayout
      title="ユーザ情報"
      description="ユーザー名、ロール、アクセス権限を変更できます。"
      backTo="/admin"
      backLabel="← ユーザー一覧へ"
    >
        {loading && <p className="text-sm text-slate-500">読み込み中...</p>}
        {loadError && (
          <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {loadError}
          </p>
        )}

        {!loading && !loadError && (
          <form onSubmit={(event) => void onSubmit(event)} className="space-y-8">
            <div>
              <label htmlFor="user-name" className="mb-2 block text-sm text-slate-800">
                ユーザー名
              </label>
              <input
                id="user-name"
                type="text"
                maxLength={50}
                value={name}
                onChange={(event) => setName(event.target.value)}
                className="w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              />
            </div>

            <div>
              <label htmlFor="user-role" className="mb-2 block text-sm text-slate-800">
                ユーザーロール
              </label>
              <select
                id="user-role"
                value={role}
                onChange={(event) => setRole(event.target.value)}
                className="w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              >
                <option value="admin">admin</option>
                <option value="member">member</option>
              </select>
            </div>

            <fieldset>
              <legend className="mb-2 text-sm text-slate-800">アクセス権限</legend>
              <div className="overflow-hidden rounded border border-slate-300 bg-white">
                {accounts.length === 0 && (
                  <p className="px-4 py-3 text-sm text-slate-500">
                    広告アカウントがありません。
                  </p>
                )}
                {accounts.map((account) => (
                  <label
                    key={account.id}
                    className="flex cursor-pointer items-center gap-3 border-b border-slate-200 px-4 py-3 last:border-b-0"
                  >
                    <input
                      type="checkbox"
                      checked={selectedIds.includes(account.id)}
                      onChange={() => toggleAccount(account.id)}
                      className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-sm text-slate-800">{account.name}</span>
                  </label>
                ))}
              </div>
            </fieldset>

            {actionError && (
              <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                {actionError}
              </p>
            )}
            {actionOk && (
              <p className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
                {actionOk}
              </p>
            )}

            <div className="flex flex-wrap gap-3">
              <button
                type="submit"
                disabled={saving || deleting}
                className="rounded bg-blue-600 px-5 py-2 text-sm font-medium text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300"
              >
                {saving ? '更新中...' : '更新'}
              </button>
              <button
                type="button"
                onClick={() => void onDelete()}
                disabled={saving || deleting}
                className="rounded bg-red-600 px-5 py-2 text-sm font-medium text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:bg-red-300"
              >
                {deleting ? '削除中...' : '削除'}
              </button>
            </div>
          </form>
        )}
    </PageLayout>
  )
}

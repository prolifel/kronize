import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import type { VisibilityTarget, UserSearchResult } from '../types'

interface Props {
  targets: VisibilityTarget[]
  onChange: (targets: VisibilityTarget[]) => void
}

export default function JobVisibilityEditor({ targets, onChange }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<UserSearchResult[]>([])
  const [open, setOpen] = useState(false)
  const searchSeq = useRef(0)

  useEffect(() => {
    const seq = ++searchSeq.current
    const t = setTimeout(() => {
      api.searchUsers(query).then((users) => {
        if (seq === searchSeq.current) setResults(users)
      }).catch(() => {})
    }, 200)
    return () => clearTimeout(t)
  }, [query])

  const hasAdmin = targets.some((t) => t.type === 'role')
  const userTargets = targets.filter((t) => t.type === 'user')

  const toggleAdmin = () => {
    if (hasAdmin) {
      onChange(targets.filter((t) => t.type !== 'role'))
    } else {
      onChange([...targets, { type: 'role', role: 'admin' }])
    }
  }

  const addUser = (u: UserSearchResult) => {
    if (userTargets.some((t) => t.user_id === u.id)) return
    onChange([...targets, { type: 'user', user_id: u.id, username: u.username }])
    setQuery('')
    setOpen(false)
  }

  const removeUser = (userId: number) => {
    onChange(targets.filter((t) => !(t.type === 'user' && t.user_id === userId)))
  }

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={hasAdmin}
          onChange={toggleAdmin}
          className="h-4 w-4"
        />
        Share with admin role (all admins)
      </label>

      <div className="relative">
        <input
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
          onFocus={() => setOpen(true)}
          placeholder="Search users to share with..."
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
        />
        {open && results.length > 0 && (
          <ul className="absolute z-10 mt-1 w-full bg-white border border-gray-300 rounded-md shadow-lg max-h-48 overflow-auto">
            {results.map((u) => (
              <li key={u.id}>
                <button
                  type="button"
                  onClick={() => addUser(u)}
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
                >
                  {u.username}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {userTargets.length > 0 && (
        <ul className="flex flex-wrap gap-2">
          {userTargets.map((t) => (
            <li key={t.user_id} className="flex items-center gap-2 bg-gray-100 rounded-full px-3 py-1 text-sm">
              {t.username}
              <button
                type="button"
                onClick={() => removeUser(t.user_id!)}
                className="text-gray-500 hover:text-red-600"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

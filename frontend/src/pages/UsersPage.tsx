import { useState, useEffect, type FormEvent } from 'react'
import { api } from '../api'
import { useAuth } from '../AuthContext'
import type { User } from '../types'

export default function UsersPage() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formUsername, setFormUsername] = useState('')
  const [formPassword, setFormPassword] = useState('')
  const [formRole, setFormRole] = useState('user')
  const [formError, setFormError] = useState('')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editUsername, setEditUsername] = useState('')
  const [editRole, setEditRole] = useState('')
  const [resetPasswordId, setResetPasswordId] = useState<number | null>(null)
  const [resetPassword, setResetPassword] = useState('')

  useEffect(() => {
    loadUsers()
  }, [])

  const loadUsers = async () => {
    try {
      const data = await api.listUsers()
      setUsers(data)
    } catch {
      // handled by api.ts redirect
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    try {
      await api.createUser({ username: formUsername, password: formPassword, role: formRole })
      setShowForm(false)
      setFormUsername('')
      setFormPassword('')
      setFormRole('user')
      await loadUsers()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create user')
    }
  }

  const handleUpdate = async (id: number) => {
    try {
      await api.updateUser(id, { username: editUsername, role: editRole })
      setEditingId(null)
      await loadUsers()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to update user')
    }
  }

  const handleResetPassword = async (id: number) => {
    if (!resetPassword.trim()) return
    try {
      await api.updateUser(id, { password: resetPassword })
      setResetPasswordId(null)
      setResetPassword('')
      alert('Password reset. User must change on next login.')
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to reset password')
    }
  }

  const handleDelete = async (id: number, username: string) => {
    if (!confirm(`Delete user "${username}"?`)) return
    try {
      await api.deleteUser(id)
      await loadUsers()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete user')
    }
  }

  if (currentUser?.role !== 'admin') {
    return (
      <div className="text-center py-12 text-gray-500">Access denied. Admin only.</div>
    )
  }

  if (loading) return null

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Users</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
        >
          {showForm ? 'Cancel' : 'Add User'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-6 rounded-lg shadow mb-6 space-y-4">
          {formError && (
            <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm">{formError}</div>
          )}
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
              <input
                type="text"
                value={formUsername}
                onChange={(e) => setFormUsername(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                value={formPassword}
                onChange={(e) => setFormPassword(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                required
                minLength={6}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Role</label>
              <select
                value={formRole}
                onChange={(e) => setFormRole(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
          </div>
          <button
            type="submit"
            className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
          >
            Create
          </button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">ID</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">Username</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">Role</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">Status</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">Created</th>
              <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className="border-b border-gray-100 hover:bg-gray-50">
                {editingId === u.id ? (
                  <>
                    <td className="px-4 py-3 text-sm text-gray-500">{u.id}</td>
                    <td className="px-4 py-3">
                      <input
                        type="text"
                        value={editUsername}
                        onChange={(e) => setEditUsername(e.target.value)}
                        className="border border-gray-300 rounded px-2 py-1 text-sm w-full"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <select
                        value={editRole}
                        onChange={(e) => setEditRole(e.target.value)}
                        className="border border-gray-300 rounded px-2 py-1 text-sm"
                      >
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                      </select>
                    </td>
                    <td className="px-4 py-3 text-sm">{u.must_change_password ? 'Force change' : 'Active'}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{new Date(u.created_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3 text-right space-x-2">
                      <button
                        onClick={() => handleUpdate(u.id)}
                        className="text-blue-600 text-sm hover:underline"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingId(null)}
                        className="text-gray-500 text-sm hover:underline"
                      >
                        Cancel
                      </button>
                    </td>
                  </>
                ) : (
                  <>
                    <td className="px-4 py-3 text-sm text-gray-500">{u.id}</td>
                    <td className="px-4 py-3 text-sm font-medium">{u.username}</td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                        u.role === 'admin' ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {u.role}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {u.must_change_password ? (
                        <span className="text-amber-600 text-xs font-medium">Force change</span>
                      ) : (
                        <span className="text-green-600 text-xs font-medium">Active</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500">{new Date(u.created_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3 text-right space-x-2">
                      {resetPasswordId === u.id ? (
                        <span className="inline-flex gap-1">
                          <input
                            type="password"
                            placeholder="New password"
                            value={resetPassword}
                            onChange={(e) => setResetPassword(e.target.value)}
                            className="border border-gray-300 rounded px-2 py-1 text-sm w-32"
                          />
                          <button
                            onClick={() => handleResetPassword(u.id)}
                            className="text-blue-600 text-sm hover:underline"
                          >
                            Set
                          </button>
                          <button
                            onClick={() => { setResetPasswordId(null); setResetPassword('') }}
                            className="text-gray-500 text-sm hover:underline"
                          >
                            X
                          </button>
                        </span>
                      ) : (
                        <>
                          <button
                            onClick={() => { setEditingId(u.id); setEditUsername(u.username); setEditRole(u.role) }}
                            className="text-blue-600 text-sm hover:underline"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => setResetPasswordId(u.id)}
                            className="text-amber-600 text-sm hover:underline"
                          >
                            Reset PW
                          </button>
                          {u.id !== currentUser?.id && (
                            <button
                              onClick={() => handleDelete(u.id, u.username)}
                              className="text-red-600 text-sm hover:underline"
                            >
                              Delete
                            </button>
                          )}
                        </>
                      )}
                    </td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

import { useState, useEffect, type FormEvent } from 'react'
import { api } from '../api'
import { useAuth } from '../AuthContext'
import type { RunnerImage } from '../types'

export default function RunnersPage() {
  const { user: currentUser } = useAuth()
  const [runners, setRunners] = useState<RunnerImage[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formName, setFormName] = useState('')
  const [formImage, setFormImage] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formError, setFormError] = useState('')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [editImage, setEditImage] = useState('')
  const [editDescription, setEditDescription] = useState('')

  useEffect(() => {
    loadRunners()
  }, [])

  const loadRunners = async () => {
    try {
      const data = await api.listRunners()
      setRunners(data)
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
      await api.createRunner({ name: formName, image: formImage, description: formDescription })
      setShowForm(false)
      setFormName('')
      setFormImage('')
      setFormDescription('')
      await loadRunners()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create runner image')
    }
  }

  const handleUpdate = async (id: number) => {
    try {
      await api.updateRunner(id, { name: editName, image: editImage, description: editDescription })
      setEditingId(null)
      await loadRunners()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to update runner image')
    }
  }

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Delete runner image "${name}"?`)) return
    try {
      await api.deleteRunner(id)
      await loadRunners()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete runner image')
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
        <h1 className="text-2xl font-bold text-gray-900">Runner Images</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
        >
          {showForm ? 'Cancel' : 'Add Runner'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-6 rounded-lg shadow mb-6 space-y-4">
          {formError && (
            <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm">{formError}</div>
          )}
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="e.g. Python 3.12"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Image</label>
              <input
                type="text"
                value={formImage}
                onChange={(e) => setFormImage(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="e.g. python:3.12-slim"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <input
                type="text"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Optional description"
              />
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
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Image</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {runners.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-sm text-gray-500">
                  No runner images configured yet.
                </td>
              </tr>
            ) : (
              runners.map((r) => (
                <tr key={r.id} className="hover:bg-gray-50">
                  {editingId === r.id ? (
                    <>
                      <td className="px-4 py-3 text-sm text-gray-500">{r.id}</td>
                      <td className="px-4 py-3">
                        <input
                          type="text"
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          className="border border-gray-300 rounded px-2 py-1 text-sm w-full"
                        />
                      </td>
                      <td className="px-4 py-3">
                        <input
                          type="text"
                          value={editImage}
                          onChange={(e) => setEditImage(e.target.value)}
                          className="border border-gray-300 rounded px-2 py-1 text-sm w-full"
                        />
                      </td>
                      <td className="px-4 py-3">
                        <input
                          type="text"
                          value={editDescription}
                          onChange={(e) => setEditDescription(e.target.value)}
                          className="border border-gray-300 rounded px-2 py-1 text-sm w-full"
                        />
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500">{new Date(r.created_at).toLocaleDateString()}</td>
                      <td className="px-4 py-3 text-right space-x-2">
                        <button
                          onClick={() => handleUpdate(r.id)}
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
                      <td className="px-4 py-3 text-sm text-gray-500">{r.id}</td>
                      <td className="px-4 py-3 text-sm font-medium">{r.name}</td>
                      <td className="px-4 py-3 text-sm text-gray-600">{r.image}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{r.description || '-'}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{new Date(r.created_at).toLocaleDateString()}</td>
                      <td className="px-4 py-3 text-right space-x-2">
                        <button
                          onClick={() => {
                            setEditingId(r.id)
                            setEditName(r.name)
                            setEditImage(r.image)
                            setEditDescription(r.description)
                          }}
                          className="text-blue-600 text-sm hover:underline"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDelete(r.id, r.name)}
                          className="text-red-600 text-sm hover:underline"
                        >
                          Delete
                        </button>
                      </td>
                    </>
                  )}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

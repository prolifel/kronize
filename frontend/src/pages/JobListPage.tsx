import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'
import type { Job } from '../types'
import { useAuth } from '../AuthContext'

export default function JobListPage() {
  const navigate = useNavigate()
  const { user: currentUser } = useAuth()
  const [jobs, setJobs] = useState<Job[]>([])
  const [error, setError] = useState('')

  const loadJobs = () => {
    api.listJobs()
      .then(setJobs)
      .catch((err) => setError(err.message))
  }

  useEffect(loadJobs, [])

  const handleToggle = async (id: number) => {
    try {
      await api.toggleJob(id)
      loadJobs()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle')
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this job?')) return
    try {
      await api.deleteJob(id)
      loadJobs()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete')
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Jobs</h1>
        <Link
          to="/jobs/new"
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
        >
          New Job
        </Link>
      </div>

      {error && <div className="bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 px-4 py-2 rounded text-sm mb-4">{error}</div>}

      {jobs.length === 0 ? (
        <p className="text-gray-500 dark:text-gray-400">No jobs yet. Create your first cron job.</p>
      ) : (
        <div className="bg-white dark:bg-gray-800 border border-transparent dark:border-gray-700 shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-900/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Schedule</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Created By</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer" onClick={() => navigate(`/jobs/${job.id}`)}>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <div className="text-sm font-medium text-gray-900 dark:text-white">{job.name}</div>
                      {currentUser && job.created_by !== currentUser.id && (
                        <span className="text-xs bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300 rounded-full px-2 py-0.5">
                          Shared
                        </span>
                      )}
                    </div>
                    {job.description && (
                      <div className="text-sm text-gray-500 dark:text-gray-400">{job.description}</div>
                    )}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 font-mono">{job.cron_expression}</td>
                  <td className="px-6 py-4"><StatusBadge enabled={job.enabled} /></td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{job.created_by_username || '-'}</td>
                  <td className="px-6 py-4 text-right text-sm space-x-2" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => handleToggle(job.id)}
                      className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
                    >
                      {job.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button
                      onClick={() => handleDelete(job.id)}
                      className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

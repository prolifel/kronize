import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'
import type { Job } from '../types'

export default function JobListPage() {
  const navigate = useNavigate()
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
        <h1 className="text-2xl font-bold text-gray-900">Jobs</h1>
        <Link
          to="/jobs/new"
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
        >
          New Job
        </Link>
      </div>

      {error && <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm mb-4">{error}</div>}

      {jobs.length === 0 ? (
        <p className="text-gray-500">No jobs yet. Create your first cron job.</p>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Schedule</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created By</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-gray-50 cursor-pointer" onClick={() => navigate(`/jobs/${job.id}`)}>
                  <td className="px-6 py-4">
                    <div className="text-sm font-medium text-gray-900">{job.name}</div>
                    {job.description && (
                      <div className="text-sm text-gray-500">{job.description}</div>
                    )}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 font-mono">{job.cron_expression}</td>
                  <td className="px-6 py-4"><StatusBadge enabled={job.enabled} /></td>
                  <td className="px-6 py-4 text-sm text-gray-500">{job.created_by_username || '-'}</td>
                  <td className="px-6 py-4 text-right text-sm space-x-2" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => handleToggle(job.id)}
                      className="text-blue-600 hover:text-blue-800"
                    >
                      {job.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button
                      onClick={() => handleDelete(job.id)}
                      className="text-red-600 hover:text-red-800"
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

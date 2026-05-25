import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'
import ExecutionLog from '../components/ExecutionLog'
import type { Job, Execution } from '../types'

export default function JobDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [job, setJob] = useState<Job | null>(null)
  const [executions, setExecutions] = useState<Execution[]>([])
  const [expandedExec, setExpandedExec] = useState<number | null>(null)
  const [expandedDetail, setExpandedDetail] = useState<Execution | null>(null)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = () => {
    if (!id) return
    const jobId = Number(id)
    api.getJob(jobId).then(setJob).catch((err) => setError(err.message))
    api.listExecutions(jobId).then(setExecutions).catch((err) => setError(err.message))
  }

  useEffect(load, [id])

  const handleRun = async () => {
    if (!id) return
    try {
      await api.runJob(Number(id))
      setSuccess('Job triggered successfully')
      setTimeout(() => setSuccess(''), 3000)
      setTimeout(load, 2000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to trigger')
    }
  }

  const handleToggle = async () => {
    if (!id) return
    try {
      await api.toggleJob(Number(id))
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle')
    }
  }

  const toggleExecution = async (execId: number) => {
    if (expandedExec === execId) {
      setExpandedExec(null)
      setExpandedDetail(null)
      return
    }
    setExpandedExec(execId)
    try {
      const detail = await api.getExecution(execId)
      setExpandedDetail(detail)
    } catch {
      setExpandedDetail(null)
    }
  }

  if (!job) return <p className="text-gray-500">Loading...</p>

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{job.name}</h1>
          {job.description && <p className="text-gray-500 mt-1">{job.description}</p>}
        </div>
        <div className="flex items-center gap-3">
          <StatusBadge enabled={job.enabled} />
          <Link to={`/jobs/${job.id}/edit`} className="text-sm text-blue-600 hover:text-blue-800">Edit</Link>
          <button onClick={handleToggle} className="text-sm text-gray-600 hover:text-gray-800">
            {job.enabled ? 'Disable' : 'Enable'}
          </button>
          <button
            onClick={handleRun}
            className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
          >
            Run Now
          </button>
        </div>
      </div>

      <div className="bg-white shadow rounded-lg p-6 mb-6">
        <dl className="grid grid-cols-2 gap-4">
          <div>
            <dt className="text-sm text-gray-500">Schedule</dt>
            <dd className="text-sm font-mono">{job.cron_expression}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500">Image</dt>
            <dd className="text-sm font-mono">{job.image}</dd>
          </div>
        </dl>
      </div>

      {error && <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm mb-4">{error}</div>}
      {success && <div className="bg-green-50 text-green-700 px-4 py-2 rounded text-sm mb-4">{success}</div>}

      <h2 className="text-lg font-semibold text-gray-900 mb-4">Execution History</h2>

      {executions.length === 0 ? (
        <p className="text-gray-500">No executions yet.</p>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Duration</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {executions.map((exec) => (
                <>
                  <tr
                    key={exec.id}
                    onClick={() => toggleExecution(exec.id)}
                    className="hover:bg-gray-50 cursor-pointer"
                  >
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          exec.status === 'success' ? 'bg-green-100 text-green-800' :
                          exec.status === 'failed' ? 'bg-red-100 text-red-800' :
                          'bg-blue-100 text-blue-800'
                        }`}
                      >
                        {exec.status === 'running' && <span className="animate-spin mr-1">&#9696;</span>}
                        {exec.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {exec.duration_ms != null ? `${exec.duration_ms}ms` : '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {new Date(exec.started_at).toLocaleString()}
                    </td>
                  </tr>
                  {expandedExec === exec.id && expandedDetail && (
                    <tr key={`${exec.id}-detail`}>
                      <td colSpan={3} className="px-6 py-4 bg-gray-50">
                        <ExecutionLog stdout={expandedDetail.stdout} stderr={expandedDetail.stderr} />
                        {expandedDetail.exit_code != null && (
                          <p className="text-sm text-gray-500 mt-2">Exit code: {expandedDetail.exit_code}</p>
                        )}
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

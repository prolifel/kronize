import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'
import ExecutionLog from '../components/ExecutionLog'
import type { Job, Execution } from '../types'
import { useAuth } from '../AuthContext'

interface LiveLog {
  id: number
  stdout: string
  stderr: string
  status: 'running' | 'success' | 'failed'
  exit_code: number | null
  duration_ms: number | null
}

export default function JobDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { user: currentUser } = useAuth()
  const [job, setJob] = useState<Job | null>(null)
  const [executions, setExecutions] = useState<Execution[]>([])
  const [expandedExec, setExpandedExec] = useState<number | null>(null)
  const [expandedDetail, setExpandedDetail] = useState<Execution | null>(null)
  const [liveLog, setLiveLog] = useState<LiveLog | null>(null)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const eventSourceRef = useRef<EventSource | null>(null)

  const load = () => {
    if (!id) return
    const jobId = Number(id)
    api.getJob(jobId).then(setJob).catch((err) => setError(err.message))
    api.listExecutions(jobId).then(setExecutions).catch((err) => setError(err.message))
  }

  useEffect(load, [id])

  const closeStream = () => {
    eventSourceRef.current?.close()
    eventSourceRef.current = null
  }

  const openStream = (execId: number) => {
    closeStream()
    setLiveLog({
      id: execId,
      stdout: '',
      stderr: '',
      status: 'running',
      exit_code: null,
      duration_ms: null,
    })

    const es = new EventSource(`/api/executions/${execId}/stream`)
    eventSourceRef.current = es

    es.addEventListener('init', (event) => {
      const data = JSON.parse((event as MessageEvent).data)
      setLiveLog((prev) => prev?.id === execId ? {
        ...prev,
        stdout: data.stdout,
        stderr: data.stderr,
        status: data.status,
        exit_code: data.exit_code,
        duration_ms: data.duration_ms,
      } : prev)
      if (data.status !== 'running') {
        closeStream()
        load()
      }
    })

    es.addEventListener('stdout', (event) => {
      const data = JSON.parse((event as MessageEvent).data)
      setLiveLog((prev) => prev?.id === execId ? { ...prev, stdout: prev.stdout + data } : prev)
    })

    es.addEventListener('stderr', (event) => {
      const data = JSON.parse((event as MessageEvent).data)
      setLiveLog((prev) => prev?.id === execId ? { ...prev, stderr: prev.stderr + data } : prev)
    })

    es.addEventListener('status', (event) => {
      const data = JSON.parse((event as MessageEvent).data)
      setLiveLog((prev) => prev?.id === execId ? {
        ...prev,
        status: data.status,
        exit_code: data.exit_code,
        duration_ms: data.duration_ms,
      } : prev)
      closeStream()
      load()
    })

    es.onerror = () => {
      closeStream()
      load()
    }
  }

  useEffect(() => () => closeStream(), [])

  const handleRun = async () => {
    if (!id) return
    try {
      const result = await api.runJob(Number(id))
      setSuccess('Job triggered successfully')
      setTimeout(() => setSuccess(''), 3000)
      setExpandedExec(result.execution_id)
      setExpandedDetail(null)
      openStream(result.execution_id)
      load()
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

  const toggleExecution = async (exec: Execution) => {
    if (expandedExec === exec.id) {
      setExpandedExec(null)
      setExpandedDetail(null)
      closeStream()
      return
    }
    setExpandedExec(exec.id)
    if (exec.status === 'running' && exec.source === 'manual') {
      setExpandedDetail(null)
      openStream(exec.id)
      return
    }
    closeStream()
    setLiveLog(null)
    try {
      const detail = await api.getExecution(exec.id)
      setExpandedDetail(detail)
    } catch {
      setExpandedDetail(null)
    }
  }

  if (!job) return <p className="text-gray-500 dark:text-gray-400">Loading...</p>

  const canManage = job.created_by === currentUser?.id || currentUser?.role === 'admin'

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{job.name}</h1>
          {job.description && <p className="text-gray-500 dark:text-gray-400 mt-1">{job.description}</p>}
          {job.visibility?.length > 0 && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Shared with:{' '}
              {(job.visibility ?? []).map((t) =>
                t.type === 'role' ? 'all admins' : t.username
              ).join(', ')}
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <StatusBadge enabled={job.enabled} />
          {canManage && (
            <>
              <Link to={`/jobs/${job.id}/edit`} className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300">Edit</Link>
              <button onClick={handleToggle} className="text-sm text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-white">
                {job.enabled ? 'Disable' : 'Enable'}
              </button>
              <button
                onClick={handleRun}
                className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
              >
                Run Now
              </button>
            </>
          )}
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 border border-transparent dark:border-gray-700 shadow rounded-lg p-6 mb-6">
        <dl className="grid grid-cols-2 gap-4">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Schedule</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-gray-100">{job.cron_expression}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Image</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-gray-100">{job.image}</dd>
          </div>
        </dl>
      </div>

      {error && <div className="bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 px-4 py-2 rounded text-sm mb-4">{error}</div>}
      {success && <div className="bg-green-50 dark:bg-green-900/30 text-green-700 dark:text-green-300 px-4 py-2 rounded text-sm mb-4">{success}</div>}

      <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Execution History</h2>

      {executions.length === 0 ? (
        <p className="text-gray-500 dark:text-gray-400">No executions yet.</p>
      ) : (
        <div className="bg-white dark:bg-gray-800 border border-transparent dark:border-gray-700 shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-900/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Duration</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {executions.map((exec) => (
                <>
                  <tr
                    key={exec.id}
                    onClick={() => toggleExecution(exec)}
                    className="hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer"
                  >
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          exec.status === 'success' ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300' :
                          exec.status === 'failed' ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300' :
                          'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300'
                        }`}
                      >
                        {exec.status === 'running' && <span className="animate-spin mr-1">&#9696;</span>}
                        {exec.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                      {exec.duration_ms != null ? `${exec.duration_ms}ms` : '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                      {new Date(exec.started_at).toLocaleString()}
                    </td>
                  </tr>
                  {expandedExec === exec.id && (liveLog?.id === exec.id ? (
                    <tr key={`${exec.id}-live`}>
                      <td colSpan={3} className="px-6 py-4 bg-gray-50 dark:bg-gray-900/60">
                        <ExecutionLog stdout={liveLog.stdout} stderr={liveLog.stderr} />
                        {liveLog.status === 'running' ? (
                          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">Running...</p>
                        ) : (
                          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                            Exit code: {liveLog.exit_code ?? '-'} | {liveLog.duration_ms ?? '-'}ms
                          </p>
                        )}
                      </td>
                    </tr>
                  ) : expandedDetail && (
                    <tr key={`${exec.id}-detail`}>
                      <td colSpan={3} className="px-6 py-4 bg-gray-50 dark:bg-gray-900/60">
                        <ExecutionLog stdout={expandedDetail.stdout} stderr={expandedDetail.stderr} />
                        {expandedDetail.exit_code != null && (
                          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">Exit code: {expandedDetail.exit_code}</p>
                        )}
                      </td>
                    </tr>
                  ))}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

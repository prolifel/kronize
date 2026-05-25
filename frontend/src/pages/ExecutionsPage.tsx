import { useState, useEffect } from 'react'
import { api } from '../api'
import type { Execution } from '../types'
import ExecutionLog from '../components/ExecutionLog'

export default function ExecutionsPage() {
  const [execs, setExecs] = useState<Array<Execution & { job_name?: string }>>([])
  const [error, setError] = useState('')

  useEffect(() => {
    api.listJobs()
      .then((jobs) => {
        return Promise.all(
          jobs.map(async (job) => {
            const execs = await api.listExecutions(job.id, 5)
            return execs.map((e) => ({ ...e, job_name: job.name, job_id: job.id }))
          })
        )
      })
      .then((nested) => setExecs(nested.flat().sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()).slice(0, 50)))
      .catch((err) => setError(err.message))
  }, [])

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Recent Executions</h1>
      {error && <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm mb-4">{error}</div>}
      {execs.length === 0 ? (
        <p className="text-gray-500">No executions yet.</p>
      ) : (
        <div className="space-y-4">
          {execs.map((exec) => (
            <div key={exec.id} className="bg-white shadow rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    exec.status === 'success' ? 'bg-green-100 text-green-800' :
                    exec.status === 'failed' ? 'bg-red-100 text-red-800' :
                    'bg-blue-100 text-blue-800'
                  }`}>{exec.status}</span>
                  <span className="text-sm font-medium text-gray-900">{exec.job_name}</span>
                  <span className="text-sm text-gray-500">#{exec.id}</span>
                </div>
                <div className="text-sm text-gray-500">
                  {exec.duration_ms != null ? `${exec.duration_ms}ms` : '-'}
                  {' | '}
                  {new Date(exec.started_at).toLocaleString()}
                </div>
              </div>
              <ExecutionLog stdout={exec.stdout} stderr={exec.stderr} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

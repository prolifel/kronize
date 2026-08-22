import { useState, useEffect, type FormEvent } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../api'
import CodeEditor from '../components/CodeEditor'
import CronHelper from '../components/CronHelper'
import EnvVarEditor from '../components/EnvVarEditor'
import HostMappingsEditor from '../components/HostMappingsEditor'
import JobVisibilityEditor from '../components/JobVisibilityEditor'
import type { JobFormData, RunnerImage } from '../types'

export default function JobFormPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = Boolean(id)

  const [form, setForm] = useState<JobFormData>({
    name: '',
    description: '',
    cron_expression: '',
    python_code: 'print("Hello from Kronize!")',
    env_vars: '{}',
    host_mappings: '[]',
    log_level: 'info',
    image_id: 0,
    visibility: [],
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [runners, setRunners] = useState<RunnerImage[]>([])

  useEffect(() => {
    api.listRunners().then(setRunners).catch(() => {})
  }, [])

  useEffect(() => {
    if (id) {
      api.getJob(Number(id)).then((job) => {
        setForm({
          name: job.name,
          description: job.description,
          cron_expression: job.cron_expression,
          python_code: job.python_code || '',
          env_vars: job.env_vars || '{}',
          host_mappings: job.host_mappings || '[]',
          log_level: job.log_level,
          image_id: job.image_id,
          visibility: job.visibility || [],
        })
      }).catch((err) => setError(err.message))
    }
  }, [id])

  // pre-select first runner when list loads and no image_id is set yet
  useEffect(() => {
    if (runners.length > 0 && form.image_id === 0) {
      setForm((prev) => ({ ...prev, image_id: runners[0].id }))
    }
  }, [runners, form.image_id])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (isEdit) {
        await api.updateJob(Number(id), form)
        await api.updateJobVisibility(Number(id), form.visibility)
      } else {
        await api.createJob(form)
      }
      navigate('/jobs')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save job')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">
        {isEdit ? 'Edit Job' : 'New Cron Job'}
      </h1>

      {error && <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm mb-4">{error}</div>}

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Log Level</label>
            <select
              value={form.log_level}
              onChange={(e) => setForm({ ...form, log_level: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="debug">DEBUG</option>
              <option value="info">INFO</option>
              <option value="warn">WARN</option>
              <option value="error">ERROR</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Image</label>
            <select
              value={form.image_id}
              onChange={(e) => setForm({ ...form, image_id: Number(e.target.value) })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              {runners.length === 0 && <option value={0}>No runners configured</option>}
              {runners.map((r) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <input
            type="text"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          />
        </div>

        <CronHelper
          value={form.cron_expression}
          onChange={(v) => setForm({ ...form, cron_expression: v })}
        />

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Python Code</label>
          <CodeEditor
            value={form.python_code}
            onChange={(v) => setForm({ ...form, python_code: v })}
          />
        </div>

        <EnvVarEditor
          value={form.env_vars}
          onChange={(v) => setForm({ ...form, env_vars: v })}
        />

        <HostMappingsEditor
          value={form.host_mappings}
          onChange={(v) => setForm({ ...form, host_mappings: v })}
        />

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Sharing</label>
          <JobVisibilityEditor
            targets={form.visibility}
            onChange={(v) => setForm((prev) => ({ ...prev, visibility: v }))}
          />
        </div>

        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={() => navigate('/jobs')}
            className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-sm text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Saving...' : isEdit ? 'Update Job' : 'Create Job'}
          </button>
        </div>
      </form>
    </div>
  )
}

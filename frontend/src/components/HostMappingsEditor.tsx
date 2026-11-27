interface HostMapping {
  host: string
  container: string
  read_only: boolean
}

interface HostMappingsEditorProps {
  value: string
  onChange: (json: string) => void
}

export default function HostMappingsEditor({ value, onChange }: HostMappingsEditorProps) {
  let mappings: HostMapping[] = []
  try {
    const parsed = JSON.parse(value || '[]')
    mappings = Array.isArray(parsed) ? parsed : []
  } catch {
    mappings = []
  }

  const update = (next: HostMapping[]) => onChange(JSON.stringify(next))

  const addMapping = () =>
    update([...mappings, { host: '', container: '', read_only: true }])

  const removeMapping = (i: number) => update(mappings.filter((_, idx) => idx !== i))

  const setField = (i: number, field: keyof HostMapping, v: string | boolean) => {
    update(mappings.map((m, idx) => (idx === i ? { ...m, [field]: v } : m)))
  }

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Host Mappings</label>
      {mappings.map((m, i) => (
        <div key={i} className="flex gap-2 items-center">
          <input
            type="text"
            placeholder="/host/path"
            value={m.host}
            onChange={(e) => setField(i, 'host', e.target.value)}
            className="flex-1 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded px-2 py-1.5 text-sm font-mono"
          />
          <input
            type="text"
            placeholder="/container/path"
            value={m.container}
            onChange={(e) => setField(i, 'container', e.target.value)}
            className="flex-1 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded px-2 py-1.5 text-sm font-mono"
          />
          <label className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-300">
            <input
              type="checkbox"
              checked={m.read_only}
              onChange={(e) => setField(i, 'read_only', e.target.checked)}
            />
            ro
          </label>
          <button type="button" onClick={() => removeMapping(i)} className="text-red-500 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 text-sm">
            X
          </button>
        </div>
      ))}
      <button type="button" onClick={addMapping} className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300">
        + Add Mapping
      </button>
      <p className="text-xs text-gray-500 dark:text-gray-400">Mounts host paths into the container (read-only by default).</p>
    </div>
  )
}

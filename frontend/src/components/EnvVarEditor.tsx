import { useState } from 'react'

interface EnvVar {
  key: string
  value: string
}

interface EnvVarEditorProps {
  value: string
  onChange: (json: string) => void
}

export default function EnvVarEditor({ value, onChange }: EnvVarEditorProps) {
  const [showValues, setShowValues] = useState(false)

  let pairs: EnvVar[] = []
  try {
    const obj = JSON.parse(value || '{}')
    pairs = Object.entries(obj).map(([k, v]) => ({ key: k, value: String(v) }))
  } catch {
    pairs = []
  }

  const update = (newPairs: EnvVar[]) => {
    const obj: Record<string, string> = {}
    newPairs.forEach((p) => {
      if (p.key) obj[p.key] = p.value
    })
    onChange(JSON.stringify(obj))
  }

  const addVar = () => {
    const obj = JSON.parse(value || '{}')
    const key = `VAR${Object.keys(obj).length + 1}`
    obj[key] = ''
    onChange(JSON.stringify(obj))
  }

  const removeVar = (i: number) => update(pairs.filter((_, idx) => idx !== i))

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="block text-sm font-medium text-gray-700">Environment Variables</label>
        <button type="button" onClick={() => setShowValues(!showValues)} className="text-xs text-gray-500 hover:text-gray-700">
          {showValues ? 'Hide' : 'Show'} Values
        </button>
      </div>
      {pairs.map((pair, i) => (
        <div key={i} className="flex gap-2 items-center">
          <input
            type="text"
            placeholder="KEY"
            value={pair.key}
            onChange={(e) => {
              const next = [...pairs]
              next[i] = { ...next[i], key: e.target.value }
              update(next)
            }}
            className="flex-1 border border-gray-300 rounded px-2 py-1.5 text-sm font-mono"
          />
          <input
            type={showValues ? 'text' : 'password'}
            placeholder="VALUE"
            value={pair.value}
            onChange={(e) => {
              const next = [...pairs]
              next[i] = { ...next[i], value: e.target.value }
              update(next)
            }}
            className="flex-1 border border-gray-300 rounded px-2 py-1.5 text-sm font-mono"
          />
          <button type="button" onClick={() => removeVar(i)} className="text-red-500 hover:text-red-700 text-sm">X</button>
        </div>
      ))}
      <button type="button" onClick={addVar} className="text-sm text-blue-600 hover:text-blue-800">+ Add Variable</button>
    </div>
  )
}

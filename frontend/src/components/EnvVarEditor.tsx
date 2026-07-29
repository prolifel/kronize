import { useState } from 'react'

interface EnvVar {
  key: string
  value: string
}

interface EnvVarEditorProps {
  value: string
  onChange: (json: string) => void
}

function parseRawText(text: string): { pairs: Record<string, string>; warnings: string[] } {
  const pairs: Record<string, string> = {}
  const warnings: string[] = []
  const lines = text.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line || line.startsWith('#')) continue
    const eqIdx = line.indexOf('=')
    if (eqIdx === -1) {
      warnings.push(`line ${i + 1} has no '='`)
      continue
    }
    const key = line.slice(0, eqIdx).trim()
    const value = line.slice(eqIdx + 1)
    if (!key) {
      warnings.push(`line ${i + 1}: empty key`)
      continue
    }
    pairs[key] = value
  }
  return { pairs, warnings }
}

export default function EnvVarEditor({ value, onChange }: EnvVarEditorProps) {
  const [showValues, setShowValues] = useState(false)
  const [mode, setMode] = useState<'simple' | 'advanced'>('simple')
  const [rawText, setRawText] = useState('')
  const [warnings, setWarnings] = useState<string[]>([])

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

  const switchToAdvanced = () => {
    const text = pairs.map(p => `${p.key}=${p.value}`).join('\n')
    setRawText(text)
    setWarnings([])
    setMode('advanced')
  }

  const switchToSimple = () => {
    const result = parseRawText(rawText)
    setWarnings(result.warnings)
    onChange(JSON.stringify(result.pairs))
    setMode('simple')
  }

  const handleRawTextChange = (text: string) => {
    setRawText(text)
    const result = parseRawText(text)
    onChange(JSON.stringify(result.pairs))
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="block text-sm font-medium text-gray-700">Environment Variables</label>
        <button type="button" onClick={() => setShowValues(!showValues)} className="text-xs text-gray-500 hover:text-gray-700">
          {showValues ? 'Hide' : 'Show'} Values
        </button>
      </div>

      {/* Mode tabs */}
      <div className="flex gap-0 border-b border-gray-300">
        <button
          type="button"
          onClick={() => mode === 'advanced' ? switchToSimple() : undefined}
          className={`px-3 py-1.5 text-sm font-medium border-b-2 -mb-px ${
            mode === 'simple' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          Simple
        </button>
        <button
          type="button"
          onClick={() => mode === 'simple' ? switchToAdvanced() : undefined}
          className={`px-3 py-1.5 text-sm font-medium border-b-2 -mb-px ${
            mode === 'advanced' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          Advanced
        </button>
      </div>

      {/* Warning banner */}
      {warnings.length > 0 && mode === 'simple' && (
        <div className="bg-yellow-50 border border-yellow-300 text-yellow-800 px-3 py-2 rounded text-sm flex items-start gap-2">
          <span>⚠</span>
          <span>
            {warnings.length} line{warnings.length > 1 ? 's' : ''} skipped:{' '}
            {warnings.join('; ')}
          </span>
          <button
            type="button"
            onClick={() => setWarnings([])}
            className="ml-auto text-yellow-600 hover:text-yellow-800 text-xs font-medium"
          >
            Dismiss
          </button>
        </div>
      )}

      {mode === 'simple' ? (
        <>
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
        </>
      ) : (
        <textarea
          value={rawText}
          onChange={(e) => handleRawTextChange(e.target.value)}
          placeholder={"KEY=VALUE\n# comments are ignored"}
          rows={10}
          className="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      )}
    </div>
  )
}

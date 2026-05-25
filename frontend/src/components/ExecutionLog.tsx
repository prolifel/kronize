interface ExecutionLogProps {
  stdout: string
  stderr: string
}

export default function ExecutionLog({ stdout, stderr }: ExecutionLogProps) {
  if (!stdout && !stderr) return <p className="text-sm text-gray-500 italic">No output</p>

  return (
    <div className="bg-gray-900 rounded-md p-4 text-sm font-mono space-y-2 max-h-64 overflow-y-auto">
      {stdout && <pre className="text-green-400 whitespace-pre-wrap">{stdout}</pre>}
      {stderr && <pre className="text-red-400 whitespace-pre-wrap">{stderr}</pre>}
    </div>
  )
}

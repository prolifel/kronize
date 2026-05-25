import { useRef } from 'react'

interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
  height?: string
}

export default function CodeEditor({ value, onChange, height = '300px' }: CodeEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  return (
    <div className="border border-gray-300 rounded-md overflow-hidden">
      <div className="bg-gray-800 text-gray-200 px-4 py-2 text-xs font-mono flex items-center">
        <span className="w-3 h-3 rounded-full bg-red-500 mr-2" />
        <span className="w-3 h-3 rounded-full bg-yellow-500 mr-2" />
        <span className="w-3 h-3 rounded-full bg-green-500 mr-2" />
        <span className="ml-2">main.py</span>
      </div>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full p-4 font-mono text-sm bg-gray-900 text-green-400 resize-vertical focus:outline-none"
        style={{ height, tabSize: 4 }}
        spellCheck={false}
      />
    </div>
  )
}

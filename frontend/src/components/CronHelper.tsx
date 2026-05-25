const descriptions: Record<string, string> = {
  '* * * * *': 'Every minute',
  '*/5 * * * *': 'Every 5 minutes',
  '*/15 * * * *': 'Every 15 minutes',
  '*/30 * * * *': 'Every 30 minutes',
  '0 * * * *': 'Every hour',
  '0 */6 * * *': 'Every 6 hours',
  '0 */12 * * *': 'Every 12 hours',
  '0 0 * * *': 'Daily at midnight',
  '0 3 * * *': 'Daily at 3:00 AM',
  '0 0 * * 0': 'Weekly on Sunday',
  '0 0 * * 1': 'Weekly on Monday',
  '0 0 1 * *': 'Monthly on the 1st',
}

export default function CronHelper({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const presetNames = Object.keys(descriptions)

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">Cron Expression</label>
      <div className="flex gap-2">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="*/5 * * * *"
          className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:ring-2 focus:ring-blue-500"
        />
        <select
          value={descriptions[value] ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          className="border border-gray-300 rounded-md px-3 py-2 text-sm"
        >
          <option value="">Presets</option>
          {presetNames.map((cron) => (
            <option key={cron} value={cron}>
              {descriptions[cron]}
            </option>
          ))}
        </select>
      </div>
      {value && (
        <p className="mt-1 text-sm text-gray-500">
          {descriptions[value] || 'Custom schedule'}
        </p>
      )}
    </div>
  )
}

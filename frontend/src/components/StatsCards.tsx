import type { Stats } from '../types'

const cards = [
  { key: 'total_jobs' as const, label: 'Total Jobs', color: 'bg-blue-500' },
  { key: 'enabled_jobs' as const, label: 'Active Jobs', color: 'bg-green-500' },
  { key: 'success_rate_24h' as const, label: 'Success Rate (24h)', color: 'bg-indigo-500', suffix: '%' as const },
  { key: 'failures_today' as const, label: 'Failures Today', color: 'bg-red-500' },
]

export default function StatsCards({ stats }: { stats: Stats }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
      {cards.map((card) => (
        <div key={card.key} className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <div className={`w-3 h-3 rounded-full ${card.color} mr-2`} />
            <span className="text-sm text-gray-500">{card.label}</span>
          </div>
          <p className="text-3xl font-bold mt-2">
            {card.suffix === '%' ? stats[card.key].toFixed(1) : stats[card.key]}
            {card.suffix || ''}
          </p>
        </div>
      ))}
    </div>
  )
}

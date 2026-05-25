import { useState, useEffect } from 'react'
import { api } from '../api'
import StatsCards from '../components/StatsCards'
import type { Stats } from '../types'

export default function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api.getStats()
      .then(setStats)
      .catch((err) => setError(err.message))
  }, [])

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>
      {error && <div className="bg-red-50 text-red-700 px-4 py-2 rounded text-sm mb-4">{error}</div>}
      {stats ? <StatsCards stats={stats} /> : <p className="text-gray-500">Loading...</p>}
    </div>
  )
}

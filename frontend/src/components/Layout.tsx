import { Link, useLocation, useNavigate, Navigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'

const publicPaths = ['/login']

export default function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout, isAuthenticated, loading } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()

  if (loading) return null

  if (!isAuthenticated) {
    if (publicPaths.includes(location.pathname)) return <>{children}</>
    return <Navigate to="/login" replace />
  }

  const navLinks = [
    { to: '/dashboard', label: 'Dashboard' },
    { to: '/jobs', label: 'Jobs' },
    { to: '/executions', label: 'Executions' },
  ]

  if (user?.role === 'admin') {
    navLinks.push({ to: '/users', label: 'Users' }, { to: '/runners', label: 'Runner' })
  }

  const isActive = (path: string) => location.pathname.startsWith(path)

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center space-x-8">
              <Link to="/dashboard" className="text-xl font-bold text-gray-900">
                Kronize
              </Link>
              {navLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className={`inline-flex items-center px-1 pt-1 text-sm font-medium border-b-2 ${
                    isActive(link.to)
                      ? 'border-blue-500 text-gray-900'
                      : 'border-transparent text-gray-500 hover:text-gray-700'
                  }`}
                >
                  {link.label}
                </Link>
              ))}
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-500">{user?.username}</span>
              <button
                onClick={() => logout().then(() => navigate('/login'))}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  )
}

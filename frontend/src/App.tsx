import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './AuthContext'
import Layout from './components/Layout'
import LoginPage from './pages/LoginPage'
import ChangePasswordPage from './pages/ChangePasswordPage'
import DashboardPage from './pages/DashboardPage'
import JobListPage from './pages/JobListPage'
import JobFormPage from './pages/JobFormPage'
import JobDetailPage from './pages/JobDetailPage'
import ExecutionsPage from './pages/ExecutionsPage'
import UsersPage from './pages/UsersPage'
import RunnersPage from './pages/RunnersPage'

export default function App() {
  return (
    <AuthProvider>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/change-password" element={<ChangePasswordPage />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/jobs" element={<JobListPage />} />
          <Route path="/jobs/new" element={<JobFormPage />} />
          <Route path="/jobs/:id" element={<JobDetailPage />} />
          <Route path="/jobs/:id/edit" element={<JobFormPage />} />
          <Route path="/executions" element={<ExecutionsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/runners" element={<RunnersPage />} />
        </Routes>
      </Layout>
    </AuthProvider>
  )
}

import type { Job, JobFormData, Execution, Stats, Setting, LoginResponse, User } from './types'

const BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/')) {
      window.location.href = '/login'
      throw new Error('redirecting to login')
    }
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export const api = {
  login: (data: { username: string; password: string }) =>
    request<LoginResponse>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),

  logout: () =>
    request<{ message: string }>('/auth/logout', { method: 'POST' }),

  getCurrentUser: () =>
    request<User>('/auth/me'),

  changePassword: (data: { current_password: string; new_password: string }) =>
    request<{ message: string }>('/auth/change-password', { method: 'POST', body: JSON.stringify(data) }),

  listUsers: () =>
    request<User[]>('/users'),

  createUser: (data: { username: string; password: string; role: string }) =>
    request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),

  updateUser: (id: number, data: { username?: string; password?: string; role?: string }) =>
    request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteUser: (id: number) =>
    request<{ message: string }>(`/users/${id}`, { method: 'DELETE' }),

  listJobs: (enabledOnly?: boolean) =>
    request<Job[]>(`/jobs${enabledOnly ? '?enabled=true' : ''}`),

  getJob: (id: number) =>
    request<Job>(`/jobs/${id}`),

  createJob: (data: JobFormData) =>
    request<Job>('/jobs', { method: 'POST', body: JSON.stringify(data) }),

  updateJob: (id: number, data: Partial<JobFormData & { enabled: boolean }>) =>
    request<Job>(`/jobs/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteJob: (id: number) =>
    request<{ message: string }>(`/jobs/${id}`, { method: 'DELETE' }),

  runJob: (id: number) =>
    request<{ message: string }>(`/jobs/${id}/run`, { method: 'POST' }),

  toggleJob: (id: number) =>
    request<Job>(`/jobs/${id}/toggle`, { method: 'PUT' }),

  listExecutions: (jobId: number, limit = 20, offset = 0) =>
    request<Execution[]>(`/jobs/${jobId}/executions?limit=${limit}&offset=${offset}`),

  getExecution: (id: number) =>
    request<Execution>(`/executions/${id}`),

  getStats: () =>
    request<Stats>('/stats'),

  getSettings: () =>
    request<Setting[]>('/settings'),

  updateSettings: (settings: Setting[]) =>
    request<Setting[]>('/settings', { method: 'PUT', body: JSON.stringify(settings) }),
}

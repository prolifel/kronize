export interface User {
  id: number
  username: string
  role: string
  must_change_password: boolean
  created_at: string
}

export interface Job {
  id: number
  name: string
  description: string
  cron_expression: string
  python_code: string
  image_id: number
  image: string
  env_vars: string
  host_mappings: string
  log_level: string
  enabled: boolean
  created_by: number
  created_by_username: string
  created_at: string
  updated_at: string
}

export interface JobFormData {
  name: string
  description: string
  cron_expression: string
  python_code: string
  env_vars: string
  host_mappings: string
  log_level: string
  image_id: number
}

export interface Execution {
  id: number
  job_id: number
  status: 'running' | 'success' | 'failed'
  stdout: string
  stderr: string
  exit_code: number | null
  duration_ms: number | null
  started_at: string
  finished_at: string | null
}

export interface Stats {
  total_jobs: number
  enabled_jobs: number
  success_rate_24h: number
  failures_today: number
}

export interface Setting {
  key: string
  value: string
}

export interface RunnerImage {
  id: number
  name: string
  image: string
  description: string
  created_at: string
}

export interface LoginResponse {
  token: string
  user: User
}

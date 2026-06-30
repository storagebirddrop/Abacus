import { getToken } from './token'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData

  // Compose headers so the bearer token is always attached when configured,
  // even for FormData uploads (which must not carry a JSON content-type).
  const headers: Record<string, string> = {}
  if (!isFormData) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  Object.assign(headers, init?.headers as Record<string, string> | undefined)

  const res = await fetch(`/api/v1${path}`, { ...init, headers })
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body?.error) msg = body.error
    } catch {}
    throw new ApiError(res.status, msg)
  }
  return res.json() as Promise<T>
}

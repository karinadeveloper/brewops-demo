import { useAuthStore } from '../stores/auth'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string

export interface ApiError {
  status: number
  message: string
}

// Requests to these paths never go through the 401 refresh-and-retry flow:
// a wrong-password login must surface as "invalid credentials" (not trigger
// a refresh attempt), and a failing refresh call must not try to refresh
// itself — that would recurse forever.
const AUTH_PATHS = ['/auth/login', '/auth/refresh', '/auth/register']

interface RefreshResponse {
  access_token: string
}

// Module-level (not per-call) so every concurrent request sees the same
// in-flight refresh instead of each one starting its own — see
// requestWithRefresh's usage below. Cleared once the refresh settles so the
// next 401, later, starts a fresh one.
let refreshPromise: Promise<string> | null = null

async function refreshAccessToken(): Promise<string> {
  const authStore = useAuthStore()
  const refreshToken = authStore.refreshToken
  if (!refreshToken) {
    throw new Error('no refresh token available')
  }

  const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })

  if (!response.ok) {
    throw new Error('refresh failed')
  }

  const body = (await response.json()) as RefreshResponse
  // The refresh endpoint only ever returns a new access token — the
  // refresh token itself doesn't rotate — so the existing one is passed
  // back in unchanged.
  authStore.setTokens(body.access_token, refreshToken)
  return body.access_token
}

async function request<T>(path: string, init: RequestInit = {}, isRetry = false): Promise<T> {
  const authStore = useAuthStore()
  const isAuthPath = AUTH_PATHS.includes(path)

  const headers: Record<string, string> = {
    ...(init.headers as Record<string, string> | undefined),
  }
  if (authStore.accessToken && !isAuthPath) {
    headers.Authorization = `Bearer ${authStore.accessToken}`
  }

  // "include" so the backend's demo_session_id cookie (only ever set when
  // DEMO_MODE=true) round-trips on cross-origin requests to the API. A
  // harmless no-op against the real product's backend, which never sets
  // that cookie.
  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers, credentials: 'include' })

  // Retrying at most once bounds this to a single extra round trip even if
  // the backend keeps returning 401 for some other reason after a
  // successful refresh — it never loops.
  if (response.status === 401 && !isAuthPath && !isRetry) {
    try {
      if (!refreshPromise) {
        refreshPromise = refreshAccessToken().finally(() => {
          refreshPromise = null
        })
      }
      await refreshPromise
    } catch {
      authStore.logout('expired')
      throw { status: 401, message: 'session expired' } satisfies ApiError
    }
    return request<T>(path, init, true)
  }

  if (!response.ok) {
    const error: ApiError = {
      status: response.status,
      message: await response.text(),
    }
    throw error
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

// useApi is the single entry point for HTTP calls in every component and
// store — never call fetch or axios directly from a component.
export function useApi() {
  return {
    get: <T>(path: string) => request<T>(path, { method: 'GET', headers: { 'Content-Type': 'application/json' } }),
    post: <T>(path: string, body?: unknown) =>
      request<T>(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
      }),
    patch: <T>(path: string, body?: unknown) =>
      request<T>(path, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
      }),
    delete: <T>(path: string) =>
      request<T>(path, { method: 'DELETE', headers: { 'Content-Type': 'application/json' } }),
    // postForm sends a multipart/form-data body (file uploads) — deliberately
    // no Content-Type header here, so fetch sets it itself with the correct
    // boundary. Used for POST /uploads/image.
    postForm: <T>(path: string, formData: FormData) => request<T>(path, { method: 'POST', body: formData }),
  }
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string

export interface ApiError {
  status: number
  message: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    ...init,
  })

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
    get: <T>(path: string) => request<T>(path, { method: 'GET' }),
    post: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
    patch: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
    delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  }
}

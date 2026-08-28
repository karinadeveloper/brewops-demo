import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { authState, setTokensMock, logoutMock } = vi.hoisted(() => {
  const state: { accessToken: string | null; refreshToken: string | null } = {
    accessToken: 'initial-access',
    refreshToken: 'initial-refresh',
  }
  return {
    authState: state,
    setTokensMock: vi.fn((accessToken: string, refreshToken: string) => {
      state.accessToken = accessToken
      state.refreshToken = refreshToken
    }),
    logoutMock: vi.fn(),
  }
})

vi.mock('../stores/auth', () => ({
  useAuthStore: () => ({
    get accessToken() {
      return authState.accessToken
    },
    get refreshToken() {
      return authState.refreshToken
    },
    setTokens: setTokensMock,
    logout: logoutMock,
  }),
}))

const { useApi } = await import('./useApi')

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

// Typed explicitly as the real fetch signature (url, init) so
// fetchMock.mock.calls[i][1] is typed as RequestInit — the test
// implementations below only need to read the URL, not init.
type FetchMock = (url: string, init?: RequestInit) => Promise<Response>

describe('useApi', () => {
  beforeEach(() => {
    authState.accessToken = 'initial-access'
    authState.refreshToken = 'initial-refresh'
    setTokensMock.mockClear()
    logoutMock.mockClear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('attaches the Authorization header from the auth store on every request', async () => {
    // Arrange
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { ok: true }))
    vi.stubGlobal('fetch', fetchMock)

    // Act
    await useApi().get('/products')

    // Assert
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer initial-access')
  })

  it('on a 401, refreshes the token and retries the original request exactly once', async () => {
    // Arrange
    let productsCallCount = 0
    const fetchMock = vi.fn<FetchMock>(async (url) => {
      if (url.includes('/auth/refresh')) {
        return jsonResponse(200, { access_token: 'new-access' })
      }
      // First call to /products fails; the retry (with the new token) succeeds.
      productsCallCount++
      return productsCallCount === 1 ? jsonResponse(401, 'unauthorized') : jsonResponse(200, { data: 'ok' })
    })
    vi.stubGlobal('fetch', fetchMock)

    // Act
    const result = await useApi().get('/products')

    // Assert
    expect(result).toEqual({ data: 'ok' })
    expect(setTokensMock).toHaveBeenCalledWith('new-access', 'initial-refresh')
    expect(logoutMock).not.toHaveBeenCalled()

    const productCalls = fetchMock.mock.calls.filter(([u]) => String(u).includes('/products'))
    expect(productCalls).toHaveLength(2)
    const retryInit = productCalls[1][1] as RequestInit
    expect((retryInit.headers as Record<string, string>).Authorization).toBe('Bearer new-access')
  })

  it('logs out with reason=expired when the refresh call itself fails', async () => {
    // Arrange
    const fetchMock = vi.fn<FetchMock>(async (url) => {
      if (url.includes('/auth/refresh')) {
        return jsonResponse(401, 'invalid refresh token')
      }
      return jsonResponse(401, 'unauthorized')
    })
    vi.stubGlobal('fetch', fetchMock)

    // Act
    await expect(useApi().get('/products')).rejects.toBeTruthy()

    // Assert
    expect(logoutMock).toHaveBeenCalledWith('expired')
    expect(setTokensMock).not.toHaveBeenCalled()
  })

  it('never retries more than once, even if the backend keeps returning 401', async () => {
    // Arrange
    const fetchMock = vi.fn<FetchMock>(async (url) => {
      if (url.includes('/auth/refresh')) {
        return jsonResponse(200, { access_token: 'new-access' })
      }
      return jsonResponse(401, 'still unauthorized')
    })
    vi.stubGlobal('fetch', fetchMock)

    // Act
    await expect(useApi().get('/products')).rejects.toMatchObject({ status: 401 })

    // Assert — exactly one refresh call, and exactly two calls to /products
    // (the original plus a single retry), never a loop.
    const refreshCalls = fetchMock.mock.calls.filter(([u]) => String(u).includes('/auth/refresh'))
    const productCalls = fetchMock.mock.calls.filter(([u]) => String(u).includes('/products'))
    expect(refreshCalls).toHaveLength(1)
    expect(productCalls).toHaveLength(2)
  })

  it('a 401 on the login request itself never triggers a refresh attempt', async () => {
    // Arrange
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(401, 'invalid credentials'))
    vi.stubGlobal('fetch', fetchMock)

    // Act
    await expect(useApi().post('/auth/login', { email: 'a', password: 'b' })).rejects.toMatchObject({
      status: 401,
    })

    // Assert
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(logoutMock).not.toHaveBeenCalled()
  })

  it('multiple concurrent 401s share a single in-flight refresh call', async () => {
    // Arrange — this is the easiest case to break: without single-flight
    // coordination, two requests failing at the same time would each kick
    // off their own POST /auth/refresh.
    const callCounts: Record<string, number> = {}
    const fetchMock = vi.fn<FetchMock>(async (url) => {
      if (url.includes('/auth/refresh')) {
        callCounts.refresh = (callCounts.refresh ?? 0) + 1
        // Simulate a network round trip so both callers' first attempts
        // land before the refresh resolves.
        await Promise.resolve()
        return jsonResponse(200, { access_token: 'new-access' })
      }
      callCounts[url] = (callCounts[url] ?? 0) + 1
      return callCounts[url] === 1 ? jsonResponse(401, 'unauthorized') : jsonResponse(200, { url })
    })
    vi.stubGlobal('fetch', fetchMock)

    // Act
    const api = useApi()
    const [a, b] = await Promise.all([api.get('/a'), api.get('/b')])

    // Assert
    expect(callCounts.refresh).toBe(1)
    expect(a).toEqual({ url: expect.stringContaining('/a') })
    expect(b).toEqual({ url: expect.stringContaining('/b') })
  })
})

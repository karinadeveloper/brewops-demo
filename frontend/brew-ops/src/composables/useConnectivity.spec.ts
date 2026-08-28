import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { __resetConnectivityForTests, useConnectivity } from './useConnectivity'

function setNavigatorOnLine(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, writable: true, configurable: true })
}

describe('useConnectivity', () => {
  beforeEach(() => {
    __resetConnectivityForTests()
  })

  afterEach(() => {
    __resetConnectivityForTests()
    vi.unstubAllGlobals()
  })

  it('is online when navigator.onLine is true and the health ping succeeds', async () => {
    // Arrange
    setNavigatorOnLine(true)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))

    // Act
    const { isOnline } = useConnectivity(60_000)

    // Assert
    await vi.waitFor(() => expect(isOnline.value).toBe(true))
  })

  it('marks offline immediately on the browser "offline" event', async () => {
    // Arrange
    setNavigatorOnLine(true)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
    const { isOnline } = useConnectivity(60_000)
    await vi.waitFor(() => expect(isOnline.value).toBe(true))

    // Act
    window.dispatchEvent(new Event('offline'))

    // Assert
    expect(isOnline.value).toBe(false)
  })

  it('the browser "online" event triggers an immediate ping instead of waiting for the interval', async () => {
    // Arrange
    setNavigatorOnLine(true)
    const fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetchMock)
    useConnectivity(60_000) // deliberately long — the online event must not wait for this
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    // Act
    window.dispatchEvent(new Event('online'))

    // Assert
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })

  it('marks offline when the health ping fails, even though navigator.onLine says true', async () => {
    // Arrange
    setNavigatorOnLine(true)
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))

    // Act
    const { isOnline } = useConnectivity(60_000)

    // Assert
    await vi.waitFor(() => expect(isOnline.value).toBe(false))
  })

  it('marks offline when the health ping responds but not ok', async () => {
    // Arrange
    setNavigatorOnLine(true)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))

    // Act
    const { isOnline } = useConnectivity(60_000)

    // Assert
    await vi.waitFor(() => expect(isOnline.value).toBe(false))
  })

  it('goes online -> offline -> online as the ping result changes over time', async () => {
    // Arrange
    setNavigatorOnLine(true)
    let healthy = true
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve({ ok: healthy })),
    )
    const { isOnline } = useConnectivity(30)
    await vi.waitFor(() => expect(isOnline.value).toBe(true))

    // Act — simulate the backend going down between pings
    healthy = false

    // Assert
    await vi.waitFor(() => expect(isOnline.value).toBe(false), { timeout: 2000 })

    // Act — and recovering
    healthy = true

    // Assert
    await vi.waitFor(() => expect(isOnline.value).toBe(true), { timeout: 2000 })
  })

  it('re-pings periodically on the configured interval', async () => {
    // Arrange
    setNavigatorOnLine(true)
    const fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetchMock)

    // Act
    useConnectivity(30)

    // Assert
    await vi.waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(3), { timeout: 2000 })
  })
})

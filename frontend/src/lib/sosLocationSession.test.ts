import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSosLocationSession } from './sosLocationSession'

const flush = async () => { for (let i = 0; i < 12; i++) await Promise.resolve() }
function setup(start = vi.fn().mockResolvedValue('session-1')) {
  let success: PositionCallback = () => undefined
  let failure: PositionErrorCallback = () => undefined
  const geo = { watchPosition: vi.fn((ok: PositionCallback, error?: PositionErrorCallback | null) => { success = ok; failure = error!; return 7 }), clearWatch: vi.fn() }
  const transport = { start, upload: vi.fn().mockResolvedValue({}), stop: vi.fn().mockResolvedValue({}) }
  const state = vi.fn()
  const control = createSosLocationSession(geo, transport, state)
  const fix = (timestamp = Date.now()) => success({ timestamp, coords: { latitude: 7.2, longitude: 8.1, accuracy: 10 } } as GeolocationPosition)
  return { geo, transport, state, control, fix, denied: () => failure({ code: 1 } as GeolocationPositionError) }
}
afterEach(() => vi.restoreAllMocks())

describe('explicit SOS location sessions', () => {
  it('cancels a start in flight and revokes its session without requesting GPS', async () => {
    let resolve!: (id: string) => void
    const s = setup(vi.fn(() => new Promise<string>((r) => { resolve = r })))
    const stopped = s.control.stop()
    resolve('late-session')
    await stopped
    expect(s.geo.watchPosition).not.toHaveBeenCalled()
    expect(s.transport.stop).toHaveBeenCalledWith('late-session')
  })
  it('throttles uploads and ignores stale device fixes', async () => {
    const clock = vi.spyOn(Date, 'now').mockReturnValue(1_000_000)
    const s = setup(); await flush()
    s.fix(980_000); await flush()
    expect(s.transport.upload).not.toHaveBeenCalled()
    s.fix(); await flush(); s.fix(); await flush()
    expect(s.transport.upload).toHaveBeenCalledTimes(1)
    clock.mockReturnValue(1_010_001); s.fix(); await flush()
    expect(s.transport.upload).toHaveBeenCalledTimes(2)
    await s.control.stop()
    expect(s.geo.clearWatch).toHaveBeenCalledWith(7)
    clock.mockReturnValue(1_030_000); s.fix(); await flush()
    expect(s.transport.upload).toHaveBeenCalledTimes(2)
  })
  it('revokes the session after GPS permission is denied', async () => {
    const s = setup(); await flush(); s.denied(); await flush()
    expect(s.transport.stop).toHaveBeenCalledWith('session-1')
    expect(s.state).toHaveBeenLastCalledWith('stopped', expect.stringContaining('permission was denied'))
  })
  it('does not queue or continue GPS uploads after a network failure', async () => {
    const s = setup(); await flush()
    s.transport.upload.mockRejectedValue(new Error('Offline'))
    s.fix(); await flush()
    expect(s.geo.clearWatch).toHaveBeenCalledWith(7)
    expect(s.transport.stop).toHaveBeenCalledWith('session-1')
    expect(s.state).toHaveBeenLastCalledWith('stopped', 'Offline')
  })
})

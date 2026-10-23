import { describe, expect, it, vi } from 'vitest'
import { copyText } from './copyText'

describe('copyText', () => {
  it('reports success only after the browser confirms the write', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    expect(await copyText('123', { writeText })).toBe(true)
    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText).toHaveBeenCalledWith('123')
  })

  it('returns false when permissions reject or the API is missing', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'))
    expect(await copyText('123', { writeText })).toBe(false)
    expect(await copyText('123', null)).toBe(false)
  })

  it('does not claim success while the clipboard promise is pending', async () => {
    let resolveCopy: (() => void) | undefined
    const writeText = vi.fn(() => new Promise<void>(resolve => { resolveCopy = resolve }))
    let finished = false
    const pending = copyText('123', { writeText }).then(ok => { finished = ok })
    expect(finished).toBe(false)
    resolveCopy?.()
    await pending
    expect(finished).toBe(true)
  })
})

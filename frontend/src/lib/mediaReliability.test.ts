import { describe, expect, it } from 'vitest'
import { enqueue } from '@/lib/offlineQueue'
import { extensionFor, uploadTypeFor } from '@/components/media/MediaRecorderField'

describe('critical media reliability', () => {
  it('keeps iPhone Safari voice as its real audio/mp4 format', () => {
    expect(uploadTypeFor('audio/mp4; codecs=mp4a.40.2', 'voice')).toBe('audio/mp4')
    expect(extensionFor('audio/mp4')).toBe('m4a')
    expect(uploadTypeFor('audio/webm;codecs=opus', 'voice')).toBe('audio/webm')
  })

  it('keeps video formats consistent with allowed backend extensions', () => {
    expect(extensionFor(uploadTypeFor('video/quicktime', 'video'))).toBe('mov')
    expect(extensionFor(uploadTypeFor('video/mp4', 'video'))).toBe('mp4')
  })

  it('does not silently queue emergency alerts for later delivery', async () => {
    await expect(enqueue('/sos/send', { latitude: 1, longitude: 1 })).rejects.toThrow(
      /Emergency SOS cannot be stored/,
    )
  })
})

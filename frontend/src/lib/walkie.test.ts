import { describe, expect, it } from 'vitest'
import {
  buildWalkieURL,
  decodeAudioChunk,
  encodeAudioChunk,
  formatCountdown,
  parseServerMessage,
  reduceWalkie,
  type PTTViewState,
} from './walkie'

describe('buildWalkieURL', () => {
  it('turns a relative API base into a ws URL on the current origin', () => {
    expect(
      buildWalkieURL('/api/v1', 'unit-1', 'token-1', {
        protocol: 'http:',
        host: 'localhost:5173',
      }),
    ).toBe('ws://localhost:5173/api/v1/ws?unitId=unit-1&token=token-1')
  })

  it('uses wss for an https page and preserves an absolute base', () => {
    expect(
      buildWalkieURL('https://api.example.com/api/v1/', 'unit-1', 'token-1', {
        protocol: 'https:',
        host: 'nativityguard.com',
      }),
    ).toBe('wss://api.example.com/api/v1/ws?unitId=unit-1&token=token-1')
  })

  it('falls back to /api/v1 when the base is empty', () => {
    expect(
      buildWalkieURL('', 'u', 't', { protocol: 'https:', host: 'example.com' }),
    ).toBe('wss://example.com/api/v1/ws?unitId=u&token=t')
  })
})

describe('audio codec', () => {
  it('round-trips signed PCM and the sample rate', () => {
    const samples = new Float32Array([-1, -0.5, 0, 0.5, 1])
    const encoded = encodeAudioChunk(samples, 48000)
    const decoded = decodeAudioChunk(encoded)

    expect(decoded).not.toBeNull()
    expect(decoded?.sampleRate).toBe(48000)
    expect(decoded?.samples).toHaveLength(samples.length)
    const decodedSamples = decoded?.samples ?? new Float32Array()
    for (let i = 0; i < samples.length; i += 1) {
      expect(decodedSamples[i]).toBeCloseTo(samples[i], 3)
    }
  })

  it('rejects frames that are not ours', () => {
    expect(decodeAudioChunk(new ArrayBuffer(4))).toBeNull()

    const wrongMagic = new ArrayBuffer(16)
    new DataView(wrongMagic).setUint32(0, 0xdeadbeef, false)
    expect(decodeAudioChunk(wrongMagic)).toBeNull()

    const oddPayload = new ArrayBuffer(8 + 3)
    const view = new DataView(oddPayload)
    view.setUint32(0, 0x50545431, false)
    view.setUint32(4, 48000, false)
    expect(decodeAudioChunk(oddPayload)).toBeNull()
  })
})

describe('parseServerMessage', () => {
  it('parses a typed control frame', () => {
    expect(parseServerMessage('{"type":"ptt.busy","speakerName":"Ada"}')).toMatchObject({
      type: 'ptt.busy',
      speakerName: 'Ada',
    })
  })

  it('rejects malformed or untyped frames', () => {
    expect(parseServerMessage('not json')).toBeNull()
    expect(parseServerMessage('[]')).toBeNull()
    expect(parseServerMessage('{"type":""}')).toBeNull()
    expect(parseServerMessage('{"type":42}')).toBeNull()
  })
})

describe('reduceWalkie', () => {
  const idle: PTTViewState = { status: 'idle' }

  it('marks the local client as transmitting when it wins the floor', () => {
    const next = reduceWalkie(
      idle,
      {
        type: 'ptt.granted',
        speakerId: 'client-1',
        speakerName: 'Me',
        expiresAt: new Date(60_000).toISOString(),
      },
      'client-1',
    )
    expect(next.status).toBe('transmitting')
    expect(next.expiresAt).toBe(60_000)
  })

  it('marks another speaker as busy', () => {
    const next = reduceWalkie(
      idle,
      { type: 'ptt.granted', speakerId: 'client-2', speakerName: 'Ada' },
      'client-1',
    )
    expect(next).toMatchObject({ status: 'busy', speakerName: 'Ada' })
  })

  it('returns to idle when the floor is released', () => {
    expect(
      reduceWalkie({ status: 'transmitting' }, { type: 'ptt.released' }, 'client-1').status,
    ).toBe('idle')
  })

  it('renders a busy snapshot and an idle snapshot', () => {
    expect(
      reduceWalkie(idle, { type: 'ptt.state', status: 'busy', speakerId: 'client-2' }, 'client-1')
        .status,
    ).toBe('busy')
    expect(
      reduceWalkie({ status: 'busy' }, { type: 'ptt.state', status: 'idle' }, 'client-1').status,
    ).toBe('idle')
  })

  it('surfaces server errors and ignores the override ack', () => {
    expect(
      reduceWalkie(idle, { type: 'ptt.error', reason: 'command_override_required' }, 'client-1'),
    ).toMatchObject({ status: 'error', error: 'command_override_required' })
    expect(reduceWalkie(idle, { type: 'ptt.override', enabled: true }, 'client-1')).toBe(idle)
  })
})

describe('formatCountdown', () => {
  it('formats whole seconds as m:ss', () => {
    expect(formatCountdown(60)).toBe('1:00')
    expect(formatCountdown(5)).toBe('0:05')
    expect(formatCountdown(-3)).toBe('0:00')
  })
})

/**
 * Walkie-talkie protocol helpers.
 *
 * Control frames are JSON; audio frames are a tiny binary envelope carrying
 * mono signed-16-bit PCM plus the sender's sample rate so the receiver can
 * resample correctly. Everything here is pure so it can be tested in the
 * default `node` vitest environment.
 */

export type PTTStatus =
  | 'disconnected'
  | 'connecting'
  | 'idle'
  | 'requesting'
  | 'transmitting'
  | 'busy'
  | 'error'

export interface PTTViewState {
  status: PTTStatus
  speakerId?: string
  speakerUserId?: string
  speakerName?: string
  /** Epoch milliseconds from the server's `expiresAt`. */
  expiresAt?: number
  error?: string
}

export interface WalkieLocation {
  protocol: string
  host: string
}

export type WalkieChannelKind = 'unit' | 'taskforce'

export interface WalkieChannel {
  kind: WalkieChannelKind
  id: string
}

/** Build the authenticated WebSocket URL for a unit or task-force channel. */
export function buildWalkieURL(
  base: string,
  channel: WalkieChannel,
  token: string,
  location: WalkieLocation,
): string {
  const url = new URL(base || '/api/v1', `${location.protocol}//${location.host}`)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.pathname = `${url.pathname.replace(/\/+$/, '')}/ws`
  url.search = ''
  url.searchParams.set(channel.kind === 'taskforce' ? 'taskforceId' : 'unitId', channel.id)
  url.searchParams.set('token', token)
  return url.toString()
}

const AUDIO_MAGIC = 0x50545431 // "PTT1"
const AUDIO_HEADER_BYTES = 8
const MIN_SAMPLE_RATE = 8000
const MAX_SAMPLE_RATE = 192000

/** Encode Float32 PCM as [magic|sampleRate|int16 PCM]. */
export function encodeAudioChunk(samples: Float32Array, sampleRate: number): ArrayBuffer {
  const buffer = new ArrayBuffer(AUDIO_HEADER_BYTES + samples.length * 2)
  const view = new DataView(buffer)
  view.setUint32(0, AUDIO_MAGIC, false)
  view.setUint32(4, sampleRate >>> 0, false)
  const pcm = new Int16Array(buffer, AUDIO_HEADER_BYTES)
  for (let i = 0; i < samples.length; i += 1) {
    const sample = Math.max(-1, Math.min(1, samples[i] ?? 0))
    pcm[i] = sample < 0 ? sample * 0x8000 : sample * 0x7fff
  }
  return buffer
}

/** Decode a [magic|sampleRate|int16 PCM] frame, or null if it is not ours. */
export function decodeAudioChunk(
  buffer: ArrayBuffer,
): { sampleRate: number; samples: Float32Array } | null {
  if (buffer.byteLength < AUDIO_HEADER_BYTES + 2) return null
  if ((buffer.byteLength - AUDIO_HEADER_BYTES) % 2 !== 0) return null

  const view = new DataView(buffer)
  if (view.getUint32(0, false) !== AUDIO_MAGIC) return null
  const sampleRate = view.getUint32(4, false)
  if (sampleRate < MIN_SAMPLE_RATE || sampleRate > MAX_SAMPLE_RATE) return null

  const pcm = new Int16Array(buffer, AUDIO_HEADER_BYTES)
  const samples = new Float32Array(pcm.length)
  for (let i = 0; i < pcm.length; i += 1) {
    samples[i] = pcm[i] / 0x8000
  }
  return { sampleRate, samples }
}

export interface WalkieServerMessage {
  type: string
  room?: string
  status?: string
  speakerId?: string
  speakerUserId?: string
  speakerName?: string
  expiresAt?: string
  remainingSeconds?: number
  reason?: string
  enabled?: boolean
  count?: number
  users?: string[]
  clientId?: string
  [key: string]: unknown
}

export function parseServerMessage(raw: string): WalkieServerMessage | null {
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') return null
    const type = (parsed as { type?: unknown }).type
    if (typeof type !== 'string' || type.length === 0) return null
    return parsed as WalkieServerMessage
  } catch {
    return null
  }
}

/** Apply one server control frame to the local UI state. */
export function reduceWalkie(
  state: PTTViewState,
  message: WalkieServerMessage,
  selfClientId: string | null,
): PTTViewState {
  const expiresAt = message.expiresAt ? Date.parse(message.expiresAt) : undefined
  const speakerId = typeof message.speakerId === 'string' ? message.speakerId : undefined
  const speakerUserId =
    typeof message.speakerUserId === 'string' ? message.speakerUserId : undefined
  const speakerName = typeof message.speakerName === 'string' ? message.speakerName : undefined

  const busy: PTTViewState = {
    status: 'busy',
    speakerId,
    speakerUserId,
    speakerName,
    expiresAt: Number.isFinite(expiresAt) ? expiresAt : undefined,
  }

  switch (message.type) {
    case 'welcome':
      return state.status === 'disconnected' || state.status === 'connecting'
        ? { status: 'idle' }
        : state

    case 'ptt.state':
    case 'ptt.granted':
      if (speakerId && selfClientId && speakerId === selfClientId) {
        return {
          status: 'transmitting',
          speakerId,
          speakerUserId,
          speakerName,
          expiresAt: Number.isFinite(expiresAt) ? expiresAt : undefined,
        }
      }
      if (speakerId) return busy
      return { status: 'idle' }

    case 'ptt.busy':
      return busy

    case 'ptt.released':
      return { status: 'idle' }

    case 'ptt.error':
      return {
        ...state,
        status: 'error',
        error: typeof message.reason === 'string' ? message.reason : 'walkie-talkie error',
      }

    default:
      return state
  }
}

export function formatCountdown(seconds: number): string {
  const safe = Math.max(0, Math.floor(seconds))
  return `${Math.floor(safe / 60)}:${String(safe % 60).padStart(2, '0')}`
}

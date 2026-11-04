import { useCallback, useEffect, useRef, useState } from 'react'
import { API_BASE, tokenStore } from '@/lib/apiClient'
import {
  buildWalkieURL,
  decodeAudioChunk,
  encodeAudioChunk,
  parseServerMessage,
  reduceWalkie,
  type PTTStatus,
  type PTTViewState,
} from '@/lib/walkie'

export interface UseWalkieTalkieOptions {
  unitId: string | null | undefined
  enabled?: boolean
}

export interface UseWalkieTalkieResult {
  status: PTTStatus
  speakerName?: string
  error?: string
  online: number
  secondsLeft: number
  muted: boolean
  setMuted: (muted: boolean) => void
  override: boolean
  setOverride: (enabled: boolean) => void
  clientId: string | null
  press: () => void
  release: () => void
  enablePlayback: () => void
}

type AudioContextCtor = typeof AudioContext

function describeMediaError(error: unknown): string {
  if (typeof DOMException !== 'undefined' && error instanceof DOMException) {
    if (error.name === 'NotAllowedError' || error.name === 'SecurityError') {
      return 'Microphone permission denied'
    }
    if (error.name === 'NotFoundError') return 'No microphone found'
    if (error.name === 'NotReadableError') return 'Microphone is already in use'
  }
  if (error instanceof Error && error.message) return error.message
  return 'Could not access the microphone'
}

/**
 * Live unit push-to-talk.
 *
 * Floor control is server-authoritative: pressing asks for the floor, audio is
 * only sent after `ptt.granted`, and the server enforces the 60-second ceiling.
 * The local timer is a UX backstop that releases at the same deadline.
 */
export function useWalkieTalkie({
  unitId,
  enabled = true,
}: UseWalkieTalkieOptions): UseWalkieTalkieResult {
  const [view, setView] = useState<PTTViewState>({ status: 'disconnected' })
  const [online, setOnline] = useState(0)
  const [secondsLeft, setSecondsLeft] = useState(60)
  const [muted, setMutedState] = useState(false)
  const [override, setOverrideState] = useState(false)
  const [clientId, setClientId] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const selfIdRef = useRef<string | null>(null)
  const reconnectRef = useRef<number | null>(null)
  const attemptsRef = useRef(0)
  const closedByUsRef = useRef(false)
  const holdingRef = useRef(false)
  const transmittingRef = useRef(false)
  const mutedRef = useRef(false)
  const playCursorRef = useRef(0)
  const statusRef = useRef<PTTStatus>('disconnected')
  const releaseRef = useRef<() => void>(() => {})

  const streamRef = useRef<MediaStream | null>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)
  const processorRef = useRef<ScriptProcessorNode | null>(null)
  const sourceNodeRef = useRef<MediaStreamAudioSourceNode | null>(null)
  const silentGainRef = useRef<GainNode | null>(null)

  useEffect(() => {
    mutedRef.current = muted
  }, [muted])

  useEffect(() => {
    statusRef.current = view.status
    if (view.status !== 'transmitting' && view.status !== 'requesting') {
      transmittingRef.current = false
    }
  }, [view.status])

  const enablePlayback = useCallback(async () => {
    if (typeof window === 'undefined') return
    if (!audioCtxRef.current) {
      const Ctor = window.AudioContext as AudioContextCtor | undefined
      if (!Ctor) return
      audioCtxRef.current = new Ctor()
    }
    if (audioCtxRef.current.state === 'suspended') {
      try {
        await audioCtxRef.current.resume()
      } catch {
        /* still waiting for a gesture */
      }
    }
  }, [])

  const stopCapture = useCallback(() => {
    processorRef.current?.disconnect()
    sourceNodeRef.current?.disconnect()
    silentGainRef.current?.disconnect()
    processorRef.current = null
    sourceNodeRef.current = null
    silentGainRef.current = null
    streamRef.current?.getTracks().forEach((track) => track.stop())
    streamRef.current = null
  }, [])

  const stopAudio = useCallback(() => {
    stopCapture()
    if (audioCtxRef.current) {
      void audioCtxRef.current.close().catch(() => undefined)
      audioCtxRef.current = null
    }
    playCursorRef.current = 0
  }, [stopCapture])

  const ensureAudio = useCallback(async () => {
    if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
      throw new Error('This browser does not support microphone capture')
    }
    if (!streamRef.current) {
      streamRef.current = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      })
    }
    await enablePlayback()
    const ctx = audioCtxRef.current
    if (!ctx) throw new Error('Audio is not available')

    if (!processorRef.current && streamRef.current) {
      const source = ctx.createMediaStreamSource(streamRef.current)
      const processor = ctx.createScriptProcessor(2048, 1, 1)
      const silent = ctx.createGain()
      silent.gain.value = 0

      processor.onaudioprocess = (event) => {
        if (!transmittingRef.current) return
        const input = event.inputBuffer.getChannelData(0)
        const payload = encodeAudioChunk(input, ctx.sampleRate)
        const socket = wsRef.current
        if (socket?.readyState === WebSocket.OPEN) socket.send(payload)
      }

      source.connect(processor)
      processor.connect(silent)
      silent.connect(ctx.destination)
      sourceNodeRef.current = source
      processorRef.current = processor
      silentGainRef.current = silent
    }
  }, [enablePlayback])

  const playChunk = useCallback((data: ArrayBuffer) => {
    const decoded = decodeAudioChunk(data)
    if (!decoded || decoded.samples.length === 0) return
    if (typeof window === 'undefined') return

    if (!audioCtxRef.current) {
      const Ctor = window.AudioContext as AudioContextCtor | undefined
      if (!Ctor) return
      audioCtxRef.current = new Ctor()
    }
    const ctx = audioCtxRef.current
    if (ctx.state === 'suspended') void ctx.resume()

    const audioBuffer = ctx.createBuffer(1, decoded.samples.length, decoded.sampleRate)
    audioBuffer.getChannelData(0).set(decoded.samples)

    const source = ctx.createBufferSource()
    source.buffer = audioBuffer
    const gain = ctx.createGain()
    gain.gain.value = mutedRef.current ? 0 : 1
    source.connect(gain)
    gain.connect(ctx.destination)

    const startAt = Math.max(ctx.currentTime + 0.01, playCursorRef.current)
    source.start(startAt)
    playCursorRef.current = startAt + audioBuffer.duration
  }, [])

  const handleMessage = useCallback((raw: string) => {
    const message = parseServerMessage(raw)
    if (!message) return

    if (message.type === 'welcome') {
      if (typeof message.clientId === 'string') {
        selfIdRef.current = message.clientId
        setClientId(message.clientId)
        setView((previous) => reduceWalkie(previous, message, message.clientId ?? null))
      }
      return
    }
    if (message.type === 'room-status') {
      if (typeof message.count === 'number') setOnline(message.count)
      return
    }
    if (message.type === 'ptt.override') {
      setOverrideState(Boolean(message.enabled))
      return
    }

    const self = selfIdRef.current
    setView((previous) => reduceWalkie(previous, message, self))

    if (message.type === 'ptt.granted') {
      const isSelf = Boolean(message.speakerId && message.speakerId === self)
      if (isSelf) {
        if (holdingRef.current) {
          transmittingRef.current = true
        } else {
          const socket = wsRef.current
          if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'ptt.release' }))
          }
          stopCapture()
        }
      } else {
        transmittingRef.current = false
        stopCapture()
      }
      return
    }
    if (message.type === 'ptt.busy' || message.type === 'ptt.released' || message.type === 'ptt.error') {
      transmittingRef.current = false
      stopCapture()
    }
  }, [stopCapture])

  useEffect(() => {
    if (!enabled || !unitId) {
      setView({ status: 'disconnected' })
      return
    }

    const token = tokenStore.get()
    if (!token) {
      setView({ status: 'error', error: 'Not signed in' })
      return
    }

    let disposed = false
    closedByUsRef.current = false
    attemptsRef.current = 0

    function connect() {
      if (disposed) return
      setView((previous) => ({ ...previous, status: 'connecting', error: undefined }))

      let socket: WebSocket
      try {
        socket = new WebSocket(buildWalkieURL(API_BASE, unitId as string, token as string, window.location))
      } catch {
        setView({ status: 'error', error: 'Could not open the unit channel' })
        return
      }
      socket.binaryType = 'arraybuffer'
      wsRef.current = socket

      socket.onopen = () => {
        attemptsRef.current = 0
      }

      socket.onmessage = (event) => {
        if (typeof event.data === 'string') {
          handleMessage(event.data)
        } else if (event.data instanceof ArrayBuffer) {
          playChunk(event.data)
        }
      }

      socket.onclose = (event) => {
        if (wsRef.current === socket) wsRef.current = null
        transmittingRef.current = false
        holdingRef.current = false
        if (disposed || closedByUsRef.current || event.code === 1000) {
          setView({ status: 'disconnected' })
          return
        }
        setView({ status: 'connecting' })
        attemptsRef.current = Math.min(attemptsRef.current + 1, 6)
        const delay = Math.min(30_000, 1000 * 2 ** (attemptsRef.current - 1))
        reconnectRef.current = window.setTimeout(connect, delay)
      }
    }

    connect()

    return () => {
      disposed = true
      closedByUsRef.current = true
      if (reconnectRef.current !== null) {
        window.clearTimeout(reconnectRef.current)
        reconnectRef.current = null
      }
      transmittingRef.current = false
      holdingRef.current = false
      const socket = wsRef.current
      wsRef.current = null
      if (socket) {
        if (socket.readyState === WebSocket.OPEN) {
          try {
            socket.send(JSON.stringify({ type: 'ptt.release' }))
          } catch {
            /* closing anyway */
          }
        }
        socket.close(1000, 'walkie-talkie closed')
      }
      stopAudio()
    }
  }, [enabled, unitId, handleMessage, playChunk, stopAudio])

  useEffect(() => {
    if (view.status !== 'transmitting' || !view.expiresAt) {
      setSecondsLeft(60)
      return
    }
    const tick = () => {
      const remaining = Math.max(0, Math.ceil((view.expiresAt! - Date.now()) / 1000))
      setSecondsLeft(remaining)
      if (remaining <= 0) releaseRef.current()
    }
    tick()
    const id = window.setInterval(tick, 250)
    return () => window.clearInterval(id)
  }, [view.status, view.expiresAt])

  const press = useCallback(() => {
    if (!enabled || !unitId || holdingRef.current) return
    const status = statusRef.current
    if (status === 'requesting' || status === 'transmitting' || status === 'disconnected' || status === 'error') {
      return
    }
    const socket = wsRef.current
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      setView({ status: 'error', error: 'Channel is not connected' })
      return
    }

    holdingRef.current = true
    setView((previous) => ({ ...previous, status: 'requesting', error: undefined }))

    void ensureAudio()
      .then(() => {
        if (!holdingRef.current) return
        const current = wsRef.current
        if (current?.readyState === WebSocket.OPEN) {
          current.send(JSON.stringify({ type: 'ptt.request' }))
        }
      })
      .catch((error: unknown) => {
        holdingRef.current = false
        setView({ status: 'error', error: describeMediaError(error) })
      })
  }, [enabled, unitId, ensureAudio])

  const release = useCallback(() => {
    const wasHolding = holdingRef.current
    holdingRef.current = false
    transmittingRef.current = false
    if (!wasHolding) return
    const socket = wsRef.current
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'ptt.release' }))
    }
    setView((previous) =>
      previous.status === 'transmitting' || previous.status === 'requesting'
        ? { status: 'idle' }
        : previous,
    )
    stopCapture()
  }, [stopCapture])

  useEffect(() => {
    releaseRef.current = release
  }, [release])

  const setMuted = useCallback((next: boolean) => {
    mutedRef.current = next
    setMutedState(next)
  }, [])

  const setOverride = useCallback((next: boolean) => {
    const socket = wsRef.current
    if (!socket || socket.readyState !== WebSocket.OPEN) return
    setOverrideState(next)
    socket.send(JSON.stringify({ type: 'ptt.command_override', data: { enabled: next } }))
  }, [])

  return {
    status: view.status,
    speakerName: view.speakerName,
    error: view.error,
    online,
    secondsLeft,
    muted,
    setMuted,
    override,
    setOverride,
    clientId,
    press,
    release,
    enablePlayback: () => void enablePlayback(),
  }
}

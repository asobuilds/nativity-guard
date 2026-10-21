import { useCallback, useEffect, useRef, useState } from 'react'
import {
  AlertTriangle,
  CheckCircle2,
  Loader2,
  Mic,
  RefreshCw,
  Square,
  Upload,
  Video,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { cn } from '@/lib/cn'
import type { PresignEvidenceResponse } from '@/types/api'

type Mode = 'voice' | 'video'
type State = 'idle' | 'permission' | 'recording' | 'preview' | 'uploading' | 'ready'

const VOICE_TYPES = ['audio/mpeg', 'audio/mp4', 'audio/wav', 'audio/ogg', 'audio/webm']
const VIDEO_TYPES = ['video/mp4', 'video/webm', 'video/quicktime']

/** Strip codec parameters and lowercase a MIME type. */
function bareType(type: string): string {
  return type.split(';')[0].trim().toLowerCase()
}

/**
 * Map a recorded MIME type onto one the backend accepts.
 *
 * Preserve Safari's audio/mp4 instead of falsely declaring MPEG bytes.
 * Backend presigning and storage now allow audio/mp4 (M4A).
 */
function uploadTypeFor(type: string, mode: Mode): string {
  const bare = bareType(type)
  if (mode === 'voice') {
    if (VOICE_TYPES.includes(bare)) return bare
    return 'audio/mpeg'
  }
  if (VIDEO_TYPES.includes(bare)) return bare
  return 'video/mp4'
}

function extensionFor(uploadType: string): string {
  switch (uploadType) {
    case 'audio/mpeg':
      return 'mp3'
    case 'audio/mp4':
      return 'm4a'
    case 'audio/ogg':
      return 'ogg'
    case 'audio/wav':
      return 'wav'
    case 'video/quicktime':
      return 'mov'
    default:
      return uploadType.split('/')[1] || 'bin'
  }
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

/** Pick a recording type the browser supports for this mode. */
function pickRecordingType(mode: Mode): string {
  if (typeof MediaRecorder === 'undefined') return ''
  const voiceCandidates = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg;codecs=opus']
  const videoCandidates = [
    'video/webm;codecs=vp9',
    'video/webm;codecs=vp8',
    'video/webm',
    'video/mp4',
    'video/quicktime',
  ]
  const candidates = mode === 'voice' ? voiceCandidates : videoCandidates
  return candidates.find((type) => MediaRecorder.isTypeSupported(type)) ?? ''
}

/**
 * Capture a short voice note or video clip and upload it through the media
 * presign → PUT flow.
 *
 * There is no confirm endpoint for these attachments: the returned object key
 * is handed straight to `onCaptured`, and the caller passes it along with the
 * SOS or community report mutation, which re-verifies the object server-side.
 */
export function MediaRecorderField({
  mode,
  presignPath,
  onCaptured,
  onFileReady,
  maxDuration = mode === 'voice' ? 30 : 60,
  maxSize = mode === 'voice' ? 5 * 1024 * 1024 : 20 * 1024 * 1024,
  label = mode === 'voice' ? 'Voice note' : 'Video clip',
  className,
}: {
  mode: Mode
  presignPath?: string
  onCaptured?: (key: string) => void
  /** For case reports: hold a file locally until the case ID exists. */
  onFileReady?: (file: File | null) => void
  maxDuration?: number
  maxSize?: number
  label?: string
  className?: string
}) {
  const { notify } = useToast()
  const [state, setState] = useState<State>('idle')
  const [error, setError] = useState<string | null>(null)
  const [elapsed, setElapsed] = useState(0)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)

  const recorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<BlobPart[]>([])
  const blobRef = useRef<Blob | null>(null)
  const timerRef = useRef<number | null>(null)
  const tickRef = useRef<number | null>(null)
  const liveVideoRef = useRef<HTMLVideoElement | null>(null)

  const stopStream = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop())
    streamRef.current = null
  }, [])

  const clearTimers = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
    if (tickRef.current !== null) {
      window.clearInterval(tickRef.current)
      tickRef.current = null
    }
  }, [])

  const clearPreviewUrl = useCallback((url: string | null) => {
    if (url) URL.revokeObjectURL(url)
  }, [])

  useEffect(() => {
    return () => {
      clearTimers()
      stopStream()
      if (recorderRef.current && recorderRef.current.state !== 'inactive') {
        recorderRef.current.stop()
      }
      recorderRef.current = null
    }
  }, [clearTimers, stopStream])

  useEffect(() => {
    return () => { if (previewUrl) URL.revokeObjectURL(previewUrl) }
  }, [previewUrl])

  // Attach the live camera feed once recording starts. The effect runs after
  // the <video> for the recording state has mounted, so the ref is populated.
  useEffect(() => {
    if (state === 'recording' && mode === 'video' && liveVideoRef.current && streamRef.current) {
      liveVideoRef.current.srcObject = streamRef.current
    }
  }, [state, mode])

  const resetCapture = useCallback(() => {
    clearTimers()
    stopStream()
    chunksRef.current = []
    blobRef.current = null
    setPreviewUrl((current) => {
      clearPreviewUrl(current)
      return null
    })
    setElapsed(0)
    setError(null)
    setState('idle')
    onCaptured?.('')
    onFileReady?.(null)
  }, [clearPreviewUrl, clearTimers, onCaptured, onFileReady, stopStream])

  async function startRecording() {
    onCaptured?.('')
    onFileReady?.(null)
    setError(null)
    setState('permission')

    const baseConstraints: MediaStreamConstraints =
      mode === 'voice'
        ? { audio: true }
        : { audio: true, video: { facingMode: 'environment' } }

    let stream: MediaStream
    try {
      stream = await navigator.mediaDevices.getUserMedia(baseConstraints)
    } catch (err) {
      const denied =
        err instanceof DOMException &&
        (err.name === 'NotAllowedError' || err.name === 'SecurityError')

      // A rear camera may not exist (desktop / front-only devices); retry with
      // the default camera before giving up. A denial is final, so retrying
      // would only prompt the user a second time.
      if (mode === 'video' && !denied) {
        try {
          stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
        } catch {
          setError('Microphone or camera access was denied. Check your browser permissions.')
          setState('idle')
          return
        }
      } else {
        setError(
          mode === 'voice'
            ? 'Microphone access was denied. Check your browser permissions.'
            : 'Microphone or camera access was denied. Check your browser permissions.',
        )
        setState('idle')
        return
      }
    }

    streamRef.current = stream
    chunksRef.current = []

    const recordingType = pickRecordingType(mode)
    let recorder: MediaRecorder
    try {
      recorder = new MediaRecorder(stream, recordingType ? { mimeType: recordingType } : undefined)
    } catch {
      setError('Recording is not supported on this browser.')
      stopStream()
      setState('idle')
      return
    }

    recorderRef.current = recorder

    recorder.ondataavailable = (event) => {
      if (event.data.size > 0) chunksRef.current.push(event.data)
    }

    recorder.onstop = () => {
      clearTimers()
      const blob = new Blob(chunksRef.current, {
        type: recorder.mimeType || recordingType || undefined,
      })

      if (blob.size === 0) {
        setError('Nothing was captured. Try again.')
        stopStream()
        setState('idle')
        return
      }
      if (blob.size > maxSize) {
        setError(`That recording exceeds the ${Math.round(maxSize / 1024 / 1024)} MB limit.`)
        stopStream()
        setState('idle')
        return
      }

      blobRef.current = blob
      const url = URL.createObjectURL(blob)
      setPreviewUrl((current) => {
        clearPreviewUrl(current)
        return url
      })
      stopStream()
      setElapsed(0)
      setState('preview')
    }

    try {
      recorder.start()
    } catch {
      setError('Recording could not start. Try again.')
      stopStream()
      setState('idle')
      return
    }

    setState('recording')
    setElapsed(0)
    const startedAt = Date.now()
    tickRef.current = window.setInterval(() => {
      setElapsed(Math.floor((Date.now() - startedAt) / 1000))
    }, 250)
    timerRef.current = window.setTimeout(() => {
      if (recorder.state === 'recording') recorder.stop()
    }, maxDuration * 1000)
  }

  function stopRecording() {
    clearTimers()
    if (recorderRef.current?.state === 'recording') recorderRef.current.stop()
  }

  function chooseFile(file: File | undefined) {
    if (!file) return
    setError(null)
    const mime = bareType(file.type)
    const allowed = mode === 'voice' ? VOICE_TYPES : VIDEO_TYPES
    if (!allowed.includes(mime)) {
      setError(`Select a supported ${mode === 'voice' ? 'audio' : 'video'} file.`)
      return
    }
    if (file.size <= 0 || file.size > maxSize) {
      setError(`Choose a file smaller than ${Math.round(maxSize / (1024 * 1024))} MB.`)
      return
    }
    onCaptured?.('')
    onFileReady?.(null)
    blobRef.current = file
    setPreviewUrl(current => {
      clearPreviewUrl(current)
      return URL.createObjectURL(file)
    })
    setState('preview')
  }

  async function upload() {
    const blob = blobRef.current
    if (!blob) return
    setState('uploading')
    setError(null)
    try {
      const uploadType = uploadTypeFor(blob.type, mode)
      const filename = `${mode === 'voice' ? 'voice-note' : 'video-clip'}.${extensionFor(uploadType)}`
      const file = new File([blob], filename, { type: uploadType })

      // A case ID is created only when the reporter submits their report.
      // Do not store a key prematurely or mislabel locally held evidence as uploaded.
      if (onFileReady) {
        onFileReady(file)
        setState('ready')
        return
      }
      if (!presignPath || !onCaptured) throw new Error('Recording upload is not configured.')

      // First attempt: browser -> object storage. Some deployments lack the
      // storage-provider CORS policy; fall back to the authenticated API.
      let key: string | null = null
      let canUseFallback = false
      try {
        const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer())
        const sha256 = Array.from(new Uint8Array(digest))
          .map(b => b.toString(16).padStart(2, '0')).join('')
        const presign = await api.post<PresignEvidenceResponse>(presignPath, {
          filename, contentType: uploadType, sizeBytes: file.size, sha256,
        })
        try {
          const response = await fetch(presign.uploadUrl, {
            method: 'PUT',
            headers: {
              'Content-Type': uploadType,
              'x-amz-meta-sha256': sha256,
              'x-amz-meta-original-filename': filename,
            },
            body: file,
          })
          if (!response.ok) {
            if (response.status === 403 || response.status >= 500) canUseFallback = true
            else throw new Error(`Storage refused upload (HTTP ${response.status}).`)
          } else key = presign.key
        } catch (error) {
          if (error instanceof TypeError) canUseFallback = true
          else throw error
        }
      } catch (error) {
        if (ApiError.isNetwork(error) || (error instanceof ApiError && error.status === 503)) {
          canUseFallback = true
        } else {
          throw error
        }
      }
      if (!key && canUseFallback) {
        const saved = await api.upload<{ key: string }>('/sos/media/upload', file)
        key = saved.key
      }
      if (!key) throw new Error('Recording was not stored. Please try again.')
      onCaptured(key)
      setState('ready')
      notify(mode === 'voice' ? 'Voice note uploaded' : 'Video clip uploaded', 'success')
    } catch (error) {
      const message = ApiError.isNetwork(error)
        ? 'Cannot reach the server. Check your internet connection and try again.'
        : error instanceof Error ? error.message : 'Recording upload failed.'
      setError(message)
      setState('preview')
      notify(message, 'error')
    }
  }

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <p className="text-xs font-medium text-ink-muted">{label}</p>

      <label className="flex items-center gap-2 text-xs text-ink-muted">
        <Upload className="size-4 shrink-0" aria-hidden />
        Or choose an existing ${mode === 'voice' ? 'audio' : 'video'} file
        <input
          type="file"
          className="max-w-[190px] text-xs"
          accept={mode === 'voice' ? 'audio/mpeg,audio/mp4,audio/wav,audio/ogg,audio/webm,.m4a' : 'video/mp4,video/webm,video/quicktime,.mov'}
          disabled={state === 'uploading' || state === 'recording' || state === 'permission'}
          onChange={event => {
            chooseFile(event.currentTarget.files?.[0])
            event.currentTarget.value = ''
          }}
        />
      </label>

      {state === 'idle' && (
        <button
          type="button"
          onClick={() => void startRecording()}
          className="flex flex-col items-center justify-center gap-2 rounded-lg border border-border-hi bg-surface-hi px-4 py-4 text-center transition-colors hover:bg-surface-hi/80 focus-visible:outline-2 focus-visible:outline-signal"
        >
          {mode === 'voice' ? (
            <Mic className="size-6 text-signal" aria-hidden />
          ) : (
            <Video className="size-6 text-signal" aria-hidden />
          )}
          <span className="text-sm font-medium text-ink">Record {label.toLowerCase()}</span>
          <span className="text-xs text-ink-muted">
            Up to {formatTime(maxDuration)} · {Math.round(maxSize / 1024 / 1024)} MB
          </span>
        </button>
      )}

      {state === 'permission' && (
        <div className="flex items-center justify-center gap-2 rounded-lg border border-border-hi bg-surface-hi px-4 py-6">
          <Loader2 className="size-4 animate-spin text-signal" aria-hidden />
          <span className="text-sm text-ink-muted">Requesting permission…</span>
        </div>
      )}

      {state === 'recording' && (
        <div className="space-y-3 rounded-lg border border-warn/30 bg-warn/5 p-4">
          <div className="flex items-center gap-2 text-warn">
            <span className="size-2 animate-pulse rounded-full bg-warn" aria-hidden />
            <span className="text-sm font-medium">Recording… {formatTime(elapsed)}</span>
            <span className="ml-auto text-xs text-ink-muted">
              {formatTime(Math.max(0, maxDuration - elapsed))} left
            </span>
          </div>

          {mode === 'video' && (
            <video
              ref={liveVideoRef}
              autoPlay
              muted
              playsInline
              className="max-h-44 w-full rounded-lg border border-border bg-black object-cover"
            />
          )}

          <Button
            size="sm"
            variant="ghost"
            icon={<Square className="size-4" />}
            onClick={stopRecording}
          >
            Stop
          </Button>
        </div>
      )}

      {state === 'preview' && previewUrl && (
        <div className="space-y-3 rounded-lg border border-border-hi bg-surface-hi p-4">
          {mode === 'video' ? (
            <video
              src={previewUrl}
              controls
              playsInline
              className="max-h-44 w-full rounded-lg border border-border bg-black object-cover"
            />
          ) : (
            <audio src={previewUrl} controls className="w-full" />
          )}

          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="ghost"
              icon={<RefreshCw className="size-4" />}
              onClick={resetCapture}
            >
              Re-record
            </Button>
            <Button
              size="sm"
              variant="primary"
              icon={<Upload className="size-4" />}
              onClick={() => void upload()}
            >
              {onFileReady ? 'Keep for report' : `Attach ${label.toLowerCase()}`}
            </Button>
          </div>
        </div>
      )}

      {state === 'uploading' && (
        <div className="flex items-center justify-center gap-2 rounded-lg border border-border-hi bg-surface-hi px-4 py-3">
          <Loader2 className="size-4 animate-spin text-signal" aria-hidden />
          <span className="text-sm text-ink-muted">Uploading…</span>
        </div>
      )}

      {state === 'ready' && (
        <div className="flex items-center gap-2 rounded-lg border border-ok/30 bg-ok/5 px-3 py-2">
          <CheckCircle2 className="size-4 shrink-0 text-ok" aria-hidden />
          <span className="text-sm text-ink">{onFileReady ? `${label} ready — attached after report submission` : `${label} uploaded`}</span>
          <button
            type="button"
            onClick={resetCapture}
            className="ml-auto text-xs text-signal underline-offset-2 hover:underline"
          >
            Remove
          </button>
        </div>
      )}

      {error && state !== 'ready' && (
        <p className="flex items-start gap-1 text-xs text-emergency" role="alert">
          <AlertTriangle className="mt-0.5 size-3 shrink-0" aria-hidden />
          <span>{error}</span>
        </p>
      )}
    </div>
  )
}

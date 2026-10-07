/** One foreground-only, explicitly started sharing session. Never queues GPS offline. */
export interface SharingTransport {
  start: () => Promise<string>
  upload: (sessionId: string, coords: GeolocationCoordinates) => Promise<unknown>
  stop: (sessionId: string) => Promise<unknown>
}

export function createSosLocationSession(
  geo: Pick<Geolocation, 'watchPosition' | 'clearWatch'>,
  transport: SharingTransport,
  onState: (state: 'starting' | 'sharing' | 'stopped', message: string) => void,
) {
  let cancelled = false
  let sessionId: string | null = null
  let watch: number | null = null
  let lastSent = -Infinity
  let pending: Promise<unknown> | null = null
  let stopping: Promise<void> | null = null

  const ready = (async () => {
    try {
      sessionId = await transport.start()
      if (cancelled) return

      watch = geo.watchPosition((position) => {
        if (cancelled || pending || Date.now() - lastSent < 10_000) return

        // Do not upload an old cached device fix as a fresh server position.
        if (
          Date.now() - position.timestamp > 15_000 ||
          position.timestamp > Date.now() + 10_000
        ) {
          return
        }

        lastSent = Date.now()

        pending = transport
          .upload(sessionId!, position.coords)
          .then(() => {
            if (!cancelled) {
              onState(
                'sharing',
                'Sharing your location for this SOS while this page is open.',
              )
            }
          })
          .catch((error: unknown) => {
            if (!cancelled) {
              const message =
                error instanceof Error
                  ? error.message
                  : 'Location upload failed.'

              void stop().then(() => onState('stopped', message))
            }
          })
          .finally(() => {
            pending = null
          })
      }, (error) => {
        void stop().then(() =>
          onState(
            'stopped',
            error.code === 1
              ? 'Location permission was denied. You can try again after allowing it.'
              : 'A GPS fix could not be obtained. Try again.',
          ),
        )
      }, {
        enableHighAccuracy: true,
        maximumAge: 0,
        timeout: 15_000,
      })
    } catch (error) {
      cancelled = true
      onState(
        'stopped',
        error instanceof Error
          ? error.message
          : 'Could not start location sharing.',
      )
    }
  })()

  function stop(): Promise<void> {
    if (stopping) return stopping

    cancelled = true

    if (watch != null) {
      geo.clearWatch(watch)
    }

    watch = null

    stopping = (async () => {
      await ready
      await pending?.catch(() => undefined)

      if (sessionId) {
        try {
          await transport.stop(sessionId)
        } catch {
          onState(
            'stopped',
            'Sharing stopped on this device. The last shared location will expire within two minutes.',
          )
          return
        }
      }

      onState('stopped', 'Location sharing stopped.')
    })()

    return stopping
  }

  onState('starting', 'Requesting location access...')

  return { stop }
}
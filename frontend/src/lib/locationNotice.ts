/**
 * Why a location request failed, and what to tell the visitor about it.
 *
 * The browser reports several very different problems as one flat
 * "permission denied", and the fix for each is different:
 *
 *  - an insecure origin (http:// on a LAN address, for instance) can never
 *    hand over a position, no matter how the visitor answers the prompt;
 *  - a refusal is fixed in the browser's site settings;
 *  - a timeout means the device or the GPS is not answering;
 *  - an unsupported browser never had the feature.
 *
 * Pure functions, so they can be pinned in this repo's node-only test
 * environment.
 */

/** `GeolocationPositionError.code` values, as the browser reports them. */
export const GEO_ERROR_DENIED = 1
export const GEO_ERROR_UNAVAILABLE = 2
export const GEO_ERROR_TIMEOUT = 3

export type LocateFailureKind =
  | 'insecure'
  | 'denied'
  | 'timeout'
  | 'unavailable'
  | 'unsupported'

export interface LocateFailureInput {
  /** The browser's error code, when the failure came from getCurrentPosition. */
  errorCode?: number | null
  /** The location hook's permission state. */
  permission?: string
  /** `window.isSecureContext` — geolocation is only exposed on secure origins. */
  secureContext?: boolean
  /** Whether the browser exposes `navigator.geolocation` at all. */
  supported?: boolean
}

/**
 * Classify a failed location request. Returns null when nothing went wrong.
 *
 * The insecure-origin check comes first: on an `http://` address the browser
 * reports code 1 (denied) no matter what the visitor answers, so calling it a
 * denial sends them to a setting that will never help.
 */
export function locateFailureKind(input: LocateFailureInput): LocateFailureKind | null {
  if (input.supported === false) return 'unsupported'
  if (input.secureContext === false) return 'insecure'
  if (input.errorCode === GEO_ERROR_DENIED) return 'denied'
  if (input.errorCode === GEO_ERROR_TIMEOUT) return 'timeout'
  if (input.errorCode === GEO_ERROR_UNAVAILABLE) return 'unavailable'
  if (input.permission === 'denied') return 'denied'
  if (input.permission === 'unavailable') return 'unsupported'
  return null
}

const MESSAGES: Record<LocateFailureKind, string> = {
  insecure:
    'Your browser only shares a location over a secure (https) connection. Open this site over https to pinpoint yourself — allowing location cannot help on an http:// address.',
  denied:
    'Location permission was denied. Allow location for this site from the padlock icon in the address bar, then press Locate My Area again.',
  timeout:
    'Your device did not return a position in time. Check that location services are switched on, then press Locate My Area again.',
  unavailable:
    'Your device could not determine a position. Check that location services are switched on and that GPS is available, then press Locate My Area again.',
  unsupported:
    'This browser does not report a location. You can still browse the map and choose an area by hand.',
}

/** The visitor-facing explanation for a failure kind. */
export function locateFailureMessage(kind: LocateFailureKind): string {
  return MESSAGES[kind]
}

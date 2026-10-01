import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { USE_MOCKS } from '@/mocks/config'
import { applyPreferences, readPreferences } from '@/lib/preferences'
import './index.css'

// ---- Chunk-load failure auto-recovery -------------------------------------
//
// When a new deploy lands, Vite's hashed chunk filenames change. If the
// browser is still running an old index.html (from an HTTP cache, a
// service-worker cache, or a tab that has been open across a deploy),
// fetching one of the referenced chunks 404s. Vite fires `vite:preloadError`
// for exactly this case.
//
// Response: reload the page ONCE. The reload fetches the current
// index.html, whose chunk references are valid. A sessionStorage guard
// ensures we never loop — if a second error fires within the same session,
// we let it surface (it means the deployment itself is broken).
//
// The guard is cleared after the app mounts successfully for a moment.
const CHUNK_RELOAD_KEY = 'cs.chunkReloadAttempted'
window.addEventListener('vite:preloadError', (event) => {
  event.preventDefault()
  try {
    if (sessionStorage.getItem(CHUNK_RELOAD_KEY) === '1') return
    sessionStorage.setItem(CHUNK_RELOAD_KEY, '1')
  } catch {
    // Private mode / storage disabled — reload blindly. Better a single
    // reload than a hard failure for a returning user.
  }
  window.location.reload()
})

async function bootstrap() {
  applyPreferences(readPreferences())

  // Install the mock API *before* rendering — the app must not fire a request
  // before the adapter is in place.
  if (USE_MOCKS) {
    const { installMockApi } = await import('./mocks/install')
    installMockApi()
  }

  const { App } = await import('./App')

  const container = document.getElementById('root')
  if (!container) throw new Error('Root element #root not found')

  createRoot(container).render(
    <StrictMode>
      {/* Outermost boundary: whatever else fails, this is what stands between a
          thrown render and an empty page. `AppShell` has its own, narrower one so
          a crashed page keeps the navigation — this one catches what that cannot,
          including a crash in the shell itself. */}
      <ErrorBoundary fullPage>
        <App />
      </ErrorBoundary>
    </StrictMode>,
  )

  // The app mounted. If we got here, the current chunk set is valid —
  // clear the reload guard so a future deploy gets a fresh single retry.
  setTimeout(() => {
    try {
      sessionStorage.removeItem(CHUNK_RELOAD_KEY)
    } catch {
      /* ignore */
    }
  }, 3000)

  // Register the service worker so the app shell loads offline.
  // Fails silently in dev or unsupported browsers.
  if ('serviceWorker' in navigator && import.meta.env.PROD) {
    navigator.serviceWorker.register('/sw.js').catch(() => undefined)
  }
}

void bootstrap()
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { USE_MOCKS } from '@/mocks/config'
import './index.css'

async function bootstrap() {
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


  // Register the service worker so the app shell loads offline.
  // Fails silently in dev or unsupported browsers.
  if ('serviceWorker' in navigator && import.meta.env.PROD) {
    navigator.serviceWorker.register('/sw.js').catch(() => undefined)
  }
}

void bootstrap()

/* Nativity Guard service worker
 * Cache-first for the app shell so the app loads instantly and works offline.
 * Network-first for API GETs with cache fallback so reads survive a dead zone.
 * Never caches POST/PUT/DELETE — those go through the write queue instead.
 */
const CACHE = 'ng-shell-v1'
const SHELL = ['/', '/index.html', '/favicon.svg', '/manifest.webmanifest']

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(SHELL).catch(() => undefined)),
  )
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))),
    ),
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const req = event.request
  const url = new URL(req.url)

  // Only handle same-origin + our backend. Skip everything else.
  if (req.method !== 'GET') return

  // App shell (HTML navigation) — cache-first, fall back to /index.html when offline.
  if (req.mode === 'navigate') {
    event.respondWith(
      fetch(req).catch(() => caches.match('/index.html').then((r) => r || Response.error())),
    )
    return
  }

  // Static assets — cache-first.
  if (url.origin === self.location.origin) {
    event.respondWith(
      caches.match(req).then((cached) => {
        if (cached) return cached
        return fetch(req)
          .then((resp) => {
            const copy = resp.clone()
            caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => undefined)
            return resp
          })
          .catch(() => caches.match('/index.html').then((r) => r || Response.error()))
      }),
    )
    return
  }

  // API GETs — network-first, fall back to last cached response.
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(req)
        .then((resp) => {
          const copy = resp.clone()
          caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => undefined)
          return resp
        })
        .catch(() => caches.match(req).then((r) => r || Response.error())),
    )
    return
  }
})
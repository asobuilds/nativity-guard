/* Nativity Guard service worker — v2
 *
 * Cache-first for immutable hashed assets (JS, CSS) so repeat visits are
 * instant. Network-first for API GETs with cache fallback so reads work
 * offline. Never caches POST/PUT/DELETE — those go through the offline
 * write queue in @/lib/offlineQueue.ts.
 */
const CACHE = 'ng-shell-v2'
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
      Promise.all(
        keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)),
      ),
    ),
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const req = event.request
  const url = new URL(req.url)

  // Only handle GET. POST/PUT/DELETE go through the offline queue.
  if (req.method !== 'GET') return

  // Same-origin assets only (plus API).
  const sameOrigin = url.origin === self.location.origin
  const isApi = sameOrigin && url.pathname.startsWith('/api/')
  const isHashedAsset = sameOrigin && url.pathname.startsWith('/assets/')
  const isHtml = sameOrigin && (url.pathname === '/' || url.pathname.endsWith('.html'))

  // 1. Hashed JS/CSS — cache-first forever. Content hashes mean a filename
  //    change signals new content, so a cache hit is always safe.
  if (isHashedAsset) {
    event.respondWith(
      caches.match(req).then((cached) => {
        if (cached) return cached
        return fetch(req).then((resp) => {
          if (resp.ok) {
            const copy = resp.clone()
            caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => undefined)
          }
          return resp
        })
      }),
    )
    return
  }

  // 2. HTML navigation — network-first, fall back to cached index.html.
  if (req.mode === 'navigate' || isHtml) {
    event.respondWith(
      fetch(req)
        .then((resp) => {
          const copy = resp.clone()
          caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => undefined)
          return resp
        })
        .catch(() =>
          caches.match('/index.html').then((r) => r || Response.error()),
        ),
    )
    return
  }

  // 3. API GETs — network-first, fall back to last cached response.
  if (isApi) {
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

  // 4. Everything else same-origin — network with cache fallback.
  if (sameOrigin) {
    event.respondWith(
      caches.match(req).then((cached) => cached || fetch(req)),
    )
  }
})
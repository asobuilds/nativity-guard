self.addEventListener('push', function (event) {
  if (!event.data) return
  const data = event.data.json()
  const title = data.title || 'Nativity Guard'
  const options = {
    body: data.message || 'You have a new notification',
    icon: '/favicon.ico',
    badge: '/favicon.ico',
    data: { url: data.url || '/' },
  }
  event.waitUntil(self.registration.showNotification(title, options))
})

self.addEventListener('notificationclick', function (event) {
  event.notification.close()
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (clientList) {
      const url = event.notification.data?.url || '/'
      for (let i = 0; i < clientList.length; i++) {
        const client = clientList[i]
        if (client.url === url && 'focus' in client) {
          return client.focus()
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(url)
      }
    })
  )
})

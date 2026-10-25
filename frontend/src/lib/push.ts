import { api } from '@/lib/apiClient'

function urlBase64ToUint8Array(base64String: string) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  const outputArray = new Uint8Array(rawData.length)
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i)
  }
  return outputArray
}

export async function registerPushNotifications(vapidPublicKey?: string) {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return false
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js')
    await navigator.serviceWorker.ready

    const permission = await Notification.requestPermission()
    if (permission !== 'granted') {
      return false
    }

    const publicVapidKey = vapidPublicKey || 'BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U'
    const convertedKey = urlBase64ToUint8Array(publicVapidKey)

    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: convertedKey,
    })

    const jsonSub = subscription.toJSON()
    await api.post('/notifications/subscribe', {
      endpoint: jsonSub.endpoint,
      keys: {
        p256dh: jsonSub.keys?.p256dh,
        auth: jsonSub.keys?.auth,
      },
    })

    return true
  } catch (err) {
    console.error('Failed to subscribe to push notifications:', err)
    return false
  }
}

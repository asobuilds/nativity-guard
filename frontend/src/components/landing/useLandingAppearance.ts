import { useEffect, useState } from 'react'
import {
  readLandingAppearance,
  saveLandingAppearance,
  type LandingAppearance,
} from './landingAppearance'

const DATABASE_NAME = 'nativity-guard-local-wallpapers'
const STORE_NAME = 'images'
const WALLPAPER_ID = 'visitor-wallpaper'
const MAX_UPLOAD_BYTES = 8 * 1024 * 1024
const MAX_IMAGE_SIDE = 1920
const ALLOWED_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])

function openWallpaperDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (!('indexedDB' in window)) {
      reject(new Error('This browser does not support saved wallpapers.'))
      return
    }
    const request = window.indexedDB.open(DATABASE_NAME, 1)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_NAME)) db.createObjectStore(STORE_NAME)
    }
    request.onerror = () => reject(new Error('Local image storage is unavailable.'))
    request.onsuccess = () => resolve(request.result)
    request.onblocked = () => reject(new Error('Local image storage is busy. Please retry.'))
  })
}

async function getSavedWallpaper(): Promise<Blob | null> {
  const db = await openWallpaperDatabase()
  try {
    return await new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, 'readonly')
      const request = tx.objectStore(STORE_NAME).get(WALLPAPER_ID)
      request.onsuccess = () => resolve(request.result instanceof Blob ? request.result : null)
      request.onerror = () => reject(new Error('Could not load your saved wallpaper.'))
    })
  } finally {
    db.close()
  }
}

async function writeSavedWallpaper(blob: Blob | null): Promise<void> {
  const db = await openWallpaperDatabase()
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, 'readwrite')
      if (blob) tx.objectStore(STORE_NAME).put(blob, WALLPAPER_ID)
      else tx.objectStore(STORE_NAME).delete(WALLPAPER_ID)
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(new Error('Could not save the wallpaper on this device.'))
      tx.onabort = () => reject(new Error('Wallpaper storage was interrupted.'))
    })
  } finally {
    db.close()
  }
}

export function validateWallpaperFile(file: Pick<File, 'type' | 'size'>): string | null {
  if (!ALLOWED_TYPES.has(file.type)) return 'Choose a JPG, PNG, or WebP picture.'
  if (file.size <= 0 || file.size > MAX_UPLOAD_BYTES) return 'Choose an image smaller than 8 MB.'
  return null
}

async function optimizeWallpaper(file: File): Promise<Blob> {
  const inputUrl = URL.createObjectURL(file)
  try {
    const image = new Image()
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve()
      image.onerror = () => reject(new Error('This picture could not be opened. Try another image.'))
      image.src = inputUrl
    })
    if (!image.naturalWidth || !image.naturalHeight) {
      throw new Error('This picture has no usable dimensions.')
    }

    const scale = Math.min(1, MAX_IMAGE_SIDE / Math.max(image.naturalWidth, image.naturalHeight))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale))
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Image processing is unavailable on this device.')
    context.fillStyle = '#0b2822'
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob(
        blob => blob ? resolve(blob) : reject(new Error('Could not prepare this picture.')),
        'image/jpeg',
        0.82,
      )
    })
  } finally {
    URL.revokeObjectURL(inputUrl)
  }
}

/**
 * Uploaded pictures remain in the current browser's IndexedDB. They are never
 * sent to Nativity Guard servers or placed in the application's public assets.
 */
export function useLandingAppearance() {
  const [appearance, setAppearance] = useState(readLandingAppearance)
  const [wallpaperUrl, setWallpaperUrl] = useState<string | null>(null)
  const [loadingWallpaper, setLoadingWallpaper] = useState(true)
  const [busy, setBusy] = useState(false)
  const [wallpaperError, setWallpaperError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    getSavedWallpaper()
      .then(blob => {
        if (!active) return
        if (blob) setWallpaperUrl(URL.createObjectURL(blob))
        else {
          setAppearance(previous => {
            if (previous.wallpaper !== 'custom') return previous
            return saveLandingAppearance({ ...previous, wallpaper: 'gate' })
          })
        }
      })
      .catch(() => {
        if (!active) return
        setAppearance(previous =>
          previous.wallpaper === 'custom'
            ? saveLandingAppearance({ ...previous, wallpaper: 'gate' })
            : previous,
        )
      })
      .finally(() => { if (active) setLoadingWallpaper(false) })
    return () => { active = false }
  }, [])

  useEffect(() => {
    return () => { if (wallpaperUrl) URL.revokeObjectURL(wallpaperUrl) }
  }, [wallpaperUrl])

  function updateAppearance(next: LandingAppearance) {
    setWallpaperError(null)
    if (next.wallpaper === 'custom' && !wallpaperUrl) return
    setAppearance(saveLandingAppearance(next))
  }

  async function uploadWallpaper(file: File) {
    setWallpaperError(null)
    const validationError = validateWallpaperFile(file)
    if (validationError) { setWallpaperError(validationError); return }
    setBusy(true)
    try {
      const optimized = await optimizeWallpaper(file)
      await writeSavedWallpaper(optimized)
      setWallpaperUrl(URL.createObjectURL(optimized))
      setAppearance(previous => saveLandingAppearance({ ...previous, wallpaper: 'custom' }))
    } catch (error) {
      setWallpaperError(error instanceof Error ? error.message : 'Could not save this picture.')
    } finally {
      setBusy(false)
    }
  }

  async function removeWallpaper() {
    setBusy(true)
    setWallpaperError(null)
    try {
      await writeSavedWallpaper(null)
      setWallpaperUrl(null)
      setAppearance(previous => saveLandingAppearance({ ...previous, wallpaper: 'gate' }))
    } catch (error) {
      setWallpaperError(error instanceof Error ? error.message : 'Could not remove the saved picture.')
    } finally {
      setBusy(false)
    }
  }

  return {
    appearance,
    updateAppearance,
    uploadWallpaper,
    removeWallpaper,
    wallpaperUrl,
    wallpaperError,
    loadingWallpaper,
    busy,
  }
}

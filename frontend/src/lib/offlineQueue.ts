/**
 * IndexedDB-backed offline write queue.
 *
 * When a POST fails because the network is down, we save it here and
 * return a synthetic "queued" response. On reconnect, the queue drains
 * sequentially. Survives browser close, page reload, and phone restart.
 *
 * Only append-only writes (SOS, reports) should go through this — anything
 * that could conflict with a server-side check should fail loudly instead.
 */

const DB_NAME = 'ng-offline'
const DB_VERSION = 1
const STORE = 'queue'

export interface QueuedWrite {
  id: string
  path: string
  method: 'POST'
  body: unknown
  createdAt: number
  attempts: number
  lastError?: string
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE, { keyPath: 'id' })
      }
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

function tx<T>(mode: IDBTransactionMode, fn: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return openDB().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const t = db.transaction(STORE, mode)
        const store = t.objectStore(STORE)
        const req = fn(store)
        req.onsuccess = () => resolve(req.result)
        req.onerror = () => reject(req.error)
      }),
  )
}

export async function enqueue(path: string, body: unknown): Promise<QueuedWrite> {
  const entry: QueuedWrite = {
    id: crypto.randomUUID(),
    path,
    method: 'POST',
    body,
    createdAt: Date.now(),
    attempts: 0,
  }
  await tx('readwrite', (s) => s.add(entry))
  notifyChange()
  return entry
}

export function listQueued(): Promise<QueuedWrite[]> {
  return tx<QueuedWrite[]>('readonly', (s) => s.getAll() as IDBRequest<QueuedWrite[]>)
}

export function removeQueued(id: string): Promise<void> {
  return tx<void>('readwrite', (s) => s.delete(id) as unknown as IDBRequest<void>).then(() => {
    notifyChange()
  })
}

export async function updateQueued(entry: QueuedWrite): Promise<void> {
  await tx('readwrite', (s) => s.put(entry))
  notifyChange()
}

export async function countQueued(): Promise<number> {
  return tx<number>('readonly', (s) => s.count())
}

/** Broadcast so React hooks can re-read the queue without polling. */
function notifyChange() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('ng:queue-changed'))
  }
}

export const QUEUE_CHANGED_EVENT = 'ng:queue-changed'
export interface ClipboardWriter {
  writeText(value: string): Promise<void>
}

/**
 * Clipboard writes can be blocked by browser permissions, insecure contexts,
 * private-mode restrictions or a missing Clipboard API. Never imply success
 * until the browser resolves the writeText promise.
 */
export async function copyText(
  value: string,
  writer?: ClipboardWriter | null,
): Promise<boolean> {
  const clipboard = writer === undefined
    ? (typeof navigator === 'undefined' ? null : navigator.clipboard)
    : writer

  if (!clipboard || typeof clipboard.writeText !== 'function') return false

  try {
    await clipboard.writeText(value)
    return true
  } catch {
    return false
  }
}

/*
 * `AppShell.tsx` writes class names like `safe-top-inset` into the markup and
 * `index.css` supplies the rules. Nothing links the two at build time: Tailwind
 * emits no CSS for a class it does not recognise, so a typo, a rename, or a
 * deleted rule silently removes the iPhone home-indicator and Android gesture-bar
 * clearance and drops the bottom navigation on top of the last row of content —
 * with no error anywhere. These tests are that missing link, and they also pin
 * the drawer geometry that stopped the panel covering 80% of a 360px phone.
 *
 * The two files are read from disk because Vitest hands CSS imports back to the
 * runner as empty modules (`?raw` and `?inline` both come back as `''`), so the
 * stylesheet cannot be read through an import.
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const shell = readFileSync(
  fileURLToPath(new URL('./AppShell.tsx', import.meta.url)),
  'utf8',
)
const css = readFileSync(
  fileURLToPath(new URL('../../index.css', import.meta.url)),
  'utf8',
)

/** The `env(safe-area-inset-*)` rule body for one shell class, or `null`. */
function ruleBody(selector: string): string | null {
  const start = css.indexOf(`.${selector}`)
  if (start === -1) return null
  const open = css.indexOf('{', start)
  if (open === -1) return null
  const close = css.indexOf('}', open)
  if (close === -1) return null
  return css.slice(open + 1, close)
}

const shellTokens = [
  ...new Set<string>(shell.match(/\bsafe-[a-z][a-z-]*\b/g) ?? []),
].sort()

describe('safe-area classes in AppShell.tsx', () => {
  it('references at least the five insets the shell is built from', () => {
    expect(shellTokens).toEqual([
      'safe-bottom-bar',
      'safe-inset-y',
      'safe-main-inset',
      'safe-sos-offset',
      'safe-top-inset',
    ])
  })

  it('defines every referenced class and resolves it against the real insets', () => {
    for (const token of shellTokens) {
      const body = ruleBody(token)
      expect(body, `.${token} is missing from index.css`).not.toBeNull()
      expect(body, `.${token} ignores env(safe-area-inset-*)`).toContain(
        'env(safe-area-inset',
      )
    }
  })

  it('reserves room for the fixed bottom bar instead of letting it overlap', () => {
    // The bar is ~62px tall and the SOS button floats above it; the content
    // padding has to clear both, which is why the value is baked into the class
    // rather than left to a Tailwind `pb-*` utility that would out-rank it.
    expect(ruleBody('safe-main-inset')).toContain('5rem')
    expect(ruleBody('safe-sos-offset')).toContain('4rem')
  })

  it('drops the mobile offsets from `lg` up, where the bar does not exist', () => {
    const lgBlock = css.slice(css.indexOf('@media (min-width: 64rem)'))
    expect(lgBlock).toContain('.safe-main-inset')
    expect(lgBlock).toContain('.safe-sos-offset')
  })
})

describe('the drawer stops short of the viewport edge', () => {
  it('caps the width and keeps a gutter rather than filling the screen', () => {
    expect(shell).toContain('w-[17rem]')
    expect(shell).toContain('max-w-[calc(100vw-3.5rem)]')
  })

  it('no longer reserves 80% of a phone screen for the panel', () => {
    expect(shell).not.toContain('max-w-[80vw]')
  })

  it('keeps the drawer, docked sidebar and bottom navigation on the same `lg` breakpoint', () => {
    const aside = shell.slice(shell.indexOf('<aside'))
    expect(aside).toContain('lg:static')
    expect(shell).toContain('lg:hidden')
  })
})

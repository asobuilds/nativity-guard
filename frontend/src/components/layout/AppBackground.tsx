import { Logo } from '@/components/brand/Logo'

/**
 * The atmospheric layers that sit behind every authenticated page.
 *
 * Layer 1 (Nigeria flag) and Layer 2 (radial glow) are pure CSS,
 * attached to body::before / body::after — see index.css. This
 * component only renders the shield watermark, which needs the SVG
 * mark and so cannot live in a pseudo-element.
 *
 * aria-hidden + pointer-events-none: it never intercepts input and
 * screen readers ignore it.
 */
export function AppBackground() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 overflow-hidden"
      style={{ zIndex: -1 }}
    >
      <div className="ng-shield-watermark absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2">
        <Logo size={560} variant="icon" theme="dark" />
      </div>
    </div>
  )
}
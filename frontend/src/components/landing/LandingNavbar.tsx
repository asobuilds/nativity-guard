import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { Menu, Palette, Search, ShieldCheck, X } from 'lucide-react'
import type { LandingAppearance, LandingPalette, LandingWallpaper } from './landingAppearance'

export type LandingPanel =
  | 'map'
  | 'track'
  | 'units'
  | 'sos-simulator'
  | 'community-alerts'
  | 'ai-safety'
  | 'governance'
  | 'transparency'

export const LANDING_FEATURES: { panel: LandingPanel; title: string; description: string }[] = [
  { panel: 'map', title: 'Safety Map', description: 'Explore public incidents and participating patrol units.' },
  { panel: 'track', title: 'Track a Case', description: 'Look up available case updates with a reference number.' },
  { panel: 'units', title: 'Security Units', description: 'Discover community patrol units and their public details.' },
  { panel: 'sos-simulator', title: 'Emergency SOS Guidance', description: 'Understand emergency support and the SOS demonstration.' },
  { panel: 'community-alerts', title: 'Community Alerts', description: 'Read notices and community safety information.' },
  { panel: 'ai-safety', title: 'AI Safety Advisor', description: 'Explore educational safety guidance and its limitations.' },
  { panel: 'governance', title: 'Governance', description: 'Learn how cases and community units are overseen.' },
  { panel: 'transparency', title: 'Financial Transparency', description: 'Explore funding transparency and supporter records.' },
]

interface LandingNavbarProps {
  onOpen: (panel: LandingPanel) => void
  onReport: () => void
  appearance?: LandingAppearance
  onAppearanceChange?: (next: LandingAppearance) => void
}

const PALETTES: { value: LandingPalette; label: string; color: string }[] = [
  { value: 'forest', label: 'Forest', color: '#06302b' },
  { value: 'midnight', label: 'Midnight', color: '#101820' },
  { value: 'ocean', label: 'Ocean', color: '#133f58' },
  { value: 'sand', label: 'Warm bronze', color: '#54402e' },
]
const WALLPAPERS: { value: LandingWallpaper; label: string }[] = [
  { value: 'gate', label: 'City Gate' },
  { value: 'illustration', label: 'Illustrated skyline' },
  { value: 'plain', label: 'Plain colour' },
]

function AppearanceChoices({ appearance, onChange }: { appearance: LandingAppearance; onChange: (next: LandingAppearance) => void }) {
  return (
    <div className="ng-ref-appearance-choices">
      <p className="ng-ref-option-label">Background colour</p>
      <div className="ng-ref-palette-options">
        {PALETTES.map(item => (
          <button
            key={item.value}
            type="button"
            aria-pressed={appearance.palette === item.value}
            onClick={() => onChange({ ...appearance, palette: item.value })}
            className="ng-ref-palette-choice"
          >
            <span className="ng-ref-palette-swatch" style={{ background: item.color }} aria-hidden />
            {item.label}
          </button>
        ))}
      </div>
      <p className="ng-ref-option-label">Wallpaper</p>
      <div className="ng-ref-wallpaper-options">
        {WALLPAPERS.map(item => (
          <button
            key={item.value}
            type="button"
            aria-pressed={appearance.wallpaper === item.value}
            onClick={() => onChange({ ...appearance, wallpaper: item.value })}
          >
            {item.label}
          </button>
        ))}
      </div>
    </div>
  )
}

export function LandingNavbar({ onOpen, onReport, appearance, onAppearanceChange }: LandingNavbarProps) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [optionsOpen, setOptionsOpen] = useState(false)
  const menuButtonRef = useRef<HTMLButtonElement>(null)
  const drawerCloseRef = useRef<HTMLButtonElement>(null)
  const optionsRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!mobileOpen) return
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    drawerCloseRef.current?.focus()
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setMobileOpen(false)
      if (event.key !== 'Tab') return
      const drawer = drawerCloseRef.current?.closest('[role="dialog"]')
      if (!drawer) return
      const focusable = Array.from(drawer.querySelectorAll<HTMLElement>('a[href],button:not([disabled])'))
      if (!focusable.length) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = previous
      menuButtonRef.current?.focus()
    }
  }, [mobileOpen])

  useEffect(() => {
    if (!optionsOpen) return
    function closeOnOutside(event: PointerEvent) {
      if (!optionsRef.current?.contains(event.target as Node)) setOptionsOpen(false)
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setOptionsOpen(false)
    }
    document.addEventListener('pointerdown', closeOnOutside)
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.removeEventListener('pointerdown', closeOnOutside)
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [optionsOpen])

  function open(panel: LandingPanel) {
    setMobileOpen(false)
    onOpen(panel)
  }

  function report() {
    setMobileOpen(false)
    onReport()
  }

  const links = (
    <>
      <Link to="/" onClick={() => setMobileOpen(false)}>Home</Link>
      <button type="button" onClick={() => open('map')}>Safety Map</button>
      <button type="button" onClick={report}>Report</button>
      <button type="button" onClick={() => open('track')}>Track Case</button>
      <Link to="/explore" onClick={() => setMobileOpen(false)}>Explore Features</Link>
      <Link to="/about" onClick={() => setMobileOpen(false)}>About</Link>
    </>
  )

  return (
    <header className="ng-ref-header">
      <div className="ng-ref-header-inner">
        <Link className="ng-ref-brand" to="/" aria-label="Nativity Guard home">
          <span className="ng-ref-brand-mark"><ShieldCheck size={26} strokeWidth={2.1} aria-hidden /></span>
          <span className="ng-ref-brand-name">NATIVITY<br />GUARD</span>
        </Link>

        <nav className="ng-ref-desktop-nav" aria-label="Main navigation">{links}</nav>

        <div className="ng-ref-header-actions">
          <button type="button" className="ng-ref-search" onClick={() => open('track')} aria-label="Search case reports">
            <Search size={23} strokeWidth={2} aria-hidden />
          </button>
          {appearance && onAppearanceChange && (
            <div className="ng-ref-appearance-holder" ref={optionsRef}>
              <button
                type="button"
                className="ng-ref-options-button"
                aria-label="Change landing page colours and wallpaper"
                aria-expanded={optionsOpen}
                onClick={() => setOptionsOpen(v => !v)}
              >
                <Palette size={17} aria-hidden /><span>Options</span>
              </button>
              {optionsOpen && (
                <div className="ng-ref-appearance-popover">
                  <div className="ng-ref-appearance-title">Appearance</div>
                  <AppearanceChoices appearance={appearance} onChange={onAppearanceChange} />
                </div>
              )}
            </div>
          )}
          <Link className="ng-ref-signin" to="/auth/login">Sign In</Link>
          <Link className="ng-ref-getstarted" to="/auth/signup">Get Started</Link>
        </div>

        <button ref={menuButtonRef} type="button" className="ng-ref-menu-trigger" onClick={() => setMobileOpen(true)} aria-label="Open navigation menu" aria-expanded={mobileOpen}>
          <Menu size={25} aria-hidden />
        </button>
      </div>

      {mobileOpen && createPortal(
        <div className="ng-ref-mobile-backdrop">
          <button type="button" className="ng-ref-mobile-dismiss" onClick={() => setMobileOpen(false)} aria-label="Close navigation menu backdrop" />
          <div className="ng-ref-mobile-drawer" role="dialog" aria-modal="true" aria-label="Navigation menu">
            <div className="ng-ref-mobile-heading">
              <span>NATIVITY GUARD</span>
              <button ref={drawerCloseRef} type="button" onClick={() => setMobileOpen(false)} aria-label="Close navigation menu"><X size={23} aria-hidden /></button>
            </div>
            <nav className="ng-ref-mobile-links" aria-label="Mobile navigation">{links}</nav>
            {appearance && onAppearanceChange && (
              <details className="ng-ref-mobile-appearance">
                <summary><Palette size={17} aria-hidden /> Appearance options</summary>
                <AppearanceChoices appearance={appearance} onChange={onAppearanceChange} />
              </details>
            )}
            <div className="ng-ref-mobile-auth">
              <Link to="/auth/login" onClick={() => setMobileOpen(false)}>Sign In</Link>
              <Link to="/auth/signup" onClick={() => setMobileOpen(false)}>Get Started</Link>
            </div>
          </div>
        </div>, document.body,
      )}
    </header>
  )
}

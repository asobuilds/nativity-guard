import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { Menu, Search, ShieldCheck, X } from 'lucide-react'

export type LandingPanel =
  | 'map'
  | 'track'
  | 'units'
  | 'sos-simulator'
  | 'community-alerts'
  | 'ai-safety'
  | 'governance'
  | 'transparency'

interface LandingNavbarProps {
  onOpen: (panel: LandingPanel) => void
  onReport: () => void
}

export function LandingNavbar({ onOpen, onReport }: LandingNavbarProps) {
  const [mobileOpen, setMobileOpen] = useState(false)

  useEffect(() => {
    if (!mobileOpen) return
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMobileOpen(false)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = previous
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [mobileOpen])

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
      <a href="#home" onClick={() => setMobileOpen(false)}>Home</a>
      <button type="button" onClick={() => open('map')}>Safety Map</button>
      <button type="button" onClick={report}>Report</button>
      <button type="button" onClick={() => open('track')}>Track Case</button>
      <button type="button" onClick={() => open('units')}>Security Units</button>
      <Link to="/about" onClick={() => setMobileOpen(false)}>About</Link>
    </>
  )

  return (
    <header className="ng-ref-header">
      <div className="ng-ref-header-inner">
        <a className="ng-ref-brand" href="#home" aria-label="Nativity Guard home">
          <span className="ng-ref-brand-mark"><ShieldCheck size={26} strokeWidth={2.1} aria-hidden /></span>
          <span className="ng-ref-brand-name">NATIVITY<br />GUARD</span>
        </a>

        <nav className="ng-ref-desktop-nav" aria-label="Main navigation">{links}</nav>

        <div className="ng-ref-header-actions">
          <button type="button" className="ng-ref-search" onClick={() => open('track')} aria-label="Search case reports">
            <Search size={23} strokeWidth={2} aria-hidden />
          </button>
          <Link className="ng-ref-signin" to="/auth/login">Sign In</Link>
          <Link className="ng-ref-getstarted" to="/auth/signup">Get Started</Link>
        </div>

        <button type="button" className="ng-ref-menu-trigger" onClick={() => setMobileOpen(true)} aria-label="Open navigation menu" aria-expanded={mobileOpen}>
          <Menu size={25} aria-hidden />
        </button>
      </div>

      {mobileOpen && createPortal(
        <div className="ng-ref-mobile-backdrop">
          <button type="button" className="ng-ref-mobile-dismiss" onClick={() => setMobileOpen(false)} aria-label="Close navigation menu backdrop" />
          <div className="ng-ref-mobile-drawer" role="dialog" aria-modal="true" aria-label="Navigation menu">
            <div className="ng-ref-mobile-heading">
              <span>NATIVITY GUARD</span>
              <button type="button" onClick={() => setMobileOpen(false)} aria-label="Close navigation menu"><X size={23} aria-hidden /></button>
            </div>
            <nav className="ng-ref-mobile-links" aria-label="Mobile navigation">{links}</nav>
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

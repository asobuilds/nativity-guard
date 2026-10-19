import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import {
  Menu,
  Palette,
  Shield,
  X,
  Check,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { usePreferences } from '@/hooks/usePreferences'
import type { Theme } from '@/lib/preferences'
import { cn } from '@/lib/cn'

const THEMES: { value: Theme; name: string; subtitle: string; preview: [string, string, string] }[] = [
  { value: 'forest', name: 'Dawn Canopy', subtitle: 'Forest Green & Gold', preview: ['#10221d', '#f4cb78', '#f8f5e9'] },
  { value: 'midnight', name: 'Lagos Highlife', subtitle: 'Midnight Onyx & Amber', preview: ['#0a0a0a', '#f4cb78', '#f5f5f5'] },
  { value: 'daylight', name: 'Savanna Stone', subtitle: 'Sunlit Ochre & Sand', preview: ['#f7f7f5', '#b8851a', '#1a1a1a'] },
  { value: 'ocean', name: 'Niger Delta Dusk', subtitle: 'Maritime Teal & Coral', preview: ['#0a1a30', '#8bd3f7', '#eaf2fb'] },
  { value: 'stone', name: 'Benin Bronze', subtitle: 'Earth Bronze & Clay', preview: ['#232019', '#e8b478', '#f4ede0'] },
]

const NAV_LINKS = [
  { href: '#map', label: 'Safety Map' },
  { href: '#case-lifecycle', label: 'Case Tracker' },
  { href: '#units', label: 'Security Units' },
  { href: '#sos-simulator', label: 'SOS Protocol' },
  { href: '#community-alerts', label: 'Alerts' },
  { href: '#ai-safety', label: 'AI Advisor' },
  { href: '#governance', label: 'Governance' },
  { href: '#transparency', label: 'Ledger' },
]

export function LandingNavbar() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [themeDropdownOpen, setThemeDropdownOpen] = useState(false)
  const { preferences, update } = usePreferences()
  const themeMenuRef = useRef<HTMLDivElement>(null)

  // Close mobile drawer and theme menu on Escape
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setMobileOpen(false)
        setThemeDropdownOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  // Close theme dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (themeMenuRef.current && !themeMenuRef.current.contains(e.target as Node)) {
        setThemeDropdownOpen(false)
      }
    }
    if (themeDropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [themeDropdownOpen])

  // Prevent background scroll when mobile drawer is open
  useEffect(() => {
    if (mobileOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => {
      document.body.style.overflow = ''
    }
  }, [mobileOpen])

  function handleThemeChange(t: Theme) {
    update({ theme: t })
    setThemeDropdownOpen(false)
  }

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border/80 bg-base/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Brand identity */}
        <a href="#" className="flex items-center gap-2.5 group">
          <div className="grid size-9 place-items-center rounded-lg bg-surface-hi text-signal ring-1 ring-border group-hover:ring-signal transition-all">
            <Shield className="size-5" aria-hidden />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-black tracking-wider text-ink uppercase">
                Nativity Guard
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-ok/10 px-1.5 py-0.2 text-[9px] font-semibold text-ok ring-1 ring-ok/20">
                <span className="size-1.5 rounded-full bg-ok animate-pulse" />
                Live
              </span>
            </div>
            <p className="text-[10px] text-ink-faint leading-none">
              Sovereign Safety Infrastructure
            </p>
          </div>
        </a>

        {/* Desktop Navigation Links */}
        <nav className="hidden lg:flex items-center gap-1 xl:gap-2" aria-label="Main Navigation">
          {NAV_LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="rounded-md px-2.5 py-1.5 text-xs font-medium text-ink-muted hover:text-ink hover:bg-surface-hi transition-colors"
            >
              {link.label}
            </a>
          ))}
        </nav>

        {/* Action Controls & Theme Picker */}
        <div className="hidden lg:flex items-center gap-3">
          {/* Theme Switcher Popover */}
          <div className="relative" ref={themeMenuRef}>
            <button
              type="button"
              onClick={() => setThemeDropdownOpen((prev) => !prev)}
              className="flex items-center gap-1.5 rounded-md border border-border bg-surface-hi/50 px-2.5 py-1.5 text-xs font-medium text-ink hover:bg-surface-hi transition-all"
              aria-label="Switch theme palette"
              aria-expanded={themeDropdownOpen}
            >
              <Palette className="size-3.5 text-signal" />
              <span className="text-[11px] capitalize">{preferences.theme}</span>
            </button>

            {themeDropdownOpen && (
              <div className="absolute right-0 mt-2 w-64 rounded-panel border border-border bg-surface p-2 shadow-panel z-50">
                <p className="px-2 py-1 text-[11px] font-semibold uppercase tracking-wider text-ink-faint">
                  Select Theme Palette
                </p>
                <div className="mt-1 space-y-1">
                  {THEMES.map((t) => {
                    const isSelected = preferences.theme === t.value
                    return (
                      <button
                        key={t.value}
                        type="button"
                        onClick={() => handleThemeChange(t.value)}
                        className={cn(
                          'flex w-full items-center justify-between rounded-md p-2 text-left text-xs transition-colors',
                          isSelected
                            ? 'bg-signal/10 text-signal font-semibold'
                            : 'hover:bg-surface-hi text-ink',
                        )}
                      >
                        <div className="flex items-center gap-2.5">
                          <span className="flex overflow-hidden rounded ring-1 ring-border">
                            {t.preview.map((c, i) => (
                              <span
                                key={i}
                                className="block size-3.5"
                                style={{ backgroundColor: c }}
                              />
                            ))}
                          </span>
                          <div>
                            <p className="leading-tight">{t.name}</p>
                            <p className="text-[10px] text-ink-faint">{t.subtitle}</p>
                          </div>
                        </div>
                        {isSelected && <Check className="size-3.5 text-signal" />}
                      </button>
                    )
                  })}
                </div>
              </div>
            )}
          </div>

          <Link
            to="/auth/login"
            className="rounded-md px-3 py-1.5 text-xs font-medium text-ink-muted hover:text-ink transition-colors"
          >
            Sign In
          </Link>

          <Link to="/auth/signup">
            <Button variant="primary" size="sm" className="text-xs">
              Get Started
            </Button>
          </Link>
        </div>

        {/* Mobile Hamburger Toggle */}
        <div className="flex items-center gap-2 lg:hidden">
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="grid size-9 place-items-center rounded-lg border border-border bg-surface-hi text-ink"
            aria-label="Open navigation menu"
            aria-expanded={mobileOpen}
          >
            <Menu className="size-5" />
          </button>
        </div>
      </div>

      {/* Mobile Drawer Navigation (PR #63 Regression Guard) */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-base/80 backdrop-blur-sm transition-opacity"
            onClick={() => setMobileOpen(false)}
            aria-hidden="true"
          />

          {/* Drawer Panel */}
          <div className="fixed inset-y-0 right-0 w-full max-w-xs bg-surface border-l border-border p-5 shadow-panel flex flex-col justify-between overflow-y-auto">
            <div>
              <div className="flex items-center justify-between border-b border-border pb-4">
                <div className="flex items-center gap-2">
                  <Shield className="size-5 text-signal" />
                  <span className="text-sm font-bold text-ink">Nativity Guard</span>
                </div>
                <button
                  type="button"
                  onClick={() => setMobileOpen(false)}
                  className="grid size-8 place-items-center rounded-md border border-border bg-surface-hi text-ink hover:text-signal"
                  aria-label="Close navigation menu"
                >
                  <X className="size-4" />
                </button>
              </div>

              {/* Navigation links */}
              <div className="mt-4 space-y-1">
                {NAV_LINKS.map((link) => (
                  <a
                    key={link.href}
                    href={link.href}
                    onClick={() => setMobileOpen(false)}
                    className="block rounded-md px-3 py-2 text-sm font-medium text-ink-muted hover:bg-surface-hi hover:text-ink transition-colors"
                  >
                    {link.label}
                  </a>
                ))}
              </div>

              {/* Theme Selection in Mobile Drawer */}
              <div className="mt-6 border-t border-border pt-4">
                <p className="px-1 text-xs font-semibold uppercase text-ink-faint">
                  Appearance Theme
                </p>
                <div className="mt-2.5 grid grid-cols-1 gap-1.5">
                  {THEMES.map((t) => (
                    <button
                      key={t.value}
                      type="button"
                      onClick={() => handleThemeChange(t.value)}
                      className={cn(
                        'flex items-center justify-between rounded-md p-2 text-xs transition-colors',
                        preferences.theme === t.value
                          ? 'bg-signal/15 text-signal font-semibold'
                          : 'bg-surface-hi/40 text-ink hover:bg-surface-hi',
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <span className="flex overflow-hidden rounded ring-1 ring-border">
                          {t.preview.map((c, i) => (
                            <span
                              key={i}
                              className="block size-3.5"
                              style={{ backgroundColor: c }}
                            />
                          ))}
                        </span>
                        <span>{t.name}</span>
                      </div>
                      {preferences.theme === t.value && <Check className="size-3 text-signal" />}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Mobile Auth Actions */}
            <div className="border-t border-border pt-4 space-y-2">
              <Link
                to="/auth/login"
                onClick={() => setMobileOpen(false)}
                className="block w-full text-center py-2 text-sm font-medium text-ink border border-border rounded-md bg-surface-hi"
              >
                Sign In
              </Link>
              <Link
                to="/auth/signup"
                onClick={() => setMobileOpen(false)}
                className="block w-full"
              >
                <Button variant="primary" className="w-full text-sm">
                  Get Started
                </Button>
              </Link>
            </div>
          </div>
        </div>
      )}
    </header>
  )
}

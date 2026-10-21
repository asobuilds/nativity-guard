import { useState, type CSSProperties } from 'react'
import { ArrowLeft, ArrowRight, BookOpen, ShieldCheck } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ActionAuthModal } from '@/components/landing/ActionAuthModal'
import { LANDING_FEATURES, LandingNavbar, type LandingPanel } from '@/components/landing/LandingNavbar'
import { LandingToolContent } from '@/components/landing/LandingToolContent'
import { useLandingAppearance } from '@/components/landing/useLandingAppearance'
import type { ActionContext } from '@/components/landing/types'
import '@/components/landing/referenceLanding.css'

/** Separate, discoverable pages for the existing public tools; no duplicated business logic. */
export function ExploreFeaturesPage() {
  const { feature } = useParams<{ feature?: string }>()
  const navigate = useNavigate()
  const selected = LANDING_FEATURES.find(item => item.panel === feature)
  const { appearance, updateAppearance, uploadWallpaper, removeWallpaper, wallpaperUrl, wallpaperError, busy } = useLandingAppearance()
  const [activeAction, setActiveAction] = useState<ActionContext | null>(null)

  function report() {
    setActiveAction({
      type: 'report_incident',
      title: 'Report an Incident',
      description: 'Sign in or create an account to submit a report and track progress.',
      returnUrl: '/report',
    })
  }

  function open(panel: LandingPanel) {
    navigate('/explore/' + panel)
  }

  return (
    <div
      className="ng-ref-page ng-ref-features-page"
      data-landing-palette={appearance.palette}
      data-landing-wallpaper={appearance.wallpaper === 'custom' && !wallpaperUrl ? 'gate' : appearance.wallpaper}
      style={{ '--ng-custom-wallpaper': wallpaperUrl ? `url("${wallpaperUrl}")` : 'none' } as CSSProperties}
    >
      <div className="ng-ref-shell">
        <LandingNavbar
          onOpen={open}
          onReport={report}
          appearance={appearance}
          onAppearanceChange={updateAppearance}
          onUploadWallpaper={uploadWallpaper}
          onRemoveWallpaper={removeWallpaper}
          hasCustomWallpaper={Boolean(wallpaperUrl)}
          wallpaperError={wallpaperError}
          wallpaperBusy={busy}
        />
        <main className="ng-ref-features-main">
          <nav className="ng-ref-feature-breadcrumbs" aria-label="Page navigation">
            {selected || feature ? (
              <Link to="/explore" className="ng-ref-feature-back"><ArrowLeft size={18} aria-hidden /> All features</Link>
            ) : null}
            <Link to="/" className="ng-ref-feature-home"><ArrowLeft size={18} aria-hidden /> Back to homepage</Link>
          </nav>
          <div className="ng-ref-features-intro">
            <span className="ng-ref-feature-eyebrow"><ShieldCheck size={16} aria-hidden /> Community safety tools</span>
            <h1>{selected ? selected.title : 'Explore everything Nativity Guard offers'}</h1>
            <p>{selected ? selected.description : 'The homepage stays simple. All the existing community safety features are available here when you need them.'}</p>
          </div>

          {selected ? (
            <section className="ng-ref-feature-workspace" aria-label={selected.title}>
              <LandingToolContent panel={selected.panel} onActionTrigger={setActiveAction} />
            </section>
          ) : feature ? (
            <div className="ng-ref-feature-workspace">
              <h2>Feature not found</h2>
              <p>Choose one of the available tools below.</p>
              <Link to="/explore">View all features</Link>
            </div>
          ) : (
            <div className="ng-ref-feature-grid" aria-label="Available community tools">
              {LANDING_FEATURES.map((item, index) => (
                <Link key={item.panel} to={'/explore/' + item.panel} className="ng-ref-feature-card">
                  <span className="ng-ref-feature-number">{String(index + 1).padStart(2, '0')}</span>
                  <span className="ng-ref-feature-copy"><strong>{item.title}</strong><small>{item.description}</small></span>
                  <ArrowRight size={20} aria-hidden />
                </Link>
              ))}
            </div>
          )}
          <div className="ng-ref-feature-note"><BookOpen size={18} aria-hidden /><p>Public information may be limited to protect personal privacy. Emergency services: dial <a href="tel:112">112</a>. In-app SOS is not a guaranteed dispatch service.</p></div>
        </main>
      </div>
      <ActionAuthModal open={Boolean(activeAction)} action={activeAction} onClose={() => setActiveAction(null)} />
    </div>
  )
}

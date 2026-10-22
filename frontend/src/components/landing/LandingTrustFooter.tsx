import { Link } from 'react-router-dom'
import { ArrowUpRight, BookOpen, ShieldCheck } from 'lucide-react'

/** Legal and accountability links without fabricated government affiliation,
 * certification, service availability, or response-time guarantees.
 */
export function LandingTrustFooter() {
  return (
    <footer className="ng-ref-trust-footer" aria-label="About, legal information and your rights">
      <div className="ng-ref-trust-summary">
        <ShieldCheck size={19} aria-hidden />
        <div>
          <strong>Know the platform before you join</strong>
          <p>Nativity Guard is a community safety coordination platform, not a police force or emergency response service. For urgent help in Nigeria, call <a href="tel:112">112</a>.</p>
        </div>
      </div>
      <nav className="ng-ref-trust-links" aria-label="Legal and trust information">
        <Link to="/about"><BookOpen size={15} aria-hidden /> About &amp; accountability</Link>
        <Link to="/terms">Terms of Service <ArrowUpRight size={14} aria-hidden /></Link>
        <Link to="/terms">Privacy notice &amp; your rights <ArrowUpRight size={14} aria-hidden /></Link>
      </nav>
      <p className="ng-ref-trust-disclaimer">No government endorsement, legal accreditation or guaranteed response is claimed. Questions about access, correction, or deletion of personal data can be raised through the Privacy Notice and support channel. Read the published policies before sharing sensitive information.</p>
    </footer>
  )
}

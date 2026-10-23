package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetBlueprint — PUBLIC. Returns a non-certifying description of platform design:
// its purpose, architecture, roles, capabilities, and how to use it.
// Designed to be rendered as the public /about page.
func GetBlueprint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "Nativity Guard",
		"tagline": "Community safety and governance infrastructure for Nigerian neighborhoods.",
		"purpose": "Nativity Guard provides software for submitting and tracking safety reports where participating units are available. A report or app alert does not guarantee investigation, dispatch, or an emergency response.",

		"principles": []gin.H{
			{"title": "Unit access is NOT case access", "body": "A security unit does not automatically see every case in its jurisdiction — only cases assigned to specific officers, or submitted to specific administrators."},
			{"title": "Governance history", "body": "Unit governance includes configured terms, votes and recall processes. A specific action depends on user eligibility, unit policy and successful processing."},
			{"title": "Presumed innocence", "body": "A citizen named as a suspect sees only that a case references them, its status, and public-safe progress. Never investigative detail, officer identity, or evidence internals."},
			{"title": "No vigilantism", "body": "The platform does not authorize citizen confrontation or replace official emergency services."},
			{"title": "Explicit consent for sensitive data", "body": "Sensitive-data processing and retention should be explained in the published Privacy Notice. Security controls require independent verification."},
		},

		"roles": []gin.H{
			{"name": "Citizen", "capabilities": []string{"Report incidents", "Track your case", "Rate the officer and unit after closure", "See community alerts and announcements", "Donate directly to a unit's bank account", "See if you are named in a case (self-view only)"}},
			{"name": "Officer", "capabilities": []string{"Work cases assigned to you (primary or paired)", "Support access on other cases you assist", "File weekly narratives", "Submit cases for review", "See your unit's officer leaderboard", "View your unit's financial ledger"}},
			{"name": "Unit Admin", "capabilities": []string{"Triage cases in your unit", "Assign cases to officers", "Review closures", "Manage elections and revocations", "Manage bank accounts and visibility", "Approve disbursements"}},
			{"name": "Head Admin", "capabilities": []string{"Everything a unit admin can do", "See all cases in the unit", "Override case archive", "Manage the unit's UnitAuth policy", "Publish unit-scoped news", "Delegate the treasurer role"}},
			{"name": "Super Admin", "capabilities": []string{"Platform-wide access", "Manage all units and users", "Confirm platform donations", "Review available platform financial and case activity records"}},
		},

		"architecture": gin.H{
			"backend": "Go 1.24 · Gin · GORM · PostgreSQL · JWT",
			"frontend": "React 19 · TypeScript · Vite · Tailwind · React Query · Leaflet",
			"infra": "Docker Compose · Render · Supabase",
		},

		"case_lifecycle": []string{
			"pending", "assigned", "dispatched", "on_scene",
			"investigating", "pending_admin_review",
			"admin_changes_requested", "closed",
		},

		"security": []string{
			"Session access controls",
			"Session refresh controls",
			"Rate limiting is designed for sensitive endpoints",
			"Password-change session management",
			"Role-based access controls",
			"Public response views are designed to limit exposed personal information",
			"Protection of sensitive records requires independent configuration review",
			"Webhook integrity controls require service configuration",
		},

		"finance": gin.H{
			"unit_model": "A participating unit may publish account details. Visitors should verify payment recipients independently before transferring funds.",
			"ledger":     "The platform stores financial activity records. The completeness and tamper resistance of a record requires independent audit.",
			"yearly_cycle": "The platform describes annual financial summaries; availability and completeness depend on actual records.",
			"platform_donation": "Platform support information is only displayed when returned by the configured backend service.",
		},

		"governance": gin.H{
			"elections": gin.H{
				"term_months":           12,
				"staggered_rotation":    true,
				"stagger_months":        6,
				"term_limit":            2,
				"cooling_off_months":    6,
				"seat_bands":            gin.H{"12-20": 5, "21-40": 7, "41-70": 9, "71-100": 10},
				"quorum_percent":        50,
			},
			"revocation": gin.H{
				"regular_admin": "10+ verified member votes + head admin approval",
				"head_admin":    "majority of verified unit members",
				"one_vote_per":  "per member per cycle",
			},
		},

		"ratings": gin.H{
			"method": "Bayesian smoothing",
			"tiers":  []string{"unranked", "bronze", "silver", "gold", "platinum", "diamond"},
			"rules": []string{
				"Only the reporter may rate",
				"Only after the case closes via the platform",
				"30-day window after closure",
				"One rating per case per ratee",
				"Officer leaderboards are within-unit only",
				"Officer of the Week is announced via platform news",
			},
		},

		"how_to_use": []gin.H{
			{"step": "1. Register", "detail": "Create a citizen account subject to eligibility and available terms."},
			{"step": "2. Report", "detail": "Submit a report with location and available evidence where the service is accessible."},
			{"step": "3. Track", "detail": "View updates if the receiving unit accepts and updates the case."},
			{"step": "4. Rate", "detail": "After closure, rate the officer and the unit."},
			{"step": "5. Join or host a unit", "detail": "Ask about joining a participating unit; eligibility and governance depend on the unit's active policy."},
		},

		"contact": gin.H{
			"support_email": "support@nativityguard.com",
			"platform_url":  "https://nativityguard.com",
		},
	})
}

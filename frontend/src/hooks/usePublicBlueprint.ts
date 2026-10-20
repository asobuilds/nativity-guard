import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { PlatformBlueprint } from '@/components/landing/types'

export const publicBlueprintKeys = {
  all: ['public-blueprint'] as const,
}

const FALLBACK_BLUEPRINT: PlatformBlueprint = {
  name: 'Nativity Guard',
  tagline: 'Community safety and governance infrastructure for Nigerian neighborhoods.',
  purpose:
    'Nativity Guard connects citizens, security units, officers, and administrators through one auditable workflow: report → assign → dispatch → investigate → review → close.',
  principles: [
    {
      title: 'Unit access is NOT case access',
      body: 'A security unit does not automatically see every case in its jurisdiction — only cases assigned to specific officers, or submitted to specific administrators.',
    },
    {
      title: 'Every hierarchy is auditable',
      body: 'Head admins, admins, and officers are all removable by defined thresholds and quorum. No role is above recall.',
    },
    {
      title: 'Presumed innocence',
      body: 'A citizen named as a suspect sees only that a case references them, its status, and public-safe progress. Never investigative detail, officer identity, or evidence internals.',
    },
    {
      title: 'No vigilantism',
      body: 'The platform coordinates lawful response. It does not arm, direct, or encourage any citizen to confront another.',
    },
    {
      title: 'Explicit consent for sensitive data',
      body: 'Medical information and government identity documents are encrypted at rest with AES-256-GCM.',
    },
  ],
  roles: [
    {
      name: 'Citizen',
      capabilities: [
        'Report incidents with geocoded coordinates',
        'Track case status in real time',
        'Rate officer and unit after closure',
        'See community alerts and announcements',
        'Donate directly to unit bank accounts',
      ],
    },
    {
      name: 'Officer',
      capabilities: [
        'Work assigned cases (primary / paired / support)',
        'File weekly narratives and progress logs',
        'Submit cases for administrative closure',
        'View unit financial ledger',
      ],
    },
    {
      name: 'Unit Admin',
      capabilities: [
        'Triage incident cases in the unit queue',
        'Assign cases to verified officers',
        'Approve or request changes on closures (requires 2 admins)',
        'Manage elections, revocations, and bank details',
      ],
    },
    {
      name: 'Super Admin',
      capabilities: [
        'Platform-level oversight and unit verification',
        'Confirm platform maintenance donations',
        'Audit any financial or case ledger',
      ],
    },
  ],
  architecture: {
    backend: 'Go 1.24 · Gin · GORM · PostgreSQL · JWT',
    frontend: 'React 19 · TypeScript · Vite · Tailwind · React Query · Leaflet',
    infra: 'Docker Compose · Render · Supabase',
  },
  caseLifecycle: [
    'pending',
    'assigned',
    'dispatched',
    'on_scene',
    'investigating',
    'pending_admin_review',
    'admin_changes_requested',
    'closed',
  ],
  security: [
    'JWT with jti-based revocation checked on every request',
    'Refresh token rotation (one-time-use, hash-only storage)',
    'Rate limiting on auth, OTP, votes, and invites',
    'Tiered case access (primary / paired / support)',
    'Content-addressed media store (5MB audio, 20MB video)',
  ],
  governance: {
    elections: {
      term_months: 12,
      staggered_rotation: true,
      stagger_months: 6,
      term_limit: 2,
      cooling_off_months: 6,
      seat_bands: { '12-20': 5, '21-40': 7, '41-70': 9, '71-100': 10 },
      quorum_percent: 50,
    },
    revocation: {
      regular_admin: '10+ verified member votes + head admin approval',
      head_admin: 'majority of verified unit members',
      one_vote_per: 'per member per cycle',
    },
  },
  contact: {
    support_email: 'support@nativityguard.app',
    platform_url: 'https://nativityguard.app',
  },
}

export function usePublicBlueprint() {
  return useQuery({
    queryKey: publicBlueprintKeys.all,
    queryFn: async () => {
      try {
        const res = await api.get<PlatformBlueprint>('/public/blueprint')
        return res && res.name ? res : FALLBACK_BLUEPRINT
      } catch {
        return FALLBACK_BLUEPRINT
      }
    },
    staleTime: 60 * 60_000,
    networkMode: 'offlineFirst',
  })
}


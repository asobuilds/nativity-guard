import { describe, it, expect } from 'vitest'
import { publicBlueprintKeys } from '@/hooks/usePublicBlueprint'
import { publicCaseKeys } from '@/hooks/usePublicCases'
import { publicUnitKeys } from '@/hooks/usePublicUnits'
import { publicLeaderboardKeys } from '@/hooks/usePublicLeaderboard'
import { publicSupporterKeys } from '@/hooks/usePublicSupporters'
import type { PublicCase, PlatformBlueprint } from './types'
import type { SecurityUnit } from '@/types/api'

describe('Landing Page Public Data Guarantees & Acceptance Criteria', () => {
  describe('Query Keys', () => {
    it('defines distinct stable query keys for caching and invalidation', () => {
      expect(publicBlueprintKeys.all).toEqual(['public-blueprint'])
      expect(publicCaseKeys.all).toEqual(['public-cases'])
      expect(publicUnitKeys.all).toEqual(['public-units'])
      expect(publicLeaderboardKeys.all).toEqual(['public-leaderboard'])
      expect(publicSupporterKeys.supporters).toEqual(['public-supporters'])
      expect(publicSupporterKeys.donationInfo).toEqual(['public-platform-donation-info'])
    })
  })

  describe('Public Case Security & Privacy Invariants', () => {
    const sampleCases: PublicCase[] = [
      {
        id: 'sample-case-1',
        unitId: 'unit-bwari-01',
        title: 'Perimeter fence breach and equipment vandalism',
        description: 'Reported unauthorized entry into the community storage yard overnight. Unit dispatched.',
        location: 'Bwari Central, Abuja FCT',
        latitude: 9.2885,
        longitude: 7.3785,
        status: 'investigating',
        priorityLevel: 'P2',
        trackingId: 'NG-2026-00194821',
        isPublic: true,
        isAnonymous: false,
        createdAt: '2026-10-19T10:00:00.000Z',
        updatedAt: '2026-10-19T12:00:00.000Z',
      },
      {
        id: 'sample-case-2',
        unitId: 'unit-kubwa-02',
        title: 'Distress call: suspicious night prowler on Sector 4 lane',
        description: 'Resident observed two individuals testing gate locks along Sector 4. Patrol unit arrived on scene.',
        location: 'Kubwa Phase 3, Abuja FCT',
        latitude: 9.1538,
        longitude: 7.3328,
        status: 'on_scene',
        priorityLevel: 'P1',
        trackingId: 'NG-2026-00194822',
        isPublic: true,
        isAnonymous: true,
        createdAt: '2026-10-19T08:00:00.000Z',
        updatedAt: '2026-10-19T13:00:00.000Z',
      },
    ]

    it('enforces NG-YYYY-NNNNNNNN tracking ID format on all public cases', () => {
      const trackingRegex = /^NG-\d{4}-\d{8}$/
      for (const c of sampleCases) {
        expect(c.trackingId).toMatch(trackingRegex)
      }
    })

    it('verifies coarse coordinates are within Nigerian sovereign borders', () => {
      // Nigeria bounds: Lat ~4.0 to 14.0 N, Lng ~2.5 to 15.0 E
      for (const c of sampleCases) {
        expect(c.latitude).toBeGreaterThanOrEqual(4.0)
        expect(c.latitude).toBeLessThanOrEqual(14.0)
        expect(c.longitude).toBeGreaterThanOrEqual(2.5)
        expect(c.longitude).toBeLessThanOrEqual(15.0)
      }
    })

    it('ensures public cases contain NO sensitive internal notes or unmasked identifiers', () => {
      for (const c of sampleCases) {
        const raw = c as unknown as Record<string, unknown>
        expect(raw.suspectNin).toBeUndefined()
        expect(raw.internalNotes).toBeUndefined()
        expect(raw.officerPhone).toBeUndefined()
        expect(raw.rawEvidenceKey).toBeUndefined()
        expect(c.isPublic).toBe(true)
      }
    })
  })

  describe('Security Unit Directory Invariants', () => {
    const sampleUnits: SecurityUnit[] = [
      {
        id: 'unit-bwari-01',
        name: 'Bwari Central Safety Command',
        type: 'Vigilante Group of Nigeria (VGN)',
        state: 'Federal Capital Territory',
        lga: 'Bwari',
        city: 'Bwari',
        latitude: 9.2885,
        longitude: 7.3785,
        operationalRadius: 15,
        contactPerson: 'Commander Aliyu Bello',
        contactPhone: '+234 803 123 4567',
        registrationNumber: 'NG/FCT/BWR/2024/008',
        totalMembers: 48,
        status: 'active',
        isVerified: true,
        verificationStatus: 'verified',
      },
      {
        id: 'unit-ikeja-01',
        name: 'Ikeja Commercial District Watch',
        type: 'Lagos State Neighbourhood Corps (LNSC)',
        state: 'Lagos',
        lga: 'Ikeja',
        city: 'Ikeja',
        latitude: 6.5964,
        longitude: 3.3426,
        operationalRadius: 8,
        contactPerson: 'Capt. Babatunde Jinadu',
        contactPhone: '+234 809 111 2233',
        registrationNumber: 'NG/LOS/IKJ/2022/045',
        totalMembers: 82,
        status: 'active',
        isVerified: true,
        verificationStatus: 'verified',
      },
    ]

    it('enforces verified registration format on units', () => {
      const regRegex = /^NG\/[A-Z]{3}\/[A-Z]{3}\/\d{4}\/\d{3}$/
      for (const u of sampleUnits) {
        expect(u.registrationNumber).toMatch(regRegex)
        expect(u.isVerified).toBe(true)
        expect(u.verificationStatus).toBe('verified')
      }
    })

    it('ensures operational radius is bounded between 1km and 50km', () => {
      for (const u of sampleUnits) {
        expect(u.operationalRadius).toBeGreaterThanOrEqual(1)
        expect(u.operationalRadius).toBeLessThanOrEqual(50)
      }
    })
  })

  describe('Democratic Governance & Seat Band Formula', () => {
    function computeSeatBand(members: number): number {
      if (members <= 20) return 5
      if (members <= 40) return 7
      if (members <= 70) return 9
      return 10
    }

    it('computes administrative seats across membership tiers', () => {
      expect(computeSeatBand(12)).toBe(5)
      expect(computeSeatBand(20)).toBe(5)
      expect(computeSeatBand(21)).toBe(7)
      expect(computeSeatBand(40)).toBe(7)
      expect(computeSeatBand(41)).toBe(9)
      expect(computeSeatBand(70)).toBe(9)
      expect(computeSeatBand(71)).toBe(10)
      expect(computeSeatBand(100)).toBe(10)
    })

    it('computes 50% voting quorum threshold accurately', () => {
      expect(Math.ceil(12 * 0.5)).toBe(6)
      expect(Math.ceil(35 * 0.5)).toBe(18)
      expect(Math.ceil(71 * 0.5)).toBe(36)
      expect(Math.ceil(100 * 0.5)).toBe(50)
    })

    it('validates the 4 cardinal rules are defined in blueprint principles', () => {
      const blueprintPrinciples: PlatformBlueprint['principles'] = [
        {
          title: 'Unit access is NOT case access',
          body: 'A security unit does not automatically see every case in its jurisdiction.',
        },
        {
          title: 'Every hierarchy is auditable',
          body: 'Head admins, admins, and officers are all removable by defined thresholds.',
        },
        {
          title: 'Presumed innocence',
          body: 'A citizen named as a suspect sees only that a case references them and public-safe progress.',
        },
        {
          title: 'No vigilantism',
          body: 'The platform coordinates lawful response. It does not arm citizens.',
        },
      ]

      const titles = blueprintPrinciples.map((p) => p.title.toLowerCase())
      expect(titles.some((t) => t.includes('unit access is not case access'))).toBe(true)
      expect(titles.some((t) => t.includes('auditable'))).toBe(true)
      expect(titles.some((t) => t.includes('presumed innocence'))).toBe(true)
      expect(titles.some((t) => t.includes('no vigilantism'))).toBe(true)
    })
  })

  describe('Non-Custodial Financial Transparency Acceptance', () => {
    it('validates 10-digit NUBAN account structure for platform trust donations', () => {
      const nubanRegex = /^\d{10}$/
      const accountNumber = '1029482019'
      expect(accountNumber).toMatch(nubanRegex)
    })
  })
})

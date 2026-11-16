export interface AgencyObserverFormValues {
  email: string
  phone: string
  firstName: string
  lastName: string
  password: string
}

/** Client-side feedback only; the API remains authoritative for validation and tenant scope. */
export function validateAgencyObserverForm(values: AgencyObserverFormValues): string | null {
  if (!values.firstName.trim() || !values.lastName.trim()) return 'Enter the observer’s first and last name.'
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email.trim())) return 'Enter a valid email address.'
  if (!values.phone.trim()) return 'Enter a phone number.'
  if (values.password.length < 6) return 'The temporary password must contain at least 6 characters.'
  return null
}


export interface AgencyUnitCandidate {
  id: string
  name: string
  type: string
  state?: string
  lga?: string
  status: string
  isVerified: boolean
}

export interface AgencyUnitScope {
  agencyType: string
  state: string
  lga: string
}

export function getEligibleAgencyUnits(
  units: AgencyUnitCandidate[],
  scope: AgencyUnitScope | undefined,
  linkedUnitIds: string[],
): AgencyUnitCandidate[] {
  if (!scope) return []
  const linked = new Set(linkedUnitIds)
  return units.filter((unit) =>
    unit.isVerified &&
    unit.status.toLowerCase() === 'active' &&
    unit.type.toLowerCase() === scope.agencyType.toLowerCase() &&
    unit.state?.trim().toLowerCase() === scope.state.trim().toLowerCase() &&
    unit.lga?.trim().toLowerCase() === scope.lga.trim().toLowerCase() &&
    !linked.has(unit.id),
  )
}

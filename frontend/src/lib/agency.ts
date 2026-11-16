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

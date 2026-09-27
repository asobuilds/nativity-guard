import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/apiClient'
import { USE_MOCKS } from '@/mocks/config'
import { ErrorState, EmptyState } from '@/components/ui/States'
import type { Case, SecurityUnit, UnitOfficer } from '@/types/api'

type Transaction = { id: string; label: string; amount: number; status: string }
type Audit = { id: string; actor: string; action: string; entity: string; time: string }
type Data = { cases: Case[]; officers: UnitOfficer[]; units: SecurityUnit[]; transactions: Transaction[]; finance: { account: string; donations: number; budget: number }; settings: { incidentTemplate: string; retentionDays: number }; audit?: Audit[] }
const inputClass = 'w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-ink'
const cardClass = 'rounded-xl border border-border bg-surface p-5'
const currency = (n: number) => `₦${n.toLocaleString()}`
const metric = (cases: Case[], start: keyof Case, end: keyof Case) => {
  const samples = cases.map((c) => [Date.parse(String(c[start] ?? '')), Date.parse(String(c[end] ?? ''))]).filter(([a, b]) => Number.isFinite(a) && Number.isFinite(b) && b >= a)
  return samples.length ? `${Math.round(samples.reduce((sum, [a, b]) => sum + (b - a), 0) / samples.length / 60000)} min · ${samples.length} cases` : 'No timed samples'
}
function Field({ label, name, value, onChange, type = 'text' }: { label: string; name: string; value: string | number; onChange: (name: string, value: string) => void; type?: string }) {
  return <label className="block text-sm text-ink-muted">{label}<input className={`${inputClass} mt-1`} type={type} name={name} value={value} onChange={(e) => onChange(name, e.target.value)} /></label>
}
function Analytics({ cases, officers }: { cases: Case[]; officers: UnitOfficer[] }) {
  const active = cases.filter((c) => c.status !== 'closed')
  return <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
    {[
      ['Total reports', cases.length], ['Open', active.length], ['Assigned', cases.filter((c) => Boolean(c.assignedTo)).length],
      ['Dispatched', cases.filter((c) => Boolean(c.dispatchedAt)).length], ['Closed', cases.filter((c) => c.status === 'closed').length],
      ['Report → assignment', metric(cases, 'createdAt', 'assignedAt')], ['Assignment → dispatch', metric(cases, 'assignedAt', 'dispatchedAt')],
      ['Dispatch → arrival', metric(cases, 'dispatchedAt', 'arrivedAt')], ['Report → resolution', metric(cases, 'createdAt', 'closedAt')],
      ['Active cases per officer', officers.length ? (active.length / officers.length).toFixed(1) : 'No officers'],
    ].map(([label, value]) => <div className={cardClass} key={label}><p className="text-xs text-ink-muted">{label}</p><p className="mt-2 text-xl font-semibold text-ink">{value}</p></div>)}
  </div>
}
export function AdminDemoPage({ section, platform = false }: { section: string; platform?: boolean }) {
  const [data, setData] = useState<Data | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [form, setForm] = useState<Record<string, string>>({})
  const [search, setSearch] = useState('')
  const load = useCallback(async () => {
    try { setData(await api.get<Data>('/demo/admin/state')); setError('') } catch (e) { setError(e instanceof Error ? e.message : 'Unable to load') }
  }, [])
  useEffect(() => { if (USE_MOCKS) void load() }, [load])
  const change = (key: string, value: string) => setForm((f) => ({ ...f, [key]: value }))
  const save = async (part: string, body: Record<string, unknown>) => {
    setNotice(''); setError('')
    try { await api.put(`/demo/admin/${part}`, body); await load(); setForm({}); setNotice('Demo change saved for this session.') }
    catch (e) { setError(e instanceof Error ? e.message : 'Save failed') }
  }
  const select = (record: Record<string, unknown>) => setForm(Object.fromEntries(Object.entries(record).filter(([, v]) => typeof v === 'string' || typeof v === 'number').map(([k, v]) => [k, String(v)])))
  if (!USE_MOCKS) return <div className="p-6"><ErrorState title="Demo screen unavailable" description="This demo uses session data. Its admin workflows still need to be connected and verified against the Go API." /></div>
  if (!data) return <div className="p-6">{error ? <ErrorState description={error} onRetry={() => void load()} /> : <p role="status">Loading demo data…</p>}</div>
  const unit = data.units[0]
  const title: Record<string, string> = { overview: platform ? 'Platform overview' : 'Unit operations', officers: 'Officer roster', analytics: platform ? 'Platform analytics and health' : 'Unit analytics', finance: 'Unit finance', settings: platform ? 'System settings and exports' : 'Unit settings', units: 'Unit registry', audit: 'Audit search' }
  const fields = (items: [string, string, string?][]) => items.map(([label, key, type]) => <Field key={key} label={label} name={key} type={type} value={form[key] ?? ''} onChange={change} />)
  const submit = (part: string, extra: Record<string, unknown> = {}) => (e: FormEvent) => { e.preventDefault(); void save(part, { ...form, ...extra }) }
  return <div className="mx-auto max-w-6xl space-y-6 p-5 md:p-8">
    <div><p className="text-xs font-semibold uppercase tracking-widest text-signal">Session demo · no live changes</p><h1 className="mt-2 text-2xl font-bold text-ink">{title[section]}</h1><p className="text-sm text-ink-muted">Sample data resets when the page reloads. Actions do not reach dispatch, accounts, or payments.</p></div>
    {notice && <p role="status" className="rounded-lg bg-signal/10 p-3 text-sm text-ink">{notice}</p>}
    {error && <p role="alert" className="rounded-lg bg-emergency/10 p-3 text-sm text-emergency">{error}</p>}
    {section === 'overview' && <><Analytics cases={data.cases} officers={data.officers} /><div className={cardClass}><h2 className="font-semibold text-ink">Case attention</h2><p className="mt-2 text-sm text-ink-muted">{data.cases.filter((c) => c.status === 'pending').length} pending · {data.cases.filter((c) => c.status === 'assigned').length} assigned · {data.cases.filter((c) => c.status === 'dispatched').length} dispatched</p><Link className="mt-3 inline-block text-sm text-signal underline" to="/admin/cases">Open case board</Link></div></>}
    {section === 'analytics' && <><Analytics cases={data.cases} officers={data.officers} />{platform && <div className={cardClass}><h2 className="font-semibold text-ink">Demo health</h2><p className="text-sm text-ink-muted">Mock API available · {data.units.length} units · {data.officers.length} officers. Live service health is not monitored here.</p></div>}</>}
    {section === 'officers' && <><div className="grid gap-3 md:grid-cols-2">{data.officers.map((o) => <button type="button" key={o.id} onClick={() => select(o as unknown as Record<string, unknown>)} className={`${cardClass} text-left text-ink hover:border-signal`}><strong>{o.name}</strong><p className="text-sm text-ink-muted">{o.badgeNumber} · {o.rank} · {o.status} · {data.cases.filter((c) => c.assignedTo === o.id && c.status !== 'closed').length} active assignments</p></button>)}</div><form onSubmit={submit('officers')} className={`${cardClass} grid gap-3 md:grid-cols-2`}><h2 className="md:col-span-2 font-semibold text-ink">{form.id ? 'Edit officer' : 'Add demo officer'}</h2>{fields([['Name','name'],['Badge number','badgeNumber'],['Rank','rank'],['Duty role','role'],['Phone','phone']])}<label className="text-sm text-ink-muted">Status<select className={`${inputClass} mt-1`} value={form.status ?? 'active'} onChange={(e) => change('status', e.target.value)}><option value="active">Active</option><option value="inactive">Inactive</option></select></label><button className="rounded-lg bg-signal px-4 py-2 font-semibold text-base md:col-span-2">Save demo officer</button></form></>}
    {section === 'finance' && <><div className="grid gap-3 md:grid-cols-3">{[['Account',data.finance.account],['Sample donations',currency(data.finance.donations)],['Budget',currency(data.finance.budget)]].map(([label,value]) => <div className={cardClass} key={label}><p className="text-xs text-ink-muted">{label}</p><strong className="text-ink">{value}</strong></div>)}</div><form className={cardClass} onSubmit={submit('finance')}><Field label="Demo budget (NGN)" name="budget" type="number" value={form.budget ?? data.finance.budget} onChange={change}/><button className="mt-3 rounded-lg bg-signal px-4 py-2 text-base font-semibold">Update budget</button></form><div className={cardClass}><h2 className="font-semibold text-ink">Transactions and approvals</h2>{data.transactions.map((t) => <div key={t.id} className="flex flex-wrap items-center justify-between gap-2 border-t border-border py-3 text-sm text-ink"><span>{t.label} · {currency(t.amount)} · {t.status}</span>{t.status === 'pending' && <span className="flex gap-2">{['approved','rejected'].map((status) => <button key={status} className="rounded border border-border px-3 py-1" onClick={() => void save('transactions',{id:t.id,status})}>{status === 'approved' ? 'Approve' : 'Reject'}</button>)}</span>}</div>)}<button className="text-sm text-signal underline" onClick={() => downloadDemo('unit-finance', data.finance, data.transactions)}>Download sample finance report</button></div></>}
    {section === 'units' && <><div className="grid gap-3 md:grid-cols-2">{data.units.map((u) => <div key={u.id} className={cardClass}><button className="text-left font-semibold text-signal underline" onClick={() => select(u as unknown as Record<string, unknown>)}>{u.name}</button><p className="text-xs text-ink-muted">{u.status} · {u.operationalRadius} km · {u.contactPhone}</p><button className="mt-2 text-xs text-emergency underline" onClick={() => { if (window.confirm(`Delete demo unit ${u.name}?`)) void save('units',{id:u.id,delete:true}) }}>Delete empty unit</button></div>)}</div><UnitForm change={change} form={form} submit={submit('units')} /></>}
    {section === 'settings' && (platform ? <><form onSubmit={(e) => { e.preventDefault(); void save('settings', { ...data.settings, ...form }) }} className={`${cardClass} space-y-3`}><h2 className="font-semibold text-ink">Demo defaults and incident template</h2><Field label="Incident template" name="incidentTemplate" value={form.incidentTemplate ?? data.settings.incidentTemplate} onChange={change} /><Field label="Retention days" name="retentionDays" type="number" value={form.retentionDays ?? data.settings.retentionDays} onChange={change} /><button className="rounded-lg bg-signal px-4 py-2 font-semibold">Save demo settings</button></form><button className="text-sm text-signal underline" onClick={() => downloadDemo('platform-export',data.units,data.cases,data.officers,data.audit ?? [])}>Export sample platform data (JSON)</button></> : <UnitForm change={change} form={{ ...Object.fromEntries(Object.entries(unit).filter(([,v]) => typeof v === 'string' || typeof v === 'number').map(([k,v]) => [k,String(v)])), ...form }} submit={(e) => { e.preventDefault(); void save('unit',{...unit,...form}) }} />)}
    {section === 'audit' && <div className={cardClass}><Field label="Search actor, action, or entity" name="search" value={search} onChange={(_,v) => setSearch(v)} />{(data.audit ?? []).filter((a) => `${a.actor} ${a.action} ${a.entity} ${a.time}`.toLowerCase().includes(search.toLowerCase())).map((a) => <div className="border-b border-border py-3 text-sm text-ink" key={a.id}>{a.actor} · {a.action} · {a.entity}<time className="block text-xs text-ink-muted">{new Date(a.time).toLocaleString()}</time></div>)}{!(data.audit ?? []).length && <EmptyState title="No demo changes yet" description="Saving a unit, officer, finance decision or setting creates a session audit entry." />}</div>}
  </div>
}
function UnitForm({ change, form, submit }: { change: (key: string, value: string) => void; form: Record<string,string>; submit: (e: FormEvent) => void }) {
  return <form onSubmit={submit} className={`${cardClass} grid gap-3 md:grid-cols-2`}><h2 className="md:col-span-2 font-semibold text-ink">{form.id ? 'Edit unit' : 'Add demo unit'}</h2>{(['name','operationalRadius','contactPhone','contactEmail','latitude','longitude','state','city'] as const).map((key) => <Field key={key} label={{name:'Unit name',operationalRadius:'Radius (km)',contactPhone:'Contact phone',contactEmail:'Contact email',latitude:'Latitude',longitude:'Longitude',state:'State',city:'City'}[key]} name={key} value={form[key] ?? ''} type={['operationalRadius','latitude','longitude'].includes(key) ? 'number' : 'text'} onChange={change} />)}<button className="rounded-lg bg-signal px-4 py-2 font-semibold md:col-span-2">Save demo unit</button></form>
}
function downloadDemo(name: string, ...items: unknown[]) {
  const blob = new Blob([JSON.stringify({ sampleOnly: true, exportedAt: new Date().toISOString(), data: items }, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href = url; a.download = `${name}-demo.json`; a.click(); URL.revokeObjectURL(url)
}

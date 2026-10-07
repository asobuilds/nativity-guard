import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { ErrorState } from '@/components/ui/States'
import { api } from '@/lib/apiClient'

type Setting = { key: string; value: string; description?: string; isPublic: boolean }

export function AdminSettingsPage() {
  const client = useQueryClient()
  const [key, setKey] = useState('')
  const [value, setValue] = useState('')
  const [description, setDescription] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const lookup = useQuery({ queryKey: ['system-setting', key], queryFn: () => api.get<{setting: Setting}>('/settings/' + encodeURIComponent(key)), enabled: false })
  const save = useMutation({ mutationFn: () => api.put('/settings/' + encodeURIComponent(key.trim()), { value, description, isPublic }), onSuccess: () => { void client.invalidateQueries({queryKey:['system-setting', key]}) } })
  async function load(e: FormEvent) { e.preventDefault(); if (!key.trim()) return; const result = await lookup.refetch(); if (result.data?.setting) { setValue(result.data.setting.value); setDescription(result.data.setting.description ?? ''); setIsPublic(result.data.setting.isPublic) } }
  return <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
    <header><h1 className="text-xl font-semibold text-ink">Unit settings</h1><p className="mt-1 text-sm text-ink-muted">Read and update persisted system settings through the live settings API.</p></header>
    <Card className="p-5"><form className="space-y-3" onSubmit={load}><Field label="Setting key" required>{p => <Input {...p} required value={key} onChange={e=>setKey(e.target.value)} placeholder="example.setting" />}</Field><Button type="submit" disabled={!key.trim()} loading={lookup.isFetching}>Load setting</Button></form>{lookup.isError && <ErrorState title="Setting not found" description="You can create this key by entering a value below and saving it." />}</Card>
    <Card className="p-5"><form className="space-y-3" onSubmit={e=>{e.preventDefault(); if(key.trim()&&value.trim()) save.mutate()}}><Field label="Value" required>{p=><Input {...p} required value={value} onChange={e=>setValue(e.target.value)} />}</Field><Field label="Description">{p=><Textarea {...p} value={description} onChange={e=>setDescription(e.target.value)} />}</Field><label className="flex items-center gap-2 text-sm text-ink"><input type="checkbox" checked={isPublic} onChange={e=>setIsPublic(e.target.checked)} /> Public setting</label><Button type="submit" variant="primary" disabled={!key.trim()||!value.trim()} loading={save.isPending}>Save setting</Button>{save.isSuccess && <p role="status" className="text-sm text-ok">Setting saved.</p>}{save.isError && <p role="alert" className="text-sm text-emergency">Could not save setting.</p>}</form></Card>
  </div>
}

import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageSquare, Send, ShieldCheck } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { useUnits } from '@/hooks/useUnits'
import { api } from '@/lib/apiClient'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Select, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'

type Room = { id: string; name: string; type: string; unitId: string; unitName?: string }
type Message = {
  id: string
  roomId: string
  senderId: string
  message: string
  createdAt: string
  sender?: { firstName?: string; lastName?: string }
}

export function MessagesPage() {
  const { user, role } = useAuth()
  const queryClient = useQueryClient()
  const [params, setParams] = useSearchParams()
  const [draft, setDraft] = useState('')
  const [unitId, setUnitId] = useState('')
  const [openingMessage, setOpeningMessage] = useState('')
  const isCitizen = role === 'citizen'
  const units = useUnits()
  const selectedRoomId = params.get('roomId') ?? ''

  const rooms = useQuery({
    queryKey: ['communication', 'my-rooms'],
    queryFn: () => api.get<{ rooms: Room[] }>('/communication/my-rooms'),
    refetchInterval: 15000,
  })
  const selectedRoom = useMemo(
    () => rooms.data?.rooms.find(room => room.id === selectedRoomId),
    [rooms.data?.rooms, selectedRoomId],
  )
  const messages = useQuery({
    queryKey: ['communication', 'messages', selectedRoomId],
    queryFn: () => api.get<{ messages: Message[] }>(`/communication/rooms/${selectedRoomId}/messages`),
    enabled: Boolean(selectedRoomId),
    refetchInterval: 5000,
  })

  useEffect(() => {
    if (!selectedRoomId && rooms.data?.rooms.length) {
      setParams({ roomId: rooms.data.rooms[0].id }, { replace: true })
    }
  }, [rooms.data?.rooms, selectedRoomId, setParams])

  const openConversation = useMutation({
    mutationFn: () => api.post<{ room: Room }>('/communication/citizen-inbox', { unitId, message: openingMessage.trim() }),
    onSuccess: data => {
      setOpeningMessage('')
      void queryClient.invalidateQueries({ queryKey: ['communication', 'my-rooms'] })
      setParams({ roomId: data.room.id })
    },
  })
  const send = useMutation({
    mutationFn: () => api.post('/communication/messages', { roomId: selectedRoomId, message: draft.trim(), messageType: 'text' }),
    onSuccess: () => {
      setDraft('')
      void queryClient.invalidateQueries({ queryKey: ['communication', 'messages', selectedRoomId] })
      void queryClient.invalidateQueries({ queryKey: ['communication', 'my-rooms'] })
    },
  })

  const verifiedUnits = (units.data ?? []).filter(unit => unit.isVerified && unit.status === 'active')
  const canOpen = isCitizen && unitId && openingMessage.trim().length > 0

  return (
    <main className="mx-auto w-full max-w-6xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-ink"><MessageSquare className="size-6 text-signal" /> Messages</h1>
        <p className="mt-1 text-sm text-ink-muted">
          {isCitizen ? 'Private conversations with verified security units.' : 'Private citizen requests and your authorised unit communication rooms.'}
        </p>
      </header>

      {isCitizen && rooms.data?.rooms.length === 0 ? (
        <Card className="space-y-3 p-4">
          <div className="flex items-center gap-2 text-sm font-semibold text-ink"><ShieldCheck className="size-4 text-signal" /> Contact a verified security unit</div>
          <label className="block text-xs text-ink-muted">Security unit
            <Select value={unitId} onChange={event => setUnitId(event.target.value)}>
              <option value="">Choose a verified unit</option>
              {verifiedUnits.map(unit => <option key={unit.id} value={unit.id}>{unit.name} · {unit.state ?? 'Nigeria'}</option>)}
            </Select>
          </label>
          <label className="block text-xs text-ink-muted">Your message
            <Textarea value={openingMessage} maxLength={4000} onChange={event => setOpeningMessage(event.target.value)} placeholder="Explain how the security unit can help…" />
          </label>
          <Button disabled={!canOpen || openConversation.isPending} loading={openConversation.isPending} onClick={() => openConversation.mutate()}>Start private conversation</Button>
          {openConversation.isError ? <p role="alert" className="text-sm text-danger">Could not open the conversation. Try again.</p> : null}
        </Card>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-[minmax(230px,0.8fr)_minmax(0,1.7fr)]">
        <Card className="space-y-2 p-3">
          <h2 className="px-2 py-1 text-sm font-semibold text-ink">Your conversations</h2>
          {rooms.isLoading ? <Skeleton className="h-16 w-full" /> : rooms.isError ? <ErrorState title="Could not load conversations" description="Try again." onRetry={() => void rooms.refetch()} /> : rooms.data?.rooms.length ? (
            <ul className="space-y-1">
              {rooms.data.rooms.map(room => (
                <li key={room.id}>
                  <button type="button" onClick={() => setParams({ roomId: room.id })} aria-current={selectedRoomId === room.id ? 'page' : undefined} className={`w-full rounded-lg px-3 py-3 text-left transition ${selectedRoomId === room.id ? 'bg-signal/10 text-ink' : 'text-ink-muted hover:bg-surface-hi'}`}>
                    <span className="block truncate text-sm font-medium">{room.unitName || room.name}</span>
                    <span className="mt-1 block text-xs capitalize text-ink-faint">{room.type === 'private' ? 'Private security inbox' : room.type.replaceAll('_', ' ')}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : <EmptyState title="No conversations yet" description={isCitizen ? 'Choose a verified security unit above to start a private conversation.' : 'Your authorised unit and private rooms will appear here.'} />}
        </Card>

        <Card className="flex min-h-[440px] flex-col p-3 sm:p-4">
          {selectedRoom ? (
            <>
              <div className="border-b border-border px-2 pb-3">
                <h2 className="font-semibold text-ink">{selectedRoom.unitName || selectedRoom.name}</h2>
                <p className="text-xs text-ink-faint">{selectedRoom.type === 'private' ? 'Private citizen-to-security conversation' : 'Security operations room'}</p>
              </div>
              <div className="flex-1 space-y-3 overflow-y-auto py-4">
                {messages.isLoading ? <Skeleton className="h-20 w-full" /> : messages.isError ? <ErrorState title="Could not load messages" description="Try again." onRetry={() => void messages.refetch()} /> : messages.data?.messages.length ? (
                  messages.data.messages.map(message => {
                    const mine = message.senderId === user?.id
                    return <div key={message.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                      <div className={`max-w-[85%] rounded-2xl px-3 py-2 ${mine ? 'bg-signal text-white' : 'bg-surface-hi text-ink'}`}>
                        <p className="mb-1 text-[11px] opacity-75">{mine ? 'You' : [message.sender?.firstName, message.sender?.lastName].filter(Boolean).join(' ') || 'Security team'}</p>
                        <p className="whitespace-pre-wrap break-words text-sm">{message.message}</p>
                        <time className="mt-1 block text-right text-[10px] opacity-70">{new Date(message.createdAt).toLocaleString()}</time>
                      </div>
                    </div>
                  })
                ) : <EmptyState title="Conversation started" description="Send a message to continue this conversation." />}
              </div>
              <form className="flex items-end gap-2 border-t border-border pt-3" onSubmit={event => { event.preventDefault(); if (draft.trim() && !send.isPending) send.mutate() }}>
                <div className="min-w-0 flex-1"><Input value={draft} maxLength={4000} onChange={event => setDraft(event.target.value)} placeholder="Write a message…" aria-label="Message" /></div>
                <Button type="submit" disabled={!draft.trim() || send.isPending} loading={send.isPending} icon={<Send className="size-4" />}>Send</Button>
              </form>
              {send.isError ? <p role="alert" className="mt-2 text-xs text-danger">Message not sent. Your text is still available above.</p> : null}
            </>
          ) : <div className="flex flex-1 items-center justify-center"><EmptyState title="Select a conversation" description="Choose a conversation from the left to read and send messages." /></div>}
        </Card>
      </div>
    </main>
  )
}

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Textarea } from '@/components/ui/Field'
import { ApiError, api } from '@/lib/apiClient'
import { caseKeys } from '@/hooks/useCases'
import { USE_MOCKS } from '@/mocks/config'
import { useAuth } from '@/auth/AuthContext'
import type { CaseFeedback } from '@/types/api'

export function CitizenFeedback({ caseId, closed, feedback }: { caseId: string; closed: boolean; feedback: CaseFeedback[] }) {
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  const { user } = useAuth()
  const client = useQueryClient()
  const submit = useMutation({
    mutationFn: () => api.post<{ message: string; feedback: CaseFeedback }>(
      `/cases/${caseId}/feedback`,
      { rating, comment: comment.trim() },
    ),
    onSuccess: () => client.invalidateQueries({ queryKey: caseKeys.detail(caseId) }),
  })
  // Show the confirmed response immediately, even while the case detail refetches.
  const mine = submit.data?.feedback ?? feedback.find((item) => item.userId === user?.id)

  return (
    <Card className="space-y-3 p-4">
      <h2 className="text-sm font-semibold text-ink">Your feedback</h2>
      {mine ? (
        <p className="text-sm text-ink-muted">You rated this response {mine.rating} out of 5. {mine.comment}</p>
      ) : !closed ? (
        <p className="text-sm text-ink-muted">Feedback opens after this report is closed.</p>
      ) : (
        <form onSubmit={(e) => { e.preventDefault(); if (rating) submit.mutate() }} className="space-y-3">
          <fieldset>
            <legend className="text-xs text-ink-muted">Rate the response</legend>
            <div className="mt-2 flex gap-2">
              {[1, 2, 3, 4, 5].map((value) => (
                <button key={value} type="button" aria-label={`${value} out of 5`} aria-pressed={rating === value} onClick={() => setRating(value)} className={`grid size-11 place-items-center rounded-lg border text-sm ${rating === value ? 'border-signal bg-signal text-signal-ink' : 'border-border-hi text-ink'}`}>
                  {value}
                </button>
              ))}
            </div>
          </fieldset>
          <label className="block text-xs text-ink-muted">Comment (optional)
            <Textarea maxLength={1000} value={comment} onChange={(e) => setComment(e.target.value)} placeholder="What went well or could improve?" />
          </label>
          {submit.isError ? <p role="alert" className="text-xs text-warn">
            {submit.error instanceof ApiError ? submit.error.message : 'Feedback was not confirmed. Please try again.'}
          </p> : null}
          <Button variant="primary" type="submit" disabled={!rating} loading={submit.isPending}>Send feedback</Button>
          {USE_MOCKS ? <p className="text-xs text-ink-faint">Demo feedback is stored only in this browser session.</p> : null}
        </form>
      )}
    </Card>
  )
}

import { useState } from 'react'
import { ExternalLink, Newspaper } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useLiveNews } from '@/hooks/useLiveNews'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import { relativeTime } from '@/lib/format'
import { cn } from '@/lib/cn'

interface SubscriptionResponse {
  subscription: {
    categories: string[]
  } | null
}

const CATEGORY_LABEL: Record<string, string> = {
  security: 'Security',
  community: 'Community',
  weather: 'Weather',
  general: 'General',
}

const ALL_CATEGORIES = ['security', 'community', 'weather', 'general']

export function NewsPage() {
  const [activeCategory, setActiveCategory] = useState<string | null>(null)

  // Fetch the caller's subscription so the filter chips reflect what they chose.
  const subQuery = useQuery({
    queryKey: ['alert-subscription'],
    queryFn: () => api.get<SubscriptionResponse>('/alerts/subscriptions'),
    staleTime: 5 * 60_000,
  })

  const subscribed = subQuery.data?.subscription?.categories ?? []
  const available = subscribed.length > 0 ? subscribed : ALL_CATEGORIES

  // Categories to send to the backend. When a chip is active, only that one.
  const requestCategories = activeCategory ? [activeCategory] : undefined

  const news = useLiveNews({ categories: requestCategories, limit: 20 })
  const items = news.data?.items ?? []

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="text-2xl font-bold text-ink">Live news</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Articles from The Guardian, filtered to the categories you follow.
        </p>
      </header>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => setActiveCategory(null)}
          className={cn(
            'rounded-full px-3 py-1.5 text-xs font-medium transition-colors',
            activeCategory === null
              ? 'bg-signal text-signal-ink'
              : 'bg-surface-hi text-ink-muted hover:text-ink',
          )}
        >
          All your subscriptions
        </button>
        {available.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => setActiveCategory(c)}
            className={cn(
              'rounded-full px-3 py-1.5 text-xs font-medium transition-colors',
              activeCategory === c
                ? 'bg-signal text-signal-ink'
                : 'bg-surface-hi text-ink-muted hover:text-ink',
            )}
          >
            {CATEGORY_LABEL[c] ?? c}
          </button>
        ))}
      </div>

      <Card>
        <CardHeader
          title={`${items.length} ${items.length === 1 ? 'article' : 'articles'}`}
          subtitle={
            subscribed.length > 0
              ? `Filtered to: ${subscribed.map((c) => CATEGORY_LABEL[c] ?? c).join(', ')}`
              : 'Showing general coverage'
          }
        />
        <CardBody>
          {news.isLoading ? (
            <div className="flex flex-col gap-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-24 w-full" />
              ))}
            </div>
          ) : news.isError ? (
            <ErrorState
              title="Could not load news"
              description="The news service is not reachable right now."
              onRetry={() => void news.refetch()}
            />
          ) : items.length === 0 ? (
            <EmptyState
              icon={<Newspaper className="size-5" aria-hidden />}
              title="No articles match"
              description="Try a different category, or adjust your subscriptions in Settings."
            />
          ) : (
            <ul className="flex flex-col gap-2">
              {items.map((item) => (
                <li key={item.id}>
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="group block rounded-xl border border-transparent p-4 transition-all hover:border-border-hi hover:bg-surface-hi/40"
                  >
                    <div className="flex items-start gap-4">
                      {item.thumbnail ? (
                        <img
                          src={item.thumbnail}
                          alt=""
                          className="hidden size-20 shrink-0 rounded-lg object-cover sm:block"
                          loading="lazy"
                        />
                      ) : null}
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="rounded-full bg-signal/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-signal">
                            {CATEGORY_LABEL[item.category] ?? item.category}
                          </span>
                          <span className="text-[11px] text-ink-faint">
                            {item.section} · {relativeTime(item.publishedAt)}
                          </span>
                        </div>
                        <p className="mt-1.5 text-base font-semibold text-ink group-hover:text-signal">
                          {item.title}
                        </p>
                        <p className="mt-1 line-clamp-2 text-sm leading-relaxed text-ink-muted">
                          {item.summary}
                        </p>
                      </div>
                      <ExternalLink
                        className="mt-1 hidden size-4 shrink-0 text-ink-faint transition-transform group-hover:translate-x-0.5 sm:block"
                        aria-hidden
                      />
                    </div>
                  </a>
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      <p className="text-center text-[11px] text-ink-faint">
        Articles sourced from{' '}
        <a
          href="https://www.theguardian.com/"
          target="_blank"
          rel="noopener noreferrer"
          className="text-signal hover:underline"
        >
          The Guardian
        </a>
        . Click any article to open the original on their site.
      </p>
    </div>
  )
}
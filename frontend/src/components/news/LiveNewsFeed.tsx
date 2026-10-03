import { ExternalLink, Newspaper } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useLiveNews } from '@/hooks/useLiveNews'
import { relativeTime } from '@/lib/format'
import { cn } from '@/lib/cn'

interface LiveNewsFeedProps {
  /** When omitted, the backend uses the caller's subscription. */
  categories?: string[]
  limit?: number
  compact?: boolean
  /** When true, includes a title link that navigates to /news. */
  showHeaderLink?: boolean
}

const CATEGORY_LABEL: Record<string, string> = {
  security: 'Security',
  community: 'Community',
  weather: 'Weather',
  general: 'General',
}

/**
 * Live news feed filtered to the caller's subscription. Renders nothing if
 * the request fails or returns zero items — better than an empty card.
 */
export function LiveNewsFeed({
  categories,
  limit = 3,
  compact = false,
  showHeaderLink = true,
}: LiveNewsFeedProps) {
  const query = useLiveNews({ categories, limit })
  const items = query.data?.items ?? []

  if (query.isError) {
    if (compact) return null
    return (
      <Card>
        <CardHeader title="Live news" />
        <CardBody>
          <ErrorState
            title="Could not load news"
            description="Check your connection and try again."
            onRetry={() => void query.refetch()}
          />
        </CardBody>
      </Card>
    )
  }

  if (query.isLoading) {
    return (
      <Card>
        <CardHeader title="Live news" />
        <CardBody className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </CardBody>
      </Card>
    )
  }

  if (items.length === 0) {
    if (compact) return null
    return (
      <Card>
        <CardHeader title="Live news" />
        <CardBody>
          <EmptyState
            icon={<Newspaper className="size-5" aria-hidden />}
            title="No news right now"
            description="Articles matching your subscriptions will appear here."
          />
        </CardBody>
      </Card>
    )
  }

  return (
    <Card as="section">
      <CardHeader
        title="Latest from the region"
        subtitle="Filtered by your alert subscriptions"
        actions={
          showHeaderLink ? (
            <Link to="/news" className="text-xs font-medium text-signal hover:underline">
              View all
            </Link>
          ) : null
        }
      />
      <CardBody className="flex flex-col gap-3">
        <ul className="flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id}>
              <a
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(
                  'group block rounded-lg border border-transparent p-3 transition-all',
                  'hover:border-border-hi hover:bg-surface-hi/40',
                )}
              >
                <div className="flex items-start gap-3">
                  {item.thumbnail ? (
                    <img
                      src={item.thumbnail}
                      alt=""
                      className="size-14 shrink-0 rounded-lg object-cover"
                      loading="lazy"
                    />
                  ) : null}
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full bg-signal/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-signal">
                        {CATEGORY_LABEL[item.category] ?? item.category}
                      </span>
                      <span className="text-[11px] text-ink-faint">
                        {relativeTime(item.publishedAt)}
                      </span>
                    </div>
                    <p className="mt-1.5 line-clamp-2 text-sm font-medium text-ink group-hover:text-signal">
                      {item.title}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-ink-muted">
                      {item.summary}
                    </p>
                  </div>
                  <ExternalLink
                    className="mt-1 size-3.5 shrink-0 text-ink-faint transition-transform group-hover:translate-x-0.5"
                    aria-hidden
                  />
                </div>
              </a>
            </li>
          ))}
        </ul>
      </CardBody>
    </Card>
  )
}
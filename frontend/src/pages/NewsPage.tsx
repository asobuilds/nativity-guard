import { useMemo, useState } from 'react'
import { ExternalLink, Newspaper } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { Button } from '@/components/ui/Button'
import { useLiveNews, type LiveNewsItem } from '@/hooks/useLiveNews'
import { relativeTime } from '@/lib/format'
import { cn } from '@/lib/cn'

const CATEGORY_LABEL: Record<string, string> = {
  local: 'Local',
  security: 'Security',
  politics: 'Politics',
  community: 'Community',
  weather: 'Weather',
  general: 'General',
}

const CATEGORY_CHIPS = [
  { id: 'all', label: 'All', value: null },
  { id: 'local', label: 'Local', value: 'local' },
  { id: 'security', label: 'Security', value: 'security' },
  { id: 'politics', label: 'Politics', value: 'politics' },
] as const

type CategoryValue = (typeof CATEGORY_CHIPS)[number]['value']

interface DateGroup {
  key: 'today' | 'yesterday' | 'earlier'
  label: string
  items: LiveNewsItem[]
}

function groupArticlesByDate(items: LiveNewsItem[]): DateGroup[] {
  const now = new Date()
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const startOfYesterday = startOfToday - 24 * 60 * 60 * 1000

  const today: LiveNewsItem[] = []
  const yesterday: LiveNewsItem[] = []
  const earlier: LiveNewsItem[] = []

  for (const item of items) {
    const itemTime = new Date(item.publishedAt).getTime()
    if (isNaN(itemTime)) {
      earlier.push(item)
    } else if (itemTime >= startOfToday) {
      today.push(item)
    } else if (itemTime >= startOfYesterday) {
      yesterday.push(item)
    } else {
      earlier.push(item)
    }
  }

  const groups: DateGroup[] = []
  if (today.length > 0) groups.push({ key: 'today', label: 'Today', items: today })
  if (yesterday.length > 0) groups.push({ key: 'yesterday', label: 'Yesterday', items: yesterday })
  if (earlier.length > 0) groups.push({ key: 'earlier', label: 'Earlier', items: earlier })

  return groups
}

export function NewsPage() {
  const [activeCategory, setActiveCategory] = useState<CategoryValue>(null)

  const requestCategories = activeCategory ? [activeCategory] : undefined
  const news = useLiveNews({ categories: requestCategories, limit: 20 })
  const items = news.data?.items ?? []

  const dateGroups = useMemo(() => groupArticlesByDate(items), [items])

  const activeCategoryLabel = activeCategory ? CATEGORY_LABEL[activeCategory] ?? activeCategory : null

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="text-2xl font-bold text-ink">Live news</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Articles from The Guardian, filtered to the categories you follow.
        </p>
      </header>

      {/* Category Chips */}
      <div className="flex flex-wrap gap-2" role="group" aria-label="News category filters">
        {CATEGORY_CHIPS.map((chip) => {
          const isActive = activeCategory === chip.value
          return (
            <button
              key={chip.id}
              type="button"
              onClick={() => setActiveCategory(chip.value)}
              className={cn(
                'rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors',
                isActive
                  ? 'bg-signal text-signal-ink font-semibold shadow-sm'
                  : 'bg-surface-hi text-ink-muted hover:text-ink hover:bg-surface-hi/80',
              )}
            >
              {chip.label}
            </button>
          )
        })}
      </div>

      <Card>
        <CardHeader
          title={`${items.length} ${items.length === 1 ? 'article' : 'articles'}`}
          subtitle={
            activeCategoryLabel
              ? `Filtered to: ${activeCategoryLabel}`
              : 'Showing all coverage'
          }
        />
        <CardBody>
          {news.isLoading ? (
            <div className="flex flex-col gap-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <div
                  key={i}
                  className="flex items-start gap-4 rounded-xl border border-border/40 p-4"
                >
                  <Skeleton className="hidden size-20 shrink-0 rounded-lg sm:block" />
                  <div className="min-w-0 flex-1 space-y-2">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-4 w-16 rounded-full" />
                      <Skeleton className="h-3 w-28 rounded" />
                    </div>
                    <Skeleton className="h-5 w-4/5 rounded" />
                    <Skeleton className="h-3.5 w-full rounded" />
                    <Skeleton className="h-3.5 w-2/3 rounded" />
                  </div>
                </div>
              ))}
            </div>
          ) : news.isError ? (
            <ErrorState
              title="Could not load news"
              description="The news service is not reachable right now."
              onRetry={() => void news.refetch()}
            />
          ) : items.length === 0 ? (
            activeCategoryLabel ? (
              <EmptyState
                icon={<Newspaper className="size-5" aria-hidden />}
                title={`No news found in ${activeCategoryLabel}`}
                description={`There are currently no articles matching the ${activeCategoryLabel.toLowerCase()} category.`}
                action={
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setActiveCategory(null)}
                  >
                    Clear filter
                  </Button>
                }
              />
            ) : (
              <EmptyState
                icon={<Newspaper className="size-5" aria-hidden />}
                title="No news articles found"
                description="There are no news articles available right now. Please check back later."
                action={
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => void news.refetch()}
                  >
                    Refresh
                  </Button>
                }
              />
            )
          ) : (
            <div className="space-y-6">
              {dateGroups.map((group) => (
                <section
                  key={group.key}
                  aria-labelledby={`date-group-${group.key}`}
                  className="space-y-3"
                >
                  <div className="flex items-center gap-3 pt-1">
                    <h2
                      id={`date-group-${group.key}`}
                      className="text-xs font-bold uppercase tracking-wider text-ink-muted"
                    >
                      {group.label}
                    </h2>
                    <div className="h-px flex-1 bg-border/60" />
                  </div>

                  <ul className="flex flex-col gap-2">
                    {group.items.map((item) => (
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
                </section>
              ))}
            </div>
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

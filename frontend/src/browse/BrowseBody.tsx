import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Pagination, Typography } from '@mui/material'
import { CloudOff, Search, SearchX } from 'lucide-react'
import { useLayoutEffect, useRef, useState } from 'react'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { unreachableNote } from '../shell/offlineText.ts'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { space } from '../theme/density.ts'
import type { InlineError } from '../toasts/report.ts'
import {
  FIRST_PAGE,
  grid,
  ICON_SIZE,
  list,
  PICTURE_PX,
  ROW_PICTURE_PX,
  STALE_OPACITY,
} from './browseConstants.ts'
import type { ResultCardProps } from './browseTypes.ts'
import { ResultCard } from './ResultCard.tsx'
import { skeletonCount } from './skeletonCount.ts'
import type { BrowseResult, Status } from './useBrowseQuery.ts'

// cardAction is what an arrow key lands on in a result card: its last control, the Add or Download action.
function cardAction(card: HTMLElement): HTMLElement | null {
  return [...card.querySelectorAll<HTMLElement>('button:not(:disabled)')].at(-1) ?? null
}

// A result card is its picture plus the card's vertical padding.
const CARD_PAD_PX = 24

type CardShared = Omit<ResultCardProps, 'row' | 'item'>

interface BrowseBodyProps {
  status: Status
  result: BrowseResult
  text: string
  hint: string
  error: InlineError | null
  // The banner's sentence when the searched sources cannot be reached; it replaces the generic failure text.
  offline: string
  view: string
  page: number
  pageCount: number
  card: CardShared
  onRetry: () => void
  onClear: () => void
  onPage: (page: number) => void
}

// Fills the results area with placeholders in the results' own grid, so nothing moves when they arrive.
function BrowseSkeleton({ view }: { view: string }) {
  const { t } = useLingui()
  const box = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState({ width: 0, height: 0 })
  useLayoutEffect(() => {
    const el = box.current
    if (!el) {
      return
    }
    const measure = () => setSize({ width: el.clientWidth, height: el.clientHeight })
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    measure()
    return () => ro.disconnect()
  }, [])
  const isGrid = view === 'grid'
  const rowPx = (isGrid ? PICTURE_PX : ROW_PICTURE_PX) + CARD_PAD_PX
  return (
    <Box ref={box} sx={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
      <SkeletonRows
        label={t`Searching…`}
        count={skeletonCount({ ...size, grid: isGrid, rowPx })}
        height={rowPx}
        sx={isGrid ? grid : list}
      />
    </Box>
  )
}

function ResultList({
  result,
  loading,
  view,
  card,
}: Pick<BrowseBodyProps, 'result' | 'view' | 'card'> & { loading: boolean }) {
  return (
    <Box
      sx={{ ...(view === 'grid' ? grid : list), opacity: loading ? STALE_OPACITY : 1 }}
      onKeyDown={(e) => arrowFocus(e, ':scope > *', cardAction)}
    >
      {result.items.map((item) => (
        <ResultCard key={`${item.source}:${item.id}`} row={view === 'list'} item={item} {...card} />
      ))}
    </Box>
  )
}

// The bar under the results: how many there are, how many the Show rows hid and any source that did not answer.
function ResultFooter({ result }: { result: BrowseResult }) {
  const { t } = useLingui()
  return (
    <Typography
      component="footer"
      sx={{
        flexShrink: 0,
        px: space.gutter,
        py: space.gap,
        fontSize: 13,
        color: 'text.secondary',
        borderTop: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      {plural(result.total, { one: '# result', other: '# results' })}
      {result.hidden ? ` · ${t`${result.hidden} hidden`}` : ''}
      {result.failed && result.failed.length > 0 ? ` · ${unreachableNote(result.failed)}` : ''}
    </Typography>
  )
}

function BrowseBody(props: BrowseBodyProps) {
  const { t } = useLingui()
  const { status, result, text, hint, error, offline, page, pageCount, onRetry, onClear, onPage } =
    props
  if (status === 'idle' || (status === 'done' && result.items.length === 0 && text.trim() === '')) {
    return (
      <EmptyState icon={<Search size={ICON_SIZE} />} title={t`Find mods to add`}>
        {hint}
      </EmptyState>
    )
  }
  if (status === 'error') {
    return (
      <EmptyState
        icon={<CloudOff size={ICON_SIZE} />}
        title={t`Search did not work`}
        action={
          <Button variant="outlined" onClick={onRetry}>
            {t`Retry`}
          </Button>
        }
      >
        <span title={error?.details}>
          {offline || (error?.message ?? t`The service may be busy. Try again in a minute.`)}
        </span>
      </EmptyState>
    )
  }
  if (result.items.length === 0) {
    return status === 'loading' ? (
      <BrowseSkeleton view={props.view} />
    ) : (
      <EmptyState
        icon={<SearchX size={ICON_SIZE} />}
        title={t`No mods match "${text}"`}
        action={
          <Button variant="outlined" onClick={onClear}>
            {t`Clear search`}
          </Button>
        }
      >
        {t`Try fewer words, or the mod's exact name.`}
      </EmptyState>
    )
  }
  return (
    <>
      <ResultList
        result={result}
        loading={status === 'loading'}
        view={props.view}
        card={props.card}
      />
      {pageCount > FIRST_PAGE ? (
        <Pagination
          count={pageCount}
          page={page}
          onChange={(_event, next) => onPage(next)}
          sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}
        />
      ) : null}
    </>
  )
}

export { BrowseBody, ResultFooter }

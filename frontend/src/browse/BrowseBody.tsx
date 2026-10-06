import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Pagination, Typography } from '@mui/material'
import { CloudOff, Search, SearchX } from 'lucide-react'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { unreachableNote } from '../shell/offlineText.ts'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import type { InlineError } from '../toasts/report.ts'
import {
  FIRST_PAGE,
  grid,
  ICON_SIZE,
  list,
  PICTURE_PX,
  ROW_PICTURE_PX,
  SKELETON_KEYS,
  STALE_OPACITY,
} from './browseConstants.ts'
import type { ResultCardProps } from './browseTypes.ts'
import { ResultCard } from './ResultCard.tsx'
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

function ResultList({
  result,
  loading,
  view,
  card,
}: Pick<BrowseBodyProps, 'result' | 'view' | 'card'> & { loading: boolean }) {
  const { t } = useLingui()
  return (
    <>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>
        {plural(result.total, { one: '# result', other: '# results' })}
        {result.hidden ? ` · ${t`${result.hidden} hidden`}` : ''}
        {result.failed && result.failed.length > 0 ? ` · ${unreachableNote(result.failed)}` : ''}
      </Typography>
      <Box
        sx={{ ...(view === 'grid' ? grid : list), opacity: loading ? STALE_OPACITY : 1 }}
        onKeyDown={(e) => arrowFocus(e, ':scope > *', cardAction)}
      >
        {result.items.map((item) => (
          <ResultCard
            key={`${item.source}:${item.id}`}
            row={view === 'list'}
            item={item}
            {...card}
          />
        ))}
      </Box>
    </>
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
      <SkeletonRows
        label={t`Searching…`}
        count={SKELETON_KEYS.length}
        height={(props.view === 'grid' ? PICTURE_PX : ROW_PICTURE_PX) + CARD_PAD_PX}
        sx={props.view === 'grid' ? grid : list}
      />
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

export { BrowseBody }

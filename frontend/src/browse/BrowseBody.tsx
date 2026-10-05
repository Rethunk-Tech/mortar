import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Pagination, Skeleton, Typography } from '@mui/material'
import { CloudOff, Search, SearchX } from 'lucide-react'
import { EmptyState } from '../shell/EmptyState.tsx'
import type { InlineError } from '../toasts/report.ts'
import {
  FIRST_PAGE,
  grid,
  ICON_SIZE,
  list,
  PICTURE_PX,
  SKELETON_KEYS,
  STALE_OPACITY,
} from './browseConstants.ts'
import type { ResultCardProps } from './browseTypes.ts'
import { ResultCard } from './ResultCard.tsx'
import type { BrowseResult, Status } from './useBrowseQuery.ts'

type CardShared = Omit<ResultCardProps, 'row' | 'item'>

interface BrowseBodyProps {
  status: Status
  result: BrowseResult
  text: string
  hint: string
  error: InlineError | null
  view: string
  page: number
  pageCount: number
  card: CardShared
  onRetry: () => void
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
        {result.failed && result.failed.length > 0
          ? ` · ${t`${result.failed.join(', ')} did not answer`}`
          : ''}
      </Typography>
      <Box sx={{ ...(view === 'grid' ? grid : list), opacity: loading ? STALE_OPACITY : 1 }}>
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
  const { status, result, text, hint, error, page, pageCount, onRetry, onPage } = props
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
          {error?.message ?? t`The service may be busy. Try again in a minute.`}
        </span>
      </EmptyState>
    )
  }
  if (result.items.length === 0) {
    return status === 'loading' ? (
      <Box sx={grid}>
        {SKELETON_KEYS.map((key) => (
          <Skeleton key={key} variant="rounded" height={PICTURE_PX + 24} />
        ))}
      </Box>
    ) : (
      <EmptyState icon={<SearchX size={ICON_SIZE} />} title={t`No mods match "${text}"`}>
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

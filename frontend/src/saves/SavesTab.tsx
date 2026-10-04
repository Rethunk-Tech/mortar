import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { History, Search, Sprout } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { BackupsDialog } from './BackupsDialog.tsx'
import { filterAndSortSaves } from './filterAndSortSaves.ts'
import { SaveRow } from './SaveRow.tsx'
import { useSaves } from './store.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

export function SavesTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const { fits, status, error: detail, load } = useSaves()
  const { name } = profile
  const [backupsOpen, setBackupsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const shown = filterAndSortSaves(fits, query)
  let body: ReactNode = null
  if (status === 'error') {
    body = (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography sx={{ fontSize: 13, color: 'error.main' }} title={errorDetails(detail)}>
          {errorMessage(detail)}
        </Typography>
        <Button
          size="small"
          sx={nowrap}
          onClick={() => {
            load(game, profile.id, String(profile.updated)).catch(reportUnexpected)
          }}
        >
          {t`Retry`}
        </Button>
      </Box>
    )
  } else if (status === 'loading' && fits.length === 0) {
    body = <LoadingRow>{t`Reading your saves…`}</LoadingRow>
  } else if (fits.length === 0) {
    body = (
      <EmptyState icon={<Sprout size={40} aria-hidden={true} />} title={t`No saves yet`}>
        {t`Play this profile and start a farm. Each save shows here with how well it fits ${name}, so you know which mods it needs.`}
      </EmptyState>
    )
  } else if (shown.length === 0 && query.trim() !== '') {
    body = (
      <EmptyState icon={<Search size={40} aria-hidden={true} />} title={t`No saves match`}>
        <Button size="small" onClick={() => setQuery('')} sx={nowrap}>
          {t`Clear filter`}
        </Button>
      </EmptyState>
    )
  } else {
    body = shown.map((fit) => <SaveRow key={fit.folder} fit={fit} profile={profile} game={game} />)
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.5, pb: 0.75 }}>
        {fits.length === 0 ? null : (
          <SearchField
            value={query}
            onChange={setQuery}
            label={t`Filter saves`}
            placeholder={plural(fits.length, { one: 'Filter # save', other: 'Filter # saves' })}
            sx={{ width: 360, maxWidth: '50%' }}
          />
        )}
        <Box sx={{ flex: 1 }} />
        <Button
          size="small"
          startIcon={<History size={14} />}
          onClick={() => setBackupsOpen(true)}
          sx={nowrap}
        >
          {t`Save backups…`}
        </Button>
      </Box>
      <Box
        sx={{
          flex: fits.length > 0 ? undefined : 1,
          display: fits.length > 0 ? 'grid' : 'flex',
          gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
          flexDirection: 'column',
          gap: 1.25,
          px: 2,
          pt: 0.75,
          pb: 1.5,
        }}
      >
        {body}
      </Box>
      <BackupsDialog open={backupsOpen} onClose={() => setBackupsOpen(false)} />
    </Box>
  )
}

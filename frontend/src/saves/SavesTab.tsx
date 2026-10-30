import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useStoredState } from '@rethunk/react-load-hooks'
import { History, Search, Sprout } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { CardsView } from '../shell/CardsView.tsx'
import { ControlsRow } from '../shell/ControlsRow.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { isViewMode, type ViewMode } from '../shell/viewMode.ts'
import { space } from '../theme/density.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { BackupsDialog } from './BackupsDialog.tsx'
import { backupsOf } from './backupsOf.ts'
import { filterAndSortSaves } from './filterAndSortSaves.ts'
import { SaveListRow, SaveListTable } from './SaveListRow.tsx'
import { SaveRow } from './SaveRow.tsx'
import { useSaves } from './store.ts'
import { useAllBackups } from './useAllBackups.ts'

export function SavesTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const fits = useSaves((s) => s.fits)
  const status = useSaves((s) => s.status)
  const detail = useSaves((s) => s.error)
  const load = useSaves((s) => s.load)
  const { name } = profile
  // A save whose format names no mods (Lethal Company) has nothing to match against a profile.
  const recordsMods = fits.some((f) => !f.unrecorded)
  const [backupsOpen, setBackupsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [view, setView] = useStoredState<ViewMode>(`mortar.savesView.${game}`, 'grid', isViewMode)
  const list = view === 'list'
  const { all, reload: reloadBackups } = useAllBackups(
    game,
    fits.map((f) => `${f.folder}:${f.played}`).join('|'),
  )
  const shown = filterAndSortSaves(fits, query)
  let body: ReactNode = null
  if (status === 'error') {
    body = (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography
          sx={{ fontSize: 'base.fontSize', color: 'error.main' }}
          title={errorDetails(detail)}
        >
          {errorMessage(detail)}
        </Typography>
        <Button
          size="small"
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
        {recordsMods
          ? t`Play this profile and save a game. Each save shows here with how well it fits ${name}, so you know which mods it needs.`
          : t`Play this profile and save a game. Each save shows here, with its backups.`}
      </EmptyState>
    )
  } else if (shown.length === 0 && query.trim() !== '') {
    body = (
      <EmptyState
        icon={<Search size={40} aria-hidden={true} />}
        title={t`No saves match`}
        action={
          <Button variant="outlined" onClick={() => setQuery('')}>
            {t`Clear filters`}
          </Button>
        }
      >
        {t`Try a different search.`}
      </EmptyState>
    )
  } else {
    const rows = shown.map((fit) => {
      const props = {
        fit,
        profile,
        game,
        backups: all === null ? null : backupsOf(all, fit.folder),
        onBackupsChanged: reloadBackups,
      }
      return list ? (
        <SaveListRow key={fit.folder} {...props} recordsMods={recordsMods} />
      ) : (
        <SaveRow key={fit.folder} {...props} />
      )
    })
    body = list ? (
      <SaveListTable name={name} recordsMods={recordsMods}>
        {rows}
      </SaveListTable>
    ) : (
      rows
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <PageActions>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<History size={16} />}
          onClick={() => setBackupsOpen(true)}
        >
          {t`Save backups…`}
        </Button>
      </PageActions>
      {fits.length === 0 ? null : (
        <ControlsRow>
          <ViewToggle value={view} onChange={setView} />
          <SearchField
            value={query}
            onChange={setQuery}
            label={t`Filter saves`}
            placeholder={plural(
              { count: fits.length },
              { one: 'Filter # save', other: 'Filter # saves' },
            )}
            grow={true}
          />
        </ControlsRow>
      )}
      <CardsView list={list} fill={fits.length === 0}>
        {body}
      </CardsView>
      <BackupsDialog
        open={backupsOpen}
        onClose={() => setBackupsOpen(false)}
        game={game}
        profile={profile.id}
      />
    </Box>
  )
}

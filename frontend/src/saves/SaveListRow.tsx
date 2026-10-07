import { useLingui } from '@lingui/react/macro'
import { Box, Table, TableBody, TableCell, TableHead, TableRow, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { saveCalendar, saveCalendarUnknown } from '../game/homeView.ts'
import { heading } from '../mods/paper.ts'
import { SaveGapLine } from './SaveGapLine.tsx'
import { FitStatus, SaveButtons } from './SaveRow.tsx'
import { saveName } from './saveName.ts'

// The last column holds up to four icon buttons; a fixed width keeps it aligned when a save has fewer.
const COLUMNS = 'minmax(0,1fr) 160px 220px 148px'

const rowGrid = {
  display: 'grid',
  gridTemplateColumns: COLUMNS,
  gap: '10px',
  alignItems: 'center',
  px: 2,
} as const

const cellReset = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function SaveCell({ children, title }: { children: ReactNode; title?: string }) {
  return (
    <TableCell role="cell" title={title} sx={{ ...cellReset, minWidth: 0 }}>
      {children}
    </TableCell>
  )
}

// The Mods list's table: a header that stays put while rows scroll, one grid row per save, and ARIA that joins
// header and rows into one table.
function SaveListTable({ name, children }: { name: string; children: ReactNode }) {
  const { t } = useLingui()
  return (
    <Table
      role="table"
      aria-label={t`Saves`}
      sx={{ display: 'block', '& thead, & tbody': { display: 'block' } }}
    >
      <TableHead role="rowgroup" sx={{ position: 'sticky', top: 0, zIndex: 1 }}>
        <TableRow
          role="row"
          sx={{
            ...rowGrid,
            height: 30,
            bgcolor: 'var(--mortar-console-90)',
            ...heading,
            borderBottom: '1px solid var(--mortar-hairline-muted)',
          }}
        >
          <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
            {t`Save`}
          </TableCell>
          <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
            {t`Season`}
          </TableCell>
          <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
            {t`Fits ${{ name }}`}
          </TableCell>
          <TableCell role="columnheader" aria-label={t`Actions`} sx={cellReset} />
        </TableRow>
      </TableHead>
      <TableBody role="rowgroup">{children}</TableBody>
    </Table>
  )
}

// The list form of a save: name, Stardew's calendar ("Unknown" with the reason when a save has no date, "—" for games without a calendar), fit and the same actions as the card.
function SaveListRow({
  fit,
  profile,
  game,
  backups,
  onBackupsChanged,
}: {
  fit: Fit
  profile: Profile
  game: string
  backups: Backup[] | null
  onBackupsChanged: () => Promise<void>
}) {
  const { t } = useLingui()
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  const calendar = saveCalendar(fit)
  const unknown = saveCalendarUnknown(fit)
  let when = '—'
  if (calendar) {
    when = t`Year ${calendar.year} ${seasons[calendar.season] ?? ''}`
  } else if (unknown) {
    when = t`Unknown`
  }
  const label = saveName(fit)
  return (
    <TableRow
      role="row"
      hover={true}
      sx={{
        ...rowGrid,
        minHeight: 36,
        fontSize: 14,
        borderBottom: '1px solid var(--mortar-hairline-faint)',
      }}
    >
      <SaveCell title={label}>
        <Typography noWrap={true} sx={{ fontWeight: 500, fontSize: 'inherit' }}>
          {label}
        </Typography>
      </SaveCell>
      <SaveCell
        {...(unknown
          ? {
              title: t`This save has no SaveGameInfo date, so Mortar cannot tell when it is in the game.`,
            }
          : {})}
      >
        <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
          {when}
        </Typography>
      </SaveCell>
      <SaveCell>
        <Box sx={{ display: 'flex' }}>
          <FitStatus missing={(fit.missing ?? []).length} unrecorded={fit.unrecorded} />
        </Box>
      </SaveCell>
      <SaveCell>
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
          <SaveButtons
            fit={fit}
            game={game}
            profileId={profile.id}
            label={label}
            backups={backups}
            onBackupsChanged={onBackupsChanged}
          />
        </Box>
      </SaveCell>
      <Box sx={{ gridColumn: '1 / -1', '&:empty': { display: 'none' }, pb: 0.5 }}>
        <SaveGapLine
          key={`${profile.updated}-${fit.lastProfileAt}`}
          fit={fit}
          profile={profile}
          game={game}
        />
      </Box>
    </TableRow>
  )
}

export { SaveListRow, SaveListTable }

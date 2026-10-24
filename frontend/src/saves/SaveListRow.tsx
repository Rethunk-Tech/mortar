import { useLingui } from '@lingui/react/macro'
import { Box, Table, TableBody, TableCell, TableHead, TableRow, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { saveCalendar, saveCalendarUnknown } from '../game/homeView.ts'
import { heading } from '../mods/paper.ts'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { space } from '../theme/density.ts'
import { SaveGapLine } from './SaveGapLine.tsx'
import { FitStatus, SaveButtons } from './SaveRow.tsx'
import { playStyleName, saveKind, saveName } from './saveName.ts'

// The last column holds up to four icon buttons; a fixed width keeps it aligned when a save has fewer. Season and fit
// columns appear only for games whose saves record a calendar and their mods (SMAPI's).
const rowGrid = (recordsMods: boolean) =>
  ({
    display: 'grid',
    gridTemplateColumns: recordsMods ? 'minmax(0,1fr) 160px 220px 148px' : 'minmax(0,1fr) 148px',
    gap: space.gap,
    alignItems: 'center',
    px: space.gutter,
  }) as const

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
function SaveListTable({
  name,
  recordsMods,
  children,
}: {
  name: string
  recordsMods: boolean
  children: ReactNode
}) {
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
            ...rowGrid(recordsMods),
            height: `calc(${space.row} - 6px)`,
            bgcolor: 'var(--mortar-console-90)',
            ...heading,
            borderBottom: '1px solid var(--mortar-hairline-muted)',
          }}
        >
          <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
            {t`Save`}
          </TableCell>
          {recordsMods ? (
            <>
              <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
                {t`Season`}
              </TableCell>
              <TableCell role="columnheader" sx={{ ...cellReset, ...heading }}>
                {t`Fits ${{ name }}`}
              </TableCell>
            </>
          ) : null}
          <TableCell role="columnheader" sx={cellReset}>
            <Box component="span" sx={visuallyHidden}>
              {t`Actions`}
            </Box>
          </TableCell>
        </TableRow>
      </TableHead>
      <TableBody role="rowgroup">{children}</TableBody>
    </Table>
  )
}

// The list form of a save: name, Stardew's calendar ("Unknown" with the reason when a save has no date), fit and the
// same actions as the card.
function SaveListRow({
  fit,
  profile,
  game,
  backups,
  onBackupsChanged,
  recordsMods,
}: {
  recordsMods: boolean
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
  let when = ''
  if (calendar) {
    when = t`Year ${{ year: calendar.year }} ${{ season: seasons[calendar.season] ?? '' }}`
  } else if (unknown) {
    when = t`Unknown`
  }
  const label = saveName(fit)
  const kind = saveKind(fit.folder) || playStyleName(fit.playStyle)
  return (
    <TableRow
      role="row"
      hover={true}
      sx={{
        ...rowGrid(recordsMods),
        minHeight: space.row,
        fontSize: 'medium.fontSize',
        borderBottom: '1px solid var(--mortar-hairline-faint)',
      }}
    >
      <SaveCell title={label}>
        <Typography noWrap={true} sx={{ fontWeight: 500, fontSize: 'inherit' }}>
          {label}
          {kind === '' ? null : (
            <Box
              component="span"
              sx={{ ml: 1, fontWeight: 400, fontSize: 'base.fontSize', color: 'text.secondary' }}
            >
              {kind}
            </Box>
          )}
        </Typography>
      </SaveCell>
      {recordsMods ? (
        <>
          <SaveCell
            {...(unknown
              ? {
                  title: t`This save has no SaveGameInfo date, so Mortar cannot tell when it is in the game.`,
                }
              : {})}
          >
            <Typography noWrap={true} sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
              {when}
            </Typography>
          </SaveCell>
          <SaveCell>
            <Box sx={{ display: 'flex' }}>
              <FitStatus missing={(fit.missing ?? []).length} />
            </Box>
          </SaveCell>
        </>
      ) : null}
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

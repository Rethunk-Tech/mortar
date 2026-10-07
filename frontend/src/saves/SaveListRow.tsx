import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { saveCalendar } from '../game/homeView.ts'
import { SaveGapLine } from './SaveGapLine.tsx'
import { FitStatus, SaveButtons } from './SaveRow.tsx'
import { saveName } from './saveName.ts'

const COLUMNS = 'minmax(0,1fr) 160px 220px auto'

function SaveListHeader({ name }: { name: string }) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: COLUMNS,
        gap: 3,
        py: 1,
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: '0.06em',
        color: 'text.secondary',
        textTransform: 'uppercase',
      }}
    >
      <span>{t`Save`}</span>
      <span>{t`Season`}</span>
      <span>{t`Fits ${{ name }}`}</span>
      <span />
    </Box>
  )
}

// The list form of a save: name, Stardew's calendar ("—" for games without one), fit and the same actions as the card.
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
  const label = saveName(fit)
  return (
    <Box sx={{ borderTop: '1px solid var(--mortar-hairline-faint)', py: 1 }}>
      <Box sx={{ display: 'grid', gridTemplateColumns: COLUMNS, gap: 3, alignItems: 'center' }}>
        <Typography noWrap={true} title={label} sx={{ fontWeight: 600 }}>
          {label}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 13 }}>
          {calendar ? t`Year ${calendar.year} ${seasons[calendar.season] ?? ''}` : '—'}
        </Typography>
        <Box sx={{ display: 'flex' }}>
          <FitStatus missing={(fit.missing ?? []).length} unrecorded={fit.unrecorded} />
        </Box>
        <SaveButtons
          fit={fit}
          game={game}
          profileId={profile.id}
          label={label}
          backups={backups}
          onBackupsChanged={onBackupsChanged}
        />
      </Box>
      <SaveGapLine
        key={`${profile.updated}-${fit.lastProfileAt}`}
        fit={fit}
        profile={profile}
        game={game}
      />
    </Box>
  )
}

export { SaveListHeader, SaveListRow }

import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import {
  FileConflicts,
  MakePackageWin,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { listNames } from '../i18n/list.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { type ConflictGroup, groupConflicts } from './fileConflicts.ts'
import { useMods } from './store.ts'

const SHOWN_PATHS = 3

/** The open profile's package file conflicts, read again whenever the profile changes. */
export function useFileConflicts(): ConflictGroup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const open = useProfiles(openProfileOf)
  const [groups, setGroups] = useState<ConflictGroup[]>([])
  useEffect(() => {
    if (!(game && open)) {
      setGroups([])
      return
    }
    FileConflicts(game, open.id)
      .then((got) => setGroups(groupConflicts(got ?? [])))
      .catch(reportUnexpected)
  }, [game, open])
  return groups
}

function ConflictRow({ group }: { group: ConflictGroup }) {
  const { t, i18n } = useLingui()
  const mods = useMods((s) => s.mods)
  const name = (key: string) => mods.find((m) => m.key === key)?.name ?? key
  const files = group.paths.slice(0, SHOWN_PATHS).join(', ')
  const more = group.paths.length - SHOWN_PATHS
  const makeWin = (loser: string) => {
    const s = useProfiles.getState()
    const open = openProfileOf(s)
    if (s.game && open) {
      MakePackageWin(s.game.id, open.id, loser, group.winner)
        .then((p) => useProfiles.getState().replace(p))
        .catch(reportUnexpected)
    }
  }
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 0.5,
        pl: space.pad,
        pr: 0.75,
        py: space.gap,
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ fontSize: 14 }}>
        {i18n._(
          plural(group.paths.length, {
            one: `${name(group.winner)} wins # file over ${listNames(group.beaten.map(name))}`,
            other: `${name(group.winner)} wins # files over ${listNames(group.beaten.map(name))}`,
          }),
        )}
      </Typography>
      <Typography
        title={group.paths.join('\n')}
        sx={{ fontSize: 13, color: 'var(--mortar-ink-sec)', wordBreak: 'break-word' }}
      >
        {more > 0 ? t`${files} and ${more} more` : files}
      </Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap }}>
        {group.beaten.map((loser) => (
          <Button key={loser} size="small" onClick={() => makeWin(loser)}>
            {t`Make ${name(loser)} win`}
          </Button>
        ))}
      </Box>
    </Box>
  )
}

// Files more than one enabled package ships: the later one in package order replaces the earlier one's copy.
export function FileConflictsSection({ groups }: { groups: ConflictGroup[] }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      {groups.map((g) => (
        <ConflictRow key={`${g.winner}\0${g.beaten.join('\0')}`} group={g} />
      ))}
    </Box>
  )
}

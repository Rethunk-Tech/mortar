import { useLingui } from '@lingui/react/macro'
import { Box, Switch, Typography } from '@mui/material'
import { ArrowDown, ArrowUp } from 'lucide-react'
import { SearchableSources } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SourceLogo } from '../../brand/sources/SourceLogo.tsx'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { IconAction } from '../../shell/IconAction.tsx'
import { useLoaded } from '../../shell/useLoaded.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useSettingsRelevant } from '../useSettingsSearch.ts'

// The game's sources in the player's order: the saved ids first, then the rest in catalog order.
function ordered(all: { id: string; name: string }[], saved: string) {
  const first = saved.split(',').flatMap((id) => all.filter((s) => s.id === id))
  return [...new Set([...first, ...all])]
}

function SourceOrder() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const label = t`Preferred source`
  const description = t`When a mod is on several sites, its card installs from the first one here. Browse leaves out a source that is off.`
  const saved = useSettings((s) => s.games?.[game]?.sourceOrder ?? '')
  const off = useSettings((s) => s.games?.[game]?.sourcesOff ?? '')
    .split(',')
    .filter(Boolean)
  const relevant = useSettingsRelevant(label, description)
  const { data: all } = useLoaded<{ id: string; name: string }[]>(
    relevant ? () => SearchableSources(game).then((found) => found ?? []) : null,
    [game, relevant],
    [],
    reportUnexpected,
  )
  const list = ordered(all, saved)
  if (list.length < 2) {
    return null
  }
  const move = (from: number, to: number) => {
    const next = [...list]
    const [item] = next.splice(from, 1)
    if (item) {
      next.splice(to, 0, item)
    }
    persist(
      () => SetByKey('sourceOrder', next.map((s) => s.id).join(','), game),
      t`Could not save that setting`,
    )
  }
  const toggle = (id: string, on: boolean) =>
    persist(
      () =>
        SetByKey('sourcesOff', (on ? off.filter((o) => o !== id) : [...off, id]).join(','), game),
      t`Could not save that setting`,
    )
  const onCount = list.filter((s) => !off.includes(s.id)).length
  return (
    <SettingRow label={label} description={description} block={true}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {list.map((s, i) => (
          <Box key={s.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Switch
              size="small"
              checked={!off.includes(s.id)}
              // The last source on stays on, or Browse would have nothing to search.
              disabled={onCount === 1 && !off.includes(s.id)}
              onChange={(_, on) => toggle(s.id, on)}
              slotProps={{ input: { 'aria-label': t`Search ${{ source: s.name }}` } }}
            />
            <SourceLogo id={s.id} size={16} />
            <Typography
              sx={{
                minWidth: 120,
                color: off.includes(s.id) ? 'text.disabled' : 'text.primary',
              }}
            >
              {s.name}
            </Typography>
            <IconAction
              label={t`Move up`}
              icon={<ArrowUp size={14} />}
              disabled={i === 0}
              onClick={() => move(i, i - 1)}
            />
            <IconAction
              label={t`Move down`}
              icon={<ArrowDown size={14} />}
              disabled={i === list.length - 1}
              onClick={() => move(i, i + 1)}
            />
          </Box>
        ))}
      </Box>
    </SettingRow>
  )
}

export { SourceOrder }

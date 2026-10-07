import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { ArrowDown, ArrowUp } from 'lucide-react'
import { useEffect, useState } from 'react'
import { SearchableSources } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SourceLogo } from '../../brand/sources/SourceLogo.tsx'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { IconAction } from '../../shell/IconAction.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// The game's sources in the player's order: the saved ids first, then the rest in catalog order.
function ordered(all: { id: string; name: string }[], saved: string) {
  const first = saved.split(',').flatMap((id) => all.filter((s) => s.id === id))
  return [...new Set([...first, ...all])]
}

function SourceOrder() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const push = useToasts((s) => s.push)
  const saved = useSettings((s) => s.games?.[game]?.sourceOrder ?? '')
  const [all, setAll] = useState<{ id: string; name: string }[]>([])
  useEffect(() => {
    SearchableSources(game)
      .then((found) => setAll(found ?? []))
      .catch(reportUnexpected)
  }, [game])
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
      push,
      t`Could not save that setting`,
    )
  }
  return (
    <SettingRow
      label={t`Preferred source`}
      description={t`When a mod is on several sites, its card installs from the first one here.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {list.map((s, i) => (
          <Box key={s.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <SourceLogo id={s.id} size={16} />
            <Typography sx={{ minWidth: 120 }}>{s.name}</Typography>
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

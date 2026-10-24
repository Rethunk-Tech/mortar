import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { HiddenLogSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { ShowLogSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// Where a hidden source's lines come from, as the Console reads it: the method (or the engine), and which object.
function useWhere() {
  const { t } = useLingui()
  return (h: HiddenLogSource) => {
    const from = h.source.startsWith('unity:')
      ? t`Engine`
      : (() => {
          const [type = '', name = ''] = h.source.split('::')
          return `${type.split('.').at(-1)}.${name}`
        })()
    const what = h.text ? '' : t`every message`
    const about = h.object ? h.object : t`every object`
    return [from, what, about].filter(Boolean).join(' · ')
  }
}

// The game's hidden log sources, each shown by a line it hid and where it comes from, with Show again.
function HiddenList({ game }: { game: string }) {
  const { t } = useLingui()
  const hidden = useSettings((s) => s.games?.[game]?.hiddenLogSources) ?? []
  const where = useWhere()
  if (hidden.length === 0) {
    return <Typography color="text.secondary">{t`None`}</Typography>
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, minWidth: 0 }}>
      {hidden.map((h) => (
        <Box
          key={`${h.source}\t${h.text}\t${h.object}`}
          sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0 }}
        >
          <Box sx={{ minWidth: 0, flex: 1 }}>
            <Typography noWrap={true} title={h.label} sx={{ fontFamily: 'monospace' }}>
              {h.text || h.label || h.source}
            </Typography>
            <Typography
              noWrap={true}
              variant="body2"
              color="text.secondary"
              title={[h.source, h.object].filter(Boolean).join('\n')}
            >
              {where(h)}
            </Typography>
          </Box>
          <Button
            size="small"
            sx={{ flexShrink: 0 }}
            aria-label={t`Show ${{ name: h.label || h.source }} again`}
            onClick={() =>
              persist(
                () => ShowLogSource(game, h.source, h.text ?? '', h.object ?? ''),
                t`Could not save that setting`,
              )
            }
          >
            {t`Show again`}
          </Button>
        </Box>
      ))}
    </Box>
  )
}

// The messages hidden from the game's Console with right-click, for every profile of the game.
export function HiddenLogSources() {
  const { t } = useLingui()
  const game = useCurrentGame()
  return (
    <SettingRow
      label={t`Hidden messages`}
      description={t`Right-click a line in the Console to hide every line from where it came from, in every profile of this game. The game stops writing them where its bridge can.`}
    >
      <HiddenList game={game} />
    </SettingRow>
  )
}

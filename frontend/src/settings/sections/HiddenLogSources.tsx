import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ShowLogSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// The messages hidden from the Console with right-click, each shown by a line it hid.
function HiddenLogSources() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const hidden = useSettings((s) => s.games?.[game]?.hiddenLogSources) ?? []
  return (
    <SettingRow
      label={t`Hidden messages`}
      description={t`Right-click a line in the Console to hide every line from where it came from. The game stops writing them where its bridge can.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {hidden.length === 0 ? <Typography color="text.secondary">{t`None`}</Typography> : null}
        {hidden.map((h) => (
          <Box
            key={`${h.source}\t${h.object}`}
            sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}
          >
            <Typography
              noWrap={true}
              title={[h.label, h.source, h.object].filter(Boolean).join('\n')}
              sx={{ maxWidth: 360, fontFamily: 'monospace' }}
            >
              {h.label || h.source}
            </Typography>
            <Button
              size="small"
              aria-label={t`Show ${{ name: h.label || h.source }} again`}
              onClick={() =>
                persist(
                  () => ShowLogSource(game, h.source, h.object ?? ''),
                  t`Could not save that setting`,
                )
              }
            >
              {t`Show again`}
            </Button>
          </Box>
        ))}
      </Box>
    </SettingRow>
  )
}

export { HiddenLogSources }
